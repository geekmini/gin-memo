package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	ServerPort    string
	GinMode       string
	MongoURI      string
	MongoDatabase string
	RedisURI      string
	JWTSecret     string
	JWTExpiry     time.Duration
}

// Load reads configuration from .env file and environment variables
func Load() *Config {
	// Load .env file (ignore error if file doesn't exist - env vars may be set directly)
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		GinMode:       getEnv("GIN_MODE", "debug"),
		MongoURI:      getEnvRequired("MONGO_URI"),
		MongoDatabase: getEnvRequired("MONGO_DATABASE"),
		RedisURI:      getEnv("REDIS_URI", "localhost:6379"),
		JWTSecret:     getEnvRequired("JWT_SECRET"),
		JWTExpiry:     parseDuration(getEnv("JWT_EXPIRY", "24h")),
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
