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
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindAll(ctx context.Context) ([]models.User, error)
	Update(ctx context.Context, id primitive.ObjectID, update *models.UpdateUserRequest) (*models.User, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// userRepository implements UserRepository using MongoDB
type userRepository struct {
	collection *mongo.Collection
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *mongo.Database) UserRepository {
	return &userRepository{
		collection: db.Collection("users"),
	}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	// Check if user with email already exists
	existing, _ := r.FindByEmail(ctx, user.Email)
	if existing != nil {
		return apperrors.ErrUserAlreadyExists
	}

	// Set timestamps
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	// Insert into database
	result, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}

	// Set the generated ID back to the user
	user.ID = result.InsertedID.(primitive.ObjectID)
	return nil
}

// FindByID finds a user by their ID
func (r *userRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User

	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// FindByEmail finds a user by their email
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User

	err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}

// FindAll returns all users
func (r *userRepository) FindAll(ctx context.Context) ([]models.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	// Return empty slice instead of nil
	if users == nil {
		users = []models.User{}
	}

	return users, nil
}

// Update updates a user's information
func (r *userRepository) Update(ctx context.Context, id primitive.ObjectID, update *models.UpdateUserRequest) (*models.User, error) {
	// Build update document
	updateDoc := bson.M{"updatedAt": time.Now()}

	if update.Email != nil {
		// Check if new email is already taken by another user
		existing, _ := r.FindByEmail(ctx, *update.Email)
		if existing != nil && existing.ID != id {
			return nil, apperrors.ErrUserAlreadyExists
		}
		updateDoc["email"] = *update.Email
	}

	if update.Name != nil {
		updateDoc["name"] = *update.Name
	}

	// Perform update
	result := r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updateDoc},
		// Return the updated document (not the original)
	)

	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, result.Err()
	}

	// Fetch and return the updated user
	return r.FindByID(ctx, id)
}

// Delete removes a user from the database
func (r *userRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}
