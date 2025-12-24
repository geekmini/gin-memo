package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TeamInvitationService handles business logic for invitation operations.
type TeamInvitationService struct {
	invitationRepo repository.TeamInvitationRepository
	memberRepo     repository.TeamMemberRepository
	teamRepo       repository.TeamRepository
	userRepo       repository.UserRepository
}

// NewTeamInvitationService creates a new TeamInvitationService.
func NewTeamInvitationService(
	invitationRepo repository.TeamInvitationRepository,
	memberRepo repository.TeamMemberRepository,
	teamRepo repository.TeamRepository,
	userRepo repository.UserRepository,
) *TeamInvitationService {
	return &TeamInvitationService{
		invitationRepo: invitationRepo,
		memberRepo:     memberRepo,
		teamRepo:       teamRepo,
		userRepo:       userRepo,
	}
}

// CreateInvitation creates a new invitation to join a team.
func (s *TeamInvitationService) CreateInvitation(ctx context.Context, teamID, inviterID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Get team to check seats
	team, err := s.teamRepo.FindByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Check if email already belongs to a team member
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err == nil {
		// User exists - check if already a member
		_, err := s.memberRepo.FindByTeamAndUser(ctx, teamID, user.ID)
		if err == nil {
			return nil, apperrors.ErrAlreadyMember
		}
	}

	// Check for existing pending invitation
	_, err = s.invitationRepo.FindByTeamAndEmail(ctx, teamID, email)
	if err == nil {
		return nil, apperrors.ErrPendingInvitation
	}
	if !errors.Is(err, apperrors.ErrInvitationNotFound) {
		return nil, err
	}

	// Check seats limit (members + pending invitations)
	memberCount, err := s.memberRepo.CountByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	invitationCount, err := s.invitationRepo.CountPendingByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if memberCount+invitationCount >= team.Seats {
		return nil, apperrors.ErrSeatsExceeded
	}

	// Create invitation
	invitation := &models.TeamInvitation{
		TeamID:    teamID,
		Email:     email,
		InvitedBy: inviterID,
		Role:      req.Role,
	}

	if err := s.invitationRepo.Create(ctx, invitation); err != nil {
		return nil, err
	}

	return invitation, nil
}

// ListTeamInvitations returns all pending invitations for a team.
func (s *TeamInvitationService) ListTeamInvitations(ctx context.Context, teamID primitive.ObjectID) (*models.InvitationListResponse, error) {
	invitations, err := s.invitationRepo.FindByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	return &models.InvitationListResponse{
		Items: invitations,
	}, nil
}

// CancelInvitation cancels a pending invitation.
func (s *TeamInvitationService) CancelInvitation(ctx context.Context, invitationID, teamID primitive.ObjectID) error {
	// Verify invitation belongs to team
	invitation, err := s.invitationRepo.FindByID(ctx, invitationID)
	if err != nil {
		return err
	}
	if invitation.TeamID != teamID {
		return apperrors.ErrInvitationNotFound
	}

	return s.invitationRepo.Delete(ctx, invitationID)
}

// ListMyInvitations returns all pending invitations for a user's email.
func (s *TeamInvitationService) ListMyInvitations(ctx context.Context, userEmail string) (*models.MyInvitationListResponse, error) {
	email := strings.ToLower(strings.TrimSpace(userEmail))

	invitations, err := s.invitationRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	// Expand team and inviter details
	invitationsWithDetails := make([]models.TeamInvitationWithDetails, 0, len(invitations))
	for _, inv := range invitations {
		detail := models.TeamInvitationWithDetails{
			ID:        inv.ID,
			Role:      inv.Role,
			ExpiresAt: inv.ExpiresAt,
			CreatedAt: inv.CreatedAt,
		}

		// Get team details
		team, err := s.teamRepo.FindByID(ctx, inv.TeamID)
		if err == nil {
			detail.Team = &models.TeamSummary{
				ID:   team.ID,
				Name: team.Name,
				Slug: team.Slug,
			}
		}

		// Get inviter details
		inviter, err := s.userRepo.FindByID(ctx, inv.InvitedBy)
		if err == nil {
			detail.InvitedBy = &models.UserSummary{
				ID:    inviter.ID,
				Email: inviter.Email,
				Name:  inviter.Name,
			}
		}

		invitationsWithDetails = append(invitationsWithDetails, detail)
	}

	return &models.MyInvitationListResponse{
		Items: invitationsWithDetails,
	}, nil
}

// AcceptInvitation accepts an invitation and adds the user to the team.
func (s *TeamInvitationService) AcceptInvitation(ctx context.Context, invitationID, userID primitive.ObjectID, userEmail string) (*models.AcceptInvitationResponse, error) {
	invitation, err := s.invitationRepo.FindByID(ctx, invitationID)
	if err != nil {
		return nil, err
	}

	// Verify email matches
	email := strings.ToLower(strings.TrimSpace(userEmail))
	if strings.ToLower(invitation.Email) != email {
		return nil, apperrors.ErrInvitationEmailMismatch
	}

	// Check if invitation is expired
	if invitation.ExpiresAt.Before(time.Now()) {
		return nil, apperrors.ErrInvitationExpired
	}

	// Get team to check seats
	team, err := s.teamRepo.FindByID(ctx, invitation.TeamID)
	if err != nil {
		return nil, err
	}

	// Check seats limit
	memberCount, err := s.memberRepo.CountByTeamID(ctx, invitation.TeamID)
	if err != nil {
		return nil, err
	}
	if memberCount >= team.Seats {
		return nil, apperrors.ErrSeatsExceeded
	}

	// Add user as team member
	member := &models.TeamMember{
		TeamID: invitation.TeamID,
		UserID: userID,
		Role:   invitation.Role,
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		return nil, err
	}

	// Delete the invitation (member already created, so log error but don't fail)
	if err := s.invitationRepo.Delete(ctx, invitationID); err != nil {
		log.Printf("Warning: failed to delete invitation %s after accepting: %v", invitationID.Hex(), err)
	}

	return &models.AcceptInvitationResponse{
		Message: "invitation accepted",
		TeamID:  invitation.TeamID.Hex(),
	}, nil
}

// DeclineInvitation declines an invitation.
func (s *TeamInvitationService) DeclineInvitation(ctx context.Context, invitationID primitive.ObjectID, userEmail string) error {
	invitation, err := s.invitationRepo.FindByID(ctx, invitationID)
	if err != nil {
		return err
	}

	// Verify email matches
	email := strings.ToLower(strings.TrimSpace(userEmail))
	if strings.ToLower(invitation.Email) != email {
		return apperrors.ErrInvitationEmailMismatch
	}

	return s.invitationRepo.Delete(ctx, invitationID)
}
