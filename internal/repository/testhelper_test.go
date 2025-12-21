package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestDB holds the test database and container.
type TestDB struct {
	Container *mongodb.MongoDBContainer
	Client    *mongo.Client
	Database  *mongo.Database
}

// SetupTestDB creates a MongoDB test container and returns a connected database.
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	ctx := context.Background()

	// Start MongoDB container
	container, err := mongodb.Run(ctx, "mongo:7.0")
	require.NoError(t, err, "Failed to start MongoDB container")

	// Get connection string
	connectionString, err := container.ConnectionString(ctx)
	require.NoError(t, err, "Failed to get connection string")

	// Connect to MongoDB
	clientOpts := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(ctx, clientOpts)
	require.NoError(t, err, "Failed to connect to MongoDB")

	// Ping to verify connection
	err = client.Ping(ctx, nil)
	require.NoError(t, err, "Failed to ping MongoDB")

	// Create test database with unique name to avoid conflicts
	dbName := "test_" + time.Now().Format("20060102150405")
	db := client.Database(dbName)

	return &TestDB{
		Container: container,
		Client:    client,
		Database:  db,
	}
}

// Cleanup cleans up the test database and container.
func (tdb *TestDB) Cleanup(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	if tdb.Database != nil {
		_ = tdb.Database.Drop(ctx)
	}

	if tdb.Client != nil {
		_ = tdb.Client.Disconnect(ctx)
	}

	if tdb.Container != nil {
		_ = tdb.Container.Terminate(ctx)
	}
}

// ClearCollection removes all documents from a collection.
func (tdb *TestDB) ClearCollection(t *testing.T, collectionName string) {
	t.Helper()

	ctx := context.Background()
	_, err := tdb.Database.Collection(collectionName).DeleteMany(ctx, map[string]interface{}{})
	require.NoError(t, err, "Failed to clear collection %s", collectionName)
}
