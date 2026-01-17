package worker

import (
	"context"
	"log"
	"time"

	"github.com/huylvt/gisty/internal/repository"
	"github.com/huylvt/gisty/internal/service"
)

const (
	// DefaultCleanupInterval is the default interval between cleanup runs
	DefaultCleanupInterval = 5 * time.Minute
	// DefaultCleanupBatchSize is the default batch size for cleanup operations
	DefaultCleanupBatchSize = 100
)

// CleanupWorkerConfig holds configuration for the cleanup worker
type CleanupWorkerConfig struct {
	Interval  time.Duration
	BatchSize int64
}

// CleanupWorker handles periodic cleanup of expired pastes
type CleanupWorker struct {
	pasteRepo *repository.PasteRepository
	storage   *service.Storage
	cache     *service.Cache
	config    CleanupWorkerConfig
	stopCh    chan struct{}
	doneCh    chan struct{}
}

// NewCleanupWorker creates a new CleanupWorker
func NewCleanupWorker(
	pasteRepo *repository.PasteRepository,
	storage *service.Storage,
	cache *service.Cache,
	config *CleanupWorkerConfig,
) *CleanupWorker {
	cfg := CleanupWorkerConfig{
		Interval:  DefaultCleanupInterval,
		BatchSize: DefaultCleanupBatchSize,
	}

	if config != nil {
		if config.Interval > 0 {
			cfg.Interval = config.Interval
		}
		if config.BatchSize > 0 {
			cfg.BatchSize = config.BatchSize
		}
	}

	return &CleanupWorker{
		pasteRepo: pasteRepo,
		storage:   storage,
		cache:     cache,
		config:    cfg,
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

// Start begins the cleanup worker
func (w *CleanupWorker) Start(ctx context.Context) {
	log.Printf("Cleanup Worker started (interval: %v, batch_size: %d)", w.config.Interval, w.config.BatchSize)

	// Run initial cleanup
	w.runCleanup(ctx)

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Cleanup Worker stopped (context cancelled)")
			close(w.doneCh)
			return
		case <-w.stopCh:
			log.Println("Cleanup Worker stopped")
			close(w.doneCh)
			return
		case <-ticker.C:
			w.runCleanup(ctx)
		}
	}
}

// Stop gracefully stops the cleanup worker
func (w *CleanupWorker) Stop() {
	close(w.stopCh)
	<-w.doneCh
}

// runCleanup performs one cleanup cycle
func (w *CleanupWorker) runCleanup(ctx context.Context) {
	totalCleaned := int64(0)

	for {
		// Get a batch of expired pastes
		expiredPastes, err := w.pasteRepo.GetExpiredBatch(ctx, w.config.BatchSize)
		if err != nil {
			log.Printf("Cleanup Worker: error fetching expired pastes: %v", err)
			return
		}

		if len(expiredPastes) == 0 {
			break
		}

		// Collect short IDs for batch deletion
		shortIDs := make([]string, len(expiredPastes))
		for i, paste := range expiredPastes {
			shortIDs[i] = paste.ShortID
		}

		// Delete from cache (best effort, ignore errors)
		for _, shortID := range shortIDs {
			_ = w.cache.Delete(ctx, shortID)
		}

		// Delete from S3 (best effort, ignore errors)
		for _, shortID := range shortIDs {
			_ = w.storage.DeleteContent(ctx, shortID)
		}

		// Delete from MongoDB
		deletedCount, err := w.pasteRepo.DeleteMany(ctx, shortIDs)
		if err != nil {
			log.Printf("Cleanup Worker: error deleting from MongoDB: %v", err)
			return
		}

		totalCleaned += deletedCount

		// If we got fewer than batch size, we're done
		if int64(len(expiredPastes)) < w.config.BatchSize {
			break
		}
	}

	if totalCleaned > 0 {
		log.Printf("Cleanup Worker: cleaned up %d expired pastes", totalCleaned)
	}
}