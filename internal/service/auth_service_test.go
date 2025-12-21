package service

import (
	"context"
	"testing"
	"time"

	cachemocks "gin-sample/internal/cache/mocks"
	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	repomocks "gin-sample/internal/repository/mocks"
	"gin-sample/pkg/auth"
	authmocks "gin-sample/pkg/auth/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

func TestNewAuthService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repomocks.NewMockUserRepository(ctrl)
	mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
	mockCache := cachemocks.NewMockCache(ctrl)
	mockJWT := authmocks.NewMockTokenManager(ctrl)

	service := NewAuthService(
		mockUserRepo,
		mockRefreshRepo,
		mockCache,
		mockJWT,
		15*time.Minute,
		7*24*time.Hour,
	)

	assert.NotNil(t, service)
}

func TestAuthService_Register(t *testing.T) {
	createUserReq := &models.CreateUserRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	t.Run("successfully registers new user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		// Expect user creation
		mockUserRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, user *models.User) error {
				user.ID = primitive.NewObjectID()
				assert.Equal(t, createUserReq.Email, user.Email)
				assert.Equal(t, createUserReq.Name, user.Name)
				assert.NotEqual(t, createUserReq.Password, user.Password) // Should be hashed
				return nil
			})

		// Expect token generation
		mockJWT.EXPECT().
			GenerateToken(gomock.Any()).
			Return("access-token", nil)

		// Expect refresh token storage
		mockRefreshRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil)

		// Expect cache set
		mockCache.EXPECT().
			SetRefreshToken(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Register(context.Background(), createUserReq)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "access-token", resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.True(t, resp.ExpiresIn > 0)
	})

	t.Run("returns error when user creation fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockUserRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(apperrors.ErrUserAlreadyExists)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Register(context.Background(), createUserReq)

		assert.Nil(t, resp)
		assert.Equal(t, apperrors.ErrUserAlreadyExists, err)
	})

	t.Run("returns error when JWT generation fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockUserRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, user *models.User) error {
				user.ID = primitive.NewObjectID()
				return nil
			})

		mockJWT.EXPECT().
			GenerateToken(gomock.Any()).
			Return("", assert.AnError)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Register(context.Background(), createUserReq)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})
}

func TestAuthService_Login(t *testing.T) {
	validUserID := primitive.NewObjectID()
	hashedPassword, _ := auth.HashPassword("password123")
	validUser := &models.User{
		ID:       validUserID,
		Email:    "test@example.com",
		Password: hashedPassword,
		Name:     "Test User",
	}

	loginReq := &models.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}

	t.Run("successfully logs in user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockUserRepo.EXPECT().
			FindByEmail(gomock.Any(), loginReq.Email).
			Return(validUser, nil)

		mockJWT.EXPECT().
			GenerateToken(validUserID.Hex()).
			Return("access-token", nil)

		mockRefreshRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil)

		mockCache.EXPECT().
			SetRefreshToken(gomock.Any(), gomock.Any(), validUserID.Hex(), gomock.Any()).
			Return(nil)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Login(context.Background(), loginReq)

		require.NoError(t, err)
		assert.Equal(t, "access-token", resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
	})

	t.Run("returns error for non-existent user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockUserRepo.EXPECT().
			FindByEmail(gomock.Any(), loginReq.Email).
			Return(nil, apperrors.ErrUserNotFound)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Login(context.Background(), loginReq)

		assert.Nil(t, resp)
		assert.Equal(t, apperrors.ErrInvalidCredentials, err)
	})

	t.Run("returns error for wrong password", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockUserRepo.EXPECT().
			FindByEmail(gomock.Any(), gomock.Any()).
			Return(validUser, nil)

		wrongPasswordReq := &models.LoginRequest{
			Email:    "test@example.com",
			Password: "wrongpassword",
		}

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Login(context.Background(), wrongPasswordReq)

		assert.Nil(t, resp)
		assert.Equal(t, apperrors.ErrInvalidCredentials, err)
	})
}

func TestAuthService_Refresh(t *testing.T) {
	validUserID := primitive.NewObjectID()
	refreshReq := &models.RefreshRequest{
		RefreshToken: "rf_valid_refresh_token",
	}

	t.Run("refreshes token from cache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockCache.EXPECT().
			GetRefreshToken(gomock.Any(), refreshReq.RefreshToken).
			Return(validUserID.Hex(), nil)

		mockJWT.EXPECT().
			GenerateToken(validUserID.Hex()).
			Return("new-access-token", nil)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		require.NoError(t, err)
		assert.Equal(t, "new-access-token", resp.AccessToken)
	})

	t.Run("refreshes token from database on cache miss", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		refreshToken := &models.RefreshToken{
			Token:     refreshReq.RefreshToken,
			UserID:    validUserID,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}

		mockCache.EXPECT().
			GetRefreshToken(gomock.Any(), refreshReq.RefreshToken).
			Return("", nil) // Cache miss

		mockRefreshRepo.EXPECT().
			FindByToken(gomock.Any(), refreshReq.RefreshToken).
			Return(refreshToken, nil)

		mockCache.EXPECT().
			SetRefreshToken(gomock.Any(), refreshReq.RefreshToken, validUserID.Hex(), gomock.Any()).
			Return(nil)

		mockJWT.EXPECT().
			GenerateToken(validUserID.Hex()).
			Return("new-access-token", nil)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		require.NoError(t, err)
		assert.Equal(t, "new-access-token", resp.AccessToken)
	})

	t.Run("returns error for invalid refresh token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockCache.EXPECT().
			GetRefreshToken(gomock.Any(), refreshReq.RefreshToken).
			Return("", nil)

		mockRefreshRepo.EXPECT().
			FindByToken(gomock.Any(), refreshReq.RefreshToken).
			Return(nil, assert.AnError)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		assert.Nil(t, resp)
		assert.Equal(t, apperrors.ErrInvalidRefreshToken, err)
	})

	t.Run("returns error on cache error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockCache.EXPECT().
			GetRefreshToken(gomock.Any(), refreshReq.RefreshToken).
			Return("", assert.AnError)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		assert.Nil(t, resp)
		assert.Error(t, err)
	})
}

func TestAuthService_Logout(t *testing.T) {
	logoutReq := &models.LogoutRequest{
		RefreshToken: "rf_refresh_token",
	}

	t.Run("successfully logs out user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockRefreshRepo.EXPECT().
			DeleteByToken(gomock.Any(), logoutReq.RefreshToken).
			Return(nil)

		mockCache.EXPECT().
			DeleteRefreshToken(gomock.Any(), logoutReq.RefreshToken).
			Return(nil)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		err := service.Logout(context.Background(), logoutReq)

		assert.NoError(t, err)
	})

	t.Run("returns error when database delete fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockRefreshRepo.EXPECT().
			DeleteByToken(gomock.Any(), logoutReq.RefreshToken).
			Return(assert.AnError)

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		err := service.Logout(context.Background(), logoutReq)

		assert.Error(t, err)
	})

	t.Run("ignores cache delete error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockRefreshRepo.EXPECT().
			DeleteByToken(gomock.Any(), logoutReq.RefreshToken).
			Return(nil)

		mockCache.EXPECT().
			DeleteRefreshToken(gomock.Any(), logoutReq.RefreshToken).
			Return(assert.AnError) // Cache error is ignored

		service := NewAuthService(
			mockUserRepo, mockRefreshRepo, mockCache, mockJWT,
			15*time.Minute, 7*24*time.Hour,
		)

		err := service.Logout(context.Background(), logoutReq)

		assert.NoError(t, err)
	})
}
