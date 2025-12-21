//go:build integration

package testdb

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoContainer wraps a MongoDB testcontainer.
type MongoContainer struct {
	Container *mongodb.MongoDBContainer
	URI       string
	Client    *mongo.Client
	Database  *mongo.Database
}

// SetupMongoDB starts a MongoDB testcontainer for integration tests.
func SetupMongoDB(t *testing.T, dbName string) *MongoContainer {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}

	database := client.Database(dbName)

	t.Cleanup(func() {
		ctx := context.Background()
		_ = client.Disconnect(ctx)
		_ = container.Terminate(ctx)
	})

	return &MongoContainer{
		Container: container,
		URI:       uri,
		Client:    client,
		Database:  database,
	}
}

// CleanupCollections drops all collections in the database.
func (mc *MongoContainer) CleanupCollections(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	collections, err := mc.Database.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to list collections: %v", err)
	}
	for _, collection := range collections {
		if err := mc.Database.Collection(collection).Drop(ctx); err != nil {
			t.Fatalf("Failed to drop collection %s: %v", collection, err)
		}
	}
}
