package repository

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config holds S3 client configuration
type S3Config struct {
	BucketName      string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // Optional: for MinIO or S3-compatible storage
}

// S3 wraps the S3 client
type S3 struct {
	Client     *s3.Client
	BucketName string
}

// NewS3Client creates a new S3 connection
func NewS3Client(ctx context.Context, cfg S3Config) (*S3, error) {
	log.Printf("[S3] Initializing client: endpoint=%s, region=%s, bucket=%s, accessKey=%s...%s",
		cfg.Endpoint, cfg.Region, cfg.BucketName,
		cfg.AccessKeyID[:4], cfg.AccessKeyID[len(cfg.AccessKeyID)-4:])

	// Create custom credentials provider
	credProvider := credentials.NewStaticCredentialsProvider(
		cfg.AccessKeyID,
		cfg.SecretAccessKey,
		"",
	)

	// Build AWS config options
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credProvider),
	}

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		log.Printf("[S3] Failed to load AWS config: %v", err)
		return nil, err
	}

	// Build S3 client options
	s3Opts := []func(*s3.Options){}

	// Add custom endpoint if specified (for MinIO or other S3-compatible storage)
	if cfg.Endpoint != "" {
		log.Printf("[S3] Using custom endpoint: %s (PathStyle=true)", cfg.Endpoint)
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true // Required for MinIO
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	log.Printf("[S3] Client initialized successfully")
	return &S3{
		Client:     client,
		BucketName: cfg.BucketName,
	}, nil
}

// HealthCheck verifies the S3 connection by checking if the bucket exists
func (s *S3) HealthCheck(ctx context.Context) error {
	_, err := s.Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.BucketName),
	})
	return err
}

// EnsureBucketExists creates the bucket if it doesn't exist
func (s *S3) EnsureBucketExists(ctx context.Context) error {
	// Check if bucket exists
	_, err := s.Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.BucketName),
	})
	if err == nil {
		// Bucket already exists
		return nil
	}

	// Bucket doesn't exist, create it
	_, err = s.Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.BucketName),
	})
	if err != nil {
		return err
	}

	return nil
}