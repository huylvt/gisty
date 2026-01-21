package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/huylvt/gisty/internal/repository"
)

const (
	// S3KeyPrefix is the prefix for all paste content in S3
	S3KeyPrefix = "gisty/"
	// S3KeySuffix is the suffix for gzipped content
	S3KeySuffix = ".gz"
)

var (
	// ErrContentNotFound is returned when the content does not exist
	ErrContentNotFound = errors.New("storage: content not found")
	// ErrAccessDenied is returned when access to the content is denied
	ErrAccessDenied = errors.New("storage: access denied")
)

// Storage handles content storage operations
type Storage struct {
	s3Client   *repository.S3
	bucketName string
}

// NewStorage creates a new Storage service
func NewStorage(s3Client *repository.S3) *Storage {
	log.Printf("[Storage] Initialized with bucket: %s", s3Client.BucketName)
	return &Storage{
		s3Client:   s3Client,
		bucketName: s3Client.BucketName,
	}
}

// SaveContent saves content to S3 with gzip compression
func (s *Storage) SaveContent(ctx context.Context, shortID, content string) error {
	// Compress content with gzip
	compressed, err := compressContent(content)
	if err != nil {
		log.Printf("[Storage.SaveContent] Compression failed: %v", err)
		return fmt.Errorf("storage: failed to compress content: %w", err)
	}

	key := s.buildKey(shortID)
	log.Printf("[Storage.SaveContent] Uploading to bucket=%s, key=%s, size=%d bytes (compressed from %d)",
		s.bucketName, key, len(compressed), len(content))

	// Note: ContentEncoding and Metadata headers removed due to Ceph S3 compatibility issues
	// Content is still gzip compressed, we handle decompression on read
	_, err = s.s3Client.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(compressed),
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		log.Printf("[Storage.SaveContent] PutObject failed: bucket=%s, key=%s, error=%v", s.bucketName, key, err)
		return fmt.Errorf("storage: failed to upload content: %w", err)
	}

	log.Printf("[Storage.SaveContent] Upload successful: %s", key)
	return nil
}

// GetContent retrieves and decompresses content from S3
func (s *Storage) GetContent(ctx context.Context, shortID string) (string, error) {
	key := s.buildKey(shortID)

	result, err := s.s3Client.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", s.handleS3Error(err)
	}
	defer result.Body.Close()

	// Read compressed data
	compressed, err := io.ReadAll(result.Body)
	if err != nil {
		return "", fmt.Errorf("storage: failed to read content: %w", err)
	}

	// Decompress content
	content, err := decompressContent(compressed)
	if err != nil {
		return "", fmt.Errorf("storage: failed to decompress content: %w", err)
	}

	return content, nil
}

// DeleteContent removes content from S3
func (s *Storage) DeleteContent(ctx context.Context, shortID string) error {
	key := s.buildKey(shortID)

	_, err := s.s3Client.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: failed to delete content: %w", err)
	}

	return nil
}

// ContentExists checks if content exists in S3
func (s *Storage) ContentExists(ctx context.Context, shortID string) (bool, error) {
	key := s.buildKey(shortID)

	_, err := s.s3Client.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// buildKey constructs the S3 key for a given shortID
func (s *Storage) buildKey(shortID string) string {
	return S3KeyPrefix + shortID + S3KeySuffix
}

// handleS3Error converts S3 errors to storage errors
func (s *Storage) handleS3Error(err error) error {
	var notFound *types.NoSuchKey
	if errors.As(err, &notFound) {
		return ErrContentNotFound
	}

	var notFoundType *types.NotFound
	if errors.As(err, &notFoundType) {
		return ErrContentNotFound
	}

	// Check for access denied
	var accessDenied interface{ ErrorCode() string }
	if errors.As(err, &accessDenied) && accessDenied.ErrorCode() == "AccessDenied" {
		return ErrAccessDenied
	}

	return fmt.Errorf("storage: S3 error: %w", err)
}

// compressContent compresses content using gzip
func compressContent(content string) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)

	if _, err := gz.Write([]byte(content)); err != nil {
		return nil, err
	}

	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompressContent decompresses gzipped content
func decompressContent(compressed []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return "", err
	}
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(decompressed), nil
}