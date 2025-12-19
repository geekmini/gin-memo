package main

import (
	"bytes"
	"context"
	"log"
	"time"

	"gin-sample/internal/config"
	"gin-sample/internal/database"
	"gin-sample/internal/storage"
	"gin-sample/pkg/auth"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// SeedUser represents a user document for seeding.
type SeedUser struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Email     string             `bson:"email"`
	Password  string             `bson:"password"`
	Name      string             `bson:"name"`
	CreatedAt time.Time          `bson:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt"`
}

// SeedVoiceMemo represents a voice memo document for seeding.
type SeedVoiceMemo struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        primitive.ObjectID `bson:"userId"`
	Title         string             `bson:"title"`
	Transcription string             `bson:"transcription"`
	AudioFileKey  string             `bson:"audioFileKey"`
	Duration      int                `bson:"duration"`
	FileSize      int64              `bson:"fileSize"`
	AudioFormat   string             `bson:"audioFormat"`
	Tags          []string           `bson:"tags"`
	IsFavorite    bool               `bson:"isFavorite"`
	CreatedAt     time.Time          `bson:"createdAt"`
}

func main() {
	log.Println("Starting seed...")

	// Load config
	cfg := config.Load()

	// Connect to MongoDB
	mongoDB := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase)
	defer mongoDB.Close()

	// Connect to S3/MinIO
	s3Client := storage.NewS3Client(
		cfg.S3Endpoint,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
		cfg.S3Bucket,
		cfg.S3UseSSL,
	)

	ctx := context.Background()

	// Seed users
	userIDs := seedUsers(ctx, mongoDB.Database)

	// Seed voice memos and audio files
	seedVoiceMemos(ctx, mongoDB.Database, s3Client, userIDs)

	log.Println("Seed completed successfully!")
}

func seedUsers(ctx context.Context, db *mongo.Database) []primitive.ObjectID {
	collection := db.Collection("users")

	// Clear existing users
	_, err := collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to clear users: %v", err)
	}

	// Hash passwords
	password1, _ := auth.HashPassword("password123")
	password2, _ := auth.HashPassword("password456")

	now := time.Now()

	users := []interface{}{
		SeedUser{
			Email:     "alice@example.com",
			Password:  password1,
			Name:      "Alice Johnson",
			CreatedAt: now,
			UpdatedAt: now,
		},
		SeedUser{
			Email:     "bob@example.com",
			Password:  password2,
			Name:      "Bob Smith",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	result, err := collection.InsertMany(ctx, users)
	if err != nil {
		log.Fatalf("Failed to seed users: %v", err)
	}

	log.Printf("Seeded %d users", len(result.InsertedIDs))

	// Convert to ObjectIDs
	var userIDs []primitive.ObjectID
	for _, id := range result.InsertedIDs {
		userIDs = append(userIDs, id.(primitive.ObjectID))
	}

	return userIDs
}

func seedVoiceMemos(ctx context.Context, db *mongo.Database, s3Client *storage.S3Client, userIDs []primitive.ObjectID) {
	collection := db.Collection("voice_memos")

	// Clear existing voice memos
	_, err := collection.DeleteMany(ctx, bson.M{})
	if err != nil {
		log.Fatalf("Failed to clear voice memos: %v", err)
	}

	now := time.Now()

	// Voice memos for Alice (userIDs[0])
	aliceMemos := []SeedVoiceMemo{
		{
			UserID:        userIDs[0],
			Title:         "Meeting Notes - Q4 Planning",
			Transcription: "Today we discussed the Q4 roadmap. Key priorities include launching the new mobile app, improving API performance, and expanding to new markets. Action items: finalize designs by Friday, schedule technical review next week.",
			AudioFileKey:  "alice/meeting-notes-q4.mp3",
			Duration:      180,
			FileSize:      2890000,
			AudioFormat:   "mp3",
			Tags:          []string{"work", "meeting", "planning"},
			IsFavorite:    true,
			CreatedAt:     now.Add(-24 * time.Hour),
		},
		{
			UserID:        userIDs[0],
			Title:         "Grocery List",
			Transcription: "Need to buy: milk, eggs, bread, chicken breast, broccoli, rice, olive oil, and some snacks for the weekend.",
			AudioFileKey:  "alice/grocery-list.mp3",
			Duration:      45,
			FileSize:      720000,
			AudioFormat:   "mp3",
			Tags:          []string{"personal", "shopping"},
			IsFavorite:    false,
			CreatedAt:     now.Add(-12 * time.Hour),
		},
		{
			UserID:        userIDs[0],
			Title:         "Book Idea",
			Transcription: "New book idea: a mystery novel set in a small coastal town. The protagonist is a retired detective who gets pulled into one last case. Themes: redemption, community, secrets.",
			AudioFileKey:  "alice/book-idea.mp3",
			Duration:      90,
			FileSize:      1440000,
			AudioFormat:   "mp3",
			Tags:          []string{"creative", "writing"},
			IsFavorite:    true,
			CreatedAt:     now.Add(-6 * time.Hour),
		},
		{
			UserID:        userIDs[0],
			Title:         "Workout Routine",
			Transcription: "Monday: chest and triceps. Wednesday: back and biceps. Friday: legs and shoulders. Each session 45 minutes. Don't forget to stretch before and after.",
			AudioFileKey:  "alice/workout.mp3",
			Duration:      60,
			FileSize:      960000,
			AudioFormat:   "mp3",
			Tags:          []string{"fitness", "health"},
			IsFavorite:    false,
			CreatedAt:     now.Add(-2 * time.Hour),
		},
	}

	// Voice memos for Bob (userIDs[1])
	bobMemos := []SeedVoiceMemo{
		{
			UserID:        userIDs[1],
			Title:         "Project Status Update",
			Transcription: "Sprint 23 is on track. We completed 8 out of 10 story points. Two items moved to next sprint due to dependency issues. Team velocity is improving.",
			AudioFileKey:  "bob/project-status.mp3",
			Duration:      120,
			FileSize:      1920000,
			AudioFormat:   "mp3",
			Tags:          []string{"work", "agile", "status"},
			IsFavorite:    false,
			CreatedAt:     now.Add(-48 * time.Hour),
		},
		{
			UserID:        userIDs[1],
			Title:         "Birthday Gift Ideas",
			Transcription: "Ideas for Sarah's birthday: cookbook, spa gift card, pottery class voucher, or that bag she mentioned last month. Budget around 100 dollars.",
			AudioFileKey:  "bob/birthday-ideas.mp3",
			Duration:      35,
			FileSize:      560000,
			AudioFormat:   "mp3",
			Tags:          []string{"personal", "gift"},
			IsFavorite:    true,
			CreatedAt:     now.Add(-5 * time.Hour),
		},
	}

	// Combine all memos
	allMemos := append(aliceMemos, bobMemos...)

	// Upload placeholder audio files to S3/MinIO
	for _, memo := range allMemos {
		uploadPlaceholderAudio(ctx, s3Client, memo.AudioFileKey, memo.FileSize)
	}

	// Convert to []interface{} for InsertMany
	var memosToInsert []interface{}
	for _, memo := range allMemos {
		memosToInsert = append(memosToInsert, memo)
	}

	result, err := collection.InsertMany(ctx, memosToInsert)
	if err != nil {
		log.Fatalf("Failed to seed voice memos: %v", err)
	}

	log.Printf("Seeded %d voice memos", len(result.InsertedIDs))
}

// uploadPlaceholderAudio uploads a placeholder audio file to S3.
func uploadPlaceholderAudio(ctx context.Context, s3Client *storage.S3Client, key string, size int64) {
	// Create placeholder content (simulated audio data)
	placeholder := bytes.Repeat([]byte{0xFF, 0xFB, 0x90, 0x00}, int(size/4)+1)
	placeholder = placeholder[:size]

	err := s3Client.PutObject(ctx, key, bytes.NewReader(placeholder), "audio/mpeg")
	if err != nil {
		log.Printf("Warning: Failed to upload %s: %v", key, err)
		return
	}

	log.Printf("Uploaded placeholder audio: %s", key)
}
