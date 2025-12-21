package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VoiceMemoStatus represents the processing status of a voice memo.
type VoiceMemoStatus string

const (
	// StatusPendingUpload indicates memo created, waiting for audio upload.
	StatusPendingUpload VoiceMemoStatus = "pending_upload"
	// StatusTranscribing indicates audio uploaded, transcription in progress.
	StatusTranscribing VoiceMemoStatus = "transcribing"
	// StatusReady indicates transcription complete, memo fully available.
	StatusReady VoiceMemoStatus = "ready"
	// StatusFailed indicates transcription failed (can retry).
	StatusFailed VoiceMemoStatus = "failed"
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
	Status        VoiceMemoStatus     `json:"status" bson:"status" example:"ready"`
	Version       int                 `json:"version" bson:"version" example:"1"` // Existing docs default to 0, increments to 1+ on first modification
	CreatedAt     time.Time           `json:"createdAt" bson:"createdAt" example:"2024-01-15T09:30:00Z"`
	UpdatedAt     time.Time           `json:"updatedAt" bson:"updatedAt" example:"2024-01-15T10:00:00Z"`
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

// CreateVoiceMemoRequest is the request body for creating a voice memo.
type CreateVoiceMemoRequest struct {
	Title       string   `json:"title" binding:"required,min=1,max=200" example:"Meeting Notes"`
	Duration    int      `json:"duration" binding:"gte=0" example:"120"`
	FileSize    int64    `json:"fileSize" binding:"required,gt=0,max=104857600" example:"1048576"` // max 100MB
	AudioFormat string   `json:"audioFormat" binding:"required,oneof=mp3 wav m4a webm aac" example:"mp3"`
	Tags        []string `json:"tags" binding:"max=10,dive,max=50" example:"work,meeting"`
	IsFavorite  bool     `json:"isFavorite" example:"false"`
}

// CreateVoiceMemoResponse is the response for creating a voice memo.
type CreateVoiceMemoResponse struct {
	Memo      VoiceMemo `json:"memo"`
	UploadURL string    `json:"uploadUrl" example:"https://s3.amazonaws.com/bucket/voice-memos/...?X-Amz-Algorithm=..."`
}
