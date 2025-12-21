package service

import (
	"context"
	"testing"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	repomocks "gin-sample/internal/repository/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

func TestNewTeamMemberService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
	mockUserRepo := repomocks.NewMockUserRepository(ctrl)
	mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

	service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)

	assert.NotNil(t, service)
}

func TestTeamMemberService_ListMembers(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	t.Run("returns members with user details", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		members := []models.TeamMember{
			{ID: primitive.NewObjectID(), TeamID: teamID, UserID: userID, Role: models.RoleMember},
		}
		user := &models.User{ID: userID, Email: "test@example.com", Name: "Test User"}

		mockMemberRepo.EXPECT().
			FindByTeamID(gomock.Any(), teamID).
			Return(members, nil)

		mockUserRepo.EXPECT().
			FindByID(gomock.Any(), userID).
			Return(user, nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		result, err := service.ListMembers(context.Background(), teamID)

		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		assert.NotNil(t, result.Items[0].User)
		assert.Equal(t, user.Email, result.Items[0].User.Email)
	})

	t.Run("returns members without user details when user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		members := []models.TeamMember{
			{ID: primitive.NewObjectID(), TeamID: teamID, UserID: userID, Role: models.RoleMember},
		}

		mockMemberRepo.EXPECT().
			FindByTeamID(gomock.Any(), teamID).
			Return(members, nil)

		mockUserRepo.EXPECT().
			FindByID(gomock.Any(), userID).
			Return(nil, apperrors.ErrUserNotFound)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		result, err := service.ListMembers(context.Background(), teamID)

		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		assert.Nil(t, result.Items[0].User) // User details not available
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		mockMemberRepo.EXPECT().
			FindByTeamID(gomock.Any(), teamID).
			Return(nil, assert.AnError)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		result, err := service.ListMembers(context.Background(), teamID)

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

func TestTeamMemberService_RemoveMember(t *testing.T) {
	teamID := primitive.NewObjectID()
	ownerID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	memberID := primitive.NewObjectID()

	t.Run("owner can remove member", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		targetMember := &models.TeamMember{TeamID: teamID, UserID: memberID, Role: models.RoleMember}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, memberID).
			Return(targetMember, nil)

		mockMemberRepo.EXPECT().
			Delete(gomock.Any(), teamID, memberID).
			Return(nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.RemoveMember(context.Background(), teamID, memberID, ownerID)

		assert.NoError(t, err)
	})

	t.Run("admin can remove member", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		targetMember := &models.TeamMember{TeamID: teamID, UserID: memberID, Role: models.RoleMember}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, memberID).
			Return(targetMember, nil)

		mockMemberRepo.EXPECT().
			Delete(gomock.Any(), teamID, memberID).
			Return(nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.RemoveMember(context.Background(), teamID, memberID, adminID)

		assert.NoError(t, err)
	})

	t.Run("cannot remove owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		targetMember := &models.TeamMember{TeamID: teamID, UserID: ownerID, Role: models.RoleOwner}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, ownerID).
			Return(targetMember, nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.RemoveMember(context.Background(), teamID, ownerID, adminID)

		assert.Equal(t, apperrors.ErrCannotRemoveOwner, err)
	})

	t.Run("only owner can remove admin", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		adminMember := &models.TeamMember{TeamID: teamID, UserID: adminID, Role: models.RoleAdmin}
		requestingMember := &models.TeamMember{TeamID: teamID, UserID: memberID, Role: models.RoleMember}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, adminID).
			Return(adminMember, nil)

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, memberID).
			Return(requestingMember, nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.RemoveMember(context.Background(), teamID, adminID, memberID)

		assert.Equal(t, apperrors.ErrInsufficientPermissions, err)
	})

	t.Run("owner can remove admin", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		adminMember := &models.TeamMember{TeamID: teamID, UserID: adminID, Role: models.RoleAdmin}
		ownerMember := &models.TeamMember{TeamID: teamID, UserID: ownerID, Role: models.RoleOwner}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, adminID).
			Return(adminMember, nil)

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, ownerID).
			Return(ownerMember, nil)

		mockMemberRepo.EXPECT().
			Delete(gomock.Any(), teamID, adminID).
			Return(nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.RemoveMember(context.Background(), teamID, adminID, ownerID)

		assert.NoError(t, err)
	})

	t.Run("cannot remove self", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		member := &models.TeamMember{TeamID: teamID, UserID: memberID, Role: models.RoleMember}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, memberID).
			Return(member, nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.RemoveMember(context.Background(), teamID, memberID, memberID) // Same user

		assert.Equal(t, apperrors.ErrCannotRemoveSelf, err)
	})

	t.Run("returns error when target not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, memberID).
			Return(nil, apperrors.ErrNotTeamMember)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.RemoveMember(context.Background(), teamID, memberID, ownerID)

		assert.Equal(t, apperrors.ErrNotTeamMember, err)
	})
}

func TestTeamMemberService_UpdateRole(t *testing.T) {
	teamID := primitive.NewObjectID()
	ownerID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	memberID := primitive.NewObjectID()

	t.Run("owner can promote member to admin", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		member := &models.TeamMember{TeamID: teamID, UserID: memberID, Role: models.RoleMember}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, memberID).
			Return(member, nil)

		mockMemberRepo.EXPECT().
			UpdateRole(gomock.Any(), teamID, memberID, models.RoleAdmin).
			Return(nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.UpdateRole(context.Background(), teamID, memberID, ownerID, models.RoleAdmin)

		assert.NoError(t, err)
	})

	t.Run("returns error for invalid role", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.UpdateRole(context.Background(), teamID, memberID, ownerID, "invalid-role")

		assert.Equal(t, apperrors.ErrInvalidRole, err)
	})

	t.Run("returns error for owner role", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.UpdateRole(context.Background(), teamID, memberID, ownerID, models.RoleOwner)

		assert.Equal(t, apperrors.ErrInvalidRole, err)
	})

	t.Run("cannot change owner role", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		ownerMember := &models.TeamMember{TeamID: teamID, UserID: ownerID, Role: models.RoleOwner}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, ownerID).
			Return(ownerMember, nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.UpdateRole(context.Background(), teamID, ownerID, adminID, models.RoleAdmin)

		assert.Equal(t, apperrors.ErrCannotChangeOwnerRole, err)
	})

	t.Run("only owner can demote admin", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		adminMember := &models.TeamMember{TeamID: teamID, UserID: adminID, Role: models.RoleAdmin}
		otherAdmin := &models.TeamMember{TeamID: teamID, UserID: primitive.NewObjectID(), Role: models.RoleAdmin}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, adminID).
			Return(adminMember, nil)

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, gomock.Any()).
			Return(otherAdmin, nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.UpdateRole(context.Background(), teamID, adminID, otherAdmin.UserID, models.RoleMember)

		assert.Equal(t, apperrors.ErrInsufficientPermissions, err)
	})
}

func TestTeamMemberService_LeaveTeam(t *testing.T) {
	teamID := primitive.NewObjectID()
	ownerID := primitive.NewObjectID()
	memberID := primitive.NewObjectID()

	t.Run("member can leave team", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		member := &models.TeamMember{TeamID: teamID, UserID: memberID, Role: models.RoleMember}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, memberID).
			Return(member, nil)

		mockMemberRepo.EXPECT().
			Delete(gomock.Any(), teamID, memberID).
			Return(nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.LeaveTeam(context.Background(), teamID, memberID)

		assert.NoError(t, err)
	})

	t.Run("owner cannot leave team", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		owner := &models.TeamMember{TeamID: teamID, UserID: ownerID, Role: models.RoleOwner}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, ownerID).
			Return(owner, nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.LeaveTeam(context.Background(), teamID, ownerID)

		assert.Equal(t, apperrors.ErrOwnerCannotLeave, err)
	})

	t.Run("returns error when not a member", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, memberID).
			Return(nil, apperrors.ErrNotTeamMember)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		err := service.LeaveTeam(context.Background(), teamID, memberID)

		assert.Equal(t, apperrors.ErrNotTeamMember, err)
	})
}

func TestTeamMemberService_GetMember(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	t.Run("returns member", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		member := &models.TeamMember{TeamID: teamID, UserID: userID, Role: models.RoleMember}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, userID).
			Return(member, nil)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		result, err := service.GetMember(context.Background(), teamID, userID)

		require.NoError(t, err)
		assert.Equal(t, member, result)
	})

	t.Run("returns error when not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, userID).
			Return(nil, apperrors.ErrNotTeamMember)

		service := NewTeamMemberService(mockMemberRepo, mockUserRepo, mockTeamRepo)
		result, err := service.GetMember(context.Background(), teamID, userID)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrNotTeamMember, err)
	})
}
