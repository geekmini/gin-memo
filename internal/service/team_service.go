package service

import (
	"context"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TeamService handles business logic for team operations.
type TeamService struct {
	teamRepo       repository.TeamRepository
	memberRepo     repository.TeamMemberRepository
	invitationRepo repository.TeamInvitationRepository
	memoRepo       repository.VoiceMemoRepository
}

// NewTeamService creates a new TeamService.
func NewTeamService(
	teamRepo repository.TeamRepository,
	memberRepo repository.TeamMemberRepository,
	invitationRepo repository.TeamInvitationRepository,
	memoRepo repository.VoiceMemoRepository,
) *TeamService {
	return &TeamService{
		teamRepo:       teamRepo,
		memberRepo:     memberRepo,
		invitationRepo: invitationRepo,
		memoRepo:       memoRepo,
	}
}

// CreateTeam creates a new team and adds the creator as owner.
func (s *TeamService) CreateTeam(ctx context.Context, userID primitive.ObjectID, req *models.CreateTeamRequest) (*models.Team, error) {
	// Check if user has reached team limit (1 for free users)
	count, err := s.teamRepo.CountByOwnerID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= 1 {
		return nil, apperrors.ErrTeamLimitReached
	}

	// Check if slug is taken
	_, err = s.teamRepo.FindBySlug(ctx, req.Slug)
	if err == nil {
		return nil, apperrors.ErrTeamSlugTaken
	}
	if err != apperrors.ErrTeamNotFound {
		return nil, err
	}

	// Create team
	team := &models.Team{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		OwnerID:     userID,
	}

	if err := s.teamRepo.Create(ctx, team); err != nil {
		return nil, err
	}

	// Add creator as owner member
	member := &models.TeamMember{
		TeamID: team.ID,
		UserID: userID,
		Role:   models.RoleOwner,
	}

	if err := s.memberRepo.Create(ctx, member); err != nil {
		// Rollback team creation on failure
		_ = s.teamRepo.SoftDelete(ctx, team.ID)
		return nil, err
	}

	return team, nil
}

// ListTeams returns paginated teams for a user.
func (s *TeamService) ListTeams(ctx context.Context, userID primitive.ObjectID, page, limit int) (*models.TeamListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 10 {
		limit = 10
	}

	teams, total, err := s.teamRepo.FindByUserID(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}

	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return &models.TeamListResponse{
		Items: teams,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

// GetTeam retrieves a team by ID.
func (s *TeamService) GetTeam(ctx context.Context, teamID primitive.ObjectID) (*models.Team, error) {
	return s.teamRepo.FindByID(ctx, teamID)
}

// UpdateTeam updates a team's information.
func (s *TeamService) UpdateTeam(ctx context.Context, teamID primitive.ObjectID, req *models.UpdateTeamRequest) (*models.Team, error) {
	team, err := s.teamRepo.FindByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		team.Name = *req.Name
	}
	if req.Slug != nil {
		// Check if new slug is taken by another team
		existing, err := s.teamRepo.FindBySlug(ctx, *req.Slug)
		if err == nil && existing.ID != teamID {
			return nil, apperrors.ErrTeamSlugTaken
		}
		if err != nil && err != apperrors.ErrTeamNotFound {
			return nil, err
		}
		team.Slug = *req.Slug
	}
	if req.Description != nil {
		team.Description = *req.Description
	}
	if req.LogoURL != nil {
		team.LogoURL = *req.LogoURL
	}

	if err := s.teamRepo.Update(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

// DeleteTeam soft deletes a team and all related data.
func (s *TeamService) DeleteTeam(ctx context.Context, teamID primitive.ObjectID) error {
	// Soft delete team voice memos
	if err := s.memoRepo.SoftDeleteByTeamID(ctx, teamID); err != nil {
		return err
	}

	// Hard delete all team members
	if err := s.memberRepo.DeleteAllByTeamID(ctx, teamID); err != nil {
		return err
	}

	// Hard delete all pending invitations
	if err := s.invitationRepo.DeleteAllByTeamID(ctx, teamID); err != nil {
		return err
	}

	// Soft delete team
	return s.teamRepo.SoftDelete(ctx, teamID)
}

// TransferOwnership transfers team ownership to another member.
func (s *TeamService) TransferOwnership(ctx context.Context, teamID, currentOwnerID, newOwnerID primitive.ObjectID) error {
	// Verify new owner is a team member
	newOwnerMember, err := s.memberRepo.FindByTeamAndUser(ctx, teamID, newOwnerID)
	if err != nil {
		return apperrors.ErrNotTeamMember
	}

	// Get the team
	team, err := s.teamRepo.FindByID(ctx, teamID)
	if err != nil {
		return err
	}

	// Update new owner's role to owner
	if err := s.memberRepo.UpdateRole(ctx, teamID, newOwnerID, models.RoleOwner); err != nil {
		return err
	}

	// Demote current owner to admin
	if err := s.memberRepo.UpdateRole(ctx, teamID, currentOwnerID, models.RoleAdmin); err != nil {
		// Rollback new owner role change
		_ = s.memberRepo.UpdateRole(ctx, teamID, newOwnerID, newOwnerMember.Role)
		return err
	}

	// Update team's ownerId
	team.OwnerID = newOwnerID
	if err := s.teamRepo.Update(ctx, team); err != nil {
		// Rollback both role changes
		_ = s.memberRepo.UpdateRole(ctx, teamID, currentOwnerID, models.RoleOwner)
		_ = s.memberRepo.UpdateRole(ctx, teamID, newOwnerID, newOwnerMember.Role)
		return err
	}
	return nil
}
