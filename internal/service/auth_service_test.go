package service

import (
	"context"
	"testing"
	"time"

	"gin-sample/internal/cache"
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

// newTestAuthService creates an AuthService with the given mocks for legacy (non-rotation) mode testing.
func newTestAuthService(
	userRepo *repomocks.MockUserRepository,
	refreshTokenRepo *repomocks.MockRefreshTokenRepository,
	cache *cachemocks.MockCache,
	jwtManager *authmocks.MockTokenManager,
) *AuthService {
	return NewAuthService(AuthServiceConfig{
		UserRepo:         userRepo,
		RefreshTokenRepo: refreshTokenRepo,
		Cache:            cache,
		JWTManager:       jwtManager,
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
		RotationEnabled:  false,
	})
}

func TestNewAuthService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockUserRepo := repomocks.NewMockUserRepository(ctrl)
	mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
	mockCache := cachemocks.NewMockCache(ctrl)
	mockJWT := authmocks.NewMockTokenManager(ctrl)

	service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

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

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

		err := service.Logout(context.Background(), logoutReq)

		assert.NoError(t, err)
	})
}

func TestAuthService_LogoutAll(t *testing.T) {
	userID := primitive.NewObjectID()

	t.Run("successfully logs out all devices", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		tokens := []models.RefreshToken{
			{Token: "rf_token_1", UserID: userID},
			{Token: "rf_token_2", UserID: userID},
		}

		mockRefreshRepo.EXPECT().
			FindAllByUserID(gomock.Any(), userID).
			Return(tokens, nil)

		mockCache.EXPECT().
			DeleteRefreshTokens(gomock.Any(), []string{"rf_token_1", "rf_token_2"}).
			Return(nil)

		mockRefreshRepo.EXPECT().
			DeleteByUserID(gomock.Any(), userID).
			Return(nil)

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

		err := service.LogoutAll(context.Background(), userID)

		assert.NoError(t, err)
	})

	t.Run("succeeds when user has no tokens", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockRefreshRepo.EXPECT().
			FindAllByUserID(gomock.Any(), userID).
			Return([]models.RefreshToken{}, nil)

		mockRefreshRepo.EXPECT().
			DeleteByUserID(gomock.Any(), userID).
			Return(nil)

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

		err := service.LogoutAll(context.Background(), userID)

		assert.NoError(t, err)
	})

	t.Run("returns error when FindAllByUserID fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		mockRefreshRepo.EXPECT().
			FindAllByUserID(gomock.Any(), userID).
			Return(nil, assert.AnError)

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

		err := service.LogoutAll(context.Background(), userID)

		assert.Error(t, err)
	})

	t.Run("ignores cache delete error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		tokens := []models.RefreshToken{
			{Token: "rf_token_1", UserID: userID},
		}

		mockRefreshRepo.EXPECT().
			FindAllByUserID(gomock.Any(), userID).
			Return(tokens, nil)

		mockCache.EXPECT().
			DeleteRefreshTokens(gomock.Any(), []string{"rf_token_1"}).
			Return(assert.AnError) // Cache error is ignored

		mockRefreshRepo.EXPECT().
			DeleteByUserID(gomock.Any(), userID).
			Return(nil)

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

		err := service.LogoutAll(context.Background(), userID)

		assert.NoError(t, err)
	})

	t.Run("returns error when DeleteByUserID fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)

		tokens := []models.RefreshToken{
			{Token: "rf_token_1", UserID: userID},
		}

		mockRefreshRepo.EXPECT().
			FindAllByUserID(gomock.Any(), userID).
			Return(tokens, nil)

		mockCache.EXPECT().
			DeleteRefreshTokens(gomock.Any(), []string{"rf_token_1"}).
			Return(nil)

		mockRefreshRepo.EXPECT().
			DeleteByUserID(gomock.Any(), userID).
			Return(assert.AnError)

		service := newTestAuthService(mockUserRepo, mockRefreshRepo, mockCache, mockJWT)

		err := service.LogoutAll(context.Background(), userID)

		assert.Error(t, err)
	})
}

// Rotation mode tests

// newTestAuthServiceWithRotation creates an AuthService with rotation enabled for testing.
func newTestAuthServiceWithRotation(
	userRepo *repomocks.MockUserRepository,
	refreshTokenRepo *repomocks.MockRefreshTokenRepository,
	cache *cachemocks.MockCache,
	tokenStore *cachemocks.MockRefreshTokenStore,
	jwtManager *authmocks.MockTokenManager,
	tokenGenerator *authmocks.MockRefreshTokenGenerator,
) *AuthService {
	return NewAuthService(AuthServiceConfig{
		UserRepo:         userRepo,
		RefreshTokenRepo: refreshTokenRepo,
		Cache:            cache,
		TokenStore:       tokenStore,
		JWTManager:       jwtManager,
		TokenGenerator:   tokenGenerator,
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  7 * 24 * time.Hour,
		RotationEnabled:  true,
	})
}

func TestAuthService_Register_WithRotation(t *testing.T) {
	createUserReq := &models.CreateUserRequest{
		Email:    "test@example.com",
		Password: "password123",
		Name:     "Test User",
	}

	t.Run("successfully registers new user with rotation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockTokenStore := cachemocks.NewMockRefreshTokenStore(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)
		mockTokenGen := authmocks.NewMockRefreshTokenGenerator(ctrl)

		// Expect user creation
		mockUserRepo.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, user *models.User) error {
				user.ID = primitive.NewObjectID()
				return nil
			})

		// Expect JWT generation
		mockJWT.EXPECT().
			GenerateToken(gomock.Any()).
			Return("access-token", nil)

		// Expect rotation token generation
		mockTokenGen.EXPECT().
			Generate().
			Return("rt_family123_random456", "family123", nil)

		mockTokenGen.EXPECT().
			Hash("rt_family123_random456").
			Return("hashed_token")

		// Expect token store creation
		mockTokenStore.EXPECT().
			Create(gomock.Any(), "family123", gomock.Any(), 7*24*time.Hour).
			Return(nil)

		service := newTestAuthServiceWithRotation(
			mockUserRepo, mockRefreshRepo, mockCache, mockTokenStore, mockJWT, mockTokenGen,
		)

		resp, err := service.Register(context.Background(), createUserReq)

		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "access-token", resp.AccessToken)
		assert.Equal(t, "rt_family123_random456", resp.RefreshToken)
	})
}

func TestAuthService_Refresh_WithRotation(t *testing.T) {
	refreshReq := &models.RefreshRequest{
		RefreshToken: "rt_family123_random456",
	}

	t.Run("successfully refreshes with rotation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockTokenStore := cachemocks.NewMockRefreshTokenStore(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)
		mockTokenGen := authmocks.NewMockRefreshTokenGenerator(ctrl)

		storedData := &cache.RefreshTokenData{
			UserID:           "user123",
			CurrentTokenHash: "current_hash",
			ExpiresAt:        time.Now().Add(1 * time.Hour),
		}

		// Extract family ID from token
		mockTokenGen.EXPECT().
			ExtractFamilyID(refreshReq.RefreshToken).
			Return("family123", nil)

		// Get stored token data
		mockTokenStore.EXPECT().
			Get(gomock.Any(), "family123").
			Return(storedData, nil)

		// Hash incoming token and compare
		mockTokenGen.EXPECT().
			Hash(refreshReq.RefreshToken).
			Return("current_hash")

		mockTokenGen.EXPECT().
			CompareHashes("current_hash", "current_hash").
			Return(true)

		// Generate new refresh token
		mockTokenGen.EXPECT().
			GenerateWithFamily("family123").
			Return("rt_family123_newrandom", nil)

		// Generate new access token
		mockJWT.EXPECT().
			GenerateToken("user123").
			Return("new-access-token", nil)

		// Hash new token
		mockTokenGen.EXPECT().
			Hash("rt_family123_newrandom").
			Return("new_hash")

		// Rotate stored data
		mockTokenStore.EXPECT().
			Rotate(gomock.Any(), "family123", "new_hash", 7*24*time.Hour).
			Return(nil)

		service := newTestAuthServiceWithRotation(
			mockUserRepo, mockRefreshRepo, mockCache, mockTokenStore, mockJWT, mockTokenGen,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		require.NoError(t, err)
		assert.Equal(t, "new-access-token", resp.AccessToken)
		assert.Equal(t, "rt_family123_newrandom", resp.RefreshToken)
	})

	t.Run("returns error for invalid token format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockTokenStore := cachemocks.NewMockRefreshTokenStore(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)
		mockTokenGen := authmocks.NewMockRefreshTokenGenerator(ctrl)

		mockTokenGen.EXPECT().
			ExtractFamilyID(refreshReq.RefreshToken).
			Return("", assert.AnError)

		service := newTestAuthServiceWithRotation(
			mockUserRepo, mockRefreshRepo, mockCache, mockTokenStore, mockJWT, mockTokenGen,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		assert.Nil(t, resp)
		assert.Equal(t, apperrors.ErrInvalidRefreshToken, err)
	})

	t.Run("returns error for expired token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockTokenStore := cachemocks.NewMockRefreshTokenStore(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)
		mockTokenGen := authmocks.NewMockRefreshTokenGenerator(ctrl)

		storedData := &cache.RefreshTokenData{
			UserID:           "user123",
			CurrentTokenHash: "current_hash",
			ExpiresAt:        time.Now().Add(-1 * time.Hour), // Expired
		}

		mockTokenGen.EXPECT().
			ExtractFamilyID(refreshReq.RefreshToken).
			Return("family123", nil)

		mockTokenStore.EXPECT().
			Get(gomock.Any(), "family123").
			Return(storedData, nil)

		// Token is expired, so delete should be called
		mockTokenStore.EXPECT().
			Delete(gomock.Any(), "family123").
			Return(nil)

		service := newTestAuthServiceWithRotation(
			mockUserRepo, mockRefreshRepo, mockCache, mockTokenStore, mockJWT, mockTokenGen,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		assert.Nil(t, resp)
		assert.Equal(t, apperrors.ErrRefreshTokenExpired, err)
	})

	t.Run("detects token reuse and invalidates family", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockTokenStore := cachemocks.NewMockRefreshTokenStore(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)
		mockTokenGen := authmocks.NewMockRefreshTokenGenerator(ctrl)

		storedData := &cache.RefreshTokenData{
			UserID:            "user123",
			CurrentTokenHash:  "current_hash",
			PreviousTokenHash: "previous_hash",
			ExpiresAt:         time.Now().Add(1 * time.Hour),
		}

		mockTokenGen.EXPECT().
			ExtractFamilyID(refreshReq.RefreshToken).
			Return("family123", nil)

		mockTokenStore.EXPECT().
			Get(gomock.Any(), "family123").
			Return(storedData, nil)

		// Hash incoming token
		mockTokenGen.EXPECT().
			Hash(refreshReq.RefreshToken).
			Return("previous_hash") // Matches previous, not current

		// Compare with current (should return false)
		mockTokenGen.EXPECT().
			CompareHashes("previous_hash", "current_hash").
			Return(false)

		// Compare with previous (should return true - reuse detected!)
		mockTokenGen.EXPECT().
			CompareHashes("previous_hash", "previous_hash").
			Return(true)

		// Should delete the family due to reuse
		mockTokenStore.EXPECT().
			Delete(gomock.Any(), "family123").
			Return(nil)

		service := newTestAuthServiceWithRotation(
			mockUserRepo, mockRefreshRepo, mockCache, mockTokenStore, mockJWT, mockTokenGen,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		assert.Nil(t, resp)
		assert.Equal(t, apperrors.ErrRefreshTokenReused, err)
	})

	t.Run("returns error for unknown token", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockTokenStore := cachemocks.NewMockRefreshTokenStore(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)
		mockTokenGen := authmocks.NewMockRefreshTokenGenerator(ctrl)

		storedData := &cache.RefreshTokenData{
			UserID:            "user123",
			CurrentTokenHash:  "current_hash",
			PreviousTokenHash: "previous_hash",
			ExpiresAt:         time.Now().Add(1 * time.Hour),
		}

		mockTokenGen.EXPECT().
			ExtractFamilyID(refreshReq.RefreshToken).
			Return("family123", nil)

		mockTokenStore.EXPECT().
			Get(gomock.Any(), "family123").
			Return(storedData, nil)

		// Hash incoming token - doesn't match current or previous
		mockTokenGen.EXPECT().
			Hash(refreshReq.RefreshToken).
			Return("unknown_hash")

		mockTokenGen.EXPECT().
			CompareHashes("unknown_hash", "current_hash").
			Return(false)

		mockTokenGen.EXPECT().
			CompareHashes("unknown_hash", "previous_hash").
			Return(false)

		service := newTestAuthServiceWithRotation(
			mockUserRepo, mockRefreshRepo, mockCache, mockTokenStore, mockJWT, mockTokenGen,
		)

		resp, err := service.Refresh(context.Background(), refreshReq)

		assert.Nil(t, resp)
		assert.Equal(t, apperrors.ErrInvalidRefreshToken, err)
	})
}

func TestAuthService_Logout_WithRotation(t *testing.T) {
	logoutReq := &models.LogoutRequest{
		RefreshToken: "rt_family123_random456",
	}

	t.Run("successfully logs out with rotation", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockTokenStore := cachemocks.NewMockRefreshTokenStore(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)
		mockTokenGen := authmocks.NewMockRefreshTokenGenerator(ctrl)

		mockTokenGen.EXPECT().
			ExtractFamilyID(logoutReq.RefreshToken).
			Return("family123", nil)

		mockTokenStore.EXPECT().
			Delete(gomock.Any(), "family123").
			Return(nil)

		service := newTestAuthServiceWithRotation(
			mockUserRepo, mockRefreshRepo, mockCache, mockTokenStore, mockJWT, mockTokenGen,
		)

		err := service.Logout(context.Background(), logoutReq)

		assert.NoError(t, err)
	})

	t.Run("logout is idempotent for invalid token format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUserRepo := repomocks.NewMockUserRepository(ctrl)
		mockRefreshRepo := repomocks.NewMockRefreshTokenRepository(ctrl)
		mockCache := cachemocks.NewMockCache(ctrl)
		mockTokenStore := cachemocks.NewMockRefreshTokenStore(ctrl)
		mockJWT := authmocks.NewMockTokenManager(ctrl)
		mockTokenGen := authmocks.NewMockRefreshTokenGenerator(ctrl)

		mockTokenGen.EXPECT().
			ExtractFamilyID(logoutReq.RefreshToken).
			Return("", assert.AnError)

		// Should not call Delete since token format is invalid
		// But logout should succeed (idempotent)

		service := newTestAuthServiceWithRotation(
			mockUserRepo, mockRefreshRepo, mockCache, mockTokenStore, mockJWT, mockTokenGen,
		)

		err := service.Logout(context.Background(), logoutReq)

		assert.NoError(t, err)
	})
}
