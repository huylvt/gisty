package service

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/huylvt/gisty/pkg/base62"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// KeyLength is the length of generated short IDs
	KeyLength = 6
	// CollectionName is the MongoDB collection for keys
	CollectionName = "keys"
	// DefaultMinKeysThreshold is the minimum number of unused keys before replenishing
	DefaultMinKeysThreshold = 100
	// DefaultBatchSize is the number of keys to generate in each batch
	DefaultBatchSize = 1000
	// DefaultCheckInterval is how often the worker checks for key availability
	DefaultCheckInterval = 1 * time.Minute
)

var (
	// ErrNoKeysAvailable is returned when no unused keys are available
	ErrNoKeysAvailable = errors.New("kgs: no unused keys available")
)

// Key represents a pre-generated key in the database
type Key struct {
	Key       string    `bson:"key"`
	Used      bool      `bson:"used"`
	CreatedAt time.Time `bson:"created_at"`
	UsedAt    time.Time `bson:"used_at,omitempty"`
}

// KGS is the Key Generation Service
type KGS struct {
	collection *mongo.Collection
}

// NewKGS creates a new Key Generation Service
func NewKGS(db *mongo.Database) (*KGS, error) {
	kgs := &KGS{
		collection: db.Collection(CollectionName),
	}

	// Create indexes
	if err := kgs.createIndexes(context.Background()); err != nil {
		return nil, err
	}

	return kgs, nil
}

// createIndexes creates the necessary MongoDB indexes
func (k *KGS) createIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "key", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "used", Value: 1}},
		},
	}

	_, err := k.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// GenerateKeys generates a batch of unique keys
func (k *KGS) GenerateKeys(ctx context.Context, count int) (int, error) {
	if count <= 0 {
		return 0, nil
	}

	generated := 0
	maxAttempts := count * 3 // Allow some retries for collisions

	for i := 0; i < maxAttempts && generated < count; i++ {
		key, err := generateRandomKey()
		if err != nil {
			return generated, err
		}

		doc := Key{
			Key:       key,
			Used:      false,
			CreatedAt: time.Now().UTC(),
		}

		_, err = k.collection.InsertOne(ctx, doc)
		if err != nil {
			// Check if it's a duplicate key error
			if mongo.IsDuplicateKeyError(err) {
				continue // Try another key
			}
			return generated, err
		}
		generated++
	}

	return generated, nil
}

// GetNextKey retrieves and marks an unused key as used atomically
func (k *KGS) GetNextKey(ctx context.Context) (string, error) {
	filter := bson.M{"used": false}
	update := bson.M{
		"$set": bson.M{
			"used":    true,
			"used_at": time.Now().UTC(),
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var key Key
	err := k.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&key)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", ErrNoKeysAvailable
		}
		return "", err
	}

	return key.Key, nil
}

// CountUnusedKeys returns the count of unused keys
func (k *KGS) CountUnusedKeys(ctx context.Context) (int64, error) {
	return k.collection.CountDocuments(ctx, bson.M{"used": false})
}

// CountTotalKeys returns the total count of keys
func (k *KGS) CountTotalKeys(ctx context.Context) (int64, error) {
	return k.collection.CountDocuments(ctx, bson.M{})
}

// generateRandomKey generates a random base62 key of KeyLength
func generateRandomKey() (string, error) {
	// Calculate max value for KeyLength digits in base62
	// 62^6 = 56,800,235,584
	maxVal := new(big.Int).Exp(big.NewInt(62), big.NewInt(KeyLength), nil)

	// Generate random number in range [0, maxVal)
	n, err := rand.Int(rand.Reader, maxVal)
	if err != nil {
		return "", err
	}

	// Encode to base62
	encoded := base62.Encode(n.Uint64())

	// Pad with leading zeros if necessary
	for len(encoded) < KeyLength {
		encoded = "0" + encoded
	}

	return encoded, nil
}

// WorkerConfig holds configuration for the KGS background worker
type WorkerConfig struct {
	MinKeysThreshold int64
	BatchSize        int
	CheckInterval    time.Duration
}

// DefaultWorkerConfig returns the default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		MinKeysThreshold: DefaultMinKeysThreshold,
		BatchSize:        DefaultBatchSize,
		CheckInterval:    DefaultCheckInterval,
	}
}

// StartReplenishWorker starts a background worker that maintains the key pool
func (k *KGS) StartReplenishWorker(ctx context.Context, cfg WorkerConfig) {
	log.Println("KGS Worker started")

	// Initial check and replenish
	k.checkAndReplenish(ctx, cfg)

	ticker := time.NewTicker(cfg.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("KGS Worker stopped")
			return
		case <-ticker.C:
			k.checkAndReplenish(ctx, cfg)
		}
	}
}

// checkAndReplenish checks if keys need to be replenished and generates them if necessary
func (k *KGS) checkAndReplenish(ctx context.Context, cfg WorkerConfig) {
	unused, err := k.CountUnusedKeys(ctx)
	if err != nil {
		log.Printf("KGS Worker: error counting unused keys: %v", err)
		return
	}

	if unused < cfg.MinKeysThreshold {
		log.Printf("KGS Worker: unused keys (%d) below threshold (%d), generating more...", unused, cfg.MinKeysThreshold)

		generated, err := k.GenerateKeys(ctx, cfg.BatchSize)
		if err != nil {
			log.Printf("KGS Worker: error generating keys: %v", err)
			return
		}

		newUnused, _ := k.CountUnusedKeys(ctx)
		log.Printf("KGS Worker: generated %d new keys, total unused: %d", generated, newUnused)
	}
}