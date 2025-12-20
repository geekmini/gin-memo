// Package authz provides authorization interfaces and implementations.
// This module is designed for future migration to SpiceDB or API Gateway.
package authz

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Action constants define the authorization actions.
const (
	ActionTeamView         = "team:view"
	ActionTeamUpdate       = "team:update"
	ActionTeamDelete       = "team:delete"
	ActionTeamTransfer     = "team:transfer"
	ActionMemberInvite     = "member:invite"
	ActionMemberRemove     = "member:remove"
	ActionMemberUpdateRole = "member:update_role"
	ActionMemoView         = "memo:view"
	ActionMemoCreate       = "memo:create"
	ActionMemoUpdate       = "memo:update"
	ActionMemoDelete       = "memo:delete"
)

// Authorizer defines the interface for authorization checks.
// Implementations can be swapped for SpiceDB or removed for API Gateway.
type Authorizer interface {
	// CanPerform checks if a user can perform an action on a team.
	CanPerform(ctx context.Context, userID, teamID primitive.ObjectID, action string) (bool, error)

	// GetUserRole returns the user's role in a team, or empty string if not a member.
	GetUserRole(ctx context.Context, userID, teamID primitive.ObjectID) (string, error)

	// IsMember checks if a user is a member of a team.
	IsMember(ctx context.Context, userID, teamID primitive.ObjectID) (bool, error)
}
