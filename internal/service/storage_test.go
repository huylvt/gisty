package service

import (
	"context"
	"strings"
	"testing"

	"github.com/huylvt/gisty/internal/repository"
)

func setupTestStorage(t *testing.T) (*Storage, func()) {
	ctx := context.Background()

	// Connect to MinIO
	s3Client, err := repository.NewS3Client(ctx, repository.S3Config{
		BucketName:      "gisty-test",
		Region:          "us-east-1",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Endpoint:        "http://localhost:9000",
	})
	if err != nil {
		t.Skipf("MinIO not available: %v", err)
	}

	// Create test bucket if not exists (ignore error if already exists)
	storage := NewStorage(s3Client)

	cleanup := func() {
		// Clean up test data - list and delete all objects with test prefix
		// For simplicity, we just leave it as is since tests use unique IDs
	}

	return storage, cleanup
}

func TestStorage_SaveAndGetContent(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test001"
	content := "Hello, World! This is a test content."

	// Save content
	err := storage.SaveContent(ctx, shortID, content)
	if err != nil {
		t.Fatalf("SaveContent() error = %v", err)
	}

	// Get content
	retrieved, err := storage.GetContent(ctx, shortID)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}

	if retrieved != content {
		t.Errorf("GetContent() = %q, want %q", retrieved, content)
	}

	// Cleanup
	storage.DeleteContent(ctx, shortID)
}

func TestStorage_SaveAndGetContent_LargeContent(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test002"

	// Create large content (100KB)
	content := strings.Repeat("This is a test line of content.\n", 3000)

	// Save content
	err := storage.SaveContent(ctx, shortID, content)
	if err != nil {
		t.Fatalf("SaveContent() error = %v", err)
	}

	// Get content
	retrieved, err := storage.GetContent(ctx, shortID)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}

	if retrieved != content {
		t.Errorf("GetContent() length = %d, want %d", len(retrieved), len(content))
	}

	// Cleanup
	storage.DeleteContent(ctx, shortID)
}

func TestStorage_GetContent_NotFound(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()

	_, err := storage.GetContent(ctx, "nonexistent")
	if err != ErrContentNotFound {
		t.Errorf("GetContent() error = %v, want ErrContentNotFound", err)
	}
}

func TestStorage_DeleteContent(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test003"
	content := "Content to be deleted"

	// Save content
	err := storage.SaveContent(ctx, shortID, content)
	if err != nil {
		t.Fatalf("SaveContent() error = %v", err)
	}

	// Verify it exists
	exists, err := storage.ContentExists(ctx, shortID)
	if err != nil {
		t.Fatalf("ContentExists() error = %v", err)
	}
	if !exists {
		t.Error("Content should exist after save")
	}

	// Delete content
	err = storage.DeleteContent(ctx, shortID)
	if err != nil {
		t.Fatalf("DeleteContent() error = %v", err)
	}

	// Verify it's gone
	exists, err = storage.ContentExists(ctx, shortID)
	if err != nil {
		t.Fatalf("ContentExists() error = %v", err)
	}
	if exists {
		t.Error("Content should not exist after delete")
	}
}

func TestStorage_ContentExists(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test004"

	// Check non-existent
	exists, err := storage.ContentExists(ctx, shortID)
	if err != nil {
		t.Fatalf("ContentExists() error = %v", err)
	}
	if exists {
		t.Error("Content should not exist before save")
	}

	// Save content
	err = storage.SaveContent(ctx, shortID, "test content")
	if err != nil {
		t.Fatalf("SaveContent() error = %v", err)
	}

	// Check exists
	exists, err = storage.ContentExists(ctx, shortID)
	if err != nil {
		t.Fatalf("ContentExists() error = %v", err)
	}
	if !exists {
		t.Error("Content should exist after save")
	}

	// Cleanup
	storage.DeleteContent(ctx, shortID)
}

func TestStorage_CompressionEfficiency(t *testing.T) {
	// Test that compression actually reduces size for compressible content
	content := strings.Repeat("aaaaaaaaaa", 1000) // Highly compressible

	compressed, err := compressContent(content)
	if err != nil {
		t.Fatalf("compressContent() error = %v", err)
	}

	compressionRatio := float64(len(compressed)) / float64(len(content))
	if compressionRatio > 0.5 {
		t.Errorf("Compression ratio = %.2f, expected < 0.5 for highly compressible content", compressionRatio)
	}

	// Verify roundtrip
	decompressed, err := decompressContent(compressed)
	if err != nil {
		t.Fatalf("decompressContent() error = %v", err)
	}

	if decompressed != content {
		t.Error("Decompressed content does not match original")
	}
}

func TestStorage_SpecialCharacters(t *testing.T) {
	storage, cleanup := setupTestStorage(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test005"

	// Content with special characters, unicode, etc.
	content := `Hello, 世界! 🎉
Special chars: <script>alert('xss')</script>
Newlines and tabs:
	- Item 1
	- Item 2
Unicode: αβγδ ∑∏∫
Emoji: 😀🎉🚀`

	// Save content
	err := storage.SaveContent(ctx, shortID, content)
	if err != nil {
		t.Fatalf("SaveContent() error = %v", err)
	}

	// Get content
	retrieved, err := storage.GetContent(ctx, shortID)
	if err != nil {
		t.Fatalf("GetContent() error = %v", err)
	}

	if retrieved != content {
		t.Errorf("GetContent() = %q, want %q", retrieved, content)
	}

	// Cleanup
	storage.DeleteContent(ctx, shortID)
}