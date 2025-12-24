package service

import (
	"context"
	"testing"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/queue"
	queuemocks "gin-sample/internal/queue/mocks"
	repomocks "gin-sample/internal/repository/mocks"
	storagemocks "gin-sample/internal/storage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

func TestNewVoiceMemoService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
	mockStorage := storagemocks.NewMockStorage(ctrl)
	mockQueue := queuemocks.NewMockQueue(ctrl)

	service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockStorage, service.s3Client)
	assert.Equal(t, mockQueue, service.queue)
}

func TestVoiceMemoService_ListByUserID(t *testing.T) {
	validUserID := primitive.NewObjectID()
	memos := []models.VoiceMemo{
		{
			ID:           primitive.NewObjectID(),
			UserID:       validUserID,
			Title:        "Memo 1",
			AudioFileKey: "voice-memos/user1/memo1.mp3",
		},
		{
			ID:           primitive.NewObjectID(),
			UserID:       validUserID,
			Title:        "Memo 2",
			AudioFileKey: "voice-memos/user1/memo2.mp3",
		},
	}

	t.Run("returns paginated memos with presigned URLs", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			FindByUserID(gomock.Any(), validUserID, 1, 10).
			Return(memos, 2, nil)

		mockStorage.EXPECT().
			GetPresignedURL(gomock.Any(), memos[0].AudioFileKey, gomock.Any()).
			Return("https://s3.example.com/memo1.mp3", nil)

		mockStorage.EXPECT().
			GetPresignedURL(gomock.Any(), memos[1].AudioFileKey, gomock.Any()).
			Return("https://s3.example.com/memo2.mp3", nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByUserID(context.Background(), validUserID.Hex(), 1, 10)

		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)
		assert.Equal(t, 1, resp.Pagination.Page)
		assert.Equal(t, 10, resp.Pagination.Limit)
		assert.Equal(t, 2, resp.Pagination.TotalItems)
		assert.Equal(t, 1, resp.Pagination.TotalPages)
		assert.Equal(t, "https://s3.example.com/memo1.mp3", resp.Items[0].AudioFileURL)
	})

	t.Run("applies default pagination values", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		// Should use defaults: page=1, limit=10
		mockRepo.EXPECT().
			FindByUserID(gomock.Any(), validUserID, 1, 10).
			Return([]models.VoiceMemo{}, 0, nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByUserID(context.Background(), validUserID.Hex(), 0, 0)

		require.NoError(t, err)
		assert.Equal(t, 1, resp.Pagination.Page)
		assert.Equal(t, 10, resp.Pagination.Limit)
	})

	t.Run("caps limit at 10", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		// Limit > 10 should be capped to 10
		mockRepo.EXPECT().
			FindByUserID(gomock.Any(), validUserID, 1, 10).
			Return([]models.VoiceMemo{}, 0, nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByUserID(context.Background(), validUserID.Hex(), 1, 100)

		require.NoError(t, err)
		assert.Equal(t, 10, resp.Pagination.Limit)
	})

	t.Run("returns error for invalid user ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByUserID(context.Background(), "invalid-id", 1, 10)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			FindByUserID(gomock.Any(), validUserID, 1, 10).
			Return(nil, 0, assert.AnError)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByUserID(context.Background(), validUserID.Hex(), 1, 10)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})

	t.Run("continues on presigned URL error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		// Create fresh memos for this test to avoid mutation
		freshMemos := []models.VoiceMemo{
			{
				ID:           primitive.NewObjectID(),
				UserID:       validUserID,
				Title:        "Memo 1",
				AudioFileKey: "voice-memos/user1/memo1.mp3",
			},
			{
				ID:           primitive.NewObjectID(),
				UserID:       validUserID,
				Title:        "Memo 2",
				AudioFileKey: "voice-memos/user1/memo2.mp3",
			},
		}

		mockRepo.EXPECT().
			FindByUserID(gomock.Any(), validUserID, 1, 10).
			Return(freshMemos, 2, nil)

		mockStorage.EXPECT().
			GetPresignedURL(gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", assert.AnError).
			Times(2)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByUserID(context.Background(), validUserID.Hex(), 1, 10)

		require.NoError(t, err)
		assert.Len(t, resp.Items, 2)
		// URLs should be empty due to error
		assert.Empty(t, resp.Items[0].AudioFileURL)
	})

	t.Run("calculates total pages correctly", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		// 15 items with limit 10 = 2 pages
		mockRepo.EXPECT().
			FindByUserID(gomock.Any(), validUserID, 1, 10).
			Return([]models.VoiceMemo{}, 15, nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByUserID(context.Background(), validUserID.Hex(), 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 2, resp.Pagination.TotalPages)
	})
}

func TestVoiceMemoService_DeleteVoiceMemo(t *testing.T) {
	memoID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	t.Run("successfully deletes memo", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			SoftDeleteWithOwnership(gomock.Any(), memoID, userID).
			Return(nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.DeleteVoiceMemo(context.Background(), memoID, userID)

		assert.NoError(t, err)
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			SoftDeleteWithOwnership(gomock.Any(), memoID, userID).
			Return(apperrors.ErrVoiceMemoNotFound)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.DeleteVoiceMemo(context.Background(), memoID, userID)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}

func TestVoiceMemoService_ListByTeamID(t *testing.T) {
	validTeamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	memos := []models.VoiceMemo{
		{
			ID:           primitive.NewObjectID(),
			UserID:       userID,
			TeamID:       &validTeamID,
			Title:        "Team Memo 1",
			AudioFileKey: "voice-memos/team1/memo1.mp3",
		},
	}

	t.Run("returns paginated team memos with presigned URLs", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			FindByTeamID(gomock.Any(), validTeamID, 1, 10).
			Return(memos, 1, nil)

		mockStorage.EXPECT().
			GetPresignedURL(gomock.Any(), memos[0].AudioFileKey, gomock.Any()).
			Return("https://s3.example.com/team-memo1.mp3", nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByTeamID(context.Background(), validTeamID.Hex(), 1, 10)

		require.NoError(t, err)
		assert.Len(t, resp.Items, 1)
		assert.Equal(t, "https://s3.example.com/team-memo1.mp3", resp.Items[0].AudioFileURL)
	})

	t.Run("returns error for invalid team ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByTeamID(context.Background(), "invalid-id", 1, 10)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})

	t.Run("applies default pagination and caps limit", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			FindByTeamID(gomock.Any(), validTeamID, 1, 10).
			Return([]models.VoiceMemo{}, 0, nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.ListByTeamID(context.Background(), validTeamID.Hex(), -1, 50)

		require.NoError(t, err)
		assert.Equal(t, 1, resp.Pagination.Page)
		assert.Equal(t, 10, resp.Pagination.Limit)
	})
}

func TestVoiceMemoService_GetVoiceMemo(t *testing.T) {
	memoID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	memo := &models.VoiceMemo{
		ID:           memoID,
		UserID:       userID,
		Title:        "Test Memo",
		AudioFileKey: "voice-memos/user1/memo1.mp3",
	}

	t.Run("returns memo with presigned URL", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			FindByID(gomock.Any(), memoID).
			Return(memo, nil)

		mockStorage.EXPECT().
			GetPresignedURL(gomock.Any(), memo.AudioFileKey, gomock.Any()).
			Return("https://s3.example.com/memo1.mp3", nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		result, err := service.GetVoiceMemo(context.Background(), memoID)

		require.NoError(t, err)
		assert.Equal(t, memoID, result.ID)
		assert.Equal(t, "https://s3.example.com/memo1.mp3", result.AudioFileURL)
	})

	t.Run("returns memo without URL on presigned URL error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		// Create fresh memo to avoid mutation from previous test
		freshMemo := &models.VoiceMemo{
			ID:           memoID,
			UserID:       userID,
			Title:        "Test Memo",
			AudioFileKey: "voice-memos/user1/memo1.mp3",
		}

		mockRepo.EXPECT().
			FindByID(gomock.Any(), memoID).
			Return(freshMemo, nil)

		mockStorage.EXPECT().
			GetPresignedURL(gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", assert.AnError)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		result, err := service.GetVoiceMemo(context.Background(), memoID)

		require.NoError(t, err)
		assert.Equal(t, memoID, result.ID)
		assert.Empty(t, result.AudioFileURL)
	})

	t.Run("returns error when memo not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			FindByID(gomock.Any(), memoID).
			Return(nil, apperrors.ErrVoiceMemoNotFound)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		result, err := service.GetVoiceMemo(context.Background(), memoID)

		assert.Nil(t, result)
		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("skips presigned URL for memo without audio key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		memoWithoutKey := &models.VoiceMemo{
			ID:           memoID,
			UserID:       userID,
			Title:        "Test Memo",
			AudioFileKey: "", // No audio key
		}

		mockRepo.EXPECT().
			FindByID(gomock.Any(), memoID).
			Return(memoWithoutKey, nil)

		// GetPresignedURL should NOT be called
		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		result, err := service.GetVoiceMemo(context.Background(), memoID)

		require.NoError(t, err)
		assert.Equal(t, memoID, result.ID)
		assert.Empty(t, result.AudioFileURL)
	})
}

func TestVoiceMemoService_DeleteTeamVoiceMemo(t *testing.T) {
	memoID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()

	t.Run("successfully deletes team memo", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			SoftDeleteWithTeam(gomock.Any(), memoID, teamID).
			Return(nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.DeleteTeamVoiceMemo(context.Background(), memoID, teamID)

		assert.NoError(t, err)
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			SoftDeleteWithTeam(gomock.Any(), memoID, teamID).
			Return(apperrors.ErrVoiceMemoNotFound)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.DeleteTeamVoiceMemo(context.Background(), memoID, teamID)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}

func TestVoiceMemoService_CreateVoiceMemo(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &models.CreateVoiceMemoRequest{
		Title:       "Test Memo",
		Duration:    60,
		FileSize:    1024,
		AudioFormat: "mp3",
		Tags:        []string{"test"},
		IsFavorite:  true,
	}

	t.Run("successfully creates memo with upload URL", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, memo *models.VoiceMemo) error {
				assert.Equal(t, userID, memo.UserID)
				assert.Equal(t, req.Title, memo.Title)
				assert.Equal(t, models.StatusPendingUpload, memo.Status)
				assert.Contains(t, memo.AudioFileKey, userID.Hex())
				assert.Contains(t, memo.AudioFileKey, ".mp3")
				return nil
			})

		mockStorage.EXPECT().
			GetPresignedPutURL(gomock.Any(), gomock.Any(), "audio/mpeg", gomock.Any()).
			Return("https://s3.example.com/upload-url", nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.CreateVoiceMemo(context.Background(), userID, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "https://s3.example.com/upload-url", resp.UploadURL)
		assert.Equal(t, models.StatusPendingUpload, resp.Memo.Status)
	})

	t.Run("initializes nil tags to empty slice", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		reqWithNilTags := &models.CreateVoiceMemoRequest{
			Title:       "Test Memo",
			Duration:    60,
			FileSize:    1024,
			AudioFormat: "wav",
			Tags:        nil,
		}

		mockRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, memo *models.VoiceMemo) error {
				assert.NotNil(t, memo.Tags)
				assert.Empty(t, memo.Tags)
				return nil
			})

		mockStorage.EXPECT().
			GetPresignedPutURL(gomock.Any(), gomock.Any(), "audio/wav", gomock.Any()).
			Return("https://s3.example.com/upload-url", nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.CreateVoiceMemo(context.Background(), userID, reqWithNilTags)

		require.NoError(t, err)
		assert.NotNil(t, resp.Memo.Tags)
	})

	t.Run("returns error when repository create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(assert.AnError)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.CreateVoiceMemo(context.Background(), userID, req)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})

	t.Run("returns error when presigned URL generation fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil)

		mockStorage.EXPECT().
			GetPresignedPutURL(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", assert.AnError)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.CreateVoiceMemo(context.Background(), userID, req)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})

	t.Run("generates correct content type for different formats", func(t *testing.T) {
		testCases := []struct {
			format      string
			contentType string
		}{
			{"mp3", "audio/mpeg"},
			{"wav", "audio/wav"},
			{"m4a", "audio/mp4"},
			{"webm", "audio/webm"},
			{"aac", "audio/aac"},
			{"unknown", "application/octet-stream"},
		}

		for _, tc := range testCases {
			t.Run(tc.format, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				defer ctrl.Finish()

				mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
				mockStorage := storagemocks.NewMockStorage(ctrl)
				mockQueue := queuemocks.NewMockQueue(ctrl)

				formatReq := &models.CreateVoiceMemoRequest{
					Title:       "Test",
					AudioFormat: tc.format,
				}

				mockRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)

				mockStorage.EXPECT().
					GetPresignedPutURL(gomock.Any(), gomock.Any(), tc.contentType, gomock.Any()).
					Return("https://s3.example.com/upload", nil)

				service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
				_, err := service.CreateVoiceMemo(context.Background(), userID, formatReq)

				assert.NoError(t, err)
			})
		}
	})
}

func TestVoiceMemoService_CreateTeamVoiceMemo(t *testing.T) {
	userID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	req := &models.CreateVoiceMemoRequest{
		Title:       "Team Memo",
		Duration:    120,
		FileSize:    2048,
		AudioFormat: "m4a",
		Tags:        []string{"team", "meeting"},
	}

	t.Run("successfully creates team memo with upload URL", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, memo *models.VoiceMemo) error {
				assert.Equal(t, userID, memo.UserID)
				assert.NotNil(t, memo.TeamID)
				assert.Equal(t, teamID, *memo.TeamID)
				assert.Contains(t, memo.AudioFileKey, teamID.Hex())
				assert.Contains(t, memo.AudioFileKey, userID.Hex())
				assert.Contains(t, memo.AudioFileKey, ".m4a")
				return nil
			})

		mockStorage.EXPECT().
			GetPresignedPutURL(gomock.Any(), gomock.Any(), "audio/mp4", gomock.Any()).
			Return("https://s3.example.com/team-upload-url", nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.CreateTeamVoiceMemo(context.Background(), userID, teamID, req)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "https://s3.example.com/team-upload-url", resp.UploadURL)
		assert.Equal(t, teamID, *resp.Memo.TeamID)
	})

	t.Run("returns error when repository create fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(assert.AnError)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		resp, err := service.CreateTeamVoiceMemo(context.Background(), userID, teamID, req)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})
}

func TestVoiceMemoService_ConfirmUpload(t *testing.T) {
	memoID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	memo := &models.VoiceMemo{
		ID:           memoID,
		UserID:       userID,
		AudioFileKey: "voice-memos/user1/memo1.mp3",
		Status:       models.StatusTranscribing,
	}

	t.Run("successfully confirms upload and enqueues transcription", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithOwnership(gomock.Any(), memoID, userID, models.StatusPendingUpload, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			DoAndReturn(func(job queue.TranscriptionJob) error {
				assert.Equal(t, memoID, job.MemoID)
				assert.Equal(t, memo.AudioFileKey, job.AudioFileKey)
				assert.Equal(t, 0, job.RetryCount)
				return nil
			})

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.ConfirmUpload(context.Background(), memoID, userID)

		assert.NoError(t, err)
	})

	t.Run("returns error when status update fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithOwnership(gomock.Any(), memoID, userID, models.StatusPendingUpload, models.StatusTranscribing).
			Return(nil, apperrors.ErrVoiceMemoNotFound)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.ConfirmUpload(context.Background(), memoID, userID)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("reverts status and returns error when queue is full", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithOwnership(gomock.Any(), memoID, userID, models.StatusPendingUpload, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			Return(queue.ErrQueueFull)

		mockRepo.EXPECT().
			UpdateStatusConditional(gomock.Any(), memoID, models.StatusTranscribing, models.StatusPendingUpload).
			Return(nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.ConfirmUpload(context.Background(), memoID, userID)

		assert.Equal(t, apperrors.ErrTranscriptionQueueFull, err)
	})

	t.Run("handles revert error gracefully when queue is full", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithOwnership(gomock.Any(), memoID, userID, models.StatusPendingUpload, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			Return(queue.ErrQueueFull)

		mockRepo.EXPECT().
			UpdateStatusConditional(gomock.Any(), memoID, models.StatusTranscribing, models.StatusPendingUpload).
			Return(assert.AnError) // Revert fails

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.ConfirmUpload(context.Background(), memoID, userID)

		// Should still return queue full error
		assert.Equal(t, apperrors.ErrTranscriptionQueueFull, err)
	})

	t.Run("returns error for non-queue errors", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithOwnership(gomock.Any(), memoID, userID, models.StatusPendingUpload, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			Return(assert.AnError) // Not ErrQueueFull

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.ConfirmUpload(context.Background(), memoID, userID)

		assert.Error(t, err)
		assert.NotEqual(t, apperrors.ErrTranscriptionQueueFull, err)
	})
}

func TestVoiceMemoService_ConfirmTeamUpload(t *testing.T) {
	memoID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	memo := &models.VoiceMemo{
		ID:           memoID,
		TeamID:       &teamID,
		AudioFileKey: "voice-memos/team1/memo1.mp3",
		Status:       models.StatusTranscribing,
	}

	t.Run("successfully confirms team upload and enqueues transcription", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithTeam(gomock.Any(), memoID, teamID, models.StatusPendingUpload, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			Return(nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.ConfirmTeamUpload(context.Background(), memoID, teamID)

		assert.NoError(t, err)
	})

	t.Run("reverts status when queue is full", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithTeam(gomock.Any(), memoID, teamID, models.StatusPendingUpload, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			Return(queue.ErrQueueFull)

		mockRepo.EXPECT().
			UpdateStatusConditional(gomock.Any(), memoID, models.StatusTranscribing, models.StatusPendingUpload).
			Return(nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.ConfirmTeamUpload(context.Background(), memoID, teamID)

		assert.Equal(t, apperrors.ErrTranscriptionQueueFull, err)
	})
}

func TestVoiceMemoService_RetryTranscription(t *testing.T) {
	memoID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	memo := &models.VoiceMemo{
		ID:           memoID,
		UserID:       userID,
		AudioFileKey: "voice-memos/user1/memo1.mp3",
		Status:       models.StatusTranscribing,
	}

	t.Run("successfully retries transcription", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithOwnership(gomock.Any(), memoID, userID, models.StatusFailed, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			DoAndReturn(func(job queue.TranscriptionJob) error {
				assert.Equal(t, 0, job.RetryCount) // Reset to 0 for manual retry
				return nil
			})

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.RetryTranscription(context.Background(), memoID, userID)

		assert.NoError(t, err)
	})

	t.Run("returns error when status update fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithOwnership(gomock.Any(), memoID, userID, models.StatusFailed, models.StatusTranscribing).
			Return(nil, apperrors.ErrVoiceMemoNotFound)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.RetryTranscription(context.Background(), memoID, userID)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("reverts status to failed when queue is full", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithOwnership(gomock.Any(), memoID, userID, models.StatusFailed, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			Return(queue.ErrQueueFull)

		mockRepo.EXPECT().
			UpdateStatusConditional(gomock.Any(), memoID, models.StatusTranscribing, models.StatusFailed).
			Return(nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.RetryTranscription(context.Background(), memoID, userID)

		assert.Equal(t, apperrors.ErrTranscriptionQueueFull, err)
	})
}

func TestVoiceMemoService_RetryTeamTranscription(t *testing.T) {
	memoID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	memo := &models.VoiceMemo{
		ID:           memoID,
		TeamID:       &teamID,
		AudioFileKey: "voice-memos/team1/memo1.mp3",
		Status:       models.StatusTranscribing,
	}

	t.Run("successfully retries team transcription", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithTeam(gomock.Any(), memoID, teamID, models.StatusFailed, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			Return(nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.RetryTeamTranscription(context.Background(), memoID, teamID)

		assert.NoError(t, err)
	})

	t.Run("reverts status to failed when queue is full", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithTeam(gomock.Any(), memoID, teamID, models.StatusFailed, models.StatusTranscribing).
			Return(memo, nil)

		mockQueue.EXPECT().
			Enqueue(gomock.Any()).
			Return(queue.ErrQueueFull)

		mockRepo.EXPECT().
			UpdateStatusConditional(gomock.Any(), memoID, models.StatusTranscribing, models.StatusFailed).
			Return(nil)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.RetryTeamTranscription(context.Background(), memoID, teamID)

		assert.Equal(t, apperrors.ErrTranscriptionQueueFull, err)
	})

	t.Run("returns error when status update fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockVoiceMemoRepository(ctrl)
		mockStorage := storagemocks.NewMockStorage(ctrl)
		mockQueue := queuemocks.NewMockQueue(ctrl)

		mockRepo.EXPECT().
			UpdateStatusWithTeam(gomock.Any(), memoID, teamID, models.StatusFailed, models.StatusTranscribing).
			Return(nil, apperrors.ErrVoiceMemoNotFound)

		service := NewVoiceMemoService(mockRepo, mockStorage, mockQueue, time.Hour, 15*time.Minute)
		err := service.RetryTeamTranscription(context.Background(), memoID, teamID)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}
