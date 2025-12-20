package service

import (
	"context"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TeamMemberService handles business logic for team member operations.
type TeamMemberService struct {
	memberRepo repository.TeamMemberRepository
	userRepo   repository.UserRepository
	teamRepo   repository.TeamRepository
}

// NewTeamMemberService creates a new TeamMemberService.
func NewTeamMemberService(
	memberRepo repository.TeamMemberRepository,
	userRepo repository.UserRepository,
	teamRepo repository.TeamRepository,
) *TeamMemberService {
	return &TeamMemberService{
		memberRepo: memberRepo,
		userRepo:   userRepo,
		teamRepo:   teamRepo,
	}
}

// ListMembers returns all members of a team with user details.
func (s *TeamMemberService) ListMembers(ctx context.Context, teamID primitive.ObjectID) (*models.TeamMemberListResponse, error) {
	members, err := s.memberRepo.FindByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Expand user details for each member
	membersWithUsers := make([]models.TeamMemberWithUser, 0, len(members))
	for _, m := range members {
		memberWithUser := models.TeamMemberWithUser{
			ID:       m.ID,
			TeamID:   m.TeamID,
			UserID:   m.UserID,
			Role:     m.Role,
			JoinedAt: m.JoinedAt,
		}

		// Get user details
		user, err := s.userRepo.FindByID(ctx, m.UserID)
		if err == nil {
			memberWithUser.User = &models.UserSummary{
				ID:    user.ID,
				Email: user.Email,
				Name:  user.Name,
			}
		}

		membersWithUsers = append(membersWithUsers, memberWithUser)
	}

	return &models.TeamMemberListResponse{
		Items: membersWithUsers,
	}, nil
}

// RemoveMember removes a member from a team.
func (s *TeamMemberService) RemoveMember(ctx context.Context, teamID, targetUserID, requestingUserID primitive.ObjectID) error {
	// Get target member
	targetMember, err := s.memberRepo.FindByTeamAndUser(ctx, teamID, targetUserID)
	if err != nil {
		return apperrors.ErrNotTeamMember
	}

	// Cannot remove owner
	if targetMember.Role == models.RoleOwner {
		return apperrors.ErrCannotRemoveOwner
	}

	// Only owner can remove an admin
	if targetMember.Role == models.RoleAdmin {
		requestingMember, err := s.memberRepo.FindByTeamAndUser(ctx, teamID, requestingUserID)
		if err != nil || requestingMember.Role != models.RoleOwner {
			return apperrors.ErrInsufficientPermissions
		}
	}

	// Cannot remove self (use leave endpoint)
	if targetUserID == requestingUserID {
		return apperrors.ErrCannotRemoveSelf
	}

	return s.memberRepo.Delete(ctx, teamID, targetUserID)
}

// UpdateRole updates a member's role in a team.
func (s *TeamMemberService) UpdateRole(ctx context.Context, teamID, targetUserID, requestingUserID primitive.ObjectID, newRole string) error {
	// Validate role
	if newRole != models.RoleAdmin && newRole != models.RoleMember {
		return apperrors.ErrInvalidRole
	}

	// Get target member
	targetMember, err := s.memberRepo.FindByTeamAndUser(ctx, teamID, targetUserID)
	if err != nil {
		return apperrors.ErrNotTeamMember
	}

	// Cannot change owner role
	if targetMember.Role == models.RoleOwner {
		return apperrors.ErrCannotChangeOwnerRole
	}

	// Only owner can change an admin's role
	if targetMember.Role == models.RoleAdmin {
		requestingMember, err := s.memberRepo.FindByTeamAndUser(ctx, teamID, requestingUserID)
		if err != nil || requestingMember.Role != models.RoleOwner {
			return apperrors.ErrInsufficientPermissions
		}
	}

	return s.memberRepo.UpdateRole(ctx, teamID, targetUserID, newRole)
}

// LeaveTeam removes the requesting user from a team.
func (s *TeamMemberService) LeaveTeam(ctx context.Context, teamID, userID primitive.ObjectID) error {
	// Get member
	member, err := s.memberRepo.FindByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		return apperrors.ErrNotTeamMember
	}

	// Owner cannot leave
	if member.Role == models.RoleOwner {
		return apperrors.ErrOwnerCannotLeave
	}

	return s.memberRepo.Delete(ctx, teamID, userID)
}

// GetMember returns a team member by team and user ID.
func (s *TeamMemberService) GetMember(ctx context.Context, teamID, userID primitive.ObjectID) (*models.TeamMember, error) {
	return s.memberRepo.FindByTeamAndUser(ctx, teamID, userID)
}
