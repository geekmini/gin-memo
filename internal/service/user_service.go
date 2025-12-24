// Package service contains business logic for the application.
package service

import (
	"context"
	"time"

	"gin-sample/internal/cache"
	"gin-sample/internal/models"
	"gin-sample/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserService handles business logic for user operations.
type UserService struct {
	repo         repository.UserRepository
	cache        cache.Cache
	userCacheTTL time.Duration
}

// NewUserService creates a new UserService.
func NewUserService(repo repository.UserRepository, cache cache.Cache, userCacheTTL time.Duration) *UserService {
	return &UserService{
		repo:         repo,
		cache:        cache,
		userCacheTTL: userCacheTTL,
	}
}

// GetUser retrieves a user by ID (with caching).
func (s *UserService) GetUser(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	// Try cache first
	cacheKey := cache.UserCacheKey(id.Hex())
	var user models.User
	found, err := s.cache.Get(ctx, cacheKey, &user)
	if err == nil && found {
		return &user, nil // Cache hit
	}

	// Cache miss - get from database
	dbUser, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Store in cache (ignore errors - cache is best effort)
	_ = s.cache.Set(ctx, cacheKey, dbUser, s.userCacheTTL)

	return dbUser, nil
}

// GetAllUsers retrieves all users.
func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	return s.repo.FindAll(ctx)
}

// UpdateUser updates a user's information.
func (s *UserService) UpdateUser(ctx context.Context, id primitive.ObjectID, req *models.UpdateUserRequest) (*models.User, error) {
	user, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, cache.UserCacheKey(id.Hex()))

	return user, nil
}

// DeleteUser removes a user.
func (s *UserService) DeleteUser(ctx context.Context, id primitive.ObjectID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, cache.UserCacheKey(id.Hex()))

	return nil
}
