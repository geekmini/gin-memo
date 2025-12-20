package authz

import (
	"context"
	"errors"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TeamMemberFinder is the interface required by LocalAuthorizer to look up team membership.
// This allows the authorizer to be decoupled from the full repository implementation.
type TeamMemberFinder interface {
	FindByTeamAndUser(ctx context.Context, teamID, userID primitive.ObjectID) (*models.TeamMember, error)
}

// LocalAuthorizer implements Authorizer using database lookups.
// This is the initial implementation that can be replaced with SpiceDBAuthorizer later.
type LocalAuthorizer struct {
	memberFinder TeamMemberFinder
}

// NewLocalAuthorizer creates a new LocalAuthorizer.
func NewLocalAuthorizer(memberFinder TeamMemberFinder) *LocalAuthorizer {
	return &LocalAuthorizer{
		memberFinder: memberFinder,
	}
}

// rolePermissions maps actions to the roles that can perform them.
var rolePermissions = map[string][]string{
	ActionTeamView:         {models.RoleOwner, models.RoleAdmin, models.RoleMember},
	ActionTeamUpdate:       {models.RoleOwner, models.RoleAdmin},
	ActionTeamDelete:       {models.RoleOwner},
	ActionTeamTransfer:     {models.RoleOwner},
	ActionMemberInvite:     {models.RoleOwner, models.RoleAdmin},
	ActionMemberRemove:     {models.RoleOwner, models.RoleAdmin},
	ActionMemberUpdateRole: {models.RoleOwner, models.RoleAdmin},
	ActionMemoView:         {models.RoleOwner, models.RoleAdmin, models.RoleMember},
	ActionMemoCreate:       {models.RoleOwner, models.RoleAdmin, models.RoleMember},
	ActionMemoUpdate:       {models.RoleOwner, models.RoleAdmin, models.RoleMember},
	ActionMemoDelete:       {models.RoleOwner, models.RoleAdmin, models.RoleMember},
}

// CanPerform checks if a user can perform an action on a team.
func (a *LocalAuthorizer) CanPerform(ctx context.Context, userID, teamID primitive.ObjectID, action string) (bool, error) {
	member, err := a.memberFinder.FindByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotTeamMember) {
			return false, nil // Expected: not a member
		}
		return false, err // Unexpected: propagate error
	}

	allowedRoles, exists := rolePermissions[action]
	if !exists {
		return false, nil // Unknown action
	}

	for _, role := range allowedRoles {
		if member.Role == role {
			return true, nil
		}
	}

	return false, nil
}

// GetUserRole returns the user's role in a team, or empty string if not a member.
func (a *LocalAuthorizer) GetUserRole(ctx context.Context, userID, teamID primitive.ObjectID) (string, error) {
	member, err := a.memberFinder.FindByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotTeamMember) {
			return "", nil // Expected: not a member
		}
		return "", err // Unexpected: propagate error
	}
	return member.Role, nil
}

// IsMember checks if a user is a member of a team.
func (a *LocalAuthorizer) IsMember(ctx context.Context, userID, teamID primitive.ObjectID) (bool, error) {
	member, err := a.memberFinder.FindByTeamAndUser(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotTeamMember) {
			return false, nil // Expected: not a member
		}
		return false, err // Unexpected: propagate error
	}
	return member != nil, nil
}
