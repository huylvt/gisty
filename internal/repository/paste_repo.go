package repository

import (
	"context"
	"errors"
	"time"

	"github.com/huylvt/gisty/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// PasteCollectionName is the MongoDB collection name for pastes
	PasteCollectionName = "pastes"
)

var (
	// ErrPasteNotFound is returned when a paste is not found
	ErrPasteNotFound = errors.New("paste: not found")
	// ErrPasteDuplicate is returned when a paste with the same short_id already exists
	ErrPasteDuplicate = errors.New("paste: duplicate short_id")
)

// PasteRepository handles paste CRUD operations
type PasteRepository struct {
	collection *mongo.Collection
}

// NewPasteRepository creates a new PasteRepository
func NewPasteRepository(db *mongo.Database) (*PasteRepository, error) {
	repo := &PasteRepository{
		collection: db.Collection(PasteCollectionName),
	}

	// Create indexes
	if err := repo.createIndexes(context.Background()); err != nil {
		return nil, err
	}

	return repo, nil
}

// createIndexes creates the required indexes for the pastes collection
func (r *PasteRepository) createIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "short_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetSparse(true),
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	}

	_, err := r.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// Create creates a new paste in the database
func (r *PasteRepository) Create(ctx context.Context, paste *model.Paste) error {
	_, err := r.collection.InsertOne(ctx, paste)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrPasteDuplicate
		}
		return err
	}
	return nil
}

// GetByShortID retrieves a paste by its short ID
func (r *PasteRepository) GetByShortID(ctx context.Context, shortID string) (*model.Paste, error) {
	var paste model.Paste
	err := r.collection.FindOne(ctx, bson.M{"short_id": shortID}).Decode(&paste)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrPasteNotFound
		}
		return nil, err
	}
	return &paste, nil
}

// Delete removes a paste by its short ID
func (r *PasteRepository) Delete(ctx context.Context, shortID string) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"short_id": shortID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return ErrPasteNotFound
	}
	return nil
}

// GetExpired retrieves all pastes that have expired
func (r *PasteRepository) GetExpired(ctx context.Context) ([]*model.Paste, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"expires_at": bson.M{
			"$lt": time.Now(),
			"$ne": nil,
		},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var pastes []*model.Paste
	if err := cursor.All(ctx, &pastes); err != nil {
		return nil, err
	}

	return pastes, nil
}

// GetExpiredBatch retrieves expired pastes in batches for efficient cleanup
func (r *PasteRepository) GetExpiredBatch(ctx context.Context, limit int64) ([]*model.Paste, error) {
	opts := options.Find().SetLimit(limit)
	cursor, err := r.collection.Find(ctx, bson.M{
		"expires_at": bson.M{
			"$lt": time.Now(),
			"$ne": nil,
		},
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var pastes []*model.Paste
	if err := cursor.All(ctx, &pastes); err != nil {
		return nil, err
	}

	return pastes, nil
}

// DeleteMany removes multiple pastes by their short IDs
func (r *PasteRepository) DeleteMany(ctx context.Context, shortIDs []string) (int64, error) {
	if len(shortIDs) == 0 {
		return 0, nil
	}

	result, err := r.collection.DeleteMany(ctx, bson.M{
		"short_id": bson.M{"$in": shortIDs},
	})
	if err != nil {
		return 0, err
	}

	return result.DeletedCount, nil
}

// Count returns the total number of pastes
func (r *PasteRepository) Count(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

// CountExpired returns the number of expired pastes
func (r *PasteRepository) CountExpired(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"expires_at": bson.M{
			"$lt": time.Now(),
			"$ne": nil,
		},
	})
}

// DeleteAll removes all pastes from the collection (for testing)
func (r *PasteRepository) DeleteAll(ctx context.Context) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{})
	return err
}