package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TeamInvitation represents an invitation to join a team.
type TeamInvitation struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	TeamID    primitive.ObjectID `json:"teamId" bson:"teamId" example:"507f1f77bcf86cd799439012"`
	Email     string             `json:"email" bson:"email" example:"newuser@example.com"`
	InvitedBy primitive.ObjectID `json:"invitedBy" bson:"invitedBy" example:"507f1f77bcf86cd799439013"`
	Role      string             `json:"role" bson:"role" example:"member"`
	ExpiresAt time.Time          `json:"expiresAt" bson:"expiresAt" example:"2024-01-22T09:30:00Z"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt" example:"2024-01-15T09:30:00Z"`
}

// TeamInvitationWithDetails is an invitation with expanded team and inviter info.
type TeamInvitationWithDetails struct {
	ID        primitive.ObjectID `json:"id" example:"507f1f77bcf86cd799439011"`
	Team      *TeamSummary       `json:"team,omitempty"`
	InvitedBy *UserSummary       `json:"invitedBy,omitempty"`
	Role      string             `json:"role" example:"member"`
	ExpiresAt time.Time          `json:"expiresAt" example:"2024-01-22T09:30:00Z"`
	CreatedAt time.Time          `json:"createdAt" example:"2024-01-15T09:30:00Z"`
}

// TeamSummary is a minimal team representation for embedding.
type TeamSummary struct {
	ID   primitive.ObjectID `json:"id" example:"507f1f77bcf86cd799439012"`
	Name string             `json:"name" example:"Engineering Team"`
	Slug string             `json:"slug" example:"engineering"`
}

// CreateInvitationRequest is the payload for creating an invitation.
type CreateInvitationRequest struct {
	Email string `json:"email" binding:"required,email" example:"newuser@example.com"`
	Role  string `json:"role" binding:"required,oneof=admin member" example:"member"`
}

// InvitationListResponse is the response for listing invitations.
type InvitationListResponse struct {
	Items []TeamInvitation `json:"items"`
}

// MyInvitationListResponse is the response for listing user's pending invitations.
type MyInvitationListResponse struct {
	Items []TeamInvitationWithDetails `json:"items"`
}

// AcceptInvitationResponse is the response for accepting an invitation.
type AcceptInvitationResponse struct {
	Message string `json:"message" example:"invitation accepted"`
	TeamID  string `json:"teamId" example:"507f1f77bcf86cd799439012"`
}
