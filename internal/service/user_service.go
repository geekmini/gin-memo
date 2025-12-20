// Package service contains business logic for the application.
package service

import (
	"context"
	"time"

	"gin-sample/internal/cache"
	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const userCacheTTL = 15 * time.Minute

// UserService handles business logic for user operations.
type UserService struct {
	repo  repository.UserRepository
	cache *cache.Redis
}

// NewUserService creates a new UserService.
func NewUserService(repo repository.UserRepository, cache *cache.Redis) *UserService {
	return &UserService{
		repo:  repo,
		cache: cache,
	}
}

// GetUser retrieves a user by ID (with caching).
func (s *UserService) GetUser(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	// Try cache first
	cacheKey := cache.UserCacheKey(id)
	var user models.User
	found, err := s.cache.Get(ctx, cacheKey, &user)
	if err == nil && found {
		return &user, nil // Cache hit
	}

	// Cache miss - get from database
	dbUser, err := s.repo.FindByID(ctx, objectID)
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors - cache is best effort)
	_ = s.cache.Set(ctx, cacheKey, dbUser, userCacheTTL)

	return dbUser, nil
}

// GetAllUsers retrieves all users.
func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	return s.repo.FindAll(ctx)
}

// UpdateUser updates a user's information.
func (s *UserService) UpdateUser(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	user, err := s.repo.Update(ctx, objectID, req)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, cache.UserCacheKey(id))

	return user, nil
}

// DeleteUser removes a user.
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	if err := s.repo.Delete(ctx, objectID); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, cache.UserCacheKey(id))

	return nil
}
