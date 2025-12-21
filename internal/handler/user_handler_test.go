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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewUserHandler(t *testing.T) {
	mockService := &mocks.MockUserService{}
	handler := NewUserHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

func TestUserHandler_GetUser(t *testing.T) {
	userID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful get user",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockUserService) {
				m.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{
						ID:        userID,
						Email:     "test@example.com",
						Name:      "Test User",
						CreatedAt: now,
						UpdatedAt: now,
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
				assert.Equal(t, "test@example.com", data["email"])
			},
		},
		{
			name:   "user not found",
			userID: primitive.NewObjectID().Hex(),
			mockSetup: func(m *mocks.MockUserService) {
				m.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return nil, apperrors.ErrUserNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockUserService) {
				m.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockUserService{}
			tt.mockSetup(mockService)

			handler := NewUserHandler(mockService)

			router := gin.New()
			router.GET("/users/:id", handler.GetUser)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.userID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestUserHandler_GetAllUsers(t *testing.T) {
	userID1 := primitive.NewObjectID()
	userID2 := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		mockSetup      func(*mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful get all users",
			mockSetup: func(m *mocks.MockUserService) {
				m.GetAllUsersFunc = func(ctx context.Context) ([]models.User, error) {
					return []models.User{
						{ID: userID1, Email: "user1@example.com", Name: "User 1", CreatedAt: now, UpdatedAt: now},
						{ID: userID2, Email: "user2@example.com", Name: "User 2", CreatedAt: now, UpdatedAt: now},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, true, resp["success"])
				data := resp["data"].([]interface{})
				assert.Len(t, data, 2)
			},
		},
		{
			name: "empty user list",
			mockSetup: func(m *mocks.MockUserService) {
				m.GetAllUsersFunc = func(ctx context.Context) ([]models.User, error) {
					return []models.User{}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].([]interface{})
				assert.Len(t, data, 0)
			},
		},
		{
			name: "internal server error",
			mockSetup: func(m *mocks.MockUserService) {
				m.GetAllUsersFunc = func(ctx context.Context) ([]models.User, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockUserService{}
			tt.mockSetup(mockService)

			handler := NewUserHandler(mockService)

			router := gin.New()
			router.GET("/users", handler.GetAllUsers)

			req := httptest.NewRequest(http.MethodGet, "/users", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestUserHandler_UpdateUser(t *testing.T) {
	userID := primitive.NewObjectID()
	now := time.Now()
	newEmail := "updated@example.com"
	newName := "Updated Name"

	tests := []struct {
		name           string
		userID         string
		body           interface{}
		mockSetup      func(*mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful update user",
			userID: userID.Hex(),
			body: models.UpdateUserRequest{
				Email: &newEmail,
				Name:  &newName,
			},
			mockSetup: func(m *mocks.MockUserService) {
				m.UpdateUserFunc = func(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
					return &models.User{
						ID:        userID,
						Email:     *req.Email,
						Name:      *req.Name,
						CreatedAt: now,
						UpdatedAt: now,
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
				assert.Equal(t, newEmail, data["email"])
			},
		},
		{
			name:           "invalid JSON body",
			userID:         userID.Hex(),
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "user not found",
			userID: primitive.NewObjectID().Hex(),
			body: models.UpdateUserRequest{
				Email: &newEmail,
			},
			mockSetup: func(m *mocks.MockUserService) {
				m.UpdateUserFunc = func(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
					return nil, apperrors.ErrUserNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "email already exists",
			userID: userID.Hex(),
			body: models.UpdateUserRequest{
				Email: &newEmail,
			},
			mockSetup: func(m *mocks.MockUserService) {
				m.UpdateUserFunc = func(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
					return nil, apperrors.ErrUserAlreadyExists
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			body: models.UpdateUserRequest{
				Email: &newEmail,
			},
			mockSetup: func(m *mocks.MockUserService) {
				m.UpdateUserFunc = func(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockUserService{}
			tt.mockSetup(mockService)

			handler := NewUserHandler(mockService)

			router := gin.New()
			router.PUT("/users/:id", handler.UpdateUser)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.userID, bytes.NewBuffer(body))
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

func TestUserHandler_DeleteUser(t *testing.T) {
	userID := primitive.NewObjectID()

	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful delete user",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockUserService) {
				m.DeleteUserFunc = func(ctx context.Context, id string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, true, resp["success"])
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "user deleted", data["message"])
			},
		},
		{
			name:   "user not found",
			userID: primitive.NewObjectID().Hex(),
			mockSetup: func(m *mocks.MockUserService) {
				m.DeleteUserFunc = func(ctx context.Context, id string) error {
					return apperrors.ErrUserNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockUserService) {
				m.DeleteUserFunc = func(ctx context.Context, id string) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockUserService{}
			tt.mockSetup(mockService)

			handler := NewUserHandler(mockService)

			router := gin.New()
			router.DELETE("/users/:id", handler.DeleteUser)

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.userID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
