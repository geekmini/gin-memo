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

// InvitationExpiryDays is the number of days until an invitation expires.
const InvitationExpiryDays = 7

// TeamInvitationRepository defines the interface for team invitation data operations.
type TeamInvitationRepository interface {
	Create(ctx context.Context, invitation *models.TeamInvitation) error
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.TeamInvitation, error)
	FindByTeamID(ctx context.Context, teamID primitive.ObjectID) ([]models.TeamInvitation, error)
	FindByEmail(ctx context.Context, email string) ([]models.TeamInvitation, error)
	FindByTeamAndEmail(ctx context.Context, teamID primitive.ObjectID, email string) (*models.TeamInvitation, error)
	CountPendingByTeamID(ctx context.Context, teamID primitive.ObjectID) (int, error)
	Delete(ctx context.Context, id primitive.ObjectID) error
	DeleteAllByTeamID(ctx context.Context, teamID primitive.ObjectID) error
	DeleteExpired(ctx context.Context) (int, error)
}

// teamInvitationRepository implements TeamInvitationRepository using MongoDB.
type teamInvitationRepository struct {
	collection *mongo.Collection
}

// NewTeamInvitationRepository creates a new TeamInvitationRepository.
func NewTeamInvitationRepository(db *mongo.Database) TeamInvitationRepository {
	return &teamInvitationRepository{
		collection: db.Collection("team_invitations"),
	}
}

// Create inserts a new invitation into the database.
func (r *teamInvitationRepository) Create(ctx context.Context, invitation *models.TeamInvitation) error {
	invitation.ID = primitive.NewObjectID()
	invitation.CreatedAt = time.Now()
	invitation.ExpiresAt = time.Now().AddDate(0, 0, InvitationExpiryDays)

	_, err := r.collection.InsertOne(ctx, invitation)
	return err
}

// FindByID retrieves an invitation by ID.
func (r *teamInvitationRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.TeamInvitation, error) {
	var invitation models.TeamInvitation
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&invitation)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrInvitationNotFound
		}
		return nil, err
	}

	return &invitation, nil
}

// FindByTeamID returns all pending invitations for a team.
func (r *teamInvitationRepository) FindByTeamID(ctx context.Context, teamID primitive.ObjectID) ([]models.TeamInvitation, error) {
	filter := bson.M{
		"teamId":    teamID,
		"expiresAt": bson.M{"$gt": time.Now()},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.TeamInvitation
	if err := cursor.All(ctx, &invitations); err != nil {
		return nil, err
	}

	if invitations == nil {
		invitations = []models.TeamInvitation{}
	}

	return invitations, nil
}

// FindByEmail returns all pending invitations for an email address.
func (r *teamInvitationRepository) FindByEmail(ctx context.Context, email string) ([]models.TeamInvitation, error) {
	filter := bson.M{
		"email":     email,
		"expiresAt": bson.M{"$gt": time.Now()},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var invitations []models.TeamInvitation
	if err := cursor.All(ctx, &invitations); err != nil {
		return nil, err
	}

	if invitations == nil {
		invitations = []models.TeamInvitation{}
	}

	return invitations, nil
}

// FindByTeamAndEmail returns a pending invitation for a specific team and email.
func (r *teamInvitationRepository) FindByTeamAndEmail(ctx context.Context, teamID primitive.ObjectID, email string) (*models.TeamInvitation, error) {
	filter := bson.M{
		"teamId":    teamID,
		"email":     email,
		"expiresAt": bson.M{"$gt": time.Now()},
	}

	var invitation models.TeamInvitation
	err := r.collection.FindOne(ctx, filter).Decode(&invitation)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, apperrors.ErrInvitationNotFound
		}
		return nil, err
	}

	return &invitation, nil
}

// CountPendingByTeamID returns the number of pending invitations for a team.
func (r *teamInvitationRepository) CountPendingByTeamID(ctx context.Context, teamID primitive.ObjectID) (int, error) {
	filter := bson.M{
		"teamId":    teamID,
		"expiresAt": bson.M{"$gt": time.Now()},
	}

	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(count), nil
}

// Delete removes an invitation.
func (r *teamInvitationRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return apperrors.ErrInvitationNotFound
	}

	return nil
}

// DeleteAllByTeamID removes all invitations for a team (used when deleting a team).
func (r *teamInvitationRepository) DeleteAllByTeamID(ctx context.Context, teamID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"teamId": teamID})
	return err
}

// DeleteExpired removes all expired invitations.
func (r *teamInvitationRepository) DeleteExpired(ctx context.Context) (int, error) {
	filter := bson.M{
		"expiresAt": bson.M{"$lte": time.Now()},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int(result.DeletedCount), nil
}
