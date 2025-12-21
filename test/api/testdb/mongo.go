//go:build api

package testdb

import (
	"context"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoContainer wraps a MongoDB testcontainer for API tests.
type MongoContainer struct {
	Container *mongodb.MongoDBContainer
	URI       string
	Client    *mongo.Client
	Database  *mongo.Database
}

// SetupMongoDB starts a MongoDB testcontainer.
// Unlike the integration test version, this doesn't use t.Cleanup since we manage
// lifecycle in TestMain.
func SetupMongoDB(ctx context.Context, dbName string) (*MongoContainer, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	container, err := mongodb.Run(ctx, "mongo:7")
	if err != nil {
		return nil, err
	}

	uri, err := container.ConnectionString(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		_ = container.Terminate(ctx)
		return nil, err
	}

	database := client.Database(dbName)

	return &MongoContainer{
		Container: container,
		URI:       uri,
		Client:    client,
		Database:  database,
	}, nil
}

// Cleanup terminates the MongoDB container.
func (mc *MongoContainer) Cleanup(ctx context.Context) error {
	if mc.Client != nil {
		_ = mc.Client.Disconnect(ctx)
	}
	if mc.Container != nil {
		return mc.Container.Terminate(ctx)
	}
	return nil
}

// CleanupCollections drops all collections in the database.
func (mc *MongoContainer) CleanupCollections(ctx context.Context) error {
	collections, err := mc.Database.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		return err
	}
	for _, collection := range collections {
		if err := mc.Database.Collection(collection).Drop(ctx); err != nil {
			return err
		}
	}
	return nil
}
