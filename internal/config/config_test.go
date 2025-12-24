package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEnv(t *testing.T) {
	t.Run("returns environment variable value when set", func(t *testing.T) {
		t.Setenv("TEST_CONFIG_VAR", "custom_value")

		result := getEnv("TEST_CONFIG_VAR", "default_value")

		assert.Equal(t, "custom_value", result)
	})

	t.Run("returns default value when env var not set", func(t *testing.T) {
		result := getEnv("NONEXISTENT_CONFIG_VAR_12345", "default_value")

		assert.Equal(t, "default_value", result)
	})

	t.Run("returns default value when env var is empty string", func(t *testing.T) {
		t.Setenv("EMPTY_CONFIG_VAR", "")

		result := getEnv("EMPTY_CONFIG_VAR", "default_value")

		assert.Equal(t, "default_value", result)
	})
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"minutes", "15m", 15 * time.Minute},
		{"hours", "168h", 168 * time.Hour},
		{"seconds", "30s", 30 * time.Second},
		{"combined", "1h30m", 90 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoad(t *testing.T) {
	t.Run("loads config with all required env vars", func(t *testing.T) {
		// Set required env vars
		t.Setenv("MONGO_URI", "mongodb://localhost:27017")
		t.Setenv("MONGO_DATABASE", "testdb")
		t.Setenv("ACCESS_TOKEN_SECRET", "test-secret-key")

		// Set optional env vars to test custom values
		t.Setenv("SERVER_PORT", "3000")
		t.Setenv("GIN_MODE", "release")
		t.Setenv("REDIS_URI", "redis.example.com:6379")
		t.Setenv("ACCESS_TOKEN_EXPIRY", "30m")
		t.Setenv("REFRESH_TOKEN_EXPIRY", "720h")
		t.Setenv("S3_ENDPOINT", "s3.example.com:9000")
		t.Setenv("S3_ACCESS_KEY", "myaccesskey")
		t.Setenv("S3_SECRET_KEY", "mysecretkey")
		t.Setenv("S3_BUCKET", "my-bucket")
		t.Setenv("S3_USE_SSL", "true")

		cfg := Load()

		require.NotNil(t, cfg)

		// Required fields
		assert.Equal(t, "mongodb://localhost:27017", cfg.MongoURI)
		assert.Equal(t, "testdb", cfg.MongoDatabase)
		assert.Equal(t, "test-secret-key", cfg.AccessTokenSecret)

		// Optional fields with custom values
		assert.Equal(t, "3000", cfg.ServerPort)
		assert.Equal(t, "release", cfg.GinMode)
		assert.Equal(t, "redis.example.com:6379", cfg.RedisURI)
		assert.Equal(t, 30*time.Minute, cfg.AccessTokenExpiry)
		assert.Equal(t, 720*time.Hour, cfg.RefreshTokenExpiry)
		assert.Equal(t, "s3.example.com:9000", cfg.S3Endpoint)
		assert.Equal(t, "myaccesskey", cfg.S3AccessKey)
		assert.Equal(t, "mysecretkey", cfg.S3SecretKey)
		assert.Equal(t, "my-bucket", cfg.S3Bucket)
		assert.True(t, cfg.S3UseSSL)
	})

	t.Run("uses default values for optional env vars", func(t *testing.T) {
		// Only set required env vars
		t.Setenv("MONGO_URI", "mongodb://localhost:27017")
		t.Setenv("MONGO_DATABASE", "testdb")
		t.Setenv("ACCESS_TOKEN_SECRET", "test-secret-key")

		cfg := Load()

		require.NotNil(t, cfg)

		// Check default values
		assert.Equal(t, "8080", cfg.ServerPort)
		assert.Equal(t, "debug", cfg.GinMode)
		assert.Equal(t, "localhost:6379", cfg.RedisURI)
		assert.Equal(t, 15*time.Minute, cfg.AccessTokenExpiry)
		assert.Equal(t, 168*time.Hour, cfg.RefreshTokenExpiry)
		assert.Equal(t, "localhost:9000", cfg.S3Endpoint)
		assert.Equal(t, "minioadmin", cfg.S3AccessKey)
		assert.Equal(t, "minioadmin", cfg.S3SecretKey)
		assert.Equal(t, "voice-memos", cfg.S3Bucket)
		assert.False(t, cfg.S3UseSSL)
	})

	t.Run("S3UseSSL is false for non-true values", func(t *testing.T) {
		t.Setenv("MONGO_URI", "mongodb://localhost:27017")
		t.Setenv("MONGO_DATABASE", "testdb")
		t.Setenv("ACCESS_TOKEN_SECRET", "test-secret-key")
		t.Setenv("S3_USE_SSL", "false")

		cfg := Load()

		assert.False(t, cfg.S3UseSSL)
	})
}
