package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/queue"
	"gin-sample/internal/repository"
	"gin-sample/internal/storage"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VoiceMemoService handles business logic for voice memo operations.
type VoiceMemoService struct {
	repo                  repository.VoiceMemoRepository
	s3Client              storage.Storage
	queue                 queue.Queue
	presignedURLExpiry    time.Duration
	presignedUploadExpiry time.Duration
}

// NewVoiceMemoService creates a new VoiceMemoService.
func NewVoiceMemoService(repo repository.VoiceMemoRepository, s3Client storage.Storage, queue queue.Queue, presignedURLExpiry, presignedUploadExpiry time.Duration) *VoiceMemoService {
	return &VoiceMemoService{
		repo:                  repo,
		s3Client:              s3Client,
		queue:                 queue,
		presignedURLExpiry:    presignedURLExpiry,
		presignedUploadExpiry: presignedUploadExpiry,
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
			url, err := s.s3Client.GetPresignedURL(ctx, memos[i].AudioFileKey, s.presignedURLExpiry)
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
			url, err := s.s3Client.GetPresignedURL(ctx, memos[i].AudioFileKey, s.presignedURLExpiry)
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
		url, err := s.s3Client.GetPresignedURL(ctx, memo.AudioFileKey, s.presignedURLExpiry)
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

// CreateVoiceMemo creates a new private voice memo and returns upload URL.
func (s *VoiceMemoService) CreateVoiceMemo(ctx context.Context, userID primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error) {
	// Generate S3 key for private memo: voice-memos/{userId}/{memoId}.{format}
	memoID := primitive.NewObjectID()
	audioKey := fmt.Sprintf("voice-memos/%s/%s.%s", userID.Hex(), memoID.Hex(), req.AudioFormat)

	// Create memo with pending_upload status
	memo := &models.VoiceMemo{
		ID:           memoID,
		UserID:       userID,
		Title:        req.Title,
		Duration:     req.Duration,
		FileSize:     req.FileSize,
		AudioFormat:  req.AudioFormat,
		Tags:         req.Tags,
		IsFavorite:   req.IsFavorite,
		AudioFileKey: audioKey,
		Status:       models.StatusPendingUpload,
	}

	// Ensure tags is not nil
	if memo.Tags == nil {
		memo.Tags = []string{}
	}

	// Save to database
	if err := s.repo.Create(ctx, memo); err != nil {
		return nil, err
	}

	// Generate pre-signed upload URL
	contentType := getContentType(req.AudioFormat)
	uploadURL, err := s.s3Client.GetPresignedPutURL(ctx, audioKey, contentType, s.presignedUploadExpiry)
	if err != nil {
		return nil, err
	}

	return &models.CreateVoiceMemoResponse{
		Memo:      *memo,
		UploadURL: uploadURL,
	}, nil
}

// CreateTeamVoiceMemo creates a new team voice memo and returns upload URL.
func (s *VoiceMemoService) CreateTeamVoiceMemo(ctx context.Context, userID, teamID primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error) {
	// Generate S3 key for team memo: voice-memos/{teamId}/{userId}/{memoId}.{format}
	memoID := primitive.NewObjectID()
	audioKey := fmt.Sprintf("voice-memos/%s/%s/%s.%s", teamID.Hex(), userID.Hex(), memoID.Hex(), req.AudioFormat)

	// Create memo with pending_upload status
	memo := &models.VoiceMemo{
		ID:           memoID,
		UserID:       userID,
		TeamID:       &teamID,
		Title:        req.Title,
		Duration:     req.Duration,
		FileSize:     req.FileSize,
		AudioFormat:  req.AudioFormat,
		Tags:         req.Tags,
		IsFavorite:   req.IsFavorite,
		AudioFileKey: audioKey,
		Status:       models.StatusPendingUpload,
	}

	// Ensure tags is not nil
	if memo.Tags == nil {
		memo.Tags = []string{}
	}

	// Save to database
	if err := s.repo.Create(ctx, memo); err != nil {
		return nil, err
	}

	// Generate pre-signed upload URL
	contentType := getContentType(req.AudioFormat)
	uploadURL, err := s.s3Client.GetPresignedPutURL(ctx, audioKey, contentType, s.presignedUploadExpiry)
	if err != nil {
		return nil, err
	}

	return &models.CreateVoiceMemoResponse{
		Memo:      *memo,
		UploadURL: uploadURL,
	}, nil
}

// ConfirmUpload confirms audio upload and triggers transcription for a private memo.
func (s *VoiceMemoService) ConfirmUpload(ctx context.Context, memoID, userID primitive.ObjectID) error {
	// Atomically update status from pending_upload to transcribing with ownership check
	// Returns the updated memo to avoid a separate FindByID call
	memo, err := s.repo.UpdateStatusWithOwnership(ctx, memoID, userID, models.StatusPendingUpload, models.StatusTranscribing)
	if err != nil {
		return err
	}

	// Enqueue transcription job
	job := queue.TranscriptionJob{
		MemoID:       memoID,
		AudioFileKey: memo.AudioFileKey,
		RetryCount:   0,
	}

	if err := s.queue.Enqueue(job); err != nil {
		if errors.Is(err, queue.ErrQueueFull) {
			// Revert status back to pending_upload if queue is full (only if still transcribing)
			if revertErr := s.repo.UpdateStatusConditional(ctx, memoID, models.StatusTranscribing, models.StatusPendingUpload); revertErr != nil {
				log.Printf("Failed to revert status for memo %s: %v", memoID.Hex(), revertErr)
			}
			return apperrors.ErrTranscriptionQueueFull
		}
		return err
	}

	return nil
}

// ConfirmTeamUpload confirms audio upload and triggers transcription for a team memo.
func (s *VoiceMemoService) ConfirmTeamUpload(ctx context.Context, memoID, teamID primitive.ObjectID) error {
	// Atomically update status from pending_upload to transcribing with team check
	// Returns the updated memo to avoid a separate FindByID call
	memo, err := s.repo.UpdateStatusWithTeam(ctx, memoID, teamID, models.StatusPendingUpload, models.StatusTranscribing)
	if err != nil {
		return err
	}

	// Enqueue transcription job
	job := queue.TranscriptionJob{
		MemoID:       memoID,
		AudioFileKey: memo.AudioFileKey,
		RetryCount:   0,
	}

	if err := s.queue.Enqueue(job); err != nil {
		if errors.Is(err, queue.ErrQueueFull) {
			// Revert status back to pending_upload if queue is full (only if still transcribing)
			if revertErr := s.repo.UpdateStatusConditional(ctx, memoID, models.StatusTranscribing, models.StatusPendingUpload); revertErr != nil {
				log.Printf("Failed to revert status for memo %s: %v", memoID.Hex(), revertErr)
			}
			return apperrors.ErrTranscriptionQueueFull
		}
		return err
	}

	return nil
}

// RetryTranscription retries transcription for a failed private memo.
func (s *VoiceMemoService) RetryTranscription(ctx context.Context, memoID, userID primitive.ObjectID) error {
	// Atomically update status from failed to transcribing with ownership check
	// Returns the updated memo to avoid a separate FindByID call
	memo, err := s.repo.UpdateStatusWithOwnership(ctx, memoID, userID, models.StatusFailed, models.StatusTranscribing)
	if err != nil {
		return err
	}

	// Enqueue transcription job
	job := queue.TranscriptionJob{
		MemoID:       memoID,
		AudioFileKey: memo.AudioFileKey,
		RetryCount:   0, // Reset retry count for manual retry
	}

	if err := s.queue.Enqueue(job); err != nil {
		if errors.Is(err, queue.ErrQueueFull) {
			// Revert status back to failed if queue is full (only if still transcribing)
			if revertErr := s.repo.UpdateStatusConditional(ctx, memoID, models.StatusTranscribing, models.StatusFailed); revertErr != nil {
				log.Printf("Failed to revert status for memo %s: %v", memoID.Hex(), revertErr)
			}
			return apperrors.ErrTranscriptionQueueFull
		}
		return err
	}

	return nil
}

// RetryTeamTranscription retries transcription for a failed team memo.
func (s *VoiceMemoService) RetryTeamTranscription(ctx context.Context, memoID, teamID primitive.ObjectID) error {
	// Atomically update status from failed to transcribing with team check
	// Returns the updated memo to avoid a separate FindByID call
	memo, err := s.repo.UpdateStatusWithTeam(ctx, memoID, teamID, models.StatusFailed, models.StatusTranscribing)
	if err != nil {
		return err
	}

	// Enqueue transcription job
	job := queue.TranscriptionJob{
		MemoID:       memoID,
		AudioFileKey: memo.AudioFileKey,
		RetryCount:   0, // Reset retry count for manual retry
	}

	if err := s.queue.Enqueue(job); err != nil {
		if errors.Is(err, queue.ErrQueueFull) {
			// Revert status back to failed if queue is full (only if still transcribing)
			if revertErr := s.repo.UpdateStatusConditional(ctx, memoID, models.StatusTranscribing, models.StatusFailed); revertErr != nil {
				log.Printf("Failed to revert status for memo %s: %v", memoID.Hex(), revertErr)
			}
			return apperrors.ErrTranscriptionQueueFull
		}
		return err
	}

	return nil
}

// getContentType returns the MIME type for an audio format.
func getContentType(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "wav":
		return "audio/wav"
	case "m4a":
		return "audio/mp4"
	case "webm":
		return "audio/webm"
	case "aac":
		return "audio/aac"
	default:
		return "application/octet-stream"
	}
}
