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
	"gin-sample/internal/middleware"
	"gin-sample/internal/models"
	"gin-sample/internal/service/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewTeamHandler(t *testing.T) {
	mockService := &mocks.MockTeamService{}
	handler := NewTeamHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

// setUserID is a helper middleware to set user ID in context
func setUserID(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.UserIDKey, userID)
		c.Next()
	}
}

// setTeamID is a helper middleware to set team ID in context
func setTeamID(teamID primitive.ObjectID) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.TeamIDKey, teamID)
		c.Next()
	}
}

func TestTeamHandler_CreateTeam(t *testing.T) {
	userID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		userID         string
		body           interface{}
		mockSetup      func(*mocks.MockTeamService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful create team",
			userID: userID.Hex(),
			body: models.CreateTeamRequest{
				Name:        "Test Team",
				Slug:        "test-team",
				Description: "A test team",
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.CreateTeamFunc = func(ctx context.Context, uID primitive.ObjectID, req *models.CreateTeamRequest) (*models.Team, error) {
					return &models.Team{
						ID:          teamID,
						Name:        req.Name,
						Slug:        req.Slug,
						Description: req.Description,
						OwnerID:     uID,
						Seats:       5,
						CreatedAt:   now,
						UpdatedAt:   now,
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
				assert.Equal(t, "Test Team", data["name"])
			},
		},
		{
			name:           "missing user ID in context",
			userID:         "",
			body:           models.CreateTeamRequest{Name: "Test", Slug: "test"},
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-id",
			body:           models.CreateTeamRequest{Name: "Test", Slug: "test"},
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid JSON body",
			userID:         userID.Hex(),
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "team limit reached",
			userID: userID.Hex(),
			body: models.CreateTeamRequest{
				Name: "Test Team",
				Slug: "test-team",
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.CreateTeamFunc = func(ctx context.Context, uID primitive.ObjectID, req *models.CreateTeamRequest) (*models.Team, error) {
					return nil, apperrors.ErrTeamLimitReached
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "slug already taken",
			userID: userID.Hex(),
			body: models.CreateTeamRequest{
				Name: "Test Team",
				Slug: "existing-slug",
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.CreateTeamFunc = func(ctx context.Context, uID primitive.ObjectID, req *models.CreateTeamRequest) (*models.Team, error) {
					return nil, apperrors.ErrTeamSlugTaken
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			body: models.CreateTeamRequest{
				Name: "Test Team",
				Slug: "test-team",
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.CreateTeamFunc = func(ctx context.Context, uID primitive.ObjectID, req *models.CreateTeamRequest) (*models.Team, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamService{}
			tt.mockSetup(mockService)

			handler := NewTeamHandler(mockService)

			router := gin.New()
			if tt.userID != "" {
				router.POST("/teams", setUserID(tt.userID), handler.CreateTeam)
			} else {
				router.POST("/teams", handler.CreateTeam)
			}

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/teams", bytes.NewBuffer(body))
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

func TestTeamHandler_ListTeams(t *testing.T) {
	userID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		userID         string
		queryParams    string
		mockSetup      func(*mocks.MockTeamService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful list teams",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamService) {
				m.ListTeamsFunc = func(ctx context.Context, uID primitive.ObjectID, page, limit int) (*models.TeamListResponse, error) {
					return &models.TeamListResponse{
						Items: []models.Team{
							{ID: teamID, Name: "Team 1", Slug: "team-1", OwnerID: uID, CreatedAt: now, UpdatedAt: now},
						},
						Pagination: models.Pagination{Page: 1, Limit: 10, TotalItems: 1, TotalPages: 1},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				items := data["items"].([]interface{})
				assert.Len(t, items, 1)
			},
		},
		{
			name:        "with pagination",
			userID:      userID.Hex(),
			queryParams: "?page=2&limit=5",
			mockSetup: func(m *mocks.MockTeamService) {
				m.ListTeamsFunc = func(ctx context.Context, uID primitive.ObjectID, page, limit int) (*models.TeamListResponse, error) {
					assert.Equal(t, 2, page)
					assert.Equal(t, 5, limit)
					return &models.TeamListResponse{
						Items:      []models.Team{},
						Pagination: models.Pagination{Page: 2, Limit: 5, TotalItems: 0, TotalPages: 0},
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing user ID",
			userID:         "",
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid user ID format",
			userID:         "invalid-id",
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "internal server error",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamService) {
				m.ListTeamsFunc = func(ctx context.Context, uID primitive.ObjectID, page, limit int) (*models.TeamListResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamService{}
			tt.mockSetup(mockService)

			handler := NewTeamHandler(mockService)

			router := gin.New()
			if tt.userID != "" {
				router.GET("/teams", setUserID(tt.userID), handler.ListTeams)
			} else {
				router.GET("/teams", handler.ListTeams)
			}

			req := httptest.NewRequest(http.MethodGet, "/teams"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamHandler_GetTeam(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		mockSetup      func(*mocks.MockTeamService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful get team",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamService) {
				m.GetTeamFunc = func(ctx context.Context, tID primitive.ObjectID) (*models.Team, error) {
					return &models.Team{
						ID:        teamID,
						Name:      "Test Team",
						Slug:      "test-team",
						OwnerID:   userID,
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
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "Test Team", data["name"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "team not found",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamService) {
				m.GetTeamFunc = func(ctx context.Context, tID primitive.ObjectID) (*models.Team, error) {
					return nil, apperrors.ErrTeamNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamService) {
				m.GetTeamFunc = func(ctx context.Context, tID primitive.ObjectID) (*models.Team, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamService{}
			tt.mockSetup(mockService)

			handler := NewTeamHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.GET("/teams/:teamId", setTeamID(*tt.teamID), handler.GetTeam)
			} else {
				router.GET("/teams/:teamId", handler.GetTeam)
			}

			req := httptest.NewRequest(http.MethodGet, "/teams/"+teamID.Hex(), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamHandler_UpdateTeam(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	now := time.Now()
	newName := "Updated Team"
	newSlug := "updated-team"

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		body           interface{}
		mockSetup      func(*mocks.MockTeamService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful update team",
			teamID: &teamID,
			body: models.UpdateTeamRequest{
				Name: &newName,
				Slug: &newSlug,
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.UpdateTeamFunc = func(ctx context.Context, tID primitive.ObjectID, req *models.UpdateTeamRequest) (*models.Team, error) {
					return &models.Team{
						ID:        teamID,
						Name:      *req.Name,
						Slug:      *req.Slug,
						OwnerID:   userID,
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
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, newName, data["name"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			body:           models.UpdateTeamRequest{Name: &newName},
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON body",
			teamID:         &teamID,
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "team not found",
			teamID: &teamID,
			body:   models.UpdateTeamRequest{Name: &newName},
			mockSetup: func(m *mocks.MockTeamService) {
				m.UpdateTeamFunc = func(ctx context.Context, tID primitive.ObjectID, req *models.UpdateTeamRequest) (*models.Team, error) {
					return nil, apperrors.ErrTeamNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "slug already taken",
			teamID: &teamID,
			body:   models.UpdateTeamRequest{Slug: &newSlug},
			mockSetup: func(m *mocks.MockTeamService) {
				m.UpdateTeamFunc = func(ctx context.Context, tID primitive.ObjectID, req *models.UpdateTeamRequest) (*models.Team, error) {
					return nil, apperrors.ErrTeamSlugTaken
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			body:   models.UpdateTeamRequest{Name: &newName},
			mockSetup: func(m *mocks.MockTeamService) {
				m.UpdateTeamFunc = func(ctx context.Context, tID primitive.ObjectID, req *models.UpdateTeamRequest) (*models.Team, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamService{}
			tt.mockSetup(mockService)

			handler := NewTeamHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.PUT("/teams/:teamId", setTeamID(*tt.teamID), handler.UpdateTeam)
			} else {
				router.PUT("/teams/:teamId", handler.UpdateTeam)
			}

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPut, "/teams/"+teamID.Hex(), bytes.NewBuffer(body))
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

func TestTeamHandler_DeleteTeam(t *testing.T) {
	teamID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		mockSetup      func(*mocks.MockTeamService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful delete team",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamService) {
				m.DeleteTeamFunc = func(ctx context.Context, tID primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "team deleted successfully", data["message"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "team not found",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamService) {
				m.DeleteTeamFunc = func(ctx context.Context, tID primitive.ObjectID) error {
					return apperrors.ErrTeamNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamService) {
				m.DeleteTeamFunc = func(ctx context.Context, tID primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamService{}
			tt.mockSetup(mockService)

			handler := NewTeamHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.DELETE("/teams/:teamId", setTeamID(*tt.teamID), handler.DeleteTeam)
			} else {
				router.DELETE("/teams/:teamId", handler.DeleteTeam)
			}

			req := httptest.NewRequest(http.MethodDelete, "/teams/"+teamID.Hex(), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamHandler_TransferOwnership(t *testing.T) {
	teamID := primitive.NewObjectID()
	currentOwnerID := primitive.NewObjectID()
	newOwnerID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		userID         string
		body           interface{}
		mockSetup      func(*mocks.MockTeamService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful transfer ownership",
			teamID: &teamID,
			userID: currentOwnerID.Hex(),
			body: models.TransferOwnershipRequest{
				NewOwnerID: newOwnerID.Hex(),
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.TransferOwnershipFunc = func(ctx context.Context, tID, curOwner, newOwner primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "ownership transferred successfully", data["message"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			userID:         currentOwnerID.Hex(),
			body:           models.TransferOwnershipRequest{NewOwnerID: newOwnerID.Hex()},
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid user ID format",
			teamID:         &teamID,
			userID:         "invalid-id",
			body:           models.TransferOwnershipRequest{NewOwnerID: newOwnerID.Hex()},
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid JSON body",
			teamID:         &teamID,
			userID:         currentOwnerID.Hex(),
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "invalid new owner ID format",
			teamID: &teamID,
			userID: currentOwnerID.Hex(),
			body: models.TransferOwnershipRequest{
				NewOwnerID: "invalid-id",
			},
			mockSetup:      func(m *mocks.MockTeamService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "new owner not a team member",
			teamID: &teamID,
			userID: currentOwnerID.Hex(),
			body: models.TransferOwnershipRequest{
				NewOwnerID: newOwnerID.Hex(),
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.TransferOwnershipFunc = func(ctx context.Context, tID, curOwner, newOwner primitive.ObjectID) error {
					return apperrors.ErrNotTeamMember
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "team not found",
			teamID: &teamID,
			userID: currentOwnerID.Hex(),
			body: models.TransferOwnershipRequest{
				NewOwnerID: newOwnerID.Hex(),
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.TransferOwnershipFunc = func(ctx context.Context, tID, curOwner, newOwner primitive.ObjectID) error {
					return apperrors.ErrTeamNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			userID: currentOwnerID.Hex(),
			body: models.TransferOwnershipRequest{
				NewOwnerID: newOwnerID.Hex(),
			},
			mockSetup: func(m *mocks.MockTeamService) {
				m.TransferOwnershipFunc = func(ctx context.Context, tID, curOwner, newOwner primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamService{}
			tt.mockSetup(mockService)

			handler := NewTeamHandler(mockService)

			router := gin.New()
			handlers := []gin.HandlerFunc{}
			if tt.teamID != nil {
				handlers = append(handlers, setTeamID(*tt.teamID))
			}
			if tt.userID != "" {
				handlers = append(handlers, setUserID(tt.userID))
			}
			handlers = append(handlers, handler.TransferOwnership)
			router.POST("/teams/:teamId/transfer", handlers...)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID.Hex()+"/transfer", bytes.NewBuffer(body))
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
