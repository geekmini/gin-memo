package main

import (
	"context"
	"log"
	"time"

	"gin-sample/internal/config"
	"gin-sample/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	log.Println("Starting migration...")

	cfg := config.Load()

	mongoDB := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase)
	defer mongoDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createIndexes(ctx, mongoDB.Database)

	log.Println("Migration completed successfully!")
}

func createIndexes(ctx context.Context, db *mongo.Database) {
	// Users indexes
	createIndex(ctx, db, "users", bson.D{{Key: "email", Value: 1}}, &options.IndexOptions{
		Unique: ptrBool(true),
	})

	// Teams indexes
	createIndex(ctx, db, "teams", bson.D{{Key: "slug", Value: 1}}, &options.IndexOptions{
		Unique: ptrBool(true),
	})
	createIndex(ctx, db, "teams", bson.D{{Key: "ownerId", Value: 1}}, nil)
	createIndex(ctx, db, "teams", bson.D{{Key: "deletedAt", Value: 1}}, nil)

	// Team members indexes
	createIndex(ctx, db, "team_members", bson.D{
		{Key: "teamId", Value: 1},
		{Key: "userId", Value: 1},
	}, &options.IndexOptions{
		Unique: ptrBool(true),
	})
	createIndex(ctx, db, "team_members", bson.D{{Key: "userId", Value: 1}}, nil)

	// Team invitations indexes
	createIndex(ctx, db, "team_invitations", bson.D{
		{Key: "teamId", Value: 1},
		{Key: "email", Value: 1},
	}, nil)
	createIndex(ctx, db, "team_invitations", bson.D{{Key: "email", Value: 1}}, nil)
	createIndex(ctx, db, "team_invitations", bson.D{{Key: "expiresAt", Value: 1}}, nil)

	// Voice memos indexes
	createIndex(ctx, db, "voice_memos", bson.D{{Key: "userId", Value: 1}}, nil)
	createIndex(ctx, db, "voice_memos", bson.D{
		{Key: "teamId", Value: 1},
		{Key: "createdAt", Value: -1},
	}, nil)
	createIndex(ctx, db, "voice_memos", bson.D{{Key: "deletedAt", Value: 1}}, nil)

	// Refresh tokens indexes
	createIndex(ctx, db, "refresh_tokens", bson.D{{Key: "userId", Value: 1}}, nil)
	createIndex(ctx, db, "refresh_tokens", bson.D{{Key: "expiresAt", Value: 1}}, nil)
}

func createIndex(ctx context.Context, db *mongo.Database, collection string, keys bson.D, opts *options.IndexOptions) {
	indexModel := mongo.IndexModel{
		Keys:    keys,
		Options: opts,
	}

	name, err := db.Collection(collection).Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Warning: Failed to create index on %s: %v", collection, err)
		return
	}

	log.Printf("Created index %s on %s", name, collection)
}

func ptrBool(b bool) *bool {
	return &b
}
