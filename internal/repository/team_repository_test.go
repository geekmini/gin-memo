package repository

import (
	"context"
	"testing"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewTeamRepository(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamRepository(tdb.Database)

	assert.NotNil(t, repo)
}

func TestTeamRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamRepository(tdb.Database)
	ctx := context.Background()

	t.Run("successfully creates team with defaults", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "Test Team",
			Slug:    "test-team",
			OwnerID: primitive.NewObjectID(),
		}

		err := repo.Create(ctx, team)

		require.NoError(t, err)
		assert.False(t, team.ID.IsZero())
		assert.NotZero(t, team.CreatedAt)
		assert.NotZero(t, team.UpdatedAt)
		assert.Equal(t, 10, team.Seats)         // Default seats
		assert.Equal(t, 30, team.RetentionDays) // Default retention
	})

	t.Run("preserves custom seats and retention", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:          "Custom Team",
			Slug:          "custom-team",
			OwnerID:       primitive.NewObjectID(),
			Seats:         50,
			RetentionDays: 90,
		}

		err := repo.Create(ctx, team)

		require.NoError(t, err)
		assert.Equal(t, 50, team.Seats)
		assert.Equal(t, 90, team.RetentionDays)
	})
}

func TestTeamRepository_FindByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds existing team", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "Find Me Team",
			Slug:    "find-me",
			OwnerID: primitive.NewObjectID(),
		}
		err := repo.Create(ctx, team)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, team.ID)

		require.NoError(t, err)
		assert.Equal(t, team.ID, found.ID)
		assert.Equal(t, team.Name, found.Name)
	})

	t.Run("returns error for non-existent team", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		found, err := repo.FindByID(ctx, primitive.NewObjectID())

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})

	t.Run("excludes soft-deleted teams", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "Deleted Team",
			Slug:    "deleted-team",
			OwnerID: primitive.NewObjectID(),
		}
		err := repo.Create(ctx, team)
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, team.ID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, team.ID)

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})
}

func TestTeamRepository_FindBySlug(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds team by slug", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "Slug Team",
			Slug:    "slug-team",
			OwnerID: primitive.NewObjectID(),
		}
		err := repo.Create(ctx, team)
		require.NoError(t, err)

		found, err := repo.FindBySlug(ctx, "slug-team")

		require.NoError(t, err)
		assert.Equal(t, team.ID, found.ID)
		assert.Equal(t, "slug-team", found.Slug)
	})

	t.Run("returns error for non-existent slug", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		found, err := repo.FindBySlug(ctx, "nonexistent-slug")

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})

	t.Run("excludes soft-deleted teams", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "To Be Deleted",
			Slug:    "to-be-deleted",
			OwnerID: primitive.NewObjectID(),
		}
		err := repo.Create(ctx, team)
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, team.ID)
		require.NoError(t, err)

		found, err := repo.FindBySlug(ctx, "to-be-deleted")

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})
}

func TestTeamRepository_FindByUserID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	teamRepo := NewTeamRepository(tdb.Database)
	memberRepo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("returns teams where user is member", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")
		tdb.ClearCollection(t, "team_members")

		userID := primitive.NewObjectID()

		// Create teams
		team1 := &models.Team{Name: "Team 1", Slug: "team-1", OwnerID: primitive.NewObjectID()}
		team2 := &models.Team{Name: "Team 2", Slug: "team-2", OwnerID: primitive.NewObjectID()}
		team3 := &models.Team{Name: "Team 3", Slug: "team-3", OwnerID: primitive.NewObjectID()}
		require.NoError(t, teamRepo.Create(ctx, team1))
		require.NoError(t, teamRepo.Create(ctx, team2))
		require.NoError(t, teamRepo.Create(ctx, team3))

		// Add user to team1 and team2 only
		require.NoError(t, memberRepo.Create(ctx, &models.TeamMember{TeamID: team1.ID, UserID: userID, Role: "member"}))
		require.NoError(t, memberRepo.Create(ctx, &models.TeamMember{TeamID: team2.ID, UserID: userID, Role: "admin"}))

		teams, total, err := teamRepo.FindByUserID(ctx, userID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, teams, 2)
	})

	t.Run("returns empty list when user has no teams", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")
		tdb.ClearCollection(t, "team_members")

		teams, total, err := teamRepo.FindByUserID(ctx, primitive.NewObjectID(), 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.NotNil(t, teams)
		assert.Len(t, teams, 0)
	})

	t.Run("paginates results", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")
		tdb.ClearCollection(t, "team_members")

		userID := primitive.NewObjectID()

		// Create 5 teams and add user to all
		for i := 0; i < 5; i++ {
			team := &models.Team{
				Name:    "Team " + string(rune('A'+i)),
				Slug:    "team-" + string(rune('a'+i)),
				OwnerID: primitive.NewObjectID(),
			}
			require.NoError(t, teamRepo.Create(ctx, team))
			require.NoError(t, memberRepo.Create(ctx, &models.TeamMember{TeamID: team.ID, UserID: userID, Role: "member"}))
		}

		// Get page 1 with limit 2
		teams, total, err := teamRepo.FindByUserID(ctx, userID, 1, 2)

		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.Len(t, teams, 2)

		// Get page 2
		teams2, total2, err := teamRepo.FindByUserID(ctx, userID, 2, 2)

		require.NoError(t, err)
		assert.Equal(t, 5, total2)
		assert.Len(t, teams2, 2)
	})
}

func TestTeamRepository_CountByOwnerID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamRepository(tdb.Database)
	ctx := context.Background()

	t.Run("counts teams owned by user", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		ownerID := primitive.NewObjectID()

		// Create teams for this owner
		for i := 0; i < 3; i++ {
			team := &models.Team{
				Name:    "Owned Team " + string(rune('A'+i)),
				Slug:    "owned-team-" + string(rune('a'+i)),
				OwnerID: ownerID,
			}
			require.NoError(t, repo.Create(ctx, team))
		}

		// Create team for different owner
		otherTeam := &models.Team{
			Name:    "Other Team",
			Slug:    "other-team",
			OwnerID: primitive.NewObjectID(),
		}
		require.NoError(t, repo.Create(ctx, otherTeam))

		count, err := repo.CountByOwnerID(ctx, ownerID)

		require.NoError(t, err)
		assert.Equal(t, 3, count)
	})

	t.Run("excludes soft-deleted teams", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		ownerID := primitive.NewObjectID()

		team1 := &models.Team{Name: "Active", Slug: "active", OwnerID: ownerID}
		team2 := &models.Team{Name: "Deleted", Slug: "deleted", OwnerID: ownerID}
		require.NoError(t, repo.Create(ctx, team1))
		require.NoError(t, repo.Create(ctx, team2))

		// Soft delete one team
		require.NoError(t, repo.SoftDelete(ctx, team2.ID))

		count, err := repo.CountByOwnerID(ctx, ownerID)

		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

func TestTeamRepository_Update(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamRepository(tdb.Database)
	ctx := context.Background()

	t.Run("updates team fields", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "Original Name",
			Slug:    "original-slug",
			OwnerID: primitive.NewObjectID(),
		}
		err := repo.Create(ctx, team)
		require.NoError(t, err)

		team.Name = "Updated Name"
		team.Description = "New Description"

		err = repo.Update(ctx, team)

		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByID(ctx, team.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", found.Name)
		assert.Equal(t, "New Description", found.Description)
	})

	t.Run("returns error for non-existent team", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			ID:      primitive.NewObjectID(),
			Name:    "Nonexistent",
			Slug:    "nonexistent",
			OwnerID: primitive.NewObjectID(),
		}

		err := repo.Update(ctx, team)

		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})

	t.Run("returns error for soft-deleted team", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "Will Be Deleted",
			Slug:    "will-be-deleted",
			OwnerID: primitive.NewObjectID(),
		}
		err := repo.Create(ctx, team)
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, team.ID)
		require.NoError(t, err)

		team.Name = "Try To Update"
		err = repo.Update(ctx, team)

		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})
}

func TestTeamRepository_SoftDelete(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamRepository(tdb.Database)
	ctx := context.Background()

	t.Run("soft deletes team", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "Delete Me",
			Slug:    "delete-me",
			OwnerID: primitive.NewObjectID(),
		}
		err := repo.Create(ctx, team)
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, team.ID)

		require.NoError(t, err)

		// Verify team is not found
		found, err := repo.FindByID(ctx, team.ID)
		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})

	t.Run("returns error for non-existent team", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		err := repo.SoftDelete(ctx, primitive.NewObjectID())

		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})

	t.Run("returns error for already deleted team", func(t *testing.T) {
		tdb.ClearCollection(t, "teams")

		team := &models.Team{
			Name:    "Double Delete",
			Slug:    "double-delete",
			OwnerID: primitive.NewObjectID(),
		}
		err := repo.Create(ctx, team)
		require.NoError(t, err)

		err = repo.SoftDelete(ctx, team.ID)
		require.NoError(t, err)

		// Try to delete again
		err = repo.SoftDelete(ctx, team.ID)
		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})
}
