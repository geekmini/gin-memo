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

func TestNewVoiceMemoHandler(t *testing.T) {
	mockService := &mocks.MockVoiceMemoService{}
	handler := NewVoiceMemoHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

func TestVoiceMemoHandler_ListVoiceMemos(t *testing.T) {
	userID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		userID         string
		query          string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful list voice memos",
			userID: userID.Hex(),
			query:  "?page=1&limit=10",
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ListByUserIDFunc = func(ctx context.Context, uid string, page, limit int) (*models.VoiceMemoListResponse, error) {
					return &models.VoiceMemoListResponse{
						Items: []models.VoiceMemo{
							{
								ID:        memoID,
								UserID:    userID,
								Title:     "Test Memo",
								Status:    models.StatusReady,
								CreatedAt: now,
								UpdatedAt: now,
							},
						},
						Pagination: models.Pagination{
							Page:       1,
							Limit:      10,
							TotalItems: 1,
							TotalPages: 1,
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
				items := data["items"].([]interface{})
				assert.Len(t, items, 1)
			},
		},
		{
			name:           "missing user ID",
			userID:         "",
			query:          "",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			query:  "",
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ListByUserIDFunc = func(ctx context.Context, uid string, page, limit int) (*models.VoiceMemoListResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.userID != "" {
				router.GET("/voice-memos", setUserID(tt.userID), handler.ListVoiceMemos)
			} else {
				router.GET("/voice-memos", handler.ListVoiceMemos)
			}

			req := httptest.NewRequest(http.MethodGet, "/voice-memos"+tt.query, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestVoiceMemoHandler_CreateVoiceMemo(t *testing.T) {
	userID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		userID         string
		body           interface{}
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful create voice memo",
			userID: userID.Hex(),
			body: models.CreateVoiceMemoRequest{
				Title:       "Test Memo",
				Duration:    120,
				FileSize:    1048576,
				AudioFormat: "mp3",
				Tags:        []string{"work", "meeting"},
			},
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.CreateVoiceMemoFunc = func(ctx context.Context, uid primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error) {
					return &models.CreateVoiceMemoResponse{
						Memo: models.VoiceMemo{
							ID:          memoID,
							UserID:      uid,
							Title:       req.Title,
							Duration:    req.Duration,
							FileSize:    req.FileSize,
							AudioFormat: req.AudioFormat,
							Tags:        req.Tags,
							Status:      models.StatusPendingUpload,
							CreatedAt:   now,
							UpdatedAt:   now,
						},
						UploadURL: "https://s3.amazonaws.com/bucket/voice-memos/test.mp3?signed=true",
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
				assert.Contains(t, data, "uploadUrl")
				assert.Contains(t, data, "memo")
			},
		},
		{
			name:           "missing user ID",
			userID:         "",
			body:           models.CreateVoiceMemoRequest{Title: "Test", FileSize: 1000, AudioFormat: "mp3"},
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-id",
			body:           models.CreateVoiceMemoRequest{Title: "Test", FileSize: 1000, AudioFormat: "mp3"},
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid JSON body",
			userID:         userID.Hex(),
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			body: models.CreateVoiceMemoRequest{
				Title:       "Test",
				FileSize:    1000,
				AudioFormat: "mp3",
			},
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.CreateVoiceMemoFunc = func(ctx context.Context, uid primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.userID != "" {
				router.POST("/voice-memos", setUserID(tt.userID), handler.CreateVoiceMemo)
			} else {
				router.POST("/voice-memos", handler.CreateVoiceMemo)
			}

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/voice-memos", bytes.NewBuffer(body))
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

func TestVoiceMemoHandler_DeleteVoiceMemo(t *testing.T) {
	userID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()

	tests := []struct {
		name           string
		userID         string
		memoID         string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
	}{
		{
			name:   "successful delete voice memo",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.DeleteVoiceMemoFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid memo ID format",
			userID:         userID.Hex(),
			memoID:         "invalid-id",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing user ID",
			userID:         "",
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-id",
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "voice memo not found",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.DeleteVoiceMemoFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "unauthorized - not owner",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.DeleteVoiceMemoFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoUnauthorized
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.DeleteVoiceMemoFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.userID != "" {
				router.DELETE("/voice-memos/:id", setUserID(tt.userID), handler.DeleteVoiceMemo)
			} else {
				router.DELETE("/voice-memos/:id", handler.DeleteVoiceMemo)
			}

			req := httptest.NewRequest(http.MethodDelete, "/voice-memos/"+tt.memoID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestVoiceMemoHandler_ConfirmUpload(t *testing.T) {
	userID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()

	tests := []struct {
		name           string
		userID         string
		memoID         string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful confirm upload",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmUploadFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "upload confirmed, transcription started", data["message"])
			},
		},
		{
			name:           "invalid memo ID format",
			userID:         userID.Hex(),
			memoID:         "invalid-id",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing user ID",
			userID:         "",
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-id",
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "voice memo not found",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmUploadFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "unauthorized - not owner",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmUploadFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoUnauthorized
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "invalid status transition",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmUploadFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoInvalidStatus
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "transcription queue full",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmUploadFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrTranscriptionQueueFull
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmUploadFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.userID != "" {
				router.POST("/voice-memos/:id/confirm-upload", setUserID(tt.userID), handler.ConfirmUpload)
			} else {
				router.POST("/voice-memos/:id/confirm-upload", handler.ConfirmUpload)
			}

			req := httptest.NewRequest(http.MethodPost, "/voice-memos/"+tt.memoID+"/confirm-upload", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestVoiceMemoHandler_RetryTranscription(t *testing.T) {
	userID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()

	tests := []struct {
		name           string
		userID         string
		memoID         string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful retry transcription",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTranscriptionFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "transcription retry started", data["message"])
			},
		},
		{
			name:           "invalid memo ID format",
			userID:         userID.Hex(),
			memoID:         "invalid-id",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing user ID",
			userID:         "",
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-id",
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "voice memo not found",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTranscriptionFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "unauthorized - not owner",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTranscriptionFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoUnauthorized
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "memo not in failed state",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTranscriptionFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoInvalidStatus
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "transcription queue full",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTranscriptionFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return apperrors.ErrTranscriptionQueueFull
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTranscriptionFunc = func(ctx context.Context, mid, uid primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.userID != "" {
				router.POST("/voice-memos/:id/retry-transcription", setUserID(tt.userID), handler.RetryTranscription)
			} else {
				router.POST("/voice-memos/:id/retry-transcription", handler.RetryTranscription)
			}

			req := httptest.NewRequest(http.MethodPost, "/voice-memos/"+tt.memoID+"/retry-transcription", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestVoiceMemoHandler_ListTeamVoiceMemos(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		query          string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful list team voice memos",
			teamID: &teamID,
			query:  "?page=1&limit=10",
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ListByTeamIDFunc = func(ctx context.Context, tid string, page, limit int) (*models.VoiceMemoListResponse, error) {
					return &models.VoiceMemoListResponse{
						Items: []models.VoiceMemo{
							{
								ID:        memoID,
								UserID:    userID,
								TeamID:    &teamID,
								Title:     "Team Memo",
								Status:    models.StatusReady,
								CreatedAt: now,
								UpdatedAt: now,
							},
						},
						Pagination: models.Pagination{
							Page:       1,
							Limit:      10,
							TotalItems: 1,
							TotalPages: 1,
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
				items := data["items"].([]interface{})
				assert.Len(t, items, 1)
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			query:          "",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			query:  "",
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ListByTeamIDFunc = func(ctx context.Context, tid string, page, limit int) (*models.VoiceMemoListResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.GET("/teams/:teamId/voice-memos", setTeamID(*tt.teamID), handler.ListTeamVoiceMemos)
			} else {
				router.GET("/teams/:teamId/voice-memos", handler.ListTeamVoiceMemos)
			}

			req := httptest.NewRequest(http.MethodGet, "/teams/"+teamID.Hex()+"/voice-memos"+tt.query, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestVoiceMemoHandler_GetTeamVoiceMemo(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()
	otherTeamID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		memoID         string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful get team voice memo",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.GetVoiceMemoFunc = func(ctx context.Context, mid primitive.ObjectID) (*models.VoiceMemo, error) {
					return &models.VoiceMemo{
						ID:        memoID,
						UserID:    userID,
						TeamID:    &teamID,
						Title:     "Team Memo",
						Status:    models.StatusReady,
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
				assert.Equal(t, "Team Memo", data["title"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid memo ID format",
			teamID:         &teamID,
			memoID:         "invalid-id",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "voice memo not found",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.GetVoiceMemoFunc = func(ctx context.Context, mid primitive.ObjectID) (*models.VoiceMemo, error) {
					return nil, apperrors.ErrVoiceMemoNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "memo belongs to different team",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.GetVoiceMemoFunc = func(ctx context.Context, mid primitive.ObjectID) (*models.VoiceMemo, error) {
					return &models.VoiceMemo{
						ID:        memoID,
						UserID:    userID,
						TeamID:    &otherTeamID, // Different team
						Title:     "Team Memo",
						Status:    models.StatusReady,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "memo is private - no team ID",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.GetVoiceMemoFunc = func(ctx context.Context, mid primitive.ObjectID) (*models.VoiceMemo, error) {
					return &models.VoiceMemo{
						ID:        memoID,
						UserID:    userID,
						TeamID:    nil, // Private memo
						Title:     "Private Memo",
						Status:    models.StatusReady,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.GetVoiceMemoFunc = func(ctx context.Context, mid primitive.ObjectID) (*models.VoiceMemo, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.GET("/teams/:teamId/voice-memos/:id", setTeamID(*tt.teamID), handler.GetTeamVoiceMemo)
			} else {
				router.GET("/teams/:teamId/voice-memos/:id", handler.GetTeamVoiceMemo)
			}

			req := httptest.NewRequest(http.MethodGet, "/teams/"+teamID.Hex()+"/voice-memos/"+tt.memoID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestVoiceMemoHandler_CreateTeamVoiceMemo(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		userID         string
		body           interface{}
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful create team voice memo",
			teamID: &teamID,
			userID: userID.Hex(),
			body: models.CreateVoiceMemoRequest{
				Title:       "Team Memo",
				Duration:    120,
				FileSize:    1048576,
				AudioFormat: "mp3",
			},
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.CreateTeamVoiceMemoFunc = func(ctx context.Context, uid, tid primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error) {
					return &models.CreateVoiceMemoResponse{
						Memo: models.VoiceMemo{
							ID:          memoID,
							UserID:      uid,
							TeamID:      &tid,
							Title:       req.Title,
							Duration:    req.Duration,
							FileSize:    req.FileSize,
							AudioFormat: req.AudioFormat,
							Status:      models.StatusPendingUpload,
							CreatedAt:   now,
							UpdatedAt:   now,
						},
						UploadURL: "https://s3.amazonaws.com/bucket/voice-memos/test.mp3?signed=true",
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
				assert.Contains(t, data, "uploadUrl")
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			userID:         userID.Hex(),
			body:           models.CreateVoiceMemoRequest{Title: "Test", FileSize: 1000, AudioFormat: "mp3"},
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing user ID",
			teamID:         &teamID,
			userID:         "",
			body:           models.CreateVoiceMemoRequest{Title: "Test", FileSize: 1000, AudioFormat: "mp3"},
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID format",
			teamID:         &teamID,
			userID:         "invalid-id",
			body:           models.CreateVoiceMemoRequest{Title: "Test", FileSize: 1000, AudioFormat: "mp3"},
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid JSON body",
			teamID:         &teamID,
			userID:         userID.Hex(),
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			userID: userID.Hex(),
			body: models.CreateVoiceMemoRequest{
				Title:       "Test",
				FileSize:    1000,
				AudioFormat: "mp3",
			},
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.CreateTeamVoiceMemoFunc = func(ctx context.Context, uid, tid primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			handlers := []gin.HandlerFunc{}
			if tt.teamID != nil {
				handlers = append(handlers, setTeamID(*tt.teamID))
			}
			if tt.userID != "" {
				handlers = append(handlers, setUserID(tt.userID))
			}
			handlers = append(handlers, handler.CreateTeamVoiceMemo)
			router.POST("/teams/:teamId/voice-memos", handlers...)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID.Hex()+"/voice-memos", bytes.NewBuffer(body))
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

func TestVoiceMemoHandler_DeleteTeamVoiceMemo(t *testing.T) {
	teamID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		memoID         string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
	}{
		{
			name:   "successful delete team voice memo",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.DeleteTeamVoiceMemoFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid memo ID format",
			teamID:         &teamID,
			memoID:         "invalid-id",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "voice memo not found",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.DeleteTeamVoiceMemoFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.DeleteTeamVoiceMemoFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.DELETE("/teams/:teamId/voice-memos/:id", setTeamID(*tt.teamID), handler.DeleteTeamVoiceMemo)
			} else {
				router.DELETE("/teams/:teamId/voice-memos/:id", handler.DeleteTeamVoiceMemo)
			}

			req := httptest.NewRequest(http.MethodDelete, "/teams/"+teamID.Hex()+"/voice-memos/"+tt.memoID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestVoiceMemoHandler_ConfirmTeamUpload(t *testing.T) {
	teamID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		memoID         string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful confirm team upload",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmTeamUploadFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "upload confirmed, transcription started", data["message"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid memo ID format",
			teamID:         &teamID,
			memoID:         "invalid-id",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "voice memo not found",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmTeamUploadFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "invalid status transition",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmTeamUploadFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoInvalidStatus
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "transcription queue full",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmTeamUploadFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return apperrors.ErrTranscriptionQueueFull
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.ConfirmTeamUploadFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.POST("/teams/:teamId/voice-memos/:id/confirm-upload", setTeamID(*tt.teamID), handler.ConfirmTeamUpload)
			} else {
				router.POST("/teams/:teamId/voice-memos/:id/confirm-upload", handler.ConfirmTeamUpload)
			}

			req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID.Hex()+"/voice-memos/"+tt.memoID+"/confirm-upload", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestVoiceMemoHandler_RetryTeamTranscription(t *testing.T) {
	teamID := primitive.NewObjectID()
	memoID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		memoID         string
		mockSetup      func(*mocks.MockVoiceMemoService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful retry team transcription",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTeamTranscriptionFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "transcription retry started", data["message"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			memoID:         memoID.Hex(),
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid memo ID format",
			teamID:         &teamID,
			memoID:         "invalid-id",
			mockSetup:      func(m *mocks.MockVoiceMemoService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "voice memo not found",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTeamTranscriptionFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "memo not in failed state",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTeamTranscriptionFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return apperrors.ErrVoiceMemoInvalidStatus
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "transcription queue full",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTeamTranscriptionFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return apperrors.ErrTranscriptionQueueFull
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			memoID: memoID.Hex(),
			mockSetup: func(m *mocks.MockVoiceMemoService) {
				m.RetryTeamTranscriptionFunc = func(ctx context.Context, mid, tid primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockVoiceMemoService{}
			tt.mockSetup(mockService)

			handler := NewVoiceMemoHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.POST("/teams/:teamId/voice-memos/:id/retry-transcription", setTeamID(*tt.teamID), handler.RetryTeamTranscription)
			} else {
				router.POST("/teams/:teamId/voice-memos/:id/retry-transcription", handler.RetryTeamTranscription)
			}

			req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID.Hex()+"/voice-memos/"+tt.memoID+"/retry-transcription", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
