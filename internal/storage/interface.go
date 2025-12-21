package storage

import (
	"context"
	"io"
	"time"
)

//go:generate mockgen -destination=mocks/mock_storage.go -package=mocks gin-sample/internal/storage Storage

// Storage defines the interface for object storage operations.
type Storage interface {
	// GetPresignedURL generates a pre-signed URL for downloading an object.
	GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	// GetPresignedPutURL generates a pre-signed URL for uploading an object.
	GetPresignedPutURL(ctx context.Context, key, contentType string, expiry time.Duration) (string, error)
	// PutObject uploads an object to storage.
	PutObject(ctx context.Context, key string, body io.Reader, contentType string) error
}

// Ensure S3Client implements Storage interface
var _ Storage = (*S3Client)(nil)
