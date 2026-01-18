package repository

import (
	"context"
	"testing"
	"time"

	"github.com/huylvt/gisty/internal/model"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestPasteDB(t *testing.T) (*mongo.Database, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use credentials from docker-compose
	uri := "mongodb://gisty:gisty123@localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	// Use a test database
	db := client.Database("gisty_paste_test")

	// Cleanup function
	cleanup := func() {
		_ = db.Collection(PasteCollectionName).Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}

	return db, cleanup
}

func TestPasteRepository_CreateAndGet(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Create a paste
	paste := &model.Paste{
		ShortID:       "test001",
		ContentKey:    "gisty/test001.gz",
		CreatedAt:     time.Now(),
		SyntaxType:    "go",
		IsPrivate:     false,
		BurnAfterRead: false,
	}

	err = repo.Create(ctx, paste)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get the paste
	retrieved, err := repo.GetByShortID(ctx, "test001")
	if err != nil {
		t.Fatalf("GetByShortID() error = %v", err)
	}

	if retrieved.ShortID != paste.ShortID {
		t.Errorf("ShortID = %q, want %q", retrieved.ShortID, paste.ShortID)
	}
	if retrieved.ContentKey != paste.ContentKey {
		t.Errorf("ContentKey = %q, want %q", retrieved.ContentKey, paste.ContentKey)
	}
	if retrieved.SyntaxType != paste.SyntaxType {
		t.Errorf("SyntaxType = %q, want %q", retrieved.SyntaxType, paste.SyntaxType)
	}
}

func TestPasteRepository_CreateWithExpiration(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Create a paste with expiration
	expiresAt := time.Now().Add(1 * time.Hour)
	paste := &model.Paste{
		ShortID:    "test002",
		ContentKey: "gisty/test002.gz",
		CreatedAt:  time.Now(),
		ExpiresAt:  &expiresAt,
		SyntaxType: "python",
	}

	err = repo.Create(ctx, paste)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get the paste
	retrieved, err := repo.GetByShortID(ctx, "test002")
	if err != nil {
		t.Fatalf("GetByShortID() error = %v", err)
	}

	if retrieved.ExpiresAt == nil {
		t.Error("ExpiresAt should not be nil")
	}
	if !retrieved.HasExpiration() {
		t.Error("HasExpiration() should return true")
	}
	if retrieved.IsExpired() {
		t.Error("Paste should not be expired yet")
	}
}

func TestPasteRepository_DuplicateShortID(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Create first paste
	paste1 := &model.Paste{
		ShortID:    "duplicate",
		ContentKey: "gisty/duplicate.gz",
		CreatedAt:  time.Now(),
		SyntaxType: "text",
	}

	err = repo.Create(ctx, paste1)
	if err != nil {
		t.Fatalf("Create() first paste error = %v", err)
	}

	// Try to create paste with same short_id
	paste2 := &model.Paste{
		ShortID:    "duplicate",
		ContentKey: "gisty/duplicate2.gz",
		CreatedAt:  time.Now(),
		SyntaxType: "text",
	}

	err = repo.Create(ctx, paste2)
	if err != ErrPasteDuplicate {
		t.Errorf("Create() should return ErrPasteDuplicate, got %v", err)
	}
}

func TestPasteRepository_GetByShortID_NotFound(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	_, err = repo.GetByShortID(ctx, "nonexistent")
	if err != ErrPasteNotFound {
		t.Errorf("GetByShortID() should return ErrPasteNotFound, got %v", err)
	}
}

func TestPasteRepository_Delete(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Create a paste
	paste := &model.Paste{
		ShortID:    "todelete",
		ContentKey: "gisty/todelete.gz",
		CreatedAt:  time.Now(),
		SyntaxType: "text",
	}

	err = repo.Create(ctx, paste)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify it exists
	_, err = repo.GetByShortID(ctx, "todelete")
	if err != nil {
		t.Fatalf("GetByShortID() error = %v", err)
	}

	// Delete it
	err = repo.Delete(ctx, "todelete")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	_, err = repo.GetByShortID(ctx, "todelete")
	if err != ErrPasteNotFound {
		t.Errorf("GetByShortID() should return ErrPasteNotFound after delete, got %v", err)
	}
}

func TestPasteRepository_Delete_NotFound(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	err = repo.Delete(ctx, "nonexistent")
	if err != ErrPasteNotFound {
		t.Errorf("Delete() should return ErrPasteNotFound, got %v", err)
	}
}

func TestPasteRepository_GetExpired(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Create non-expired paste
	nonExpired := &model.Paste{
		ShortID:    "nonexpired",
		ContentKey: "gisty/nonexpired.gz",
		CreatedAt:  time.Now(),
		SyntaxType: "text",
	}
	err = repo.Create(ctx, nonExpired)
	if err != nil {
		t.Fatalf("Create() non-expired error = %v", err)
	}

	// Create expired paste (past time)
	expiredTime := time.Now().Add(-1 * time.Hour)
	expired := &model.Paste{
		ShortID:    "expired",
		ContentKey: "gisty/expired.gz",
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		ExpiresAt:  &expiredTime,
		SyntaxType: "text",
	}
	err = repo.Create(ctx, expired)
	if err != nil {
		t.Fatalf("Create() expired error = %v", err)
	}

	// Create future expiration paste
	futureTime := time.Now().Add(1 * time.Hour)
	future := &model.Paste{
		ShortID:    "future",
		ContentKey: "gisty/future.gz",
		CreatedAt:  time.Now(),
		ExpiresAt:  &futureTime,
		SyntaxType: "text",
	}
	err = repo.Create(ctx, future)
	if err != nil {
		t.Fatalf("Create() future error = %v", err)
	}

	// Get expired pastes
	expiredPastes, err := repo.GetExpired(ctx)
	if err != nil {
		t.Fatalf("GetExpired() error = %v", err)
	}

	if len(expiredPastes) != 1 {
		t.Errorf("GetExpired() returned %d pastes, want 1", len(expiredPastes))
	}

	if len(expiredPastes) > 0 && expiredPastes[0].ShortID != "expired" {
		t.Errorf("GetExpired() returned wrong paste: %q, want 'expired'", expiredPastes[0].ShortID)
	}
}

func TestPasteRepository_DeleteMany(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Create multiple pastes
	for i := 1; i <= 5; i++ {
		paste := &model.Paste{
			ShortID:    "batch" + string(rune('0'+i)),
			ContentKey: "gisty/batch.gz",
			CreatedAt:  time.Now(),
			SyntaxType: "text",
		}
		err = repo.Create(ctx, paste)
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	// Delete multiple
	deleted, err := repo.DeleteMany(ctx, []string{"batch1", "batch2", "batch3"})
	if err != nil {
		t.Fatalf("DeleteMany() error = %v", err)
	}

	if deleted != 3 {
		t.Errorf("DeleteMany() deleted %d, want 3", deleted)
	}

	// Verify remaining count
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}

	if count != 2 {
		t.Errorf("Count() = %d, want 2", count)
	}
}

func TestPasteRepository_Count(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Initially empty
	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Initial count = %d, want 0", count)
	}

	// Create some pastes
	for i := 1; i <= 3; i++ {
		paste := &model.Paste{
			ShortID:    "count" + string(rune('0'+i)),
			ContentKey: "gisty/count.gz",
			CreatedAt:  time.Now(),
			SyntaxType: "text",
		}
		_ = repo.Create(ctx, paste)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestPasteRepository_BurnAfterRead(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Create burn after read paste
	paste := &model.Paste{
		ShortID:       "burnme",
		ContentKey:    "gisty/burnme.gz",
		CreatedAt:     time.Now(),
		SyntaxType:    "text",
		BurnAfterRead: true,
	}

	err = repo.Create(ctx, paste)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get it
	retrieved, err := repo.GetByShortID(ctx, "burnme")
	if err != nil {
		t.Fatalf("GetByShortID() error = %v", err)
	}

	if !retrieved.BurnAfterRead {
		t.Error("BurnAfterRead should be true")
	}
}

func TestPasteRepository_WithUserID(t *testing.T) {
	db, cleanup := setupTestPasteDB(t)
	defer cleanup()

	repo, err := NewPasteRepository(db)
	if err != nil {
		t.Fatalf("NewPasteRepository() error = %v", err)
	}

	ctx := context.Background()

	// Create paste with user ID
	userID := "user123"
	paste := &model.Paste{
		ShortID:    "withuser",
		UserID:     &userID,
		ContentKey: "gisty/withuser.gz",
		CreatedAt:  time.Now(),
		SyntaxType: "text",
		IsPrivate:  true,
	}

	err = repo.Create(ctx, paste)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Get it
	retrieved, err := repo.GetByShortID(ctx, "withuser")
	if err != nil {
		t.Fatalf("GetByShortID() error = %v", err)
	}

	if retrieved.UserID == nil {
		t.Error("UserID should not be nil")
	}
	if *retrieved.UserID != userID {
		t.Errorf("UserID = %q, want %q", *retrieved.UserID, userID)
	}
	if !retrieved.IsPrivate {
		t.Error("IsPrivate should be true")
	}
}