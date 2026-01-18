package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/huylvt/gisty/internal/model"
	"github.com/huylvt/gisty/internal/repository"
	"github.com/huylvt/gisty/internal/service"
)

func setupCleanupTest(t *testing.T) (*CleanupWorker, *repository.PasteRepository, *service.Storage, *service.Cache, func()) {
	// Setup MongoDB
	mongoClient, err := repository.NewMongoClient(context.Background(), "mongodb://gisty:gisty123@localhost:27017/gisty_test?authSource=admin", "gisty_test")
	if err != nil {
		t.Skipf("Skipping test, MongoDB not available: %v", err)
	}

	// Setup Redis
	redisClient, err := repository.NewRedisClient(context.Background(), "redis://localhost:6379")
	if err != nil {
		mongoClient.Close(context.Background())
		t.Skipf("Skipping test, Redis not available: %v", err)
	}

	// Setup S3
	s3Client, err := repository.NewS3Client(context.Background(), repository.S3Config{
		BucketName:      "gisty-test",
		Region:          "us-east-1",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Endpoint:        "http://localhost:9000",
	})
	if err != nil {
		redisClient.Close()
		mongoClient.Close(context.Background())
		t.Skipf("Skipping test, S3 not available: %v", err)
	}
	_ = s3Client.EnsureBucketExists(context.Background())

	// Create repositories and services
	pasteRepo, err := repository.NewPasteRepository(mongoClient.Database)
	if err != nil {
		redisClient.Close()
		mongoClient.Close(context.Background())
		t.Fatalf("Failed to create paste repository: %v", err)
	}

	storage := service.NewStorage(s3Client)
	cache := service.NewCache(redisClient)

	worker := NewCleanupWorker(pasteRepo, storage, cache, &CleanupWorkerConfig{
		Interval:  100 * time.Millisecond, // Short interval for testing
		BatchSize: 10,
	})

	cleanup := func() {
		// Clean up test data
		_ = mongoClient.Database.Collection("pastes").Drop(context.Background())
		redisClient.Close()
		mongoClient.Close(context.Background())
	}

	return worker, pasteRepo, storage, cache, cleanup
}

func TestCleanupWorker_CleansExpiredPastes(t *testing.T) {
	worker, pasteRepo, storage, cache, cleanup := setupCleanupTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create expired paste directly
	expiredTime := time.Now().Add(-1 * time.Hour)
	expiredPaste := &model.Paste{
		ShortID:    "expired1",
		ContentKey: "pastes/expired1",
		ExpiresAt:  &expiredTime,
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		SyntaxType: "text",
	}

	// Save to MongoDB
	if err := pasteRepo.Create(ctx, expiredPaste); err != nil {
		t.Fatalf("Failed to create expired paste: %v", err)
	}

	// Save to S3
	if err := storage.SaveContent(ctx, "expired1", "test content"); err != nil {
		t.Fatalf("Failed to save content: %v", err)
	}

	// Save to cache
	if err := cache.Set(ctx, "expired1", "test content", time.Hour); err != nil {
		t.Fatalf("Failed to cache content: %v", err)
	}

	// Verify paste exists
	_, err := pasteRepo.GetByShortID(ctx, "expired1")
	if err != nil {
		t.Fatalf("Paste should exist before cleanup: %v", err)
	}

	// Run cleanup
	worker.runCleanup(ctx)

	// Verify paste is deleted from MongoDB
	_, err = pasteRepo.GetByShortID(ctx, "expired1")
	if err != repository.ErrPasteNotFound {
		t.Errorf("Paste should be deleted from MongoDB, got error: %v", err)
	}

	// Verify content is deleted from S3
	_, err = storage.GetContent(ctx, "expired1")
	if err == nil {
		t.Error("Content should be deleted from S3")
	}

	// Verify content is deleted from cache
	_, found, _ := cache.Get(ctx, "expired1")
	if found {
		t.Error("Content should be deleted from cache")
	}
}

func TestCleanupWorker_DoesNotCleanNonExpiredPastes(t *testing.T) {
	worker, pasteRepo, storage, _, cleanup := setupCleanupTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create non-expired paste
	futureTime := time.Now().Add(1 * time.Hour)
	nonExpiredPaste := &model.Paste{
		ShortID:    "nonexpired1",
		ContentKey: "pastes/nonexpired1",
		ExpiresAt:  &futureTime,
		CreatedAt:  time.Now(),
		SyntaxType: "text",
	}

	if err := pasteRepo.Create(ctx, nonExpiredPaste); err != nil {
		t.Fatalf("Failed to create paste: %v", err)
	}

	if err := storage.SaveContent(ctx, "nonexpired1", "test content"); err != nil {
		t.Fatalf("Failed to save content: %v", err)
	}

	// Run cleanup
	worker.runCleanup(ctx)

	// Verify paste still exists
	_, err := pasteRepo.GetByShortID(ctx, "nonexpired1")
	if err != nil {
		t.Errorf("Non-expired paste should not be deleted: %v", err)
	}

	// Verify content still exists
	content, err := storage.GetContent(ctx, "nonexpired1")
	if err != nil || content != "test content" {
		t.Errorf("Content should still exist: %v", err)
	}
}

func TestCleanupWorker_DoesNotCleanPastesWithoutExpiration(t *testing.T) {
	worker, pasteRepo, storage, _, cleanup := setupCleanupTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create paste without expiration
	neverExpiresPaste := &model.Paste{
		ShortID:    "neverexpires1",
		ContentKey: "pastes/neverexpires1",
		ExpiresAt:  nil, // No expiration
		CreatedAt:  time.Now(),
		SyntaxType: "text",
	}

	if err := pasteRepo.Create(ctx, neverExpiresPaste); err != nil {
		t.Fatalf("Failed to create paste: %v", err)
	}

	if err := storage.SaveContent(ctx, "neverexpires1", "test content"); err != nil {
		t.Fatalf("Failed to save content: %v", err)
	}

	// Run cleanup
	worker.runCleanup(ctx)

	// Verify paste still exists
	_, err := pasteRepo.GetByShortID(ctx, "neverexpires1")
	if err != nil {
		t.Errorf("Paste without expiration should not be deleted: %v", err)
	}
}

func TestCleanupWorker_BatchProcessing(t *testing.T) {
	worker, pasteRepo, storage, _, cleanup := setupCleanupTest(t)
	defer cleanup()

	ctx := context.Background()

	// Get initial count of expired pastes (from other tests running in parallel)
	initialCount, err := pasteRepo.CountExpired(ctx)
	if err != nil {
		t.Fatalf("Failed to get initial count: %v", err)
	}

	// Create multiple expired pastes
	expiredTime := time.Now().Add(-1 * time.Hour)
	numPastes := 25 // More than batch size (10)
	for i := 0; i < numPastes; i++ {
		shortID := fmt.Sprintf("batch%02d", i)
		paste := &model.Paste{
			ShortID:    shortID,
			ContentKey: "pastes/" + shortID,
			ExpiresAt:  &expiredTime,
			CreatedAt:  time.Now().Add(-2 * time.Hour),
			SyntaxType: "text",
		}

		if err := pasteRepo.Create(ctx, paste); err != nil {
			t.Fatalf("Failed to create paste %s: %v", shortID, err)
		}

		if err := storage.SaveContent(ctx, shortID, "content for "+shortID); err != nil {
			t.Fatalf("Failed to save content for %s: %v", shortID, err)
		}
	}

	// Verify count before cleanup - should be at least our created pastes
	count, err := pasteRepo.CountExpired(ctx)
	if err != nil {
		t.Fatalf("Failed to count expired: %v", err)
	}
	if count < int64(numPastes) {
		t.Errorf("Expected at least %d expired pastes, got %d", numPastes, count)
	}

	// Run cleanup
	worker.runCleanup(ctx)

	// Verify all expired pastes are deleted
	count, err = pasteRepo.CountExpired(ctx)
	if err != nil {
		t.Fatalf("Failed to count expired after cleanup: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 expired pastes after cleanup, got %d", count)
	}

	// Verify our specific pastes are deleted
	for i := 0; i < numPastes; i++ {
		shortID := fmt.Sprintf("batch%02d", i)
		_, err := pasteRepo.GetByShortID(ctx, shortID)
		if err != repository.ErrPasteNotFound {
			t.Errorf("Paste %s should be deleted", shortID)
		}
	}

	_ = initialCount // used for documentation, actual count may include other tests' pastes
}

func TestCleanupWorker_StartStop(t *testing.T) {
	worker, _, _, _, cleanup := setupCleanupTest(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	// Start worker in goroutine
	done := make(chan struct{})
	go func() {
		worker.Start(ctx)
		close(done)
	}()

	// Let it run for a bit
	time.Sleep(50 * time.Millisecond)

	// Cancel context to stop
	cancel()

	// Wait for worker to stop
	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Worker did not stop within timeout")
	}
}