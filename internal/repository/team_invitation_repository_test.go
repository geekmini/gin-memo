package repository

import (
	"context"
	"testing"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewTeamInvitationRepository(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)

	assert.NotNil(t, repo)
}

func TestTeamInvitationRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("successfully creates invitation", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		invitation := &models.TeamInvitation{
			TeamID:    primitive.NewObjectID(),
			Email:     "invitee@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}

		err := repo.Create(ctx, invitation)

		require.NoError(t, err)
		assert.False(t, invitation.ID.IsZero())
		assert.NotZero(t, invitation.CreatedAt)
		assert.NotZero(t, invitation.ExpiresAt)
		// Verify expiry is approximately 7 days in the future
		assert.True(t, invitation.ExpiresAt.After(time.Now().Add(6*24*time.Hour)))
	})
}

func TestTeamInvitationRepository_FindByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds invitation by ID", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		invitation := &models.TeamInvitation{
			TeamID:    primitive.NewObjectID(),
			Email:     "findbyid@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, invitation))

		found, err := repo.FindByID(ctx, invitation.ID)

		require.NoError(t, err)
		assert.Equal(t, invitation.ID, found.ID)
		assert.Equal(t, "findbyid@example.com", found.Email)
	})

	t.Run("returns error for non-existent invitation", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		found, err := repo.FindByID(ctx, primitive.NewObjectID())

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrInvitationNotFound, err)
	})
}

func TestTeamInvitationRepository_FindByTeamID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("returns pending invitations for team", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		teamID := primitive.NewObjectID()

		// Create valid invitations
		for i := 0; i < 3; i++ {
			invitation := &models.TeamInvitation{
				TeamID:    teamID,
				Email:     "user" + string(rune('a'+i)) + "@example.com",
				Role:      "member",
				InvitedBy: primitive.NewObjectID(),
			}
			require.NoError(t, repo.Create(ctx, invitation))
		}

		// Create invitation for different team
		otherInvitation := &models.TeamInvitation{
			TeamID:    primitive.NewObjectID(),
			Email:     "other@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, otherInvitation))

		invitations, err := repo.FindByTeamID(ctx, teamID)

		require.NoError(t, err)
		assert.Len(t, invitations, 3)
	})

	t.Run("excludes expired invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		teamID := primitive.NewObjectID()

		// Create valid invitation
		validInvitation := &models.TeamInvitation{
			TeamID:    teamID,
			Email:     "valid@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, validInvitation))

		// Create expired invitation manually
		expiredInvitation := &models.TeamInvitation{
			ID:        primitive.NewObjectID(),
			TeamID:    teamID,
			Email:     "expired@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
			CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
			ExpiresAt: time.Now().Add(-3 * 24 * time.Hour), // Expired 3 days ago
		}
		_, err := tdb.Database.Collection("team_invitations").InsertOne(ctx, expiredInvitation)
		require.NoError(t, err)

		invitations, err := repo.FindByTeamID(ctx, teamID)

		require.NoError(t, err)
		assert.Len(t, invitations, 1)
		assert.Equal(t, "valid@example.com", invitations[0].Email)
	})

	t.Run("returns empty slice when no invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		invitations, err := repo.FindByTeamID(ctx, primitive.NewObjectID())

		require.NoError(t, err)
		assert.NotNil(t, invitations)
		assert.Len(t, invitations, 0)
	})
}

func TestTeamInvitationRepository_FindByEmail(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("returns invitations for email", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		email := "multi@example.com"

		// Create invitations from multiple teams
		for i := 0; i < 3; i++ {
			invitation := &models.TeamInvitation{
				TeamID:    primitive.NewObjectID(),
				Email:     email,
				Role:      "member",
				InvitedBy: primitive.NewObjectID(),
			}
			require.NoError(t, repo.Create(ctx, invitation))
		}

		invitations, err := repo.FindByEmail(ctx, email)

		require.NoError(t, err)
		assert.Len(t, invitations, 3)
	})

	t.Run("returns empty slice when no invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		invitations, err := repo.FindByEmail(ctx, "nobody@example.com")

		require.NoError(t, err)
		assert.NotNil(t, invitations)
		assert.Len(t, invitations, 0)
	})
}

func TestTeamInvitationRepository_FindByTeamAndEmail(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds invitation by team and email", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		teamID := primitive.NewObjectID()
		email := "specific@example.com"

		invitation := &models.TeamInvitation{
			TeamID:    teamID,
			Email:     email,
			Role:      "admin",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, invitation))

		found, err := repo.FindByTeamAndEmail(ctx, teamID, email)

		require.NoError(t, err)
		assert.Equal(t, teamID, found.TeamID)
		assert.Equal(t, email, found.Email)
		assert.Equal(t, "admin", found.Role)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		found, err := repo.FindByTeamAndEmail(ctx, primitive.NewObjectID(), "notfound@example.com")

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrInvitationNotFound, err)
	})

	t.Run("excludes expired invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		teamID := primitive.NewObjectID()
		email := "expired@example.com"

		// Create expired invitation manually
		expiredInvitation := &models.TeamInvitation{
			ID:        primitive.NewObjectID(),
			TeamID:    teamID,
			Email:     email,
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
			CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
			ExpiresAt: time.Now().Add(-3 * 24 * time.Hour),
		}
		_, err := tdb.Database.Collection("team_invitations").InsertOne(ctx, expiredInvitation)
		require.NoError(t, err)

		found, err := repo.FindByTeamAndEmail(ctx, teamID, email)

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrInvitationNotFound, err)
	})
}

func TestTeamInvitationRepository_CountPendingByTeamID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("counts pending invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		teamID := primitive.NewObjectID()

		for i := 0; i < 5; i++ {
			invitation := &models.TeamInvitation{
				TeamID:    teamID,
				Email:     "pending" + string(rune('a'+i)) + "@example.com",
				Role:      "member",
				InvitedBy: primitive.NewObjectID(),
			}
			require.NoError(t, repo.Create(ctx, invitation))
		}

		count, err := repo.CountPendingByTeamID(ctx, teamID)

		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("excludes expired invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		teamID := primitive.NewObjectID()

		// Create valid invitation
		validInvitation := &models.TeamInvitation{
			TeamID:    teamID,
			Email:     "valid@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, validInvitation))

		// Create expired invitation manually
		expiredInvitation := &models.TeamInvitation{
			ID:        primitive.NewObjectID(),
			TeamID:    teamID,
			Email:     "expired@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
			CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
			ExpiresAt: time.Now().Add(-3 * 24 * time.Hour),
		}
		_, err := tdb.Database.Collection("team_invitations").InsertOne(ctx, expiredInvitation)
		require.NoError(t, err)

		count, err := repo.CountPendingByTeamID(ctx, teamID)

		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestTeamInvitationRepository_Delete(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes invitation", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		invitation := &models.TeamInvitation{
			TeamID:    primitive.NewObjectID(),
			Email:     "delete@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, invitation))

		err := repo.Delete(ctx, invitation.ID)

		require.NoError(t, err)

		// Verify deletion
		found, err := repo.FindByID(ctx, invitation.ID)
		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrInvitationNotFound, err)
	})

	t.Run("returns error for non-existent invitation", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		err := repo.Delete(ctx, primitive.NewObjectID())

		assert.Equal(t, apperrors.ErrInvitationNotFound, err)
	})
}

func TestTeamInvitationRepository_DeleteAllByTeamID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes all invitations for team", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		teamID := primitive.NewObjectID()
		otherTeamID := primitive.NewObjectID()

		// Create invitations for first team
		for i := 0; i < 3; i++ {
			invitation := &models.TeamInvitation{
				TeamID:    teamID,
				Email:     "user" + string(rune('a'+i)) + "@example.com",
				Role:      "member",
				InvitedBy: primitive.NewObjectID(),
			}
			require.NoError(t, repo.Create(ctx, invitation))
		}

		// Create invitation for other team
		otherInvitation := &models.TeamInvitation{
			TeamID:    otherTeamID,
			Email:     "other@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, otherInvitation))

		err := repo.DeleteAllByTeamID(ctx, teamID)

		require.NoError(t, err)

		// Verify invitations deleted
		invitations, err := repo.FindByTeamID(ctx, teamID)
		require.NoError(t, err)
		assert.Len(t, invitations, 0)

		// Verify other team's invitation still exists
		otherInvitations, err := repo.FindByTeamID(ctx, otherTeamID)
		require.NoError(t, err)
		assert.Len(t, otherInvitations, 1)
	})

	t.Run("succeeds when team has no invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		err := repo.DeleteAllByTeamID(ctx, primitive.NewObjectID())

		assert.NoError(t, err)
	})
}

func TestTeamInvitationRepository_DeleteExpired(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamInvitationRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes expired invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		teamID := primitive.NewObjectID()

		// Create valid invitation
		validInvitation := &models.TeamInvitation{
			TeamID:    teamID,
			Email:     "valid@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, validInvitation))

		// Create expired invitations manually
		for i := 0; i < 3; i++ {
			expiredInvitation := &models.TeamInvitation{
				ID:        primitive.NewObjectID(),
				TeamID:    teamID,
				Email:     "expired" + string(rune('a'+i)) + "@example.com",
				Role:      "member",
				InvitedBy: primitive.NewObjectID(),
				CreatedAt: time.Now().Add(-10 * 24 * time.Hour),
				ExpiresAt: time.Now().Add(-3 * 24 * time.Hour),
			}
			_, err := tdb.Database.Collection("team_invitations").InsertOne(ctx, expiredInvitation)
			require.NoError(t, err)
		}

		count, err := repo.DeleteExpired(ctx)

		require.NoError(t, err)
		assert.Equal(t, 3, count)

		// Verify valid invitation still exists
		found, err := repo.FindByID(ctx, validInvitation.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
	})

	t.Run("returns zero when no expired invitations", func(t *testing.T) {
		tdb.ClearCollection(t, "team_invitations")

		// Create only valid invitation
		invitation := &models.TeamInvitation{
			TeamID:    primitive.NewObjectID(),
			Email:     "valid@example.com",
			Role:      "member",
			InvitedBy: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, invitation))

		count, err := repo.DeleteExpired(ctx)

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
