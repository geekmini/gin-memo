// Package cache provides caching functionality using Redis.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis wraps the Redis client.
type Redis struct {
	client *redis.Client
}

// NewRedis creates a new Redis connection.
func NewRedis(uri string) *Redis {
	opt, err := redis.ParseURL("redis://" + uri)
	if err != nil {
		log.Fatalf("Failed to parse Redis URI: %v", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Connected to Redis")

	return &Redis{client: client}
}

// Close closes the Redis connection.
func (r *Redis) Close() {
	if err := r.client.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}
	log.Println("Disconnected from Redis")
}

// Set stores a value in cache with TTL.
func (r *Redis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(ctx, key, data, ttl).Err()
}

// Get retrieves a value from cache.
// Returns false if key doesn't exist.
func (r *Redis) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil // Key doesn't exist
		}
		return false, err
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return true, nil
}

// Delete removes a key from cache.
func (r *Redis) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// UserCacheKey generates a cache key for a user.
func UserCacheKey(userID string) string {
	return fmt.Sprintf("user:%s", userID)
}

// RefreshTokenCacheKey generates a cache key for a refresh token.
func RefreshTokenCacheKey(token string) string {
	return fmt.Sprintf("refresh:%s", token)
}

// SetRefreshToken stores a refresh token in cache.
func (r *Redis) SetRefreshToken(ctx context.Context, token string, userID string, ttl time.Duration) error {
	return r.client.Set(ctx, RefreshTokenCacheKey(token), userID, ttl).Err()
}

// GetRefreshToken retrieves a user ID from a refresh token.
// Returns empty string if token doesn't exist.
func (r *Redis) GetRefreshToken(ctx context.Context, token string) (string, error) {
	userID, err := r.client.Get(ctx, RefreshTokenCacheKey(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", nil // Key doesn't exist
		}
		return "", err
	}
	return userID, nil
}

// DeleteRefreshToken removes a refresh token from cache.
func (r *Redis) DeleteRefreshToken(ctx context.Context, token string) error {
	return r.client.Del(ctx, RefreshTokenCacheKey(token)).Err()
}
