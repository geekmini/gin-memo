// Package database provides database connection and management.
package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB holds the database connection
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// NewMongoDB creates a new MongoDB connection
func NewMongoDB(uri, dbName string) *MongoDB {
	// Create a context with timeout for connection
	// Context = carries deadlines, cancellation signals, and request-scoped values
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // Always call cancel to release resources

	// Configure client options
	clientOptions := options.Client().ApplyURI(uri)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	log.Printf("Connected to MongoDB: %s", dbName)

	return &MongoDB{
		Client:   client,
		Database: client.Database(dbName),
	}
}

// Close disconnects from MongoDB
func (m *MongoDB) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := m.Client.Disconnect(ctx); err != nil {
		log.Printf("Error disconnecting from MongoDB: %v", err)
	}
	log.Println("Disconnected from MongoDB")
}

// Collection returns a collection from the database
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}
