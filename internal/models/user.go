package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Email     string             `json:"email" bson:"email"`
	Password  string             `json:"-" bson:"password"` // "-" = never include in JSON response
	Name      string             `json:"name" bson:"name"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// CreateUserRequest is the payload for creating a user
type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required,min=2"`
}

// UpdateUserRequest is the payload for updating a user
type UpdateUserRequest struct {
	Email *string `json:"email" binding:"omitempty,email"`
	Name  *string `json:"name" binding:"omitempty,min=2"`
}

// LoginRequest is the payload for user login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse is the response after successful login
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
