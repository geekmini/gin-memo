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

// TeamMemberRepository defines the interface for team member data operations.
type TeamMemberRepository interface {
	Create(ctx context.Context, member *models.TeamMember) error
	FindByTeamID(ctx context.Context, teamID primitive.ObjectID) ([]models.TeamMember, error)
	FindByTeamAndUser(ctx context.Context, teamID, userID primitive.ObjectID) (*models.TeamMember, error)
	FindByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.TeamMember, error)
	CountByTeamID(ctx context.Context, teamID primitive.ObjectID) (int, error)
	UpdateRole(ctx context.Context, teamID, userID primitive.ObjectID, role string) error
	Delete(ctx context.Context, teamID, userID primitive.ObjectID) error
	DeleteAllByTeamID(ctx context.Context, teamID primitive.ObjectID) error
}

// teamMemberRepository implements TeamMemberRepository using MongoDB.
type teamMemberRepository struct {
	collection *mongo.Collection
}

// NewTeamMemberRepository creates a new TeamMemberRepository.
func NewTeamMemberRepository(db *mongo.Database) TeamMemberRepository {
	return &teamMemberRepository{
		collection: db.Collection("team_members"),
	}
}

// Create inserts a new team member into the database.
func (r *teamMemberRepository) Create(ctx context.Context, member *models.TeamMember) error {
	member.ID = primitive.NewObjectID()
	member.JoinedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, member)
	return err
}

// FindByTeamID returns all members of a team.
func (r *teamMemberRepository) FindByTeamID(ctx context.Context, teamID primitive.ObjectID) ([]models.TeamMember, error) {
	filter := bson.M{"teamId": teamID}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []models.TeamMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}

	if members == nil {
		members = []models.TeamMember{}
	}

	return members, nil
}

// FindByTeamAndUser returns a team member by team and user ID.
func (r *teamMemberRepository) FindByTeamAndUser(ctx context.Context, teamID, userID primitive.ObjectID) (*models.TeamMember, error) {
	filter := bson.M{
		"teamId": teamID,
		"userId": userID,
	}

	var member models.TeamMember
	err := r.collection.FindOne(ctx, filter).Decode(&member)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrNotTeamMember
		}
		return nil, err
	}

	return &member, nil
}

// FindByUserID returns all team memberships for a user.
func (r *teamMemberRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.TeamMember, error) {
	filter := bson.M{"userId": userID}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []models.TeamMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}

	if members == nil {
		members = []models.TeamMember{}
	}

	return members, nil
}

// CountByTeamID returns the number of members in a team.
func (r *teamMemberRepository) CountByTeamID(ctx context.Context, teamID primitive.ObjectID) (int, error) {
	filter := bson.M{"teamId": teamID}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// UpdateRole updates a team member's role.
func (r *teamMemberRepository) UpdateRole(ctx context.Context, teamID, userID primitive.ObjectID, role string) error {
	filter := bson.M{
		"teamId": teamID,
		"userId": userID,
	}

	update := bson.M{
		"$set": bson.M{"role": role},
	}

	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return apperrors.ErrNotTeamMember
	}

	return nil
}

// Delete removes a team member.
func (r *teamMemberRepository) Delete(ctx context.Context, teamID, userID primitive.ObjectID) error {
	filter := bson.M{
		"teamId": teamID,
		"userId": userID,
	}

	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return apperrors.ErrNotTeamMember
	}

	return nil
}

// DeleteAllByTeamID removes all members of a team (used when deleting a team).
func (r *teamMemberRepository) DeleteAllByTeamID(ctx context.Context, teamID primitive.ObjectID) error {
	filter := bson.M{"teamId": teamID}

	_, err := r.collection.DeleteMany(ctx, filter)
	return err
}
