package service

import (
	"context"
	"testing"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	repomocks "gin-sample/internal/repository/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

func TestNewTeamInvitationService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
	mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
	mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
	mockUserRepo := repomocks.NewMockUserRepository(ctrl)

	service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)

	assert.NotNil(t, service)
}

func TestTeamInvitationService_CreateInvitation(t *testing.T) {
	teamID := primitive.NewObjectID()
	inviterID := primitive.NewObjectID()
	createReq := &models.CreateInvitationRequest{
		Email: "invitee@example.com",
		Role:  models.RoleMember,
	}

	t.Run("successfully creates invitation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		team := &models.Team{ID: teamID, Seats: 10}

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockUserRepo.EXPECT().
			FindByEmail(gomock.Any(), createReq.Email).
			Return(nil, apperrors.ErrUserNotFound)

		mockInvitationRepo.EXPECT().
			FindByTeamAndEmail(gomock.Any(), teamID, createReq.Email).
			Return(nil, apperrors.ErrInvitationNotFound)

		mockMemberRepo.EXPECT().
			CountByTeamID(gomock.Any(), teamID).
			Return(3, nil)

		mockInvitationRepo.EXPECT().
			CountPendingByTeamID(gomock.Any(), teamID).
			Return(2, nil)

		mockInvitationRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, inv *models.TeamInvitation) error {
				inv.ID = primitive.NewObjectID()
				assert.Equal(t, createReq.Email, inv.Email)
				assert.Equal(t, createReq.Role, inv.Role)
				return nil
			})

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.CreateInvitation(context.Background(), teamID, inviterID, createReq)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, createReq.Email, result.Email)
	})

	t.Run("returns error when user is already a member", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		team := &models.Team{ID: teamID, Seats: 10}
		existingUserID := primitive.NewObjectID()
		existingUser := &models.User{ID: existingUserID, Email: createReq.Email}

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockUserRepo.EXPECT().
			FindByEmail(gomock.Any(), createReq.Email).
			Return(existingUser, nil)

		mockMemberRepo.EXPECT().
			FindByTeamAndUser(gomock.Any(), teamID, existingUserID).
			Return(&models.TeamMember{}, nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.CreateInvitation(context.Background(), teamID, inviterID, createReq)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrAlreadyMember, err)
	})

	t.Run("returns error when invitation already pending", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		team := &models.Team{ID: teamID, Seats: 10}

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockUserRepo.EXPECT().
			FindByEmail(gomock.Any(), createReq.Email).
			Return(nil, apperrors.ErrUserNotFound)

		mockInvitationRepo.EXPECT().
			FindByTeamAndEmail(gomock.Any(), teamID, createReq.Email).
			Return(&models.TeamInvitation{}, nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.CreateInvitation(context.Background(), teamID, inviterID, createReq)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrPendingInvitation, err)
	})

	t.Run("returns error when seats exceeded", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		team := &models.Team{ID: teamID, Seats: 5}

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockUserRepo.EXPECT().
			FindByEmail(gomock.Any(), createReq.Email).
			Return(nil, apperrors.ErrUserNotFound)

		mockInvitationRepo.EXPECT().
			FindByTeamAndEmail(gomock.Any(), teamID, createReq.Email).
			Return(nil, apperrors.ErrInvitationNotFound)

		mockMemberRepo.EXPECT().
			CountByTeamID(gomock.Any(), teamID).
			Return(3, nil)

		mockInvitationRepo.EXPECT().
			CountPendingByTeamID(gomock.Any(), teamID).
			Return(2, nil) // 3 + 2 = 5 >= 5 seats

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.CreateInvitation(context.Background(), teamID, inviterID, createReq)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrSeatsExceeded, err)
	})
}

func TestTeamInvitationService_ListTeamInvitations(t *testing.T) {
	teamID := primitive.NewObjectID()

	t.Run("returns team invitations", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		invitations := []models.TeamInvitation{
			{ID: primitive.NewObjectID(), Email: "user1@example.com"},
			{ID: primitive.NewObjectID(), Email: "user2@example.com"},
		}

		mockInvitationRepo.EXPECT().
			FindByTeamID(gomock.Any(), teamID).
			Return(invitations, nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.ListTeamInvitations(context.Background(), teamID)

		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
	})
}

func TestTeamInvitationService_CancelInvitation(t *testing.T) {
	teamID := primitive.NewObjectID()
	invitationID := primitive.NewObjectID()

	t.Run("successfully cancels invitation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		invitation := &models.TeamInvitation{ID: invitationID, TeamID: teamID}

		mockInvitationRepo.EXPECT().
			FindByID(gomock.Any(), invitationID).
			Return(invitation, nil)

		mockInvitationRepo.EXPECT().
			Delete(gomock.Any(), invitationID).
			Return(nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		err := service.CancelInvitation(context.Background(), invitationID, teamID)

		assert.NoError(t, err)
	})

	t.Run("returns error when invitation belongs to different team", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		otherTeamID := primitive.NewObjectID()
		invitation := &models.TeamInvitation{ID: invitationID, TeamID: otherTeamID}

		mockInvitationRepo.EXPECT().
			FindByID(gomock.Any(), invitationID).
			Return(invitation, nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		err := service.CancelInvitation(context.Background(), invitationID, teamID)

		assert.Equal(t, apperrors.ErrInvitationNotFound, err)
	})
}

func TestTeamInvitationService_ListMyInvitations(t *testing.T) {
	userEmail := "user@example.com"

	t.Run("returns user invitations with details", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		teamID := primitive.NewObjectID()
		inviterID := primitive.NewObjectID()
		invitations := []models.TeamInvitation{
			{
				ID:        primitive.NewObjectID(),
				TeamID:    teamID,
				Email:     userEmail,
				InvitedBy: inviterID,
				Role:      models.RoleMember,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
		}

		team := &models.Team{ID: teamID, Name: "Test Team", Slug: "test-team"}
		inviter := &models.User{ID: inviterID, Email: "inviter@example.com", Name: "Inviter"}

		mockInvitationRepo.EXPECT().
			FindByEmail(gomock.Any(), userEmail).
			Return(invitations, nil)

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockUserRepo.EXPECT().
			FindByID(gomock.Any(), inviterID).
			Return(inviter, nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.ListMyInvitations(context.Background(), userEmail)

		require.NoError(t, err)
		assert.Len(t, result.Items, 1)
		assert.NotNil(t, result.Items[0].Team)
		assert.NotNil(t, result.Items[0].InvitedBy)
	})
}

func TestTeamInvitationService_AcceptInvitation(t *testing.T) {
	invitationID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	userEmail := "user@example.com"

	t.Run("successfully accepts invitation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		invitation := &models.TeamInvitation{
			ID:        invitationID,
			TeamID:    teamID,
			Email:     userEmail,
			Role:      models.RoleMember,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		team := &models.Team{ID: teamID, Seats: 10}

		mockInvitationRepo.EXPECT().
			FindByID(gomock.Any(), invitationID).
			Return(invitation, nil)

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockMemberRepo.EXPECT().
			CountByTeamID(gomock.Any(), teamID).
			Return(5, nil)

		mockMemberRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil)

		mockInvitationRepo.EXPECT().
			Delete(gomock.Any(), invitationID).
			Return(nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.AcceptInvitation(context.Background(), invitationID, userID, userEmail)

		require.NoError(t, err)
		assert.Equal(t, teamID.Hex(), result.TeamID)
	})

	t.Run("returns error when email does not match", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		invitation := &models.TeamInvitation{
			ID:    invitationID,
			Email: "other@example.com",
		}

		mockInvitationRepo.EXPECT().
			FindByID(gomock.Any(), invitationID).
			Return(invitation, nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.AcceptInvitation(context.Background(), invitationID, userID, userEmail)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrInvitationEmailMismatch, err)
	})

	t.Run("returns error when invitation expired", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		invitation := &models.TeamInvitation{
			ID:        invitationID,
			Email:     userEmail,
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
		}

		mockInvitationRepo.EXPECT().
			FindByID(gomock.Any(), invitationID).
			Return(invitation, nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.AcceptInvitation(context.Background(), invitationID, userID, userEmail)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrInvitationExpired, err)
	})

	t.Run("returns error when seats exceeded", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		invitation := &models.TeamInvitation{
			ID:        invitationID,
			TeamID:    teamID,
			Email:     userEmail,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}

		team := &models.Team{ID: teamID, Seats: 5}

		mockInvitationRepo.EXPECT().
			FindByID(gomock.Any(), invitationID).
			Return(invitation, nil)

		mockTeamRepo.EXPECT().
			FindByID(gomock.Any(), teamID).
			Return(team, nil)

		mockMemberRepo.EXPECT().
			CountByTeamID(gomock.Any(), teamID).
			Return(5, nil) // At capacity

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		result, err := service.AcceptInvitation(context.Background(), invitationID, userID, userEmail)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrSeatsExceeded, err)
	})
}

func TestTeamInvitationService_DeclineInvitation(t *testing.T) {
	invitationID := primitive.NewObjectID()
	userEmail := "user@example.com"

	t.Run("successfully declines invitation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		invitation := &models.TeamInvitation{
			ID:    invitationID,
			Email: userEmail,
		}

		mockInvitationRepo.EXPECT().
			FindByID(gomock.Any(), invitationID).
			Return(invitation, nil)

		mockInvitationRepo.EXPECT().
			Delete(gomock.Any(), invitationID).
			Return(nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		err := service.DeclineInvitation(context.Background(), invitationID, userEmail)

		assert.NoError(t, err)
	})

	t.Run("returns error when email does not match", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockInvitationRepo := repomocks.NewMockTeamInvitationRepository(ctrl)
		mockMemberRepo := repomocks.NewMockTeamMemberRepository(ctrl)
		mockTeamRepo := repomocks.NewMockTeamRepository(ctrl)
		mockUserRepo := repomocks.NewMockUserRepository(ctrl)

		invitation := &models.TeamInvitation{
			ID:    invitationID,
			Email: "other@example.com",
		}

		mockInvitationRepo.EXPECT().
			FindByID(gomock.Any(), invitationID).
			Return(invitation, nil)

		service := NewTeamInvitationService(mockInvitationRepo, mockMemberRepo, mockTeamRepo, mockUserRepo)
		err := service.DeclineInvitation(context.Background(), invitationID, userEmail)

		assert.Equal(t, apperrors.ErrInvitationEmailMismatch, err)
	})
}
