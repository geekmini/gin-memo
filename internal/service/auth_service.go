// Package service contains business logic for the application.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"gin-sample/internal/cache"
	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/repository"
	"gin-sample/pkg/auth"
)

// AuthService handles authentication business logic.
type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	cache            cache.Cache
	jwtManager       auth.TokenManager
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo repository.UserRepository,
	refreshTokenRepo repository.RefreshTokenRepository,
	cache cache.Cache,
	jwtManager auth.TokenManager,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		refreshTokenRepo: refreshTokenRepo,
		cache:            cache,
		jwtManager:       jwtManager,
		accessTokenTTL:   accessTokenTTL,
		refreshTokenTTL:  refreshTokenTTL,
	}
}

// Register creates a new user account and returns auth tokens.
func (s *AuthService) Register(ctx context.Context, req *models.CreateUserRequest) (*models.AuthResponse, error) {
	// Hash the password
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user model
	user := &models.User{
		Email:    req.Email,
		Password: hashedPassword,
		Name:     req.Name,
	}

	// Save to database
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user)
}

// Login authenticates a user and returns auth tokens.
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	// Find user by email
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Check password
	if err := auth.CheckPassword(req.Password, user.Password); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Generate tokens
	return s.generateAuthResponse(ctx, user)
}

// Refresh exchanges a refresh token for a new access token.
func (s *AuthService) Refresh(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error) {
	// Try cache first
	userID, err := s.cache.GetRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Cache miss - check database
	if userID == "" {
		refreshToken, err := s.refreshTokenRepo.FindByToken(ctx, req.RefreshToken)
		if err != nil {
			return nil, apperrors.ErrInvalidRefreshToken
		}
		userID = refreshToken.UserID.Hex()

		// Cache the token for next time
		ttl := time.Until(refreshToken.ExpiresAt)
		if ttl > 0 {
			_ = s.cache.SetRefreshToken(ctx, req.RefreshToken, userID, ttl)
		}
	}

	// Generate new access token
	accessToken, err := s.jwtManager.GenerateToken(userID)
	if err != nil {
		return nil, err
	}

	return &models.RefreshResponse{
		AccessToken: accessToken,
		ExpiresIn:   int(s.accessTokenTTL.Seconds()),
	}, nil
}

// Logout invalidates a refresh token.
func (s *AuthService) Logout(ctx context.Context, req *models.LogoutRequest) error {
	// Delete from database
	if err := s.refreshTokenRepo.DeleteByToken(ctx, req.RefreshToken); err != nil {
		return err
	}

	// Delete from cache
	_ = s.cache.DeleteRefreshToken(ctx, req.RefreshToken)

	return nil
}

// generateAuthResponse creates access and refresh tokens for a user.
func (s *AuthService) generateAuthResponse(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	// Generate access token
	accessToken, err := s.jwtManager.GenerateToken(user.ID.Hex())
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshTokenStr, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Store refresh token in database
	refreshToken := &models.RefreshToken{
		Token:     refreshTokenStr,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, err
	}

	// Cache refresh token
	_ = s.cache.SetRefreshToken(ctx, refreshTokenStr, user.ID.Hex(), s.refreshTokenTTL)

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
		User:         *user,
	}, nil
}

// generateRefreshToken creates a cryptographically secure random token.
func generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "rf_" + hex.EncodeToString(bytes), nil
}
