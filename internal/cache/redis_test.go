package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		expected string
	}{
		{"simple id", "123", "user:123"},
		{"uuid format", "550e8400-e29b-41d4-a716-446655440000", "user:550e8400-e29b-41d4-a716-446655440000"},
		{"objectid format", "507f1f77bcf86cd799439011", "user:507f1f77bcf86cd799439011"},
		{"empty string", "", "user:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UserCacheKey(tt.userID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRefreshTokenCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{"simple token", "abc123", "refresh:abc123"},
		{"jwt-like token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", "refresh:eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"},
		{"with special chars", "rf_abc-123_xyz", "refresh:rf_abc-123_xyz"},
		{"empty string", "", "refresh:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RefreshTokenCacheKey(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}
