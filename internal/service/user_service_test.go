package service

import (
	"context"
	"testing"
	"time"

	cachemocks "gin-sample/internal/cache/mocks"
	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	repomocks "gin-sample/internal/repository/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

func TestNewUserService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := repomocks.NewMockUserRepository(ctrl)
	mockCache := cachemocks.NewMockCache(ctrl)

	service := NewUserService(mockRepo, mockCache)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.repo)
	assert.Equal(t, mockCache, service.cache)
}

func TestUserService_GetUser(t *testing.T) {
	validUserID := primitive.NewObjectID()
	validUser := &models.User{
		ID:    validUserID,
		Email: "test@example.com",
		Name:  "Test User",
	}

	t.Run("returns user from cache when cached", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockCache.EXPECT().
			Get(gomock.Any(), "user:"+validUserID.Hex(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, key string, dest interface{}) (bool, error) {
				// Simulate cache hit by populating dest
				user := dest.(*models.User)
				*user = *validUser
				return true, nil
			})

		service := NewUserService(mockRepo, mockCache)
		user, err := service.GetUser(context.Background(), validUserID.Hex())

		require.NoError(t, err)
		assert.Equal(t, validUser.ID, user.ID)
		assert.Equal(t, validUser.Email, user.Email)
	})

	t.Run("fetches from database on cache miss and caches result", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockCache.EXPECT().
			Get(gomock.Any(), "user:"+validUserID.Hex(), gomock.Any()).
			Return(false, nil) // Cache miss

		mockRepo.EXPECT().
			FindByID(gomock.Any(), validUserID).
			Return(validUser, nil)

		mockCache.EXPECT().
			Set(gomock.Any(), "user:"+validUserID.Hex(), validUser, 15*time.Minute).
			Return(nil)

		service := NewUserService(mockRepo, mockCache)
		user, err := service.GetUser(context.Background(), validUserID.Hex())

		require.NoError(t, err)
		assert.Equal(t, validUser.ID, user.ID)
	})

	t.Run("returns error for invalid user ID format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		service := NewUserService(mockRepo, mockCache)
		user, err := service.GetUser(context.Background(), "invalid-id")

		assert.Nil(t, user)
		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})

	t.Run("returns error when user not found in database", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockCache.EXPECT().
			Get(gomock.Any(), "user:"+validUserID.Hex(), gomock.Any()).
			Return(false, nil)

		mockRepo.EXPECT().
			FindByID(gomock.Any(), validUserID).
			Return(nil, apperrors.ErrUserNotFound)

		service := NewUserService(mockRepo, mockCache)
		user, err := service.GetUser(context.Background(), validUserID.Hex())

		assert.Nil(t, user)
		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})

	t.Run("continues on cache set error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockCache.EXPECT().
			Get(gomock.Any(), "user:"+validUserID.Hex(), gomock.Any()).
			Return(false, nil)

		mockRepo.EXPECT().
			FindByID(gomock.Any(), validUserID).
			Return(validUser, nil)

		mockCache.EXPECT().
			Set(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(assert.AnError) // Cache set fails

		service := NewUserService(mockRepo, mockCache)
		user, err := service.GetUser(context.Background(), validUserID.Hex())

		require.NoError(t, err) // Should not fail on cache error
		assert.Equal(t, validUser.ID, user.ID)
	})
}

func TestUserService_GetAllUsers(t *testing.T) {
	t.Run("returns all users", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		users := []models.User{
			{ID: primitive.NewObjectID(), Email: "user1@example.com"},
			{ID: primitive.NewObjectID(), Email: "user2@example.com"},
		}

		mockRepo.EXPECT().
			FindAll(gomock.Any()).
			Return(users, nil)

		service := NewUserService(mockRepo, mockCache)
		result, err := service.GetAllUsers(context.Background())

		require.NoError(t, err)
		assert.Len(t, result, 2)
	})

	t.Run("returns error on repository failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockRepo.EXPECT().
			FindAll(gomock.Any()).
			Return(nil, assert.AnError)

		service := NewUserService(mockRepo, mockCache)
		result, err := service.GetAllUsers(context.Background())

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	validUserID := primitive.NewObjectID()
	updateReq := &models.UpdateUserRequest{Name: strPtr("Updated Name")}
	updatedUser := &models.User{
		ID:    validUserID,
		Email: "test@example.com",
		Name:  "Updated Name",
	}

	t.Run("updates user and invalidates cache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockRepo.EXPECT().
			Update(gomock.Any(), validUserID, updateReq).
			Return(updatedUser, nil)

		mockCache.EXPECT().
			Delete(gomock.Any(), "user:"+validUserID.Hex()).
			Return(nil)

		service := NewUserService(mockRepo, mockCache)
		user, err := service.UpdateUser(context.Background(), validUserID.Hex(), updateReq)

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", user.Name)
	})

	t.Run("returns error for invalid user ID format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		service := NewUserService(mockRepo, mockCache)
		user, err := service.UpdateUser(context.Background(), "invalid-id", updateReq)

		assert.Nil(t, user)
		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})

	t.Run("returns error when user not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockRepo.EXPECT().
			Update(gomock.Any(), validUserID, updateReq).
			Return(nil, apperrors.ErrUserNotFound)

		service := NewUserService(mockRepo, mockCache)
		user, err := service.UpdateUser(context.Background(), validUserID.Hex(), updateReq)

		assert.Nil(t, user)
		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	validUserID := primitive.NewObjectID()

	t.Run("deletes user and invalidates cache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockRepo.EXPECT().
			Delete(gomock.Any(), validUserID).
			Return(nil)

		mockCache.EXPECT().
			Delete(gomock.Any(), "user:"+validUserID.Hex()).
			Return(nil)

		service := NewUserService(mockRepo, mockCache)
		err := service.DeleteUser(context.Background(), validUserID.Hex())

		assert.NoError(t, err)
	})

	t.Run("returns error for invalid user ID format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		service := NewUserService(mockRepo, mockCache)
		err := service.DeleteUser(context.Background(), "invalid-id")

		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})

	t.Run("returns error when delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockRepo := repomocks.NewMockUserRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)

		mockRepo.EXPECT().
			Delete(gomock.Any(), validUserID).
			Return(apperrors.ErrUserNotFound)

		service := NewUserService(mockRepo, mockCache)
		err := service.DeleteUser(context.Background(), validUserID.Hex())

		assert.Equal(t, apperrors.ErrUserNotFound, err)
	})
}

// Helper function
func strPtr(s string) *string {
	return &s
}
