// Package models defines data structures for the application.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system.
type User struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	Email     string             `json:"email" bson:"email" example:"user@example.com"`
	Password  string             `json:"-" bson:"password"` // "-" = never include in JSON response
	Name      string             `json:"name" bson:"name" example:"John Doe"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt" example:"2024-01-15T09:30:00Z"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt" example:"2024-01-15T09:30:00Z"`
}

// CreateUserRequest is the payload for creating a user.
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=6" example:"secret123"`
	Name     string `json:"name" binding:"required,min=2" example:"John Doe"`
}

// UpdateUserRequest is the payload for updating a user.
type UpdateUserRequest struct {
	Email *string `json:"email" binding:"omitempty,email" example:"newemail@example.com"`
	Name  *string `json:"name" binding:"omitempty,min=2" example:"Jane Doe"`
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"secret123"`
}

// LoginResponse is the response after successful login.
type LoginResponse struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	User  User   `json:"user"`
}
