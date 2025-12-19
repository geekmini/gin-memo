package service

import (
	"context"
	"time"

	"gin-sample/internal/models"
	"gin-sample/internal/repository"
	"gin-sample/internal/storage"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const presignedURLExpiry = 1 * time.Hour

// VoiceMemoService handles business logic for voice memo operations.
type VoiceMemoService struct {
	repo     repository.VoiceMemoRepository
	s3Client *storage.S3Client
}

// NewVoiceMemoService creates a new VoiceMemoService.
func NewVoiceMemoService(repo repository.VoiceMemoRepository, s3Client *storage.S3Client) *VoiceMemoService {
	return &VoiceMemoService{
		repo:     repo,
		s3Client: s3Client,
	}
}

// ListByUserID retrieves paginated voice memos for a user with pre-signed URLs.
func (s *VoiceMemoService) ListByUserID(ctx context.Context, userID string, page, limit int) (*models.VoiceMemoListResponse, error) {
	// Parse user ID
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, err
	}

	// Set defaults
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 10 {
		limit = 10
	}

	// Get memos from repository
	memos, total, err := s.repo.FindByUserID(ctx, objectID, page, limit)
	if err != nil {
		return nil, err
	}

	// Generate pre-signed URLs for each memo
	for i := range memos {
		if memos[i].AudioFileKey != "" {
			url, err := s.s3Client.GetPresignedURL(ctx, memos[i].AudioFileKey, presignedURLExpiry)
			if err != nil {
				// Log error but continue - URL will be empty
				continue
			}
			memos[i].AudioFileURL = url
		}
	}

	// Calculate total pages
	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return &models.VoiceMemoListResponse{
		Items: memos,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}
