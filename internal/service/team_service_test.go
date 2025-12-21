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

func TestNewTeamService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
	mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
	mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
	mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

	service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)

	assert.NotNil(t, service)
}

func TestTeamService_CreateTeam(t *testing.T) {
	userID := primitive.NewObjectID()
	createReq := &models.CreateTeamRequest{
		Name:        "Test Team",
		Slug:        "test-team",
		Description: "A test team",
	}

	t.Run("successfully creates team with owner membership", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockTeamRepo.EXPECT().
			CountByOwnerID(gomock.Any(), userID).
			Return(0, nil)

		mockTeamRepo.EXPECT().
			FindBySlug(gomock.Any(), createReq.Slug).
			Return(nil, apperrors.ErrTeamNotFound)

		mockTeamRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, team *models.Team) error {
				team.ID = primitive.NewObjectID()
				assert.Equal(t, createReq.Name, team.Name)
				assert.Equal(t, createReq.Slug, team.Slug)
				assert.Equal(t, userID, team.OwnerID)
				return nil
			})

		mockMemberRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, member *models.TeamMember) error {
				assert.Equal(t, userID, member.UserID)
				assert.Equal(t, models.RoleOwner, member.Role)
				return nil
			})

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		team, err := service.CreateTeam(context.Background(), userID, createReq)

		require.NoError(t, err)
		assert.NotNil(t, team)
		assert.Equal(t, createReq.Name, team.Name)
	})

	t.Run("returns error when team limit reached", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockTeamRepo.EXPECT().
			CountByOwnerID(gomock.Any(), userID).
			Return(1, nil) // Already has 1 team

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		team, err := service.CreateTeam(context.Background(), userID, createReq)

		assert.Nil(t, team)
		assert.Equal(t, apperrors.ErrTeamLimitReached, err)
	})

	t.Run("returns error when slug is taken", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockTeamRepo.EXPECT().
			CountByOwnerID(gomock.Any(), userID).
			Return(0, nil)

		existingTeam := &models.Team{ID: primitive.NewObjectID(), Slug: createReq.Slug}
		mockTeamRepo.EXPECT().
			FindBySlug(gomock.Any(), createReq.Slug).
			Return(existingTeam, nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		team, err := service.CreateTeam(context.Background(), userID, createReq)

		assert.Nil(t, team)
		assert.Equal(t, apperrors.ErrTeamSlugTaken, err)
	})

	t.Run("rolls back team on member creation failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		var createdTeamID primitive.ObjectID
		mockTeamRepo.EXPECT().
			CountByOwnerID(gomock.Any(), userID).
			Return(0, nil)

		mockTeamRepo.EXPECT().
			FindBySlug(gomock.Any(), createReq.Slug).
			Return(nil, apperrors.ErrTeamNotFound)

		mockTeamRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, team *models.Team) error {
				createdTeamID = primitive.NewObjectID()
				team.ID = createdTeamID
				return nil
			})

		mockMemberRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(assert.AnError)

		mockTeamRepo.EXPECT().
			SoftDelete(gomock.Any(), gomock.Any()).
			Return(nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		team, err := service.CreateTeam(context.Background(), userID, createReq)

		assert.Nil(t, team)
		assert.Error(t, err)
	})
}

func TestTeamService_ListTeams(t *testing.T) {
	userID := primitive.NewObjectID()

	t.Run("returns paginated teams", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		teams := []models.Team{
			{ID: primitive.NewObjectID(), Name: "Team 1"},
			{ID: primitive.NewObjectID(), Name: "Team 2"},
		}

		mockTeamRepo.EXPECT().
			FindByUserID(gomock.Any(), userID, 1, 10).
			Return(teams, 2, nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		result, err := service.ListTeams(context.Background(), userID, 1, 10)

		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
		assert.Equal(t, 1, result.Pagination.Page)
		assert.Equal(t, 1, result.Pagination.TotalPages)
	})

	t.Run("sets default page and limit values", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockTeamRepo.EXPECT().
			FindByUserID(gomock.Any(), userID, 1, 10). // Default values
			Return([]models.Team{}, 0, nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		_, err := service.ListTeams(context.Background(), userID, 0, 0) // Invalid values

		assert.NoError(t, err)
	})

	t.Run("caps limit at 10", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockTeamRepo.EXPECT().
			FindByUserID(gomock.Any(), userID, 1, 10). // Capped at 10
			Return([]models.Team{}, 0, nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		_, err := service.ListTeams(context.Background(), userID, 1, 100) // Request 100

		assert.NoError(t, err)
	})
}

func TestTeamService_GetTeam(t *testing.T) {
	teamID := primitive.NewObjectID()

	t.Run("returns team by ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		team := &models.Team{ID: teamID, Name: "Test Team"}
		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		result, err := service.GetTeam(context.Background(), teamID)

		require.NoError(t, err)
		assert.Equal(t, team, result)
	})

	t.Run("returns error when team not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(nil, apperrors.ErrTeamNotFound)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		result, err := service.GetTeam(context.Background(), teamID)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrTeamNotFound, err)
	})
}

func TestTeamService_UpdateTeam(t *testing.T) {
	teamID := primitive.NewObjectID()

	t.Run("updates team name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		newName := "Updated Name"
		updateReq := &models.UpdateTeamRequest{Name: &newName}

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(&models.Team{ID: teamID, Name: "Original"}, nil)

		mockTeamRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		result, err := service.UpdateTeam(context.Background(), teamID, updateReq)

		require.NoError(t, err)
		assert.Equal(t, newName, result.Name)
	})

	t.Run("updates slug when available", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		newSlug := "new-slug"
		updateReq := &models.UpdateTeamRequest{Slug: &newSlug}

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(&models.Team{ID: teamID, Slug: "old-slug"}, nil)

		mockTeamRepo.EXPECT().
			FindBySlug(gomock.Any(), newSlug).
			Return(nil, apperrors.ErrTeamNotFound) // Slug available

		mockTeamRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		result, err := service.UpdateTeam(context.Background(), teamID, updateReq)

		require.NoError(t, err)
		assert.Equal(t, newSlug, result.Slug)
	})

	t.Run("returns error when new slug is taken", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		newSlug := "taken-slug"
		updateReq := &models.UpdateTeamRequest{Slug: &newSlug}

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(&models.Team{ID: teamID, Slug: "old-slug"}, nil)

		otherTeamID := primitive.NewObjectID()
		mockTeamRepo.EXPECT().
			FindBySlug(gomock.Any(), newSlug).
			Return(&models.Team{ID: otherTeamID, Slug: newSlug}, nil) // Different team has slug

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		result, err := service.UpdateTeam(context.Background(), teamID, updateReq)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrTeamSlugTaken, err)
	})

	t.Run("allows same slug for same team", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		sameSlug := "same-slug"
		updateReq := &models.UpdateTeamRequest{Slug: &sameSlug}

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(&models.Team{ID: teamID, Slug: sameSlug}, nil)

		mockTeamRepo.EXPECT().
			FindBySlug(gomock.Any(), sameSlug).
			Return(&models.Team{ID: teamID, Slug: sameSlug}, nil) // Same team

		mockTeamRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		_, err := service.UpdateTeam(context.Background(), teamID, updateReq)

		assert.NoError(t, err)
	})
}

func TestTeamService_DeleteTeam(t *testing.T) {
	teamID := primitive.NewObjectID()

	t.Run("deletes team and all related data", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockMemoRepo.EXPECT().
			SoftDeleteByTeamID(gomock.Any(), teamID).
			Return(nil)

		mockMemberRepo.EXPECT().
			DeleteAllByTeamID(gomock.Any(), teamID).
			Return(nil)

		mockInvitationRepo.EXPECT().
			DeleteAllByTeamID(gomock.Any(), teamID).
			Return(nil)

		mockTeamRepo.EXPECT().
			SoftDelete(gomock.Any(), teamID).
			Return(nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		err := service.DeleteTeam(context.Background(), teamID)

		assert.NoError(t, err)
	})

	t.Run("returns error if memo deletion fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockMemoRepo.EXPECT().
			SoftDeleteByTeamID(gomock.Any(), teamID).
			Return(assert.AnError)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		err := service.DeleteTeam(context.Background(), teamID)

		assert.Error(t, err)
	})
}

func TestTeamService_TransferOwnership(t *testing.T) {
	teamID := primitive.NewObjectID()
	currentOwnerID := primitive.NewObjectID()
	newOwnerID := primitive.NewObjectID()

	t.Run("successfully transfers ownership", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		newOwnerMember := &models.TeamMember{
			TeamID: teamID,
			UserID: newOwnerID,
			Role:   models.RoleMember,
		}

		team := &models.Team{
			ID:      teamID,
			OwnerID: currentOwnerID,
		}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, newOwnerID).
			Return(newOwnerMember, nil)

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockMemberRepo.EXPECT().
			UpdateRole(gomock.Any(), teamID, newOwnerID, models.RoleOwner).
			Return(nil)

		mockMemberRepo.EXPECT().
			UpdateRole(gomock.Any(), teamID, currentOwnerID, models.RoleAdmin).
			Return(nil)

		mockTeamRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		err := service.TransferOwnership(context.Background(), teamID, currentOwnerID, newOwnerID)

		assert.NoError(t, err)
	})

	t.Run("returns error when new owner is not a member", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, newOwnerID).
			Return(nil, apperrors.ErrNotTeamMember)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		err := service.TransferOwnership(context.Background(), teamID, currentOwnerID, newOwnerID)

		assert.Equal(t, apperrors.ErrNotTeamMember, err)
	})

	t.Run("rolls back on team update failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemoRepo := repomocks.NewMockVoiceMemoRepository(ctrl)

		newOwnerMember := &models.TeamMember{
			TeamID: teamID,
			UserID: newOwnerID,
			Role:   models.RoleMember,
		}

		team := &models.Team{ID: teamID, OwnerID: currentOwnerID}

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, newOwnerID).
			Return(newOwnerMember, nil)

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockMemberRepo.EXPECT().
			UpdateRole(gomock.Any(), teamID, newOwnerID, models.RoleOwner).
			Return(nil)

		mockMemberRepo.EXPECT().
			UpdateRole(gomock.Any(), teamID, currentOwnerID, models.RoleAdmin).
			Return(nil)

		mockTeamRepo.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			Return(assert.AnError) // Update fails

		// Rollback calls
		mockMemberRepo.EXPECT().
			UpdateRole(gomock.Any(), teamID, currentOwnerID, models.RoleOwner).
			Return(nil)

		mockMemberRepo.EXPECT().
			UpdateRole(gomock.Any(), teamID, newOwnerID, models.RoleMember).
			Return(nil)

		service := NewTeamService(mockTeamRepo, mockMemberRepo, mockInvitationRepo, mockMemoRepo)
		err := service.TransferOwnership(context.Background(), teamID, currentOwnerID, newOwnerID)

		assert.Error(t, err)
	})
}
