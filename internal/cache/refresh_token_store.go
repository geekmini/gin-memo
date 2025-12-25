// Package cache provides caching functionality using Redis.
package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrRefreshTokenFamilyNotFound is returned when a refresh token family does not exist.
var ErrRefreshTokenFamilyNotFound = errors.New("refresh token family not found")

// RefreshTokenData represents the data stored in Redis for a refresh token family.
type RefreshTokenData struct {
	UserID            string    `json:"user_id"`
	CurrentTokenHash  string    `json:"current_token_hash"`
	PreviousTokenHash string    `json:"previous_token_hash,omitempty"`
	ExpiresAt         time.Time `json:"expires_at"`
	CreatedAt         time.Time `json:"created_at"`
}

// RefreshTokenStore manages refresh token storage in Redis.
type RefreshTokenStore interface {
	// Create stores a new refresh token family.
	Create(ctx context.Context, familyID string, data *RefreshTokenData, ttl time.Duration) error
	// Get retrieves refresh token data by family ID.
	Get(ctx context.Context, familyID string) (*RefreshTokenData, error)
	// Rotate updates the token hashes for rotation.
	Rotate(ctx context.Context, familyID string, newTokenHash string, ttl time.Duration) error
	// Delete removes a refresh token family.
	Delete(ctx context.Context, familyID string) error
	// DeleteAllByUserID removes all refresh token families for a user.
	DeleteAllByUserID(ctx context.Context, userID string) error
}

// RedisClientProvider provides access to the underlying Redis client.
type RedisClientProvider interface {
	Client() *redis.Client
}

type refreshTokenStore struct {
	cache  Cache
	client *redis.Client
}

// NewRefreshTokenStore creates a new RefreshTokenStore.
// The cache must implement RedisClientProvider (e.g., *Redis) to support atomic operations.
func NewRefreshTokenStore(cache Cache) RefreshTokenStore {
	store := &refreshTokenStore{cache: cache}
	if provider, ok := cache.(RedisClientProvider); ok {
		store.client = provider.Client()
	}
	return store
}

// refreshTokenFamilyKey generates a cache key for a refresh token family.
func refreshTokenFamilyKey(familyID string) string {
	return fmt.Sprintf("refresh_token:%s", familyID)
}

// userRefreshTokensKey generates a cache key for a user's refresh token families set.
func userRefreshTokensKey(userID string) string {
	return fmt.Sprintf("user_refresh_tokens:%s", userID)
}

// Create stores a new refresh token family.
func (s *refreshTokenStore) Create(ctx context.Context, familyID string, data *RefreshTokenData, ttl time.Duration) error {
	if err := s.cache.Set(ctx, refreshTokenFamilyKey(familyID), data, ttl); err != nil {
		return err
	}

	// Add family ID to user's set for logout-all support
	if s.client != nil {
		userKey := userRefreshTokensKey(data.UserID)
		if err := s.client.SAdd(ctx, userKey, familyID).Err(); err != nil {
			// Best-effort: don't fail the create if index update fails
			return nil
		}
		// Set expiry on the user's set to match the longest possible token TTL
		_ = s.client.Expire(ctx, userKey, ttl)
	}

	return nil
}

// Get retrieves refresh token data by family ID.
func (s *refreshTokenStore) Get(ctx context.Context, familyID string) (*RefreshTokenData, error) {
	var data RefreshTokenData
	found, err := s.cache.Get(ctx, refreshTokenFamilyKey(familyID), &data)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &data, nil
}

// rotateScript is a Lua script that atomically rotates refresh token hashes.
// It reads the current data, rotates hashes, and writes back in one atomic operation.
var rotateScript = redis.NewScript(`
local key = KEYS[1]
local newTokenHash = ARGV[1]
local ttlSeconds = tonumber(ARGV[2])

-- Get existing data
local data = redis.call('GET', key)
if not data then
    return redis.error_reply("refresh token family not found")
end

-- Parse JSON data
local decoded = cjson.decode(data)

-- Rotate hashes: current becomes previous, new becomes current
decoded.previous_token_hash = decoded.current_token_hash
decoded.current_token_hash = newTokenHash

-- Encode and store with TTL
local encoded = cjson.encode(decoded)
redis.call('SET', key, encoded, 'EX', ttlSeconds)

return "OK"
`)

// Rotate updates the token hashes for rotation (current becomes previous, new becomes current).
// This operation is atomic to prevent race conditions.
func (s *refreshTokenStore) Rotate(ctx context.Context, familyID string, newTokenHash string, ttl time.Duration) error {
	if s.client != nil {
		// Use atomic Lua script
		key := refreshTokenFamilyKey(familyID)
		ttlSeconds := int(ttl.Seconds())
		_, err := rotateScript.Run(ctx, s.client, []string{key}, newTokenHash, ttlSeconds).Result()
		if err != nil {
			if err.Error() == "refresh token family not found" {
				return ErrRefreshTokenFamilyNotFound
			}
			return fmt.Errorf("rotate script failed: %w", err)
		}
		return nil
	}

	// Fallback for non-Redis clients (e.g., mocks in tests)
	return s.rotateFallback(ctx, familyID, newTokenHash, ttl)
}

// rotateFallback provides non-atomic rotation for testing/mocking scenarios.
func (s *refreshTokenStore) rotateFallback(ctx context.Context, familyID string, newTokenHash string, ttl time.Duration) error {
	data, err := s.Get(ctx, familyID)
	if err != nil {
		return err
	}
	if data == nil {
		return ErrRefreshTokenFamilyNotFound
	}

	data.PreviousTokenHash = data.CurrentTokenHash
	data.CurrentTokenHash = newTokenHash

	return s.cache.Set(ctx, refreshTokenFamilyKey(familyID), data, ttl)
}

// Delete removes a refresh token family and updates the user index.
func (s *refreshTokenStore) Delete(ctx context.Context, familyID string) error {
	// Get the token data first to find the user ID for index cleanup
	if s.client != nil {
		data, err := s.Get(ctx, familyID)
		if err == nil && data != nil {
			// Remove family ID from user's set (best-effort)
			userKey := userRefreshTokensKey(data.UserID)
			_ = s.client.SRem(ctx, userKey, familyID)
		}
	}

	return s.cache.Delete(ctx, refreshTokenFamilyKey(familyID))
}

// DeleteAllByUserID removes all refresh token families for a user.
func (s *refreshTokenStore) DeleteAllByUserID(ctx context.Context, userID string) error {
	if s.client == nil {
		// Fallback: no-op for non-Redis clients (mocks)
		return nil
	}

	userKey := userRefreshTokensKey(userID)

	// Get all family IDs for this user
	familyIDs, err := s.client.SMembers(ctx, userKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user token families: %w", err)
	}

	// Delete each token family
	for _, familyID := range familyIDs {
		_ = s.cache.Delete(ctx, refreshTokenFamilyKey(familyID))
	}

	// Delete the user's family set
	_ = s.client.Del(ctx, userKey)

	return nil
}
