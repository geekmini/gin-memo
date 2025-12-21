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

// DeleteVoiceMemo soft deletes a voice memo with atomic ownership check.
// Idempotent - returns nil if memo is already deleted.
func (s *VoiceMemoService) DeleteVoiceMemo(ctx context.Context, memoID, userID primitive.ObjectID) error {
	return s.repo.SoftDeleteWithOwnership(ctx, memoID, userID)
}

// ListByTeamID retrieves paginated voice memos for a team with pre-signed URLs.
func (s *VoiceMemoService) ListByTeamID(ctx context.Context, teamID string, page, limit int) (*models.VoiceMemoListResponse, error) {
	objectID, err := primitive.ObjectIDFromHex(teamID)
	if err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 10 {
		limit = 10
	}

	memos, total, err := s.repo.FindByTeamID(ctx, objectID, page, limit)
	if err != nil {
		return nil, err
	}

	// Generate pre-signed URLs for each memo
	for i := range memos {
		if memos[i].AudioFileKey != "" {
			url, err := s.s3Client.GetPresignedURL(ctx, memos[i].AudioFileKey, presignedURLExpiry)
			if err != nil {
				continue
			}
			memos[i].AudioFileURL = url
		}
	}

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

// GetVoiceMemo retrieves a voice memo by ID with pre-signed URL.
func (s *VoiceMemoService) GetVoiceMemo(ctx context.Context, memoID primitive.ObjectID) (*models.VoiceMemo, error) {
	memo, err := s.repo.FindByID(ctx, memoID)
	if err != nil {
		return nil, err
	}

	// Generate pre-signed URL
	if memo.AudioFileKey != "" {
		url, err := s.s3Client.GetPresignedURL(ctx, memo.AudioFileKey, presignedURLExpiry)
		if err == nil {
			memo.AudioFileURL = url
		}
	}

	return memo, nil
}

// DeleteTeamVoiceMemo soft deletes a team voice memo with atomic team check.
// Idempotent - returns nil if memo is already deleted.
func (s *VoiceMemoService) DeleteTeamVoiceMemo(ctx context.Context, memoID, teamID primitive.ObjectID) error {
	return s.repo.SoftDeleteWithTeam(ctx, memoID, teamID)
}
