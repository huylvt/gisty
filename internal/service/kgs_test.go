package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func setupTestDB(t *testing.T) (*mongo.Database, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use credentials from docker-compose
	uri := "mongodb://gisty:gisty123@localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	// Use a test database
	db := client.Database("gisty_test")

	// Cleanup function
	cleanup := func() {
		db.Drop(context.Background())
		client.Disconnect(context.Background())
	}

	return db, cleanup
}

func TestKGS_GenerateKeys(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	kgs, err := NewKGS(db)
	if err != nil {
		t.Fatalf("NewKGS() error = %v", err)
	}

	// Generate 100 keys
	count, err := kgs.GenerateKeys(context.Background(), 100)
	if err != nil {
		t.Fatalf("GenerateKeys() error = %v", err)
	}
	if count != 100 {
		t.Errorf("GenerateKeys() generated %d keys, want 100", count)
	}

	// Verify count in database
	total, err := kgs.CountTotalKeys(context.Background())
	if err != nil {
		t.Fatalf("CountTotalKeys() error = %v", err)
	}
	if total != 100 {
		t.Errorf("CountTotalKeys() = %d, want 100", total)
	}

	// All should be unused
	unused, err := kgs.CountUnusedKeys(context.Background())
	if err != nil {
		t.Fatalf("CountUnusedKeys() error = %v", err)
	}
	if unused != 100 {
		t.Errorf("CountUnusedKeys() = %d, want 100", unused)
	}
}

func TestKGS_GetNextKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	kgs, err := NewKGS(db)
	if err != nil {
		t.Fatalf("NewKGS() error = %v", err)
	}

	// Generate some keys
	_, err = kgs.GenerateKeys(context.Background(), 10)
	if err != nil {
		t.Fatalf("GenerateKeys() error = %v", err)
	}

	// Get a key
	key, err := kgs.GetNextKey(context.Background())
	if err != nil {
		t.Fatalf("GetNextKey() error = %v", err)
	}
	if len(key) != KeyLength {
		t.Errorf("GetNextKey() returned key of length %d, want %d", len(key), KeyLength)
	}

	// Verify unused count decreased
	unused, _ := kgs.CountUnusedKeys(context.Background())
	if unused != 9 {
		t.Errorf("CountUnusedKeys() = %d, want 9", unused)
	}
}

func TestKGS_GetNextKey_NoKeysAvailable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	kgs, err := NewKGS(db)
	if err != nil {
		t.Fatalf("NewKGS() error = %v", err)
	}

	// Don't generate any keys
	_, err = kgs.GetNextKey(context.Background())
	if err != ErrNoKeysAvailable {
		t.Errorf("GetNextKey() error = %v, want ErrNoKeysAvailable", err)
	}
}

func TestKGS_GetNextKey_Unique(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	kgs, err := NewKGS(db)
	if err != nil {
		t.Fatalf("NewKGS() error = %v", err)
	}

	// Generate 50 keys
	_, err = kgs.GenerateKeys(context.Background(), 50)
	if err != nil {
		t.Fatalf("GenerateKeys() error = %v", err)
	}

	// Get all keys and verify uniqueness
	keys := make(map[string]bool)
	for i := 0; i < 50; i++ {
		key, err := kgs.GetNextKey(context.Background())
		if err != nil {
			t.Fatalf("GetNextKey() error = %v", err)
		}
		if keys[key] {
			t.Errorf("GetNextKey() returned duplicate key: %s", key)
		}
		keys[key] = true
	}
}

func TestKGS_ConcurrentGetNextKey(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	kgs, err := NewKGS(db)
	if err != nil {
		t.Fatalf("NewKGS() error = %v", err)
	}

	// Generate 100 keys
	_, err = kgs.GenerateKeys(context.Background(), 100)
	if err != nil {
		t.Fatalf("GenerateKeys() error = %v", err)
	}

	// Concurrently get keys
	var wg sync.WaitGroup
	keys := make(chan string, 100)
	errChan := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, err := kgs.GetNextKey(context.Background())
			if err != nil {
				errChan <- err
				return
			}
			keys <- key
		}()
	}

	wg.Wait()
	close(keys)
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("GetNextKey() error = %v", err)
	}

	// Verify all keys are unique
	keySet := make(map[string]bool)
	for key := range keys {
		if keySet[key] {
			t.Errorf("GetNextKey() returned duplicate key in concurrent access: %s", key)
		}
		keySet[key] = true
	}

	if len(keySet) != 100 {
		t.Errorf("Expected 100 unique keys, got %d", len(keySet))
	}
}

func TestKGS_KeyFormat(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	kgs, err := NewKGS(db)
	if err != nil {
		t.Fatalf("NewKGS() error = %v", err)
	}

	// Generate keys
	_, err = kgs.GenerateKeys(context.Background(), 10)
	if err != nil {
		t.Fatalf("GenerateKeys() error = %v", err)
	}

	// Check key format
	var key Key
	err = kgs.collection.FindOne(context.Background(), bson.M{}).Decode(&key)
	if err != nil {
		t.Fatalf("FindOne() error = %v", err)
	}

	if len(key.Key) != KeyLength {
		t.Errorf("Key length = %d, want %d", len(key.Key), KeyLength)
	}

	// Verify key contains only valid base62 characters
	for _, c := range key.Key {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			t.Errorf("Key contains invalid character: %c", c)
		}
	}
}

func TestKGS_ReplenishWorker(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	kgs, err := NewKGS(db)
	if err != nil {
		t.Fatalf("NewKGS() error = %v", err)
	}

	// Create a context with cancel for worker
	ctx, cancel := context.WithCancel(context.Background())

	// Configure worker with short interval for testing
	cfg := WorkerConfig{
		MinKeysThreshold: 50,
		BatchSize:        100,
		CheckInterval:    100 * time.Millisecond,
	}

	// Start worker in background
	go kgs.StartReplenishWorker(ctx, cfg)

	// Wait for worker to generate keys
	time.Sleep(200 * time.Millisecond)

	// Verify keys were generated
	unused, err := kgs.CountUnusedKeys(context.Background())
	if err != nil {
		t.Fatalf("CountUnusedKeys() error = %v", err)
	}
	if unused < cfg.MinKeysThreshold {
		t.Errorf("Worker should have generated keys, got %d unused", unused)
	}

	// Stop worker
	cancel()
	time.Sleep(50 * time.Millisecond) // Give worker time to stop
}

func TestKGS_ReplenishWorker_AutoReplenish(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	kgs, err := NewKGS(db)
	if err != nil {
		t.Fatalf("NewKGS() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := WorkerConfig{
		MinKeysThreshold: 20,
		BatchSize:        50,
		CheckInterval:    100 * time.Millisecond,
	}

	// Start worker
	go kgs.StartReplenishWorker(ctx, cfg)

	// Wait for initial replenish
	time.Sleep(150 * time.Millisecond)

	// Use up most keys
	for i := 0; i < 40; i++ {
		_, err := kgs.GetNextKey(context.Background())
		if err != nil {
			t.Fatalf("GetNextKey() error = %v", err)
		}
	}

	// Wait for worker to detect and replenish
	time.Sleep(200 * time.Millisecond)

	// Verify keys were replenished
	unused, _ := kgs.CountUnusedKeys(context.Background())
	if unused < cfg.MinKeysThreshold {
		t.Errorf("Worker should have replenished keys, got %d unused", unused)
	}
}