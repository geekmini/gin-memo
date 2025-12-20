// Package repository provides data access operations for the application.
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

// RefreshTokenRepository defines the interface for refresh token data operations.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	FindByToken(ctx context.Context, token string) (*models.RefreshToken, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID primitive.ObjectID) error
}

// refreshTokenRepository implements RefreshTokenRepository using MongoDB.
type refreshTokenRepository struct {
	collection *mongo.Collection
}

// NewRefreshTokenRepository creates a new RefreshTokenRepository.
func NewRefreshTokenRepository(db *mongo.Database) RefreshTokenRepository {
	collection := db.Collection("refresh_tokens")

	// Create indexes
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "userId", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expiresAt", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0), // TTL index
		},
	}

	_, _ = collection.Indexes().CreateMany(ctx, indexes)

	return &refreshTokenRepository{
		collection: collection,
	}
}

// Create inserts a new refresh token into the database.
func (r *refreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	token.CreatedAt = time.Now()

	result, err := r.collection.InsertOne(ctx, token)
	if err != nil {
		return err
	}

	token.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByToken finds a refresh token by its token string.
func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*models.RefreshToken, error) {
	var refreshToken models.RefreshToken

	err := r.collection.FindOne(ctx, bson.M{
		"token":     token,
		"expiresAt": bson.M{"$gt": time.Now()},
	}).Decode(&refreshToken)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrInvalidRefreshToken
		}
		return nil, err
	}

	return &refreshToken, nil
}

// DeleteByToken removes a refresh token by its token string.
func (r *refreshTokenRepository) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"token": token})
	return err
}

// DeleteByUserID removes all refresh tokens for a user.
func (r *refreshTokenRepository) DeleteByUserID(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"userId": userID})
	return err
}
