// Package config handles application configuration from environment variables.
package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	ServerPort         string
	GinMode            string
	MongoURI           string
	MongoDatabase      string
	RedisURI           string
	AccessTokenSecret  string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	S3Endpoint         string
	S3AccessKey        string
	S3SecretKey        string
	S3Bucket           string
	S3UseSSL           bool
}

// Load reads configuration from .env file and environment variables
func Load() *Config {
	// Load .env file (ignore error if file doesn't exist - env vars may be set directly)
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		GinMode:            getEnv("GIN_MODE", "debug"),
		MongoURI:           getEnvRequired("MONGO_URI"),
		MongoDatabase:      getEnvRequired("MONGO_DATABASE"),
		RedisURI:           getEnv("REDIS_URI", "localhost:6379"),
		AccessTokenSecret:  getEnvRequired("ACCESS_TOKEN_SECRET"),
		AccessTokenExpiry:  parseDuration(getEnv("ACCESS_TOKEN_EXPIRY", "15m")),
		RefreshTokenExpiry: parseDuration(getEnv("REFRESH_TOKEN_EXPIRY", "168h")),
		S3Endpoint:         getEnv("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey:        getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:        getEnv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:           getEnv("S3_BUCKET", "voice-memos"),
		S3UseSSL:           getEnv("S3_USE_SSL", "false") == "true",
	}

	return cfg
}

// getEnv reads an environment variable with a fallback default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvRequired reads an environment variable and panics if not set
func getEnvRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// parseDuration parses a duration string, panics on error
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Fatalf("Invalid duration format: %s", s)
	}
	return d
}
