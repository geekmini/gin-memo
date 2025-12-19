// Package service contains business logic for the application.
package service

import (
	"context"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/repository"
	"gin-sample/pkg/auth"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserService handles business logic for user operations.
type UserService struct {
	repo       repository.UserRepository
	jwtManager *auth.JWTManager
}

// NewUserService creates a new UserService.
func NewUserService(repo repository.UserRepository, jwtManager *auth.JWTManager) *UserService {
	return &UserService{
		repo:       repo,
		jwtManager: jwtManager,
	}
}

// Register creates a new user account.
func (s *UserService) Register(ctx context.Context, req *models.CreateUserRequest) (*models.User, error) {
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
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user and returns a JWT token.
func (s *UserService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	// Find user by email
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Check password
	if err := auth.CheckPassword(req.Password, user.Password); err != nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(user.ID.Hex())
	if err != nil {
		return nil, err
	}

	return &models.LoginResponse{
		Token: token,
		User:  *user,
	}, nil
}

// GetUser retrieves a user by ID.
func (s *UserService) GetUser(ctx context.Context, id string) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	return s.repo.FindByID(ctx, objectID)
}

// GetAllUsers retrieves all users.
func (s *UserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	return s.repo.FindAll(ctx)
}

// UpdateUser updates a user's information.
func (s *UserService) UpdateUser(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	return s.repo.Update(ctx, objectID, req)
}

// DeleteUser removes a user.
func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return apperrors.ErrUserNotFound
	}

	return s.repo.Delete(ctx, objectID)
}
