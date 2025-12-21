package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTManager(t *testing.T) {
	t.Run("creates manager with valid config", func(t *testing.T) {
		manager := NewJWTManager("testsecret", 15*time.Minute)

		assert.NotNil(t, manager)
	})

	t.Run("creates manager with empty secret", func(t *testing.T) {
		manager := NewJWTManager("", 15*time.Minute)

		assert.NotNil(t, manager)
	})
}

func TestJWTManager_GenerateToken(t *testing.T) {
	manager := NewJWTManager("testsecret123", 15*time.Minute)

	t.Run("generates valid token for user ID", func(t *testing.T) {
		userID := "507f1f77bcf86cd799439011"

		token, err := manager.GenerateToken(userID)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		// Token should be a valid JWT format (3 parts separated by dots)
		assert.Regexp(t, `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`, token)
	})

	t.Run("generates different tokens for same user after time passes", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping slow test in short mode")
		}
		userID := "507f1f77bcf86cd799439011"

		token1, _ := manager.GenerateToken(userID)
		time.Sleep(1100 * time.Millisecond) // JWT timestamps have second granularity
		token2, _ := manager.GenerateToken(userID)

		assert.NotEqual(t, token1, token2, "tokens should have different timestamps")
	})

	t.Run("generates token for empty user ID", func(t *testing.T) {
		token, err := manager.GenerateToken("")

		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("token contains correct user ID", func(t *testing.T) {
		userID := "test-user-123"

		token, _ := manager.GenerateToken(userID)
		claims, err := manager.ValidateToken(token)

		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
	})
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := NewJWTManager("testsecret123", 15*time.Minute)

	t.Run("validates correctly signed token", func(t *testing.T) {
		userID := "507f1f77bcf86cd799439011"
		token, _ := manager.GenerateToken(userID)

		claims, err := manager.ValidateToken(token)

		require.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.UserID)
	})

	t.Run("returns error for expired token", func(t *testing.T) {
		// Create a manager with 1ms expiry
		shortManager := NewJWTManager("testsecret123", 1*time.Millisecond)
		token, _ := shortManager.GenerateToken("user123")

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		claims, err := shortManager.ValidateToken(token)

		assert.Error(t, err)
		assert.Nil(t, claims)
		assert.ErrorIs(t, err, jwt.ErrTokenExpired)
	})

	t.Run("returns error for wrong secret", func(t *testing.T) {
		manager1 := NewJWTManager("secret1", 15*time.Minute)
		manager2 := NewJWTManager("secret2", 15*time.Minute)

		token, _ := manager1.GenerateToken("user123")
		claims, err := manager2.ValidateToken(token)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("returns error for invalid token format", func(t *testing.T) {
		claims, err := manager.ValidateToken("not.a.valid.token")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("returns error for empty token", func(t *testing.T) {
		claims, err := manager.ValidateToken("")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("returns error for tampered token", func(t *testing.T) {
		token, _ := manager.GenerateToken("user123")
		// Tamper with the token by changing a character
		tamperedToken := token[:len(token)-5] + "XXXXX"

		claims, err := manager.ValidateToken(tamperedToken)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("validates token expiry time is set correctly", func(t *testing.T) {
		expiry := 30 * time.Minute
		manager := NewJWTManager("secret", expiry)
		beforeGeneration := time.Now()

		token, _ := manager.GenerateToken("user123")
		claims, err := manager.ValidateToken(token)

		require.NoError(t, err)
		// Expiry should be approximately expiry duration from now
		expectedExpiry := beforeGeneration.Add(expiry)
		assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 2*time.Second)
	})

	t.Run("validates issued at time is set", func(t *testing.T) {
		beforeGeneration := time.Now()

		token, _ := manager.GenerateToken("user123")
		claims, err := manager.ValidateToken(token)

		require.NoError(t, err)
		assert.WithinDuration(t, beforeGeneration, claims.IssuedAt.Time, 2*time.Second)
	})
}

func TestJWTManager_TokenManager_Interface(t *testing.T) {
	t.Run("JWTManager implements TokenManager interface", func(t *testing.T) {
		var _ TokenManager = (*JWTManager)(nil)
	})
}

func BenchmarkJWTManager_GenerateToken(b *testing.B) {
	manager := NewJWTManager("benchmarksecret", 15*time.Minute)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateToken("user123")
	}
}

func BenchmarkJWTManager_ValidateToken(b *testing.B) {
	manager := NewJWTManager("benchmarksecret", 15*time.Minute)
	token, _ := manager.GenerateToken("user123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateToken(token)
	}
}
