package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gin-sample/internal/cache"
	"gin-sample/internal/cache/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewRefreshTokenStore(t *testing.T) {
	t.Run("creates store with cache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		store := cache.NewRefreshTokenStore(mockCache)

		assert.NotNil(t, store)
	})
}

func TestRefreshTokenStore_Create(t *testing.T) {
	ctx := context.Background()
	familyID := "test-family-123"
	ttl := 24 * time.Hour
	data := &cache.RefreshTokenData{
		UserID:           "user123",
		CurrentTokenHash: "hash123",
		ExpiresAt:        time.Now().Add(ttl),
		CreatedAt:        time.Now(),
	}

	t.Run("creates refresh token successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		mockCache.EXPECT().
			Set(ctx, "refresh_token:test-family-123", data, ttl).
			Return(nil)

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.Create(ctx, familyID, data, ttl)

		require.NoError(t, err)
	})

	t.Run("returns error when cache set fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		expectedErr := errors.New("cache error")
		mockCache.EXPECT().
			Set(ctx, "refresh_token:test-family-123", data, ttl).
			Return(expectedErr)

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.Create(ctx, familyID, data, ttl)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestRefreshTokenStore_Get(t *testing.T) {
	ctx := context.Background()
	familyID := "test-family-123"

	t.Run("returns data when found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		expectedData := cache.RefreshTokenData{
			UserID:           "user123",
			CurrentTokenHash: "hash123",
			ExpiresAt:        time.Now().Add(24 * time.Hour),
			CreatedAt:        time.Now(),
		}

		mockCache := mocks.NewMockCache(ctrl)
		mockCache.EXPECT().
			Get(ctx, "refresh_token:test-family-123", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, dest any) (bool, error) {
				// Copy data to destination
				d := dest.(*cache.RefreshTokenData)
				*d = expectedData
				return true, nil
			})

		store := cache.NewRefreshTokenStore(mockCache)
		data, err := store.Get(ctx, familyID)

		require.NoError(t, err)
		assert.NotNil(t, data)
		assert.Equal(t, expectedData.UserID, data.UserID)
		assert.Equal(t, expectedData.CurrentTokenHash, data.CurrentTokenHash)
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		mockCache.EXPECT().
			Get(ctx, "refresh_token:test-family-123", gomock.Any()).
			Return(false, nil)

		store := cache.NewRefreshTokenStore(mockCache)
		data, err := store.Get(ctx, familyID)

		require.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("returns error when cache fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		expectedErr := errors.New("cache error")
		mockCache.EXPECT().
			Get(ctx, "refresh_token:test-family-123", gomock.Any()).
			Return(false, expectedErr)

		store := cache.NewRefreshTokenStore(mockCache)
		data, err := store.Get(ctx, familyID)

		assert.Error(t, err)
		assert.Nil(t, data)
	})
}

func TestRefreshTokenStore_Rotate_Fallback(t *testing.T) {
	ctx := context.Background()
	familyID := "test-family-123"
	newTokenHash := "new-hash-456"
	ttl := 24 * time.Hour

	t.Run("rotates token hashes successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		existingData := cache.RefreshTokenData{
			UserID:           "user123",
			CurrentTokenHash: "old-hash-123",
			ExpiresAt:        time.Now().Add(ttl),
			CreatedAt:        time.Now(),
		}

		mockCache := mocks.NewMockCache(ctrl)
		// Get existing data
		mockCache.EXPECT().
			Get(ctx, "refresh_token:test-family-123", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, dest any) (bool, error) {
				d := dest.(*cache.RefreshTokenData)
				*d = existingData
				return true, nil
			})
		// Set rotated data
		mockCache.EXPECT().
			Set(ctx, "refresh_token:test-family-123", gomock.Any(), ttl).
			DoAndReturn(func(_ context.Context, _ string, data any, _ time.Duration) error {
				d := data.(*cache.RefreshTokenData)
				// Verify rotation happened correctly
				assert.Equal(t, newTokenHash, d.CurrentTokenHash)
				assert.Equal(t, existingData.CurrentTokenHash, d.PreviousTokenHash)
				return nil
			})

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.Rotate(ctx, familyID, newTokenHash, ttl)

		require.NoError(t, err)
	})

	t.Run("returns error when family not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		mockCache.EXPECT().
			Get(ctx, "refresh_token:test-family-123", gomock.Any()).
			Return(false, nil)

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.Rotate(ctx, familyID, newTokenHash, ttl)

		assert.Error(t, err)
		assert.ErrorIs(t, err, cache.ErrRefreshTokenFamilyNotFound)
	})

	t.Run("returns error when get fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		expectedErr := errors.New("cache error")
		mockCache.EXPECT().
			Get(ctx, "refresh_token:test-family-123", gomock.Any()).
			Return(false, expectedErr)

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.Rotate(ctx, familyID, newTokenHash, ttl)

		assert.Error(t, err)
	})

	t.Run("returns error when set fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		existingData := cache.RefreshTokenData{
			UserID:           "user123",
			CurrentTokenHash: "old-hash-123",
			ExpiresAt:        time.Now().Add(ttl),
			CreatedAt:        time.Now(),
		}

		mockCache := mocks.NewMockCache(ctrl)
		mockCache.EXPECT().
			Get(ctx, "refresh_token:test-family-123", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, dest any) (bool, error) {
				d := dest.(*cache.RefreshTokenData)
				*d = existingData
				return true, nil
			})
		expectedErr := errors.New("set error")
		mockCache.EXPECT().
			Set(ctx, "refresh_token:test-family-123", gomock.Any(), ttl).
			Return(expectedErr)

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.Rotate(ctx, familyID, newTokenHash, ttl)

		assert.Error(t, err)
	})
}

func TestRefreshTokenStore_Delete(t *testing.T) {
	ctx := context.Background()
	familyID := "test-family-123"

	t.Run("deletes refresh token successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		mockCache.EXPECT().
			Delete(ctx, "refresh_token:test-family-123").
			Return(nil)

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.Delete(ctx, familyID)

		require.NoError(t, err)
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		expectedErr := errors.New("delete error")
		mockCache.EXPECT().
			Delete(ctx, "refresh_token:test-family-123").
			Return(expectedErr)

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.Delete(ctx, familyID)

		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})
}

func TestRefreshTokenStore_DeleteAllByUserID(t *testing.T) {
	ctx := context.Background()
	userID := "user123"

	t.Run("returns nil for non-Redis client (fallback)", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCache := mocks.NewMockCache(ctrl)
		// No Redis client, so it should just return nil

		store := cache.NewRefreshTokenStore(mockCache)
		err := store.DeleteAllByUserID(ctx, userID)

		require.NoError(t, err)
	})
}
