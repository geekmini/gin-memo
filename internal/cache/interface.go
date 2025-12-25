package cache

import (
	"context"
	"time"
)

//go:generate mockgen -destination=mocks/mock_cache.go -package=mocks gin-sample/internal/cache Cache

// Cache defines the interface for caching operations.
type Cache interface {
	// Set stores a value in cache with TTL.
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	// Get retrieves a value from cache. Returns false if key doesn't exist.
	Get(ctx context.Context, key string, dest interface{}) (bool, error)
	// Delete removes a key from cache.
	Delete(ctx context.Context, key string) error
	// SetRefreshToken stores a refresh token in cache.
	SetRefreshToken(ctx context.Context, token string, userID string, ttl time.Duration) error
	// GetRefreshToken retrieves a user ID from a refresh token.
	GetRefreshToken(ctx context.Context, token string) (string, error)
	// DeleteRefreshToken removes a refresh token from cache.
	DeleteRefreshToken(ctx context.Context, token string) error
	// DeleteRefreshTokens removes multiple refresh tokens from cache.
	DeleteRefreshTokens(ctx context.Context, tokens []string) error
}

// Ensure Redis implements Cache interface
var _ Cache = (*Redis)(nil)
