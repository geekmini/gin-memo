package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/service/mocks"
	"gin-sample/internal/validator"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
}

func TestNewAuthHandler(t *testing.T) {
	mockService := &mocks.MockAuthService{}
	handler := NewAuthHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

func TestAuthHandler_Register(t *testing.T) {
	userID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func(*mocks.MockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful registration",
			body: models.CreateUserRequest{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.RegisterFunc = func(ctx context.Context, req *models.CreateUserRequest) (*models.AuthResponse, error) {
					return &models.AuthResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    900,
						User: models.User{
							ID:        userID,
							Email:     req.Email,
							Name:      req.Name,
							CreatedAt: now,
							UpdatedAt: now,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, true, resp["success"])
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "access-token", data["accessToken"])
				assert.Equal(t, "refresh-token", data["refreshToken"])
			},
		},
		{
			name:           "invalid JSON body",
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing required fields",
			body: map[string]string{
				"email": "test@example.com",
			},
			mockSetup:      func(m *mocks.MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "user already exists",
			body: models.CreateUserRequest{
				Email:    "existing@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.RegisterFunc = func(ctx context.Context, req *models.CreateUserRequest) (*models.AuthResponse, error) {
					return nil, apperrors.ErrUserAlreadyExists
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "internal server error",
			body: models.CreateUserRequest{
				Email:    "test@example.com",
				Password: "password123",
				Name:     "Test User",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.RegisterFunc = func(ctx context.Context, req *models.CreateUserRequest) (*models.AuthResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			router := gin.New()
			router.POST("/auth/register", handler.Register)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandler_Login(t *testing.T) {
	userID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func(*mocks.MockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful login",
			body: models.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.LoginFunc = func(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
					return &models.AuthResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
						ExpiresIn:    900,
						User: models.User{
							ID:        userID,
							Email:     req.Email,
							Name:      "Test User",
							CreatedAt: now,
							UpdatedAt: now,
						},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, true, resp["success"])
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "access-token", data["accessToken"])
			},
		},
		{
			name:           "invalid JSON body",
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "missing password",
			body: map[string]string{
				"email": "test@example.com",
			},
			mockSetup:      func(m *mocks.MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid credentials",
			body: models.LoginRequest{
				Email:    "test@example.com",
				Password: "wrongpassword",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.LoginFunc = func(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
					return nil, apperrors.ErrInvalidCredentials
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "internal server error",
			body: models.LoginRequest{
				Email:    "test@example.com",
				Password: "password123",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.LoginFunc = func(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			router := gin.New()
			router.POST("/auth/login", handler.Login)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandler_Refresh(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func(*mocks.MockAuthService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful refresh",
			body: models.RefreshRequest{
				RefreshToken: "valid-refresh-token",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.RefreshFunc = func(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error) {
					return &models.RefreshResponse{
						AccessToken: "new-access-token",
						ExpiresIn:   900,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, true, resp["success"])
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "new-access-token", data["accessToken"])
			},
		},
		{
			name:           "invalid JSON body",
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing refresh token",
			body:           map[string]string{},
			mockSetup:      func(m *mocks.MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid refresh token",
			body: models.RefreshRequest{
				RefreshToken: "invalid-token",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.RefreshFunc = func(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error) {
					return nil, apperrors.ErrInvalidRefreshToken
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "internal server error",
			body: models.RefreshRequest{
				RefreshToken: "valid-token",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.RefreshFunc = func(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			router := gin.New()
			router.POST("/auth/refresh", handler.Refresh)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestAuthHandler_Logout(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		mockSetup      func(*mocks.MockAuthService)
		expectedStatus int
	}{
		{
			name: "successful logout",
			body: models.LogoutRequest{
				RefreshToken: "valid-refresh-token",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.LogoutFunc = func(ctx context.Context, req *models.LogoutRequest) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid JSON body",
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing refresh token",
			body:           map[string]string{},
			mockSetup:      func(m *mocks.MockAuthService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "internal server error",
			body: models.LogoutRequest{
				RefreshToken: "valid-token",
			},
			mockSetup: func(m *mocks.MockAuthService) {
				m.LogoutFunc = func(ctx context.Context, req *models.LogoutRequest) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockAuthService{}
			tt.mockSetup(mockService)

			handler := NewAuthHandler(mockService)

			router := gin.New()
			router.POST("/auth/logout", handler.Logout)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
