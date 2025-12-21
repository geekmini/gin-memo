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

func TestNewTeamMemberRepository(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)

	assert.NotNil(t, repo)
}

func TestTeamMemberRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("successfully creates team member", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		member := &models.TeamMember{
			TeamID: primitive.NewObjectID(),
			UserID: primitive.NewObjectID(),
			Role:   "member",
		}

		err := repo.Create(ctx, member)

		require.NoError(t, err)
		assert.False(t, member.ID.IsZero())
		assert.NotZero(t, member.JoinedAt)
	})
}

func TestTeamMemberRepository_FindByTeamID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("returns all team members", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		teamID := primitive.NewObjectID()

		// Add 3 members
		for i := 0; i < 3; i++ {
			member := &models.TeamMember{
				TeamID: teamID,
				UserID: primitive.NewObjectID(),
				Role:   "member",
			}
			require.NoError(t, repo.Create(ctx, member))
		}

		// Add member to different team
		otherMember := &models.TeamMember{
			TeamID: primitive.NewObjectID(),
			UserID: primitive.NewObjectID(),
			Role:   "member",
		}
		require.NoError(t, repo.Create(ctx, otherMember))

		members, err := repo.FindByTeamID(ctx, teamID)

		require.NoError(t, err)
		assert.Len(t, members, 3)
	})

	t.Run("returns empty slice when no members", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		members, err := repo.FindByTeamID(ctx, primitive.NewObjectID())

		require.NoError(t, err)
		assert.NotNil(t, members)
		assert.Len(t, members, 0)
	})
}

func TestTeamMemberRepository_FindByTeamAndUser(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds member by team and user", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		teamID := primitive.NewObjectID()
		userID := primitive.NewObjectID()

		member := &models.TeamMember{
			TeamID: teamID,
			UserID: userID,
			Role:   "admin",
		}
		require.NoError(t, repo.Create(ctx, member))

		found, err := repo.FindByTeamAndUser(ctx, teamID, userID)

		require.NoError(t, err)
		assert.Equal(t, teamID, found.TeamID)
		assert.Equal(t, userID, found.UserID)
		assert.Equal(t, "admin", found.Role)
	})

	t.Run("returns error when member not found", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		found, err := repo.FindByTeamAndUser(ctx, primitive.NewObjectID(), primitive.NewObjectID())

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrNotTeamMember, err)
	})
}

func TestTeamMemberRepository_FindByUserID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("returns all memberships for user", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		userID := primitive.NewObjectID()

		// Add user to 3 teams
		for i := 0; i < 3; i++ {
			member := &models.TeamMember{
				TeamID: primitive.NewObjectID(),
				UserID: userID,
				Role:   "member",
			}
			require.NoError(t, repo.Create(ctx, member))
		}

		members, err := repo.FindByUserID(ctx, userID)

		require.NoError(t, err)
		assert.Len(t, members, 3)
	})

	t.Run("returns empty slice when user has no memberships", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		members, err := repo.FindByUserID(ctx, primitive.NewObjectID())

		require.NoError(t, err)
		assert.NotNil(t, members)
		assert.Len(t, members, 0)
	})
}

func TestTeamMemberRepository_CountByTeamID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("counts team members", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		teamID := primitive.NewObjectID()

		for i := 0; i < 5; i++ {
			member := &models.TeamMember{
				TeamID: teamID,
				UserID: primitive.NewObjectID(),
				Role:   "member",
			}
			require.NoError(t, repo.Create(ctx, member))
		}

		count, err := repo.CountByTeamID(ctx, teamID)

		require.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("returns zero when no members", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		count, err := repo.CountByTeamID(ctx, primitive.NewObjectID())

		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestTeamMemberRepository_UpdateRole(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("updates member role", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		teamID := primitive.NewObjectID()
		userID := primitive.NewObjectID()

		member := &models.TeamMember{
			TeamID: teamID,
			UserID: userID,
			Role:   "member",
		}
		require.NoError(t, repo.Create(ctx, member))

		err := repo.UpdateRole(ctx, teamID, userID, "admin")

		require.NoError(t, err)

		// Verify update
		found, err := repo.FindByTeamAndUser(ctx, teamID, userID)
		require.NoError(t, err)
		assert.Equal(t, "admin", found.Role)
	})

	t.Run("returns error for non-existent member", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		err := repo.UpdateRole(ctx, primitive.NewObjectID(), primitive.NewObjectID(), "admin")

		assert.Equal(t, apperrors.ErrNotTeamMember, err)
	})
}

func TestTeamMemberRepository_Delete(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes member", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		teamID := primitive.NewObjectID()
		userID := primitive.NewObjectID()

		member := &models.TeamMember{
			TeamID: teamID,
			UserID: userID,
			Role:   "member",
		}
		require.NoError(t, repo.Create(ctx, member))

		err := repo.Delete(ctx, teamID, userID)

		require.NoError(t, err)

		// Verify deletion
		found, err := repo.FindByTeamAndUser(ctx, teamID, userID)
		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrNotTeamMember, err)
	})

	t.Run("returns error for non-existent member", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		err := repo.Delete(ctx, primitive.NewObjectID(), primitive.NewObjectID())

		assert.Equal(t, apperrors.ErrNotTeamMember, err)
	})
}

func TestTeamMemberRepository_DeleteAllByTeamID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewTeamMemberRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes all members of team", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		teamID := primitive.NewObjectID()
		otherTeamID := primitive.NewObjectID()

		// Add members to first team
		for i := 0; i < 3; i++ {
			member := &models.TeamMember{
				TeamID: teamID,
				UserID: primitive.NewObjectID(),
				Role:   "member",
			}
			require.NoError(t, repo.Create(ctx, member))
		}

		// Add member to other team
		otherMember := &models.TeamMember{
			TeamID: otherTeamID,
			UserID: primitive.NewObjectID(),
			Role:   "member",
		}
		require.NoError(t, repo.Create(ctx, otherMember))

		err := repo.DeleteAllByTeamID(ctx, teamID)

		require.NoError(t, err)

		// Verify team members deleted
		members, err := repo.FindByTeamID(ctx, teamID)
		require.NoError(t, err)
		assert.Len(t, members, 0)

		// Verify other team member still exists
		otherMembers, err := repo.FindByTeamID(ctx, otherTeamID)
		require.NoError(t, err)
		assert.Len(t, otherMembers, 1)
	})

	t.Run("succeeds when team has no members", func(t *testing.T) {
		tdb.ClearCollection(t, "team_members")

		err := repo.DeleteAllByTeamID(ctx, primitive.NewObjectID())

		assert.NoError(t, err)
	})
}
