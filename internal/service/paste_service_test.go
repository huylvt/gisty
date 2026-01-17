package service

import (
	"context"
	"testing"
	"time"

	"github.com/huylvt/gisty/internal/repository"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupPasteServiceTest(t *testing.T) (*PasteService, func()) {
	ctx := context.Background()

	// Connect to MongoDB
	mongoURI := "mongodb://gisty:gisty123@localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	db := client.Database("gisty_paste_service_test")

	// Connect to Redis
	redisClient, err := repository.NewRedisClient(ctx, "redis://localhost:6379")
	if err != nil {
		client.Disconnect(ctx)
		t.Skipf("Redis not available: %v", err)
	}

	// Connect to MinIO
	s3Client, err := repository.NewS3Client(ctx, repository.S3Config{
		BucketName:      "gisty-test",
		Region:          "us-east-1",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Endpoint:        "http://localhost:9000",
	})
	if err != nil {
		redisClient.Close()
		client.Disconnect(ctx)
		t.Skipf("MinIO not available: %v", err)
	}

	// Initialize services
	kgs, err := NewKGS(db)
	if err != nil {
		redisClient.Close()
		client.Disconnect(ctx)
		t.Fatalf("Failed to create KGS: %v", err)
	}

	// Pre-generate some keys
	kgs.GenerateKeys(ctx, 100)

	storage := NewStorage(s3Client)
	cache := NewCache(redisClient)

	pasteRepo, err := repository.NewPasteRepository(db)
	if err != nil {
		redisClient.Close()
		client.Disconnect(ctx)
		t.Fatalf("Failed to create paste repository: %v", err)
	}

	pasteService := NewPasteService(kgs, storage, cache, pasteRepo, "http://localhost:8080")

	cleanup := func() {
		db.Drop(ctx)
		redisClient.Close()
		client.Disconnect(ctx)
	}

	return pasteService, cleanup
}

func TestPasteService_CreatePaste(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	req := &CreatePasteRequest{
		Content:    "Hello, World!",
		SyntaxType: "plaintext",
		ExpiresIn:  "1h",
	}

	resp, err := svc.CreatePaste(ctx, req)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	if resp.ShortID == "" {
		t.Error("ShortID should not be empty")
	}
	if resp.URL == "" {
		t.Error("URL should not be empty")
	}
	if resp.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil for 1h expiration")
	}

	t.Logf("Created paste: %s at %s", resp.ShortID, resp.URL)
}

func TestPasteService_CreatePaste_NoExpiration(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	req := &CreatePasteRequest{
		Content:    "Never expires",
		SyntaxType: "go",
		ExpiresIn:  "never",
	}

	resp, err := svc.CreatePaste(ctx, req)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	if resp.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil for 'never' expiration")
	}
}

func TestPasteService_CreatePaste_BurnAfterRead(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	req := &CreatePasteRequest{
		Content:   "This will be burned",
		ExpiresIn: "burn",
	}

	resp, err := svc.CreatePaste(ctx, req)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	if resp.ShortID == "" {
		t.Error("ShortID should not be empty")
	}

	// Verify the paste is marked as burn after read
	paste, err := svc.pasteRepo.GetByShortID(ctx, resp.ShortID)
	if err != nil {
		t.Fatalf("GetByShortID() error = %v", err)
	}

	if !paste.BurnAfterRead {
		t.Error("BurnAfterRead should be true")
	}
}

func TestPasteService_CreatePaste_EmptyContent(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	req := &CreatePasteRequest{
		Content: "",
	}

	_, err := svc.CreatePaste(ctx, req)
	if err != ErrEmptyContent {
		t.Errorf("CreatePaste() should return ErrEmptyContent, got %v", err)
	}
}

func TestPasteService_CreatePaste_ContentTooLarge(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create content larger than 1MB
	largeContent := make([]byte, MaxContentSize+1)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	req := &CreatePasteRequest{
		Content: string(largeContent),
	}

	_, err := svc.CreatePaste(ctx, req)
	if err != ErrContentTooLarge {
		t.Errorf("CreatePaste() should return ErrContentTooLarge, got %v", err)
	}
}

func TestPasteService_CreatePaste_InvalidExpiresIn(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	req := &CreatePasteRequest{
		Content:   "Test",
		ExpiresIn: "invalid",
	}

	_, err := svc.CreatePaste(ctx, req)
	if err != ErrInvalidExpiresIn {
		t.Errorf("CreatePaste() should return ErrInvalidExpiresIn, got %v", err)
	}
}

func TestPasteService_CreatePaste_DefaultSyntaxType(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	req := &CreatePasteRequest{
		Content: "Some content without syntax type",
	}

	resp, err := svc.CreatePaste(ctx, req)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	// Verify default syntax type is set
	paste, err := svc.pasteRepo.GetByShortID(ctx, resp.ShortID)
	if err != nil {
		t.Fatalf("GetByShortID() error = %v", err)
	}

	if paste.SyntaxType != DefaultSyntaxType {
		t.Errorf("SyntaxType = %q, want %q", paste.SyntaxType, DefaultSyntaxType)
	}
}

func TestPasteService_CreatePaste_Private(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	req := &CreatePasteRequest{
		Content:   "Private content",
		IsPrivate: true,
	}

	resp, err := svc.CreatePaste(ctx, req)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	// Verify private flag
	paste, err := svc.pasteRepo.GetByShortID(ctx, resp.ShortID)
	if err != nil {
		t.Fatalf("GetByShortID() error = %v", err)
	}

	if !paste.IsPrivate {
		t.Error("IsPrivate should be true")
	}
}

func TestPasteService_ParseExpiration(t *testing.T) {
	svc := &PasteService{}

	tests := []struct {
		input         string
		wantDuration  time.Duration
		wantBurn      bool
		wantNil       bool
		wantErr       bool
	}{
		{"", 0, false, true, false},
		{"never", 0, false, true, false},
		{"burn", 0, true, true, false},
		{"10m", 10 * time.Minute, false, false, false},
		{"30m", 30 * time.Minute, false, false, false},
		{"1h", 1 * time.Hour, false, false, false},
		{"6h", 6 * time.Hour, false, false, false},
		{"12h", 12 * time.Hour, false, false, false},
		{"1d", 24 * time.Hour, false, false, false},
		{"3d", 3 * 24 * time.Hour, false, false, false},
		{"1w", 7 * 24 * time.Hour, false, false, false},
		{"2w", 14 * 24 * time.Hour, false, false, false},
		{"1M", 30 * 24 * time.Hour, false, false, false},
		{"2h30m", 2*time.Hour + 30*time.Minute, false, false, false}, // Go duration
		{"invalid", 0, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expiresAt, burn, err := svc.parseExpiration(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if burn != tt.wantBurn {
				t.Errorf("burn = %v, want %v", burn, tt.wantBurn)
			}

			if tt.wantNil {
				if expiresAt != nil && !tt.wantBurn {
					t.Error("Expected nil expiresAt")
				}
				return
			}

			if expiresAt == nil {
				t.Fatal("Expected non-nil expiresAt")
			}

			// Check duration is approximately correct (within 1 second)
			expectedTime := time.Now().Add(tt.wantDuration)
			diff := expectedTime.Sub(*expiresAt)
			if diff < 0 {
				diff = -diff
			}
			if diff > time.Second {
				t.Errorf("expiresAt differs by %v, expected within 1s", diff)
			}
		})
	}
}

func TestPasteService_GetPaste(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a paste first
	createReq := &CreatePasteRequest{
		Content:    "Hello, World!",
		SyntaxType: "go",
		ExpiresIn:  "1h",
	}

	createResp, err := svc.CreatePaste(ctx, createReq)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	// Get the paste
	getResp, err := svc.GetPaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("GetPaste() error = %v", err)
	}

	if getResp.ShortID != createResp.ShortID {
		t.Errorf("ShortID = %q, want %q", getResp.ShortID, createResp.ShortID)
	}
	if getResp.Content != createReq.Content {
		t.Errorf("Content = %q, want %q", getResp.Content, createReq.Content)
	}
	if getResp.SyntaxType != createReq.SyntaxType {
		t.Errorf("SyntaxType = %q, want %q", getResp.SyntaxType, createReq.SyntaxType)
	}
	if getResp.CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}
	if getResp.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil for paste with expiration")
	}
}

func TestPasteService_GetPaste_NotFound(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.GetPaste(ctx, "nonexistent")
	if err != ErrPasteNotFound {
		t.Errorf("GetPaste() should return ErrPasteNotFound, got %v", err)
	}
}

func TestPasteService_GetPaste_BurnAfterRead(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a burn-after-read paste
	createReq := &CreatePasteRequest{
		Content:   "Secret content",
		ExpiresIn: "burn",
	}

	createResp, err := svc.CreatePaste(ctx, createReq)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	// First read should succeed
	getResp, err := svc.GetPaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("First GetPaste() error = %v", err)
	}
	if getResp.Content != createReq.Content {
		t.Errorf("Content = %q, want %q", getResp.Content, createReq.Content)
	}

	// Wait for async delete to complete
	time.Sleep(100 * time.Millisecond)

	// Second read should fail
	_, err = svc.GetPaste(ctx, createResp.ShortID)
	if err != ErrPasteNotFound {
		t.Errorf("Second GetPaste() should return ErrPasteNotFound, got %v", err)
	}
}

func TestPasteService_GetPaste_CacheHit(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a paste
	createReq := &CreatePasteRequest{
		Content:    "Cached content",
		SyntaxType: "text",
	}

	createResp, err := svc.CreatePaste(ctx, createReq)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	// First read (should populate cache)
	_, err = svc.GetPaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("First GetPaste() error = %v", err)
	}

	// Second read (should hit cache)
	getResp, err := svc.GetPaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("Second GetPaste() error = %v", err)
	}

	if getResp.Content != createReq.Content {
		t.Errorf("Content = %q, want %q", getResp.Content, createReq.Content)
	}
}

func TestPasteService_GetPaste_NoExpiration(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a paste without expiration
	createReq := &CreatePasteRequest{
		Content:   "Never expires",
		ExpiresIn: "never",
	}

	createResp, err := svc.CreatePaste(ctx, createReq)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	// Get the paste
	getResp, err := svc.GetPaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("GetPaste() error = %v", err)
	}

	if getResp.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil for paste without expiration")
	}
}

func TestPasteService_DeletePaste(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a paste first
	createReq := &CreatePasteRequest{
		Content:    "To be deleted",
		SyntaxType: "text",
	}

	createResp, err := svc.CreatePaste(ctx, createReq)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	// Verify it exists
	_, err = svc.GetPaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("GetPaste() before delete error = %v", err)
	}

	// Delete it
	err = svc.DeletePaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("DeletePaste() error = %v", err)
	}

	// Verify it's gone
	_, err = svc.GetPaste(ctx, createResp.ShortID)
	if err != ErrPasteNotFound {
		t.Errorf("GetPaste() after delete should return ErrPasteNotFound, got %v", err)
	}
}

func TestPasteService_DeletePaste_NotFound(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	err := svc.DeletePaste(ctx, "nonexistent")
	if err != ErrPasteNotFound {
		t.Errorf("DeletePaste() should return ErrPasteNotFound, got %v", err)
	}
}

func TestPasteService_DeletePaste_DeletesFromAllLayers(t *testing.T) {
	svc, cleanup := setupPasteServiceTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a paste
	createReq := &CreatePasteRequest{
		Content:    "Delete from all layers",
		SyntaxType: "text",
	}

	createResp, err := svc.CreatePaste(ctx, createReq)
	if err != nil {
		t.Fatalf("CreatePaste() error = %v", err)
	}

	// Access it to populate cache
	_, err = svc.GetPaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("GetPaste() error = %v", err)
	}

	// Verify cache has the content
	_, found, _ := svc.cache.Get(ctx, createResp.ShortID)
	if !found {
		t.Error("Expected content to be in cache before delete")
	}

	// Delete the paste
	err = svc.DeletePaste(ctx, createResp.ShortID)
	if err != nil {
		t.Fatalf("DeletePaste() error = %v", err)
	}

	// Verify cache is cleared
	_, found, _ = svc.cache.Get(ctx, createResp.ShortID)
	if found {
		t.Error("Expected cache to be cleared after delete")
	}

	// Verify S3 content is gone
	_, err = svc.storage.GetContent(ctx, createResp.ShortID)
	if err == nil {
		t.Error("Expected S3 content to be deleted")
	}

	// Verify MongoDB record is gone
	_, err = svc.pasteRepo.GetByShortID(ctx, createResp.ShortID)
	if err == nil {
		t.Error("Expected MongoDB record to be deleted")
	}
}
