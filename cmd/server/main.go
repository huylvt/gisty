package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/huylvt/gisty/internal/config"
	"github.com/huylvt/gisty/internal/handler"
	"github.com/huylvt/gisty/internal/middleware"
	"github.com/huylvt/gisty/internal/repository"
	"github.com/huylvt/gisty/internal/service"
	"github.com/huylvt/gisty/internal/worker"

	_ "github.com/huylvt/gisty/docs" // Swagger docs
)

// @title Gisty API
// @version 1.0
// @description Fast snippet sharing platform - API documentation
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/huylvt/gisty
// @contact.email support@gisty.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @schemes http https

func main() {
	fmt.Println("Gisty Server")
	fmt.Printf("Version: %s\n", "0.1.0")

	if len(os.Args) > 1 && os.Args[1] == "--help" {
		printHelp()
		return
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Environment: %s", cfg.Server.Env)

	// Connect to MongoDB
	ctx := context.Background()
	mongoDB, err := repository.NewMongoClient(ctx, cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB")

	// Connect to Redis
	redisClient, err := repository.NewRedisClient(ctx, cfg.Redis.URI)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")

	// Connect to S3
	s3Client, err := repository.NewS3Client(ctx, repository.S3Config{
		BucketName:      cfg.S3.BucketName,
		Region:          cfg.S3.Region,
		AccessKeyID:     cfg.S3.AccessKeyID,
		SecretAccessKey: cfg.S3.SecretAccessKey,
		Endpoint:        cfg.S3.Endpoint,
	})
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}
	if err := s3Client.EnsureBucketExists(ctx); err != nil {
		log.Fatalf("Failed to verify S3 bucket '%s': %v", cfg.S3.BucketName, err)
	}
	log.Printf("Connected to S3, bucket '%s' verified", cfg.S3.BucketName)

	// Initialize KGS (Key Generation Service)
	kgs, err := service.NewKGS(mongoDB.Database)
	if err != nil {
		log.Fatalf("Failed to initialize KGS: %v", err)
	}

	// Start KGS background worker with cancellable context
	kgsCtx, kgsCancel := context.WithCancel(context.Background())
	go kgs.StartReplenishWorker(kgsCtx, service.DefaultWorkerConfig())

	// Initialize services
	storageService := service.NewStorage(s3Client)
	cacheService := service.NewCache(redisClient)

	// Initialize repositories
	pasteRepo, err := repository.NewPasteRepository(mongoDB.Database)
	if err != nil {
		log.Fatalf("Failed to initialize paste repository: %v", err)
	}

	// Initialize paste service
	baseURL := fmt.Sprintf("http://localhost:%s", cfg.Server.Port)
	if cfg.Server.Env == "production" {
		baseURL = cfg.Server.BaseURL
	}
	pasteService := service.NewPasteService(kgs, storageService, cacheService, pasteRepo, baseURL)

	// Initialize and start cleanup worker
	cleanupInterval, err := time.ParseDuration(cfg.Cleanup.Interval)
	if err != nil {
		log.Printf("Invalid cleanup interval '%s', using default 5m", cfg.Cleanup.Interval)
		cleanupInterval = 5 * time.Minute
	}
	cleanupWorker := worker.NewCleanupWorker(pasteRepo, storageService, cacheService, &worker.CleanupWorkerConfig{
		Interval:  cleanupInterval,
		BatchSize: cfg.Cleanup.BatchSize,
	})
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	go cleanupWorker.Start(cleanupCtx)

	// Initialize rate limiter
	rateLimiter := middleware.NewRateLimiter(&middleware.RateLimitConfig{
		RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
		Enabled:           cfg.RateLimit.Enabled,
	})
	if cfg.RateLimit.Enabled {
		log.Printf("Rate limiting enabled: %d requests/minute", cfg.RateLimit.RequestsPerMinute)
	}

	// Initialize handlers
	pasteHandler := handler.NewPasteHandler(pasteService)

	// Setup router with dependencies
	deps := &handler.RouterDeps{
		PasteHandler: pasteHandler,
		RateLimiter:  rateLimiter,
	}
	router := handler.NewRouter(cfg, deps)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Stop KGS worker
	kgsCancel()

	// Stop Cleanup worker
	cleanupCancel()

	// Give outstanding requests 5 seconds to complete
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	} else {
		log.Println("Redis connection closed")
	}

	// Close MongoDB connection
	if err := mongoDB.Close(shutdownCtx); err != nil {
		log.Printf("Error closing MongoDB connection: %v", err)
	} else {
		log.Println("MongoDB connection closed")
	}

	log.Println("Server exited gracefully")
}

func printHelp() {
	fmt.Print(`Gisty - Fast snippet sharing platform

Usage:
  gisty [flags]

Flags:
  --help    Show this help message

Environment Variables:
  PORT                 Server port (default: 8080)
  ENV                  Environment (development/production)
  MONGO_URI            MongoDB connection string
  REDIS_URI            Redis connection string
  S3_BUCKET_NAME       S3 bucket name
  S3_REGION            S3 region
  S3_ACCESS_KEY_ID     S3 access key
  S3_SECRET_ACCESS_KEY S3 secret key
  S3_ENDPOINT          S3 endpoint URL
  CLEANUP_INTERVAL     Cleanup worker interval (default: 5m)
  CLEANUP_BATCH_SIZE   Cleanup batch size (default: 100)
  RATE_LIMIT_REQUESTS_PER_MINUTE  Rate limit per IP (default: 5)
  RATE_LIMIT_ENABLED   Enable rate limiting (default: true)
`)
}