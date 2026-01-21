package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/huylvt/gisty/internal/config"
	"github.com/huylvt/gisty/internal/handler"
	"github.com/huylvt/gisty/internal/middleware"
	"github.com/huylvt/gisty/internal/repository"
	"github.com/huylvt/gisty/internal/service"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

// TestEnv holds all test environment resources
type TestEnv struct {
	Router       *gin.Engine
	Server       *httptest.Server
	MongoC       *mongodb.MongoDBContainer
	RedisC       *redis.RedisContainer
	MinioC       *minio.MinioContainer
	PasteService *service.PasteService
	Cleanup      func()
}

// terminateContainer safely terminates a container, ignoring errors during cleanup
func terminateContainer(ctx context.Context, container testcontainers.Container) {
	if container != nil {
		_ = container.Terminate(ctx)
	}
}

// SetupTestEnv creates a complete test environment with all dependencies
func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()
	ctx := context.Background()

	// Start MongoDB container
	mongoC, err := mongodb.Run(ctx, "mongo:7.0")
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	mongoURI, err := mongoC.ConnectionString(ctx)
	if err != nil {
		terminateContainer(ctx, mongoC)
		t.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	// Start Redis container
	redisC, err := redis.Run(ctx, "redis:7-alpine")
	if err != nil {
		terminateContainer(ctx, mongoC)
		t.Fatalf("Failed to start Redis container: %v", err)
	}

	redisURI, err := redisC.ConnectionString(ctx)
	if err != nil {
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		t.Fatalf("Failed to get Redis connection string: %v", err)
	}

	// Start MinIO container
	minioC, err := minio.Run(ctx, "minio/minio:latest",
		minio.WithUsername("minioadmin"),
		minio.WithPassword("minioadmin"),
	)
	if err != nil {
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		t.Fatalf("Failed to start MinIO container: %v", err)
	}

	minioEndpoint, err := minioC.ConnectionString(ctx)
	if err != nil {
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to get MinIO endpoint: %v", err)
	}

	// Connect to MongoDB
	mongoDB, err := repository.NewMongoClient(ctx, mongoURI, "gisty_integration_test")
	if err != nil {
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Connect to Redis
	redisClient, err := repository.NewRedisClient(ctx, redisURI)
	if err != nil {
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Connect to S3 (MinIO)
	s3Client, err := repository.NewS3Client(ctx, repository.S3Config{
		BucketName:      "gisty-integration-test",
		Region:          "us-east-1",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Endpoint:        "http://" + minioEndpoint,
	})
	if err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to create S3 client: %v", err)
	}

	if err := s3Client.EnsureBucketExists(ctx); err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to create S3 bucket: %v", err)
	}

	// Initialize services
	kgs, err := service.NewKGS(mongoDB.Database)
	if err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to initialize KGS: %v", err)
	}

	// Generate initial keys
	if _, err := kgs.GenerateKeys(ctx, 100); err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to generate keys: %v", err)
	}

	storageService := service.NewStorage(s3Client)
	cacheService := service.NewCache(redisClient)

	pasteRepo, err := repository.NewPasteRepository(mongoDB.Database)
	if err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to initialize paste repository: %v", err)
	}

	pasteService := service.NewPasteService(kgs, storageService, cacheService, pasteRepo, "http://localhost:8080")

	// Initialize rate limiter (disabled by default for tests, enabled in specific tests)
	rateLimiter := middleware.NewRateLimiter(&middleware.RateLimitConfig{
		RequestsPerMinute: 5,
		Enabled:           false, // Disabled by default
	})

	// Initialize handlers
	pasteHandler := handler.NewPasteHandler(pasteService)

	// Setup router
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: "8080",
			Env:  "test",
		},
	}
	deps := &handler.RouterDeps{
		PasteHandler: pasteHandler,
		RateLimiter:  rateLimiter,
		S3Client:     s3Client,
	}
	router := handler.NewRouter(cfg, deps)

	// Create test server
	server := httptest.NewServer(router)

	cleanup := func() {
		server.Close()
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
	}

	return &TestEnv{
		Router:       router,
		Server:       server,
		MongoC:       mongoC,
		RedisC:       redisC,
		MinioC:       minioC,
		PasteService: pasteService,
		Cleanup:      cleanup,
	}
}

// SetupTestEnvWithRateLimit creates a test environment with rate limiting enabled
func SetupTestEnvWithRateLimit(t *testing.T, requestsPerMinute int) *TestEnv {
	t.Helper()
	ctx := context.Background()

	// Start MongoDB container
	mongoC, err := mongodb.Run(ctx, "mongo:7.0")
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	mongoURI, err := mongoC.ConnectionString(ctx)
	if err != nil {
		terminateContainer(ctx, mongoC)
		t.Fatalf("Failed to get MongoDB connection string: %v", err)
	}

	// Start Redis container
	redisC, err := redis.Run(ctx, "redis:7-alpine")
	if err != nil {
		terminateContainer(ctx, mongoC)
		t.Fatalf("Failed to start Redis container: %v", err)
	}

	redisURI, err := redisC.ConnectionString(ctx)
	if err != nil {
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		t.Fatalf("Failed to get Redis connection string: %v", err)
	}

	// Start MinIO container
	minioC, err := minio.Run(ctx, "minio/minio:latest",
		minio.WithUsername("minioadmin"),
		minio.WithPassword("minioadmin"),
	)
	if err != nil {
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		t.Fatalf("Failed to start MinIO container: %v", err)
	}

	minioEndpoint, err := minioC.ConnectionString(ctx)
	if err != nil {
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to get MinIO endpoint: %v", err)
	}

	// Connect to MongoDB
	mongoDB, err := repository.NewMongoClient(ctx, mongoURI, "gisty_integration_test")
	if err != nil {
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Connect to Redis
	redisClient, err := repository.NewRedisClient(ctx, redisURI)
	if err != nil {
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Connect to S3 (MinIO)
	s3Client, err := repository.NewS3Client(ctx, repository.S3Config{
		BucketName:      "gisty-integration-test",
		Region:          "us-east-1",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Endpoint:        "http://" + minioEndpoint,
	})
	if err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to create S3 client: %v", err)
	}

	if err := s3Client.EnsureBucketExists(ctx); err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to create S3 bucket: %v", err)
	}

	// Initialize services
	kgs, err := service.NewKGS(mongoDB.Database)
	if err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to initialize KGS: %v", err)
	}

	if _, err := kgs.GenerateKeys(ctx, 100); err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to generate keys: %v", err)
	}

	storageService := service.NewStorage(s3Client)
	cacheService := service.NewCache(redisClient)

	pasteRepo, err := repository.NewPasteRepository(mongoDB.Database)
	if err != nil {
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
		t.Fatalf("Failed to initialize paste repository: %v", err)
	}

	pasteService := service.NewPasteService(kgs, storageService, cacheService, pasteRepo, "http://localhost:8080")

	// Initialize rate limiter with specified limit
	rateLimiter := middleware.NewRateLimiter(&middleware.RateLimitConfig{
		RequestsPerMinute: requestsPerMinute,
		Enabled:           true,
	})

	// Initialize handlers
	pasteHandler := handler.NewPasteHandler(pasteService)

	// Setup router
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: "8080",
			Env:  "test",
		},
	}
	deps := &handler.RouterDeps{
		PasteHandler: pasteHandler,
		RateLimiter:  rateLimiter,
		S3Client:     s3Client,
	}
	router := handler.NewRouter(cfg, deps)

	server := httptest.NewServer(router)

	cleanup := func() {
		server.Close()
		redisClient.Close()
		mongoDB.Close(ctx)
		terminateContainer(ctx, mongoC)
		terminateContainer(ctx, redisC)
		terminateContainer(ctx, minioC)
	}

	return &TestEnv{
		Router:       router,
		Server:       server,
		MongoC:       mongoC,
		RedisC:       redisC,
		MinioC:       minioC,
		PasteService: pasteService,
		Cleanup:      cleanup,
	}
}

// Helper functions for making HTTP requests

// CreatePasteRequest represents the request body for creating a paste
type CreatePasteRequest struct {
	Content    string `json:"content"`
	SyntaxType string `json:"syntax_type,omitempty"`
	ExpiresIn  string `json:"expires_in,omitempty"`
	IsPrivate  bool   `json:"is_private,omitempty"`
}

// CreatePasteResponse represents the response after creating a paste
type CreatePasteResponse struct {
	ShortID   string `json:"short_id"`
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at,omitempty"`
}

// GetPasteResponse represents the response when retrieving a paste
type GetPasteResponse struct {
	ShortID    string `json:"short_id"`
	Content    string `json:"content"`
	SyntaxType string `json:"syntax_type"`
	CreatedAt  string `json:"created_at"`
	ExpiresAt  string `json:"expires_at,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error      string `json:"error"`
	MaxSize    string `json:"max_size,omitempty"`
	RetryAfter int64  `json:"retry_after,omitempty"`
}

// DoCreatePaste sends a POST request to create a paste
func DoCreatePaste(t *testing.T, serverURL string, req CreatePasteRequest) (*http.Response, []byte) {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	resp, err := http.Post(serverURL+"/api/v1/pastes", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	return resp, respBody
}

// DoGetPaste sends a GET request to retrieve a paste
func DoGetPaste(t *testing.T, serverURL, shortID string) (*http.Response, []byte) {
	t.Helper()

	resp, err := http.Get(serverURL + "/api/v1/pastes/" + shortID)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	return resp, respBody
}

// DoDeletePaste sends a DELETE request to delete a paste
func DoDeletePaste(t *testing.T, serverURL, shortID string) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, serverURL+"/api/v1/pastes/"+shortID, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	return resp, respBody
}

// DoShortURL sends a GET request to the short URL endpoint
func DoShortURL(t *testing.T, serverURL, shortID string, acceptJSON bool) (*http.Response, []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, serverURL+"/"+shortID, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	if acceptJSON {
		req.Header.Set("Accept", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	return resp, respBody
}

// WaitForExpiration waits for a paste to expire
func WaitForExpiration(d time.Duration) {
	time.Sleep(d + 100*time.Millisecond)
}

// SkipIfNoDocker skips the test if Docker is not available
func SkipIfNoDocker(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to ping Docker by creating a simple container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "alpine:latest",
		},
		Started: false,
	})
	if err != nil {
		t.Skipf("Docker not available, skipping integration test: %v", err)
	}
	// Clean up the test container
	if container != nil {
		terminateContainer(ctx, container)
	}
}

// ParseCreateResponse parses a create paste response
func ParseCreateResponse(t *testing.T, body []byte) CreatePasteResponse {
	t.Helper()
	var resp CreatePasteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to parse response: %v, body: %s", err, string(body))
	}
	return resp
}

// ParseGetResponse parses a get paste response
func ParseGetResponse(t *testing.T, body []byte) GetPasteResponse {
	t.Helper()
	var resp GetPasteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to parse response: %v, body: %s", err, string(body))
	}
	return resp
}

// ParseErrorResponse parses an error response
func ParseErrorResponse(t *testing.T, body []byte) ErrorResponse {
	t.Helper()
	var resp ErrorResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Failed to parse error response: %v, body: %s", err, string(body))
	}
	return resp
}

// AssertStatusCode checks if the response status code matches expected
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Errorf("Expected status code %d, got %d", expected, resp.StatusCode)
	}
}

// Printf is a helper for test logging
func Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
