// Package models defines data structures for the application.
package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RefreshToken represents a refresh token stored in the database.
type RefreshToken struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Token     string             `json:"token" bson:"token"`
	UserID    primitive.ObjectID `json:"userId" bson:"userId"`
	ExpiresAt time.Time          `json:"expiresAt" bson:"expiresAt"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
}

// RefreshRequest is the payload for refreshing an access token.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"rf_8a7b3c9d..."`
}

// LogoutRequest is the payload for logging out.
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required" example:"rf_8a7b3c9d..."`
}

// AuthResponse is the response after successful login or registration.
type AuthResponse struct {
	AccessToken  string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIs..."`
	RefreshToken string `json:"refreshToken" example:"rf_8a7b3c9d..."`
	ExpiresIn    int    `json:"expiresIn" example:"900"`
	User         User   `json:"user"`
}

// RefreshResponse is the response after successful token refresh.
type RefreshResponse struct {
	AccessToken string `json:"accessToken" example:"eyJhbGciOiJIUzI1NiIs..."`
	ExpiresIn   int    `json:"expiresIn" example:"900"`
}
