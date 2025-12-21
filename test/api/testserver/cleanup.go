//go:build api

package testserver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// CleanupBetweenTests clears all data between tests.
// Call this at the start of each test function for isolation.
func (ts *TestServer) CleanupBetweenTests(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	// Clear MongoDB collections
	err := ts.MongoDB.CleanupCollections(ctx)
	require.NoError(t, err, "failed to cleanup MongoDB collections")

	// Clear Redis
	err = ts.Redis.FlushDB(ctx)
	require.NoError(t, err, "failed to flush Redis")

	// Clear MinIO bucket
	err = ts.MinIO.ClearBucket(ctx)
	require.NoError(t, err, "failed to clear MinIO bucket")
}

// CleanupMongoDB clears only MongoDB collections.
func (ts *TestServer) CleanupMongoDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	err := ts.MongoDB.CleanupCollections(ctx)
	require.NoError(t, err, "failed to cleanup MongoDB collections")
}

// CleanupRedis clears only Redis.
func (ts *TestServer) CleanupRedis(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	err := ts.Redis.FlushDB(ctx)
	require.NoError(t, err, "failed to flush Redis")
}

// CleanupMinIO clears only MinIO bucket.
func (ts *TestServer) CleanupMinIO(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	err := ts.MinIO.ClearBucket(ctx)
	require.NoError(t, err, "failed to clear MinIO bucket")
}
