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

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthService handles authentication business logic.
type AuthService struct {
	userRepo         repository.UserRepository
	refreshTokenRepo repository.RefreshTokenRepository
	cache            cache.Cache
	tokenStore       cache.RefreshTokenStore
	jwtManager       auth.TokenManager
	tokenGenerator   auth.RefreshTokenGenerator
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	rotationEnabled  bool
}

// AuthServiceConfig holds configuration for AuthService.
type AuthServiceConfig struct {
	UserRepo         repository.UserRepository
	RefreshTokenRepo repository.RefreshTokenRepository
	Cache            cache.Cache
	TokenStore       cache.RefreshTokenStore
	JWTManager       auth.TokenManager
	TokenGenerator   auth.RefreshTokenGenerator
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	RotationEnabled  bool
}

// NewAuthService creates a new AuthService.
func NewAuthService(cfg AuthServiceConfig) *AuthService {
	return &AuthService{
		userRepo:         cfg.UserRepo,
		refreshTokenRepo: cfg.RefreshTokenRepo,
		cache:            cfg.Cache,
		tokenStore:       cfg.TokenStore,
		jwtManager:       cfg.JWTManager,
		tokenGenerator:   cfg.TokenGenerator,
		accessTokenTTL:   cfg.AccessTokenTTL,
		refreshTokenTTL:  cfg.RefreshTokenTTL,
		rotationEnabled:  cfg.RotationEnabled,
	}
}

// Register creates a new user account and returns auth tokens.
func (s *AuthService) Register(ctx context.Context, req *models.CreateUserRequest) (*models.AuthResponse, error) {
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:    req.Email,
		Password: hashedPassword,
		Name:     req.Name,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.generateAuthResponse(ctx, user)
}

// Login authenticates a user and returns auth tokens.
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	if err := auth.CheckPassword(req.Password, user.Password); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	return s.generateAuthResponse(ctx, user)
}

// Refresh exchanges a refresh token for a new access token.
// If rotation is enabled, returns a new refresh token as well.
func (s *AuthService) Refresh(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error) {
	if s.rotationEnabled && s.tokenGenerator != nil {
		return s.refreshWithRotation(ctx, req)
	}
	return s.refreshWithoutRotation(ctx, req)
}

// refreshWithRotation handles refresh with token rotation.
func (s *AuthService) refreshWithRotation(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error) {
	// Extract family ID from token
	familyID, err := s.tokenGenerator.ExtractFamilyID(req.RefreshToken)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	// Get stored token data from Redis
	storedData, err := s.tokenStore.Get(ctx, familyID)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}
	if storedData == nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	// Check if token has expired
	if time.Now().After(storedData.ExpiresAt) {
		_ = s.tokenStore.Delete(ctx, familyID)
		return nil, apperrors.ErrRefreshTokenExpired
	}

	// Verify token hash (reuse detection)
	incomingHash := s.tokenGenerator.Hash(req.RefreshToken)

	// Check against current token
	if s.tokenGenerator.CompareHashes(incomingHash, storedData.CurrentTokenHash) {
		// Valid current token - perform rotation
		return s.performRotation(ctx, familyID, storedData)
	}

	// Check against previous token (1-token lookback for reuse detection)
	if storedData.PreviousTokenHash != "" && s.tokenGenerator.CompareHashes(incomingHash, storedData.PreviousTokenHash) {
		// REUSE DETECTED - invalidate entire family
		_ = s.tokenStore.Delete(ctx, familyID)
		return nil, apperrors.ErrRefreshTokenReused
	}

	// Token doesn't match current or previous - invalid
	return nil, apperrors.ErrInvalidRefreshToken
}

// performRotation generates new tokens and rotates the stored token data.
func (s *AuthService) performRotation(ctx context.Context, familyID string, storedData *cache.RefreshTokenData) (*models.RefreshResponse, error) {
	// Generate new refresh token with same family
	newRefreshToken, err := s.tokenGenerator.GenerateWithFamily(familyID)
	if err != nil {
		return nil, err
	}

	// Generate new access token
	accessToken, err := s.jwtManager.GenerateToken(storedData.UserID)
	if err != nil {
		return nil, err
	}

	// Hash new refresh token
	newHash := s.tokenGenerator.Hash(newRefreshToken)

	// Rotate stored data (current becomes previous)
	if err := s.tokenStore.Rotate(ctx, familyID, newHash, s.refreshTokenTTL); err != nil {
		return nil, err
	}

	return &models.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
	}, nil
}

// refreshWithoutRotation handles refresh without token rotation (legacy behavior).
func (s *AuthService) refreshWithoutRotation(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error) {
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
	if s.rotationEnabled && s.tokenGenerator != nil {
		return s.logoutWithRotation(ctx, req)
	}
	return s.logoutWithoutRotation(ctx, req)
}

// logoutWithRotation handles logout with token rotation.
func (s *AuthService) logoutWithRotation(ctx context.Context, req *models.LogoutRequest) error {
	familyID, err := s.tokenGenerator.ExtractFamilyID(req.RefreshToken)
	if err != nil {
		// Invalid format, but logout should be idempotent
		return nil
	}
	// Delete the token family - ignore errors for idempotency
	_ = s.tokenStore.Delete(ctx, familyID)
	return nil
}

// logoutWithoutRotation handles logout without token rotation (legacy behavior).
func (s *AuthService) logoutWithoutRotation(ctx context.Context, req *models.LogoutRequest) error {
	// Delete from database
	if err := s.refreshTokenRepo.DeleteByToken(ctx, req.RefreshToken); err != nil {
		return err
	}

	// Delete from cache
	_ = s.cache.DeleteRefreshToken(ctx, req.RefreshToken)

	return nil
}

// LogoutAll invalidates all refresh tokens for a user.
func (s *AuthService) LogoutAll(ctx context.Context, userID primitive.ObjectID) error {
	// Get all tokens for user from MongoDB
	tokens, err := s.refreshTokenRepo.FindAllByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// Extract token strings for cache deletion
	if len(tokens) > 0 {
		tokenStrings := make([]string, len(tokens))
		for i, t := range tokens {
			tokenStrings[i] = t.Token
		}

		// Batch delete from cache (best-effort)
		_ = s.cache.DeleteRefreshTokens(ctx, tokenStrings)
	}

	// Delete all from MongoDB
	return s.refreshTokenRepo.DeleteByUserID(ctx, userID)
}

// generateAuthResponse creates access and refresh tokens for a user.
func (s *AuthService) generateAuthResponse(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	accessToken, err := s.jwtManager.GenerateToken(user.ID.Hex())
	if err != nil {
		return nil, err
	}

	var refreshTokenStr string

	if s.rotationEnabled && s.tokenGenerator != nil {
		// Use family-based rotation tokens
		refreshTokenStr, err = s.generateRotationToken(ctx, user.ID.Hex())
		if err != nil {
			return nil, err
		}
	} else {
		// Use legacy token storage
		refreshTokenStr, err = s.generateLegacyToken(ctx, user.ID)
		if err != nil {
			return nil, err
		}
	}

	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
		User:         *user,
	}, nil
}

// generateRotationToken creates a new refresh token with family-based rotation.
func (s *AuthService) generateRotationToken(ctx context.Context, userID string) (string, error) {
	token, familyID, err := s.tokenGenerator.Generate()
	if err != nil {
		return "", err
	}

	tokenData := &cache.RefreshTokenData{
		UserID:           userID,
		CurrentTokenHash: s.tokenGenerator.Hash(token),
		ExpiresAt:        time.Now().Add(s.refreshTokenTTL),
		CreatedAt:        time.Now(),
	}

	if err := s.tokenStore.Create(ctx, familyID, tokenData, s.refreshTokenTTL); err != nil {
		return "", err
	}

	return token, nil
}

// generateRandomToken creates a cryptographically secure random token.
func generateRandomToken() string {
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	return "rf_" + hex.EncodeToString(bytes)
}

// generateLegacyToken creates a refresh token using the legacy MongoDB storage.
func (s *AuthService) generateLegacyToken(ctx context.Context, userID primitive.ObjectID) (string, error) {
	refreshTokenStr := generateRandomToken()

	refreshToken := &models.RefreshToken{
		Token:     refreshTokenStr,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.refreshTokenTTL),
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return "", err
	}

	// Cache refresh token
	_ = s.cache.SetRefreshToken(ctx, refreshTokenStr, userID.Hex(), s.refreshTokenTTL)

	return refreshTokenStr, nil
}
