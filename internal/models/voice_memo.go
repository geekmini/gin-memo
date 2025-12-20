package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VoiceMemo represents a voice memo in the system.
type VoiceMemo struct {
	ID            primitive.ObjectID  `json:"id" bson:"_id,omitempty" example:"507f1f77bcf86cd799439011"`
	UserID        primitive.ObjectID  `json:"userId" bson:"userId" example:"507f1f77bcf86cd799439012"`
	TeamID        *primitive.ObjectID `json:"teamId,omitempty" bson:"teamId,omitempty" example:"507f1f77bcf86cd799439013"` // nil = private memo, set = team memo
	Title         string              `json:"title" bson:"title" example:"Meeting notes"`
	Transcription string              `json:"transcription" bson:"transcription" example:"Today we discussed the Q4 roadmap..."`
	AudioFileKey  string              `json:"-" bson:"audioFileKey"`                                                                             // S3 key, not exposed in JSON
	AudioFileURL  string              `json:"audioFileUrl" bson:"-" example:"https://bucket.s3.amazonaws.com/audio/123.mp3?X-Amz-Signature=..."` // Pre-signed URL, not stored in DB
	Duration      int                 `json:"duration" bson:"duration" example:"180"`
	FileSize      int64               `json:"fileSize" bson:"fileSize" example:"2890000"`
	AudioFormat   string              `json:"audioFormat" bson:"audioFormat" example:"mp3"`
	Tags          []string            `json:"tags" bson:"tags" example:"work,meeting"`
	IsFavorite    bool                `json:"isFavorite" bson:"isFavorite" example:"false"`
	CreatedAt     time.Time           `json:"createdAt" bson:"createdAt" example:"2024-01-15T09:30:00Z"`
	DeletedAt     *time.Time          `json:"deletedAt,omitempty" bson:"deletedAt,omitempty"`
}

// VoiceMemoListResponse is the response for listing voice memos.
type VoiceMemoListResponse struct {
	Items      []VoiceMemo `json:"items"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination contains pagination metadata.
type Pagination struct {
	Page       int `json:"page" example:"1"`
	Limit      int `json:"limit" example:"10"`
	TotalItems int `json:"totalItems" example:"42"`
	TotalPages int `json:"totalPages" example:"5"`
}
