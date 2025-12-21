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
	FindByTeamID(ctx context.Context, teamID primitive.ObjectID, page, limit int) ([]models.VoiceMemo, int, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.VoiceMemo, error)
	SoftDeleteByID(ctx context.Context, id primitive.ObjectID) error
	SoftDeleteWithOwnership(ctx context.Context, id, userID primitive.ObjectID) error
	SoftDeleteWithTeam(ctx context.Context, id, teamID primitive.ObjectID) error
	SoftDeleteByTeamID(ctx context.Context, teamID primitive.ObjectID) error
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

// FindByUserID returns paginated private voice memos for a user, sorted by createdAt descending.
// Excludes soft-deleted records and team memos.
func (r *voiceMemoRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int) ([]models.VoiceMemo, int, error) {
	filter := bson.M{
		"userId":    userID,
		"teamId":    bson.M{"$exists": false}, // Only private memos
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

// SoftDeleteByID marks a voice memo as deleted by setting deletedAt timestamp.
// Note: Use SoftDeleteWithOwnership or SoftDeleteWithTeam instead for atomic ownership/team
// checks. This method is intended for batch operations where authorization is handled separately.
func (r *voiceMemoRepository) SoftDeleteByID(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	filter := bson.M{
		"_id":       id,
		"deletedAt": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"deletedAt": now,
			"updatedAt": now,
		},
		"$inc": bson.M{"version": 1},
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

// SoftDeleteWithOwnership atomically soft-deletes a voice memo if the user owns it.
// Returns nil if memo is already deleted (idempotent).
// Returns ErrVoiceMemoNotFound if memo doesn't exist.
// Returns ErrVoiceMemoUnauthorized if memo exists but user doesn't own it.
func (r *voiceMemoRepository) SoftDeleteWithOwnership(ctx context.Context, id, userID primitive.ObjectID) error {
	now := time.Now()
	filter := bson.M{
		"_id":       id,
		"userId":    userID,
		"deletedAt": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"deletedAt": now,
			"updatedAt": now,
		},
		"$inc": bson.M{"version": 1},
	}

	result := r.collection.FindOneAndUpdate(ctx, filter, update)

	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			// Atomic update failed - need to determine why for proper error response.
			// Trade-off: This extra query only runs on failure path (not found, unauthorized,
			// or already deleted). The happy path remains a single atomic operation.
			var existingMemo models.VoiceMemo
			checkErr := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&existingMemo)

			if checkErr != nil {
				if errors.Is(checkErr, mongo.ErrNoDocuments) {
					return apperrors.ErrVoiceMemoNotFound // Doesn't exist
				}
				return checkErr
			}

			// Document exists - check why update failed
			if existingMemo.UserID != userID {
				return apperrors.ErrVoiceMemoUnauthorized // Wrong owner
			}
			// Must be already deleted - idempotent success
			return nil
		}
		return result.Err()
	}

	return nil
}

// SoftDeleteWithTeam atomically soft-deletes a voice memo if it belongs to the team.
// Returns nil if memo is already deleted (idempotent).
// Returns ErrVoiceMemoNotFound if memo doesn't exist or doesn't belong to team.
func (r *voiceMemoRepository) SoftDeleteWithTeam(ctx context.Context, id, teamID primitive.ObjectID) error {
	now := time.Now()
	filter := bson.M{
		"_id":       id,
		"teamId":    teamID,
		"deletedAt": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"deletedAt": now,
			"updatedAt": now,
		},
		"$inc": bson.M{"version": 1},
	}

	result := r.collection.FindOneAndUpdate(ctx, filter, update)

	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			// Atomic update failed - need to determine why for proper error response.
			// Trade-off: This extra query only runs on failure path (not found, wrong team,
			// or already deleted). The happy path remains a single atomic operation.
			var existingMemo models.VoiceMemo
			checkErr := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&existingMemo)

			if checkErr != nil {
				if errors.Is(checkErr, mongo.ErrNoDocuments) {
					return apperrors.ErrVoiceMemoNotFound
				}
				return checkErr
			}

			// Check team membership
			if existingMemo.TeamID == nil || *existingMemo.TeamID != teamID {
				return apperrors.ErrVoiceMemoNotFound // Wrong team
			}
			// Already deleted - idempotent
			return nil
		}
		return result.Err()
	}

	return nil
}

// FindByTeamID returns paginated voice memos for a team, sorted by createdAt descending.
// Excludes soft-deleted records.
func (r *voiceMemoRepository) FindByTeamID(ctx context.Context, teamID primitive.ObjectID, page, limit int) ([]models.VoiceMemo, int, error) {
	filter := bson.M{
		"teamId":    teamID,
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

// SoftDeleteByTeamID soft deletes all voice memos for a team.
func (r *voiceMemoRepository) SoftDeleteByTeamID(ctx context.Context, teamID primitive.ObjectID) error {
	now := time.Now()
	filter := bson.M{
		"teamId":    teamID,
		"deletedAt": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"deletedAt": now,
			"updatedAt": now,
		},
		"$inc": bson.M{"version": 1},
	}

	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}
