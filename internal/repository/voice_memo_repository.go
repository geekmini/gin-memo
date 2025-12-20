package repository

import (
	"context"
	"errors"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// VoiceMemoRepository defines the interface for voice memo data operations.
type VoiceMemoRepository interface {
	FindByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int) ([]models.VoiceMemo, int, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.VoiceMemo, error)
	SoftDelete(ctx context.Context, id primitive.ObjectID) error
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
// Excludes soft-deleted records.
func (r *voiceMemoRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int) ([]models.VoiceMemo, int, error) {
	filter := bson.M{
		"userId":    userID,
		"deletedAt": bson.M{"$exists": false},
	}

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

// FindByID retrieves a voice memo by ID. Excludes soft-deleted records.
func (r *voiceMemoRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.VoiceMemo, error) {
	filter := bson.M{
		"_id":       id,
		"deletedAt": bson.M{"$exists": false},
	}

	var memo models.VoiceMemo
	err := r.collection.FindOne(ctx, filter).Decode(&memo)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrVoiceMemoNotFound
		}
		return nil, err
	}

	return &memo, nil
}

// SoftDelete marks a voice memo as deleted by setting deletedAt timestamp.
func (r *voiceMemoRepository) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{
		"_id":       id,
		"deletedAt": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"deletedAt": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return apperrors.ErrVoiceMemoNotFound
	}

	return nil
}
