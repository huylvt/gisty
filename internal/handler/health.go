package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/huylvt/gisty/internal/repository"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	s3Client *repository.S3
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(s3Client *repository.S3) *HealthHandler {
	return &HealthHandler{
		s3Client: s3Client,
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status" example:"ok"`
	Timestamp string `json:"timestamp" example:"2024-01-15T14:00:00Z"`
}

// Health godoc
// @Summary Health check
// @Description Check if the service is running
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse "Service is healthy"
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	c.JSON(http.StatusOK, response)
}

// DebugS3Response represents the S3 debug response
type DebugS3Response struct {
	Bucket      string `json:"bucket"`
	TestKey     string `json:"test_key"`
	ListBuckets string `json:"list_buckets"`
	HeadBucket  string `json:"head_bucket"`
	PutObject   string `json:"put_object"`
	GetObject   string `json:"get_object"`
	DeleteObject string `json:"delete_object"`
}

// DebugS3 tests S3 connectivity
func (h *HealthHandler) DebugS3(c *gin.Context) {
	if h.s3Client == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "S3 client not initialized"})
		return
	}

	ctx := context.Background()
	response := DebugS3Response{
		Bucket:  h.s3Client.BucketName,
		TestKey: fmt.Sprintf("debug/test-%d.txt", time.Now().Unix()),
	}

	// Test 1: List buckets
	listResult, err := h.s3Client.Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		response.ListBuckets = fmt.Sprintf("FAIL: %v", err)
	} else {
		var names []string
		for _, b := range listResult.Buckets {
			names = append(names, *b.Name)
		}
		response.ListBuckets = fmt.Sprintf("OK: %d buckets (%s)", len(names), strings.Join(names, ", "))
	}

	// Test 2: Head bucket
	_, err = h.s3Client.Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(h.s3Client.BucketName),
	})
	if err != nil {
		response.HeadBucket = fmt.Sprintf("FAIL: %v", err)
	} else {
		response.HeadBucket = "OK"
	}

	// Test 3: Put object (simple)
	testContent := "Hello from Gisty debug endpoint!"
	_, err = h.s3Client.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(h.s3Client.BucketName),
		Key:         aws.String(response.TestKey),
		Body:        strings.NewReader(testContent),
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		response.PutObject = fmt.Sprintf("FAIL: %v", err)
		c.JSON(http.StatusOK, response)
		return
	}
	response.PutObject = "OK"

	// Test 3b: Put object with gzip encoding and metadata (like SaveContent)
	testKeyGzip := response.TestKey + ".gz"
	_, err = h.s3Client.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:          aws.String(h.s3Client.BucketName),
		Key:             aws.String(testKeyGzip),
		Body:            strings.NewReader(testContent),
		ContentType:     aws.String("text/plain"),
		ContentEncoding: aws.String("gzip"),
		Metadata: map[string]string{
			"original-size": "100",
		},
	})
	if err != nil {
		response.PutObject = fmt.Sprintf("OK (simple), FAIL with gzip+metadata: %v", err)
	} else {
		response.PutObject = "OK (simple + gzip+metadata)"
		// Cleanup gzip test file
		_, _ = h.s3Client.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(h.s3Client.BucketName),
			Key:    aws.String(testKeyGzip),
		})
	}

	// Test 4: Get object
	_, err = h.s3Client.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(h.s3Client.BucketName),
		Key:    aws.String(response.TestKey),
	})
	if err != nil {
		response.GetObject = fmt.Sprintf("FAIL: %v", err)
	} else {
		response.GetObject = "OK"
	}

	// Test 5: Delete object
	_, err = h.s3Client.Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(h.s3Client.BucketName),
		Key:    aws.String(response.TestKey),
	})
	if err != nil {
		response.DeleteObject = fmt.Sprintf("FAIL: %v", err)
	} else {
		response.DeleteObject = "OK"
	}

	c.JSON(http.StatusOK, response)
}