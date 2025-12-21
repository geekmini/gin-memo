//go:build api

package testdb

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// RedisContainer wraps a Redis testcontainer for API tests.
type RedisContainer struct {
	Container testcontainers.Container
	URI       string
	Client    *redis.Client
}

// SetupRedis starts a Redis testcontainer.
func SetupRedis(ctx context.Context) (*RedisContainer, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	uri := host + ":" + port.Port()

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: uri,
	})

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		_ = container.Terminate(ctx)
		return nil, err
	}

	return &RedisContainer{
		Container: container,
		URI:       uri,
		Client:    client,
	}, nil
}

// Cleanup terminates the Redis container.
func (rc *RedisContainer) Cleanup(ctx context.Context) error {
	if rc.Client != nil {
		_ = rc.Client.Close()
	}
	if rc.Container != nil {
		return rc.Container.Terminate(ctx)
	}
	return nil
}

// FlushDB clears all keys from Redis.
func (rc *RedisContainer) FlushDB(ctx context.Context) error {
	return rc.Client.FlushDB(ctx).Err()
}
