package repository

import (
	"context"
	"errors"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// TeamRepository defines the interface for team data operations.
type TeamRepository interface {
	Create(ctx context.Context, team *models.Team) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Team, error)
	FindBySlug(ctx context.Context, slug string) (*models.Team, error)
	FindByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int) ([]models.Team, int, error)
	CountByOwnerID(ctx context.Context, ownerID primitive.ObjectID) (int, error)
	Update(ctx context.Context, team *models.Team) error
	SoftDelete(ctx context.Context, id primitive.ObjectID) error
}

// teamRepository implements TeamRepository using MongoDB.
type teamRepository struct {
	collection *mongo.Collection
}

// NewTeamRepository creates a new TeamRepository.
func NewTeamRepository(db *mongo.Database) TeamRepository {
	return &teamRepository{
		collection: db.Collection("teams"),
	}
}

// Create inserts a new team into the database.
func (r *teamRepository) Create(ctx context.Context, team *models.Team) error {
	team.ID = primitive.NewObjectID()
	team.CreatedAt = time.Now()
	team.UpdatedAt = time.Now()

	if team.Seats == 0 {
		team.Seats = 10 // Default seats
	}
	if team.RetentionDays == 0 {
		team.RetentionDays = 30 // Default retention
	}

	_, err := r.collection.InsertOne(ctx, team)
	return err
}

// FindByID retrieves a team by ID. Excludes soft-deleted teams.
func (r *teamRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Team, error) {
	filter := bson.M{
		"_id":       id,
		"deletedAt": bson.M{"$exists": false},
	}

	var team models.Team
	err := r.collection.FindOne(ctx, filter).Decode(&team)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrTeamNotFound
		}
		return nil, err
	}

	return &team, nil
}

// FindBySlug retrieves a team by slug. Excludes soft-deleted teams.
func (r *teamRepository) FindBySlug(ctx context.Context, slug string) (*models.Team, error) {
	filter := bson.M{
		"slug":      slug,
		"deletedAt": bson.M{"$exists": false},
	}

	var team models.Team
	err := r.collection.FindOne(ctx, filter).Decode(&team)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrTeamNotFound
		}
		return nil, err
	}

	return &team, nil
}

// FindByUserID returns paginated teams for a user (teams where user is a member).
// This requires a lookup with the team_members collection.
func (r *teamRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID, page, limit int) ([]models.Team, int, error) {
	skip := (page - 1) * limit

	// Pipeline to join teams with team_members
	pipeline := mongo.Pipeline{
		// Match non-deleted teams
		{{Key: "$match", Value: bson.M{"deletedAt": bson.M{"$exists": false}}}},
		// Lookup team members
		{{Key: "$lookup", Value: bson.M{
			"from":         "team_members",
			"localField":   "_id",
			"foreignField": "teamId",
			"as":           "members",
		}}},
		// Filter to teams where user is a member
		{{Key: "$match", Value: bson.M{"members.userId": userID}}},
		// Remove members array from output
		{{Key: "$project", Value: bson.M{"members": 0}}},
		// Sort by createdAt descending
		{{Key: "$sort", Value: bson.D{{Key: "createdAt", Value: -1}}}},
	}

	// Count pipeline
	countPipeline := append(pipeline, bson.D{{Key: "$count", Value: "total"}})
	countCursor, err := r.collection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, 0, err
	}
	defer countCursor.Close(ctx)

	var countResult []struct {
		Total int `bson:"total"`
	}
	if err := countCursor.All(ctx, &countResult); err != nil {
		return nil, 0, err
	}

	total := 0
	if len(countResult) > 0 {
		total = countResult[0].Total
	}

	// Add pagination to pipeline
	pipeline = append(pipeline,
		bson.D{{Key: "$skip", Value: int64(skip)}},
		bson.D{{Key: "$limit", Value: int64(limit)}},
	)

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var teams []models.Team
	if err := cursor.All(ctx, &teams); err != nil {
		return nil, 0, err
	}

	if teams == nil {
		teams = []models.Team{}
	}

	return teams, total, nil
}

// CountByOwnerID returns the number of teams owned by a user.
func (r *teamRepository) CountByOwnerID(ctx context.Context, ownerID primitive.ObjectID) (int, error) {
	filter := bson.M{
		"ownerId":   ownerID,
		"deletedAt": bson.M{"$exists": false},
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// Update updates an existing team.
func (r *teamRepository) Update(ctx context.Context, team *models.Team) error {
	team.UpdatedAt = time.Now()

	filter := bson.M{
		"_id":       team.ID,
		"deletedAt": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"name":        team.Name,
			"slug":        team.Slug,
			"description": team.Description,
			"logoUrl":     team.LogoURL,
			"ownerId":     team.OwnerID,
			"updatedAt":   team.UpdatedAt,
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return apperrors.ErrTeamNotFound
	}

	return nil
}

// SoftDelete marks a team as deleted.
func (r *teamRepository) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{
		"_id":       id,
		"deletedAt": bson.M{"$exists": false},
	}

	update := bson.M{
		"$set": bson.M{
			"deletedAt": time.Now(),
		},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return apperrors.ErrTeamNotFound
	}

	return nil
}
