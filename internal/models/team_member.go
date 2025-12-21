package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Team role constants.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// TeamMember represents a user's membership in a team.
type TeamMember struct {
	ID       primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	TeamID   primitive.ObjectID `json:"teamId" bson:"teamId" example:"507f1f77bcf86cd799439012"`
	UserID   primitive.ObjectID `json:"userId" bson:"userId" example:"507f1f77bcf86cd799439013"`
	Role     string             `json:"role" bson:"role" example:"member"`
	JoinedAt time.Time          `json:"joinedAt" bson:"joinedAt" example:"2024-01-15T09:30:00Z"`
}

// TeamMemberWithUser is a team member with expanded user information.
type TeamMemberWithUser struct {
	ID       primitive.ObjectID `json:"id" example:"507f1f77bcf86cd799439011"`
	TeamID   primitive.ObjectID `json:"teamId" example:"507f1f77bcf86cd799439012"`
	UserID   primitive.ObjectID `json:"userId" example:"507f1f77bcf86cd799439013"`
	User     *UserSummary       `json:"user,omitempty"`
	Role     string             `json:"role" example:"member"`
	JoinedAt time.Time          `json:"joinedAt" example:"2024-01-15T09:30:00Z"`
}

// UserSummary is a minimal user representation for embedding.
type UserSummary struct {
	ID    primitive.ObjectID `json:"id" example:"507f1f77bcf86cd799439013"`
	Email string             `json:"email" example:"user@example.com"`
	Name  string             `json:"name" example:"John Doe"`
}

// UpdateRoleRequest is the payload for updating a member's role.
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin member" example:"admin"`
}

// TeamMemberListResponse is the response for listing team members.
type TeamMemberListResponse struct {
	Items []TeamMemberWithUser `json:"items"`
}
