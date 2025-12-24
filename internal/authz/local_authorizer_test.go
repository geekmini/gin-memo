package authz

import (
	"context"
	"errors"
	"testing"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// mockMemberFinder is a test double for TeamMemberFinder.
type mockMemberFinder struct {
	member *models.TeamMember
	err    error
}

func (m *mockMemberFinder) FindByTeamAndUser(_ context.Context, _, _ primitive.ObjectID) (*models.TeamMember, error) {
	return m.member, m.err
}

func TestNewLocalAuthorizer(t *testing.T) {
	finder := &mockMemberFinder{}

	auth := NewLocalAuthorizer(finder)

	require.NotNil(t, auth)
	assert.Equal(t, finder, auth.memberFinder)
}

func TestLocalAuthorizer_CanPerform(t *testing.T) {
	userID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	ctx := context.Background()

	// Test all role/action combinations
	roleActionTests := []struct {
		name     string
		role     string
		action   string
		expected bool
	}{
		// Owner permissions - can do everything
		{"owner can view team", models.RoleOwner, ActionTeamView, true},
		{"owner can update team", models.RoleOwner, ActionTeamUpdate, true},
		{"owner can delete team", models.RoleOwner, ActionTeamDelete, true},
		{"owner can transfer team", models.RoleOwner, ActionTeamTransfer, true},
		{"owner can invite members", models.RoleOwner, ActionMemberInvite, true},
		{"owner can remove members", models.RoleOwner, ActionMemberRemove, true},
		{"owner can update roles", models.RoleOwner, ActionMemberUpdateRole, true},
		{"owner can view memos", models.RoleOwner, ActionMemoView, true},
		{"owner can create memos", models.RoleOwner, ActionMemoCreate, true},
		{"owner can update memos", models.RoleOwner, ActionMemoUpdate, true},
		{"owner can delete memos", models.RoleOwner, ActionMemoDelete, true},

		// Admin permissions - most things except delete/transfer team
		{"admin can view team", models.RoleAdmin, ActionTeamView, true},
		{"admin can update team", models.RoleAdmin, ActionTeamUpdate, true},
		{"admin cannot delete team", models.RoleAdmin, ActionTeamDelete, false},
		{"admin cannot transfer team", models.RoleAdmin, ActionTeamTransfer, false},
		{"admin can invite members", models.RoleAdmin, ActionMemberInvite, true},
		{"admin can remove members", models.RoleAdmin, ActionMemberRemove, true},
		{"admin can update roles", models.RoleAdmin, ActionMemberUpdateRole, true},
		{"admin can view memos", models.RoleAdmin, ActionMemoView, true},
		{"admin can create memos", models.RoleAdmin, ActionMemoCreate, true},
		{"admin can update memos", models.RoleAdmin, ActionMemoUpdate, true},
		{"admin can delete memos", models.RoleAdmin, ActionMemoDelete, true},

		// Member permissions - limited to view/memo operations
		{"member can view team", models.RoleMember, ActionTeamView, true},
		{"member cannot update team", models.RoleMember, ActionTeamUpdate, false},
		{"member cannot delete team", models.RoleMember, ActionTeamDelete, false},
		{"member cannot transfer team", models.RoleMember, ActionTeamTransfer, false},
		{"member cannot invite members", models.RoleMember, ActionMemberInvite, false},
		{"member cannot remove members", models.RoleMember, ActionMemberRemove, false},
		{"member cannot update roles", models.RoleMember, ActionMemberUpdateRole, false},
		{"member can view memos", models.RoleMember, ActionMemoView, true},
		{"member can create memos", models.RoleMember, ActionMemoCreate, true},
		{"member can update memos", models.RoleMember, ActionMemoUpdate, true},
		{"member can delete memos", models.RoleMember, ActionMemoDelete, true},
	}

	for _, tt := range roleActionTests {
		t.Run(tt.name, func(t *testing.T) {
			finder := &mockMemberFinder{
				member: &models.TeamMember{Role: tt.role},
			}
			auth := NewLocalAuthorizer(finder)

			can, err := auth.CanPerform(ctx, userID, teamID, tt.action)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, can)
		})
	}

	t.Run("non-member returns false without error", func(t *testing.T) {
		finder := &mockMemberFinder{
			err: apperrors.ErrNotTeamMember,
		}
		auth := NewLocalAuthorizer(finder)

		can, err := auth.CanPerform(ctx, userID, teamID, ActionTeamView)

		require.NoError(t, err)
		assert.False(t, can)
	})

	t.Run("unknown action returns false", func(t *testing.T) {
		finder := &mockMemberFinder{
			member: &models.TeamMember{Role: models.RoleOwner},
		}
		auth := NewLocalAuthorizer(finder)

		can, err := auth.CanPerform(ctx, userID, teamID, "unknown:action")

		require.NoError(t, err)
		assert.False(t, can)
	})

	t.Run("unknown role returns false", func(t *testing.T) {
		finder := &mockMemberFinder{
			member: &models.TeamMember{Role: "unknown_role"},
		}
		auth := NewLocalAuthorizer(finder)

		can, err := auth.CanPerform(ctx, userID, teamID, ActionTeamView)

		require.NoError(t, err)
		assert.False(t, can)
	})

	t.Run("database error is propagated", func(t *testing.T) {
		dbError := errors.New("database connection failed")
		finder := &mockMemberFinder{
			err: dbError,
		}
		auth := NewLocalAuthorizer(finder)

		can, err := auth.CanPerform(ctx, userID, teamID, ActionTeamView)

		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		assert.False(t, can)
	})
}

func TestLocalAuthorizer_GetUserRole(t *testing.T) {
	userID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	ctx := context.Background()

	t.Run("returns role for team member", func(t *testing.T) {
		finder := &mockMemberFinder{
			member: &models.TeamMember{Role: models.RoleAdmin},
		}
		auth := NewLocalAuthorizer(finder)

		role, err := auth.GetUserRole(ctx, userID, teamID)

		require.NoError(t, err)
		assert.Equal(t, models.RoleAdmin, role)
	})

	t.Run("returns empty string for non-member", func(t *testing.T) {
		finder := &mockMemberFinder{
			err: apperrors.ErrNotTeamMember,
		}
		auth := NewLocalAuthorizer(finder)

		role, err := auth.GetUserRole(ctx, userID, teamID)

		require.NoError(t, err)
		assert.Empty(t, role)
	})

	t.Run("propagates database error", func(t *testing.T) {
		dbError := errors.New("database error")
		finder := &mockMemberFinder{err: dbError}
		auth := NewLocalAuthorizer(finder)

		role, err := auth.GetUserRole(ctx, userID, teamID)

		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		assert.Empty(t, role)
	})
}

func TestLocalAuthorizer_IsMember(t *testing.T) {
	userID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	ctx := context.Background()

	t.Run("returns true for team member", func(t *testing.T) {
		finder := &mockMemberFinder{
			member: &models.TeamMember{Role: models.RoleMember},
		}
		auth := NewLocalAuthorizer(finder)

		isMember, err := auth.IsMember(ctx, userID, teamID)

		require.NoError(t, err)
		assert.True(t, isMember)
	})

	t.Run("returns false for non-member", func(t *testing.T) {
		finder := &mockMemberFinder{
			err: apperrors.ErrNotTeamMember,
		}
		auth := NewLocalAuthorizer(finder)

		isMember, err := auth.IsMember(ctx, userID, teamID)

		require.NoError(t, err)
		assert.False(t, isMember)
	})

	t.Run("propagates database error", func(t *testing.T) {
		dbError := errors.New("database error")
		finder := &mockMemberFinder{err: dbError}
		auth := NewLocalAuthorizer(finder)

		isMember, err := auth.IsMember(ctx, userID, teamID)

		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		assert.False(t, isMember)
	})
}
