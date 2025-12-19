package repository

import (
	"context"

	"gin-sample/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VoiceMemoRepository defines the interface for voice memo data operations.
type VoiceMemoRepository interface {
	FindByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int) ([]models.VoiceMemo, int, error)
}

// voiceMemoRepository implements VoiceMemoRepository using MongoDB.
type voiceMemoRepository struct {
	collection *mongo.Collection
}

// NewVoiceMemoRepository creates a new VoiceMemoRepository.
func NewVoiceMemoRepository(db *mongo.Database) VoiceMemoRepository {
	return &voiceMemoRepository{
		collection: db.Collection("voice_memos"),
	}
}

// FindByUserID returns paginated voice memos for a user, sorted by createdAt descending.
func (r *voiceMemoRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int) ([]models.VoiceMemo, int, error) {
	filter := bson.M{"userId": userID}

	// Count total documents
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Find with pagination and sorting (newest first)
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var memos []models.VoiceMemo
	if err := cursor.All(ctx, &memos); err != nil {
		return nil, 0, err
	}

	// Return empty slice instead of nil
	if memos == nil {
		memos = []models.VoiceMemo{}
	}

	return memos, int(total), nil
}
