package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Team represents a team in the system.
type Team struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	Name          string             `json:"name" bson:"name" example:"Engineering Team"`
	Slug          string             `json:"slug" bson:"slug" example:"engineering"`
	Description   string             `json:"description" bson:"description" example:"Our engineering team workspace"`
	LogoURL       string             `json:"logoUrl" bson:"logoUrl" example:"https://example.com/logo.png"`
	OwnerID       primitive.ObjectID `json:"ownerId" bson:"ownerId" example:"507f1f77bcf86cd799439012"`
	Seats         int                `json:"seats" bson:"seats" example:"10"`
	RetentionDays int                `json:"retentionDays" bson:"retentionDays" example:"30"`
	CreatedAt     time.Time          `json:"createdAt" bson:"createdAt" example:"2024-01-15T09:30:00Z"`
	UpdatedAt     time.Time          `json:"updatedAt" bson:"updatedAt" example:"2024-01-15T09:30:00Z"`
	DeletedAt     *time.Time         `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
}

// CreateTeamRequest is the payload for creating a team.
type CreateTeamRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100" example:"Engineering Team"`
	Slug        string `json:"slug" binding:"required,min=2,max=50,alphanum" example:"engineering"`
	Description string `json:"description" binding:"omitempty,max=500" example:"Our engineering team workspace"`
	LogoURL     string `json:"logoUrl" binding:"omitempty,url" example:"https://example.com/logo.png"`
}

// UpdateTeamRequest is the payload for updating a team.
type UpdateTeamRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=100" example:"Updated Team Name"`
	Slug        *string `json:"slug" binding:"omitempty,min=2,max=50,alphanum" example:"updated-slug"`
	Description *string `json:"description" binding:"omitempty,max=500" example:"Updated description"`
	LogoURL     *string `json:"logoUrl" binding:"omitempty" example:"https://example.com/new-logo.png"`
}

// TransferOwnershipRequest is the payload for transferring team ownership.
type TransferOwnershipRequest struct {
	NewOwnerID string `json:"newOwnerId" binding:"required" example:"507f1f77bcf86cd799439013"`
}

// TeamListResponse is the response for listing teams.
type TeamListResponse struct {
	Items      []Team     `json:"items"`
	Pagination Pagination `json:"pagination"`
}
