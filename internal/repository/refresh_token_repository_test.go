package repository

import (
	"context"
	"testing"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewRefreshTokenRepository(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewRefreshTokenRepository(tdb.Database)

	assert.NotNil(t, repo)
}

func TestRefreshTokenRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewRefreshTokenRepository(tdb.Database)
	ctx := context.Background()

	t.Run("successfully creates refresh token", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		token := &models.RefreshToken{
			Token:     "rf_test_token_123",
			UserID:    primitive.NewObjectID(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		err := repo.Create(ctx, token)

		require.NoError(t, err)
		assert.False(t, token.ID.IsZero())
		assert.NotZero(t, token.CreatedAt)
	})

	t.Run("creates multiple tokens for same user", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		userID := primitive.NewObjectID()

		token1 := &models.RefreshToken{
			Token:     "rf_token_1",
			UserID:    userID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		token2 := &models.RefreshToken{
			Token:     "rf_token_2",
			UserID:    userID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}

		err := repo.Create(ctx, token1)
		require.NoError(t, err)

		err = repo.Create(ctx, token2)
		require.NoError(t, err)
	})
}

func TestRefreshTokenRepository_FindByToken(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewRefreshTokenRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds valid token", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		userID := primitive.NewObjectID()
		token := &models.RefreshToken{
			Token:     "rf_findme_token",
			UserID:    userID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		found, err := repo.FindByToken(ctx, "rf_findme_token")

		require.NoError(t, err)
		assert.Equal(t, token.Token, found.Token)
		assert.Equal(t, userID, found.UserID)
	})

	t.Run("returns error for non-existent token", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		found, err := repo.FindByToken(ctx, "rf_nonexistent")

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrInvalidRefreshToken, err)
	})

	t.Run("returns error for expired token", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		token := &models.RefreshToken{
			Token:     "rf_expired_token",
			UserID:    primitive.NewObjectID(),
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		found, err := repo.FindByToken(ctx, "rf_expired_token")

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrInvalidRefreshToken, err)
	})
}

func TestRefreshTokenRepository_DeleteByToken(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewRefreshTokenRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes existing token", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		token := &models.RefreshToken{
			Token:     "rf_delete_me",
			UserID:    primitive.NewObjectID(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := repo.Create(ctx, token)
		require.NoError(t, err)

		err = repo.DeleteByToken(ctx, "rf_delete_me")
		require.NoError(t, err)

		// Verify deletion
		found, err := repo.FindByToken(ctx, "rf_delete_me")
		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrInvalidRefreshToken, err)
	})

	t.Run("succeeds for non-existent token", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		err := repo.DeleteByToken(ctx, "rf_nonexistent")

		// Should not return error for non-existent token
		assert.NoError(t, err)
	})
}

func TestRefreshTokenRepository_DeleteByUserID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewRefreshTokenRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes all tokens for user", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		userID := primitive.NewObjectID()

		// Create multiple tokens for same user
		for i := 0; i < 3; i++ {
			token := &models.RefreshToken{
				Token:     "rf_user_token_" + string(rune('a'+i)),
				UserID:    userID,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}
			err := repo.Create(ctx, token)
			require.NoError(t, err)
		}

		// Create token for different user
		otherToken := &models.RefreshToken{
			Token:     "rf_other_user_token",
			UserID:    primitive.NewObjectID(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := repo.Create(ctx, otherToken)
		require.NoError(t, err)

		// Delete all tokens for first user
		err = repo.DeleteByUserID(ctx, userID)
		require.NoError(t, err)

		// Verify user's tokens are deleted
		for i := 0; i < 3; i++ {
			found, err := repo.FindByToken(ctx, "rf_user_token_"+string(rune('a'+i)))
			assert.Nil(t, found)
			assert.Equal(t, apperrors.ErrInvalidRefreshToken, err)
		}

		// Verify other user's token still exists
		found, err := repo.FindByToken(ctx, "rf_other_user_token")
		require.NoError(t, err)
		assert.NotNil(t, found)
	})

	t.Run("succeeds when user has no tokens", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		err := repo.DeleteByUserID(ctx, primitive.NewObjectID())

		assert.NoError(t, err)
	})
}

func TestRefreshTokenRepository_FindAllByUserID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewRefreshTokenRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds all tokens for user", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		userID := primitive.NewObjectID()

		// Create multiple tokens for same user
		for i := 0; i < 3; i++ {
			token := &models.RefreshToken{
				Token:     "rf_findall_token_" + string(rune('a'+i)),
				UserID:    userID,
				ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
			}
			err := repo.Create(ctx, token)
			require.NoError(t, err)
		}

		// Create token for different user (should not be included)
		otherToken := &models.RefreshToken{
			Token:     "rf_other_user_findall",
			UserID:    primitive.NewObjectID(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := repo.Create(ctx, otherToken)
		require.NoError(t, err)

		// Find all tokens for first user
		tokens, err := repo.FindAllByUserID(ctx, userID)

		require.NoError(t, err)
		assert.Len(t, tokens, 3)
		for _, token := range tokens {
			assert.Equal(t, userID, token.UserID)
		}
	})

	t.Run("returns empty slice when user has no tokens", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		tokens, err := repo.FindAllByUserID(ctx, primitive.NewObjectID())

		require.NoError(t, err)
		assert.Empty(t, tokens)
	})

	t.Run("excludes expired tokens", func(t *testing.T) {
		tdb.ClearCollection(t, "refresh_tokens")

		userID := primitive.NewObjectID()

		// Create valid token
		validToken := &models.RefreshToken{
			Token:     "rf_valid_token",
			UserID:    userID,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
		err := repo.Create(ctx, validToken)
		require.NoError(t, err)

		// Create expired token
		expiredToken := &models.RefreshToken{
			Token:     "rf_expired_findall",
			UserID:    userID,
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		err = repo.Create(ctx, expiredToken)
		require.NoError(t, err)

		// Find all should only return valid token
		tokens, err := repo.FindAllByUserID(ctx, userID)

		require.NoError(t, err)
		assert.Len(t, tokens, 1)
		assert.Equal(t, "rf_valid_token", tokens[0].Token)
	})
}
