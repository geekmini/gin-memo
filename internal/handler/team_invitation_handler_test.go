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

func TestNewTeamInvitationHandler(t *testing.T) {
	mockInvitationService := &mocks.MockTeamInvitationService{}
	mockUserService := &mocks.MockUserService{}
	handler := NewTeamInvitationHandler(mockInvitationService, mockUserService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockInvitationService, handler.invitationService)
	assert.Equal(t, mockUserService, handler.userService)
}

func TestTeamInvitationHandler_CreateInvitation(t *testing.T) {
	teamID := primitive.NewObjectID()
	inviterID := primitive.NewObjectID()
	invitationID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		userID         string
		body           interface{}
		mockSetup      func(*mocks.MockTeamInvitationService, *mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful create invitation",
			teamID: &teamID,
			userID: inviterID.Hex(),
			body: models.CreateInvitationRequest{
				Email: "newuser@example.com",
				Role:  models.RoleMember,
			},
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.CreateInvitationFunc = func(ctx context.Context, tID, iID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error) {
					return &models.TeamInvitation{
						ID:        invitationID,
						TeamID:    tID,
						Email:     req.Email,
						InvitedBy: iID,
						Role:      req.Role,
						ExpiresAt: now.Add(7 * 24 * time.Hour),
						CreatedAt: now,
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
				assert.Equal(t, "newuser@example.com", data["email"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			userID:         inviterID.Hex(),
			body:           models.CreateInvitationRequest{Email: "test@example.com", Role: models.RoleMember},
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON body",
			teamID:         &teamID,
			userID:         inviterID.Hex(),
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "user already a member",
			teamID: &teamID,
			userID: inviterID.Hex(),
			body: models.CreateInvitationRequest{
				Email: "existing@example.com",
				Role:  models.RoleMember,
			},
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.CreateInvitationFunc = func(ctx context.Context, tID, iID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error) {
					return nil, apperrors.ErrAlreadyMember
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "pending invitation exists",
			teamID: &teamID,
			userID: inviterID.Hex(),
			body: models.CreateInvitationRequest{
				Email: "pending@example.com",
				Role:  models.RoleMember,
			},
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.CreateInvitationFunc = func(ctx context.Context, tID, iID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error) {
					return nil, apperrors.ErrPendingInvitation
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name:   "seats exceeded",
			teamID: &teamID,
			userID: inviterID.Hex(),
			body: models.CreateInvitationRequest{
				Email: "test@example.com",
				Role:  models.RoleMember,
			},
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.CreateInvitationFunc = func(ctx context.Context, tID, iID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error) {
					return nil, apperrors.ErrSeatsExceeded
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			userID: inviterID.Hex(),
			body: models.CreateInvitationRequest{
				Email: "test@example.com",
				Role:  models.RoleMember,
			},
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.CreateInvitationFunc = func(ctx context.Context, tID, iID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInvitationService := &mocks.MockTeamInvitationService{}
			mockUserService := &mocks.MockUserService{}
			tt.mockSetup(mockInvitationService, mockUserService)

			handler := NewTeamInvitationHandler(mockInvitationService, mockUserService)

			router := gin.New()
			handlers := []gin.HandlerFunc{}
			if tt.teamID != nil {
				handlers = append(handlers, setTeamID(*tt.teamID))
			}
			if tt.userID != "" {
				handlers = append(handlers, setUserID(tt.userID))
			}
			handlers = append(handlers, handler.CreateInvitation)
			router.POST("/teams/:teamId/invitations", handlers...)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID.Hex()+"/invitations", bytes.NewBuffer(body))
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

func TestTeamInvitationHandler_ListTeamInvitations(t *testing.T) {
	teamID := primitive.NewObjectID()
	inviterID := primitive.NewObjectID()
	invitationID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		mockSetup      func(*mocks.MockTeamInvitationService, *mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful list invitations",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.ListTeamInvitationsFunc = func(ctx context.Context, tID primitive.ObjectID) (*models.InvitationListResponse, error) {
					return &models.InvitationListResponse{
						Items: []models.TeamInvitation{
							{
								ID:        invitationID,
								TeamID:    tID,
								Email:     "invited@example.com",
								InvitedBy: inviterID,
								Role:      models.RoleMember,
								ExpiresAt: now.Add(7 * 24 * time.Hour),
								CreatedAt: now,
							},
						},
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
			name:           "missing team ID in context",
			teamID:         nil,
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.ListTeamInvitationsFunc = func(ctx context.Context, tID primitive.ObjectID) (*models.InvitationListResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInvitationService := &mocks.MockTeamInvitationService{}
			mockUserService := &mocks.MockUserService{}
			tt.mockSetup(mockInvitationService, mockUserService)

			handler := NewTeamInvitationHandler(mockInvitationService, mockUserService)

			router := gin.New()
			if tt.teamID != nil {
				router.GET("/teams/:teamId/invitations", setTeamID(*tt.teamID), handler.ListTeamInvitations)
			} else {
				router.GET("/teams/:teamId/invitations", handler.ListTeamInvitations)
			}

			req := httptest.NewRequest(http.MethodGet, "/teams/"+teamID.Hex()+"/invitations", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamInvitationHandler_CancelInvitation(t *testing.T) {
	teamID := primitive.NewObjectID()
	invitationID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		invitationID   string
		mockSetup      func(*mocks.MockTeamInvitationService, *mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "successful cancel invitation",
			teamID:       &teamID,
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.CancelInvitationFunc = func(ctx context.Context, iID, tID primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "invitation cancelled", data["message"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			invitationID:   invitationID.Hex(),
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid invitation ID format",
			teamID:         &teamID,
			invitationID:   "invalid-id",
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "invitation not found",
			teamID:       &teamID,
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.CancelInvitationFunc = func(ctx context.Context, iID, tID primitive.ObjectID) error {
					return apperrors.ErrInvitationNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "internal server error",
			teamID:       &teamID,
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				m.CancelInvitationFunc = func(ctx context.Context, iID, tID primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInvitationService := &mocks.MockTeamInvitationService{}
			mockUserService := &mocks.MockUserService{}
			tt.mockSetup(mockInvitationService, mockUserService)

			handler := NewTeamInvitationHandler(mockInvitationService, mockUserService)

			router := gin.New()
			if tt.teamID != nil {
				router.DELETE("/teams/:teamId/invitations/:id", setTeamID(*tt.teamID), handler.CancelInvitation)
			} else {
				router.DELETE("/teams/:teamId/invitations/:id", handler.CancelInvitation)
			}

			req := httptest.NewRequest(http.MethodDelete, "/teams/"+teamID.Hex()+"/invitations/"+tt.invitationID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamInvitationHandler_ListMyInvitations(t *testing.T) {
	userID := primitive.NewObjectID()
	invitationID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		userID         string
		mockSetup      func(*mocks.MockTeamInvitationService, *mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful list my invitations",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{
						ID:    userID,
						Email: "user@example.com",
						Name:  "Test User",
					}, nil
				}
				m.ListMyInvitationsFunc = func(ctx context.Context, email string) (*models.MyInvitationListResponse, error) {
					return &models.MyInvitationListResponse{
						Items: []models.TeamInvitationWithDetails{
							{
								ID:        invitationID,
								Team:      &models.TeamSummary{ID: primitive.NewObjectID(), Name: "Team A", Slug: "team-a"},
								InvitedBy: &models.UserSummary{ID: primitive.NewObjectID(), Email: "inviter@example.com", Name: "Inviter"},
								Role:      models.RoleMember,
								ExpiresAt: now.Add(7 * 24 * time.Hour),
								CreatedAt: now,
							},
						},
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
			name:           "missing user ID",
			userID:         "",
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "user not found",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return nil, errors.New("user not found")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "internal server error on list invitations",
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com"}, nil
				}
				m.ListMyInvitationsFunc = func(ctx context.Context, email string) (*models.MyInvitationListResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInvitationService := &mocks.MockTeamInvitationService{}
			mockUserService := &mocks.MockUserService{}
			tt.mockSetup(mockInvitationService, mockUserService)

			handler := NewTeamInvitationHandler(mockInvitationService, mockUserService)

			router := gin.New()
			if tt.userID != "" {
				router.GET("/invitations", setUserID(tt.userID), handler.ListMyInvitations)
			} else {
				router.GET("/invitations", handler.ListMyInvitations)
			}

			req := httptest.NewRequest(http.MethodGet, "/invitations", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamInvitationHandler_AcceptInvitation(t *testing.T) {
	userID := primitive.NewObjectID()
	invitationID := primitive.NewObjectID()
	teamID := primitive.NewObjectID()

	tests := []struct {
		name           string
		userID         string
		invitationID   string
		mockSetup      func(*mocks.MockTeamInvitationService, *mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "successful accept invitation",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com", Name: "Test User"}, nil
				}
				m.AcceptInvitationFunc = func(ctx context.Context, iID, uID primitive.ObjectID, email string) (*models.AcceptInvitationResponse, error) {
					return &models.AcceptInvitationResponse{
						Message: "invitation accepted",
						TeamID:  teamID.Hex(),
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "invitation accepted", data["message"])
			},
		},
		{
			name:           "missing user ID",
			userID:         "",
			invitationID:   invitationID.Hex(),
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid invitation ID format",
			userID:         userID.Hex(),
			invitationID:   "invalid-id",
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "user not found",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return nil, errors.New("user not found")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "invitation not found",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com"}, nil
				}
				m.AcceptInvitationFunc = func(ctx context.Context, iID, uID primitive.ObjectID, email string) (*models.AcceptInvitationResponse, error) {
					return nil, apperrors.ErrInvitationNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "email mismatch",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "wrong@example.com"}, nil
				}
				m.AcceptInvitationFunc = func(ctx context.Context, iID, uID primitive.ObjectID, email string) (*models.AcceptInvitationResponse, error) {
					return nil, apperrors.ErrInvitationEmailMismatch
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:         "invitation expired",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com"}, nil
				}
				m.AcceptInvitationFunc = func(ctx context.Context, iID, uID primitive.ObjectID, email string) (*models.AcceptInvitationResponse, error) {
					return nil, apperrors.ErrInvitationExpired
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "seats exceeded",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com"}, nil
				}
				m.AcceptInvitationFunc = func(ctx context.Context, iID, uID primitive.ObjectID, email string) (*models.AcceptInvitationResponse, error) {
					return nil, apperrors.ErrSeatsExceeded
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:         "internal server error",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com"}, nil
				}
				m.AcceptInvitationFunc = func(ctx context.Context, iID, uID primitive.ObjectID, email string) (*models.AcceptInvitationResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInvitationService := &mocks.MockTeamInvitationService{}
			mockUserService := &mocks.MockUserService{}
			tt.mockSetup(mockInvitationService, mockUserService)

			handler := NewTeamInvitationHandler(mockInvitationService, mockUserService)

			router := gin.New()
			if tt.userID != "" {
				router.POST("/invitations/:id/accept", setUserID(tt.userID), handler.AcceptInvitation)
			} else {
				router.POST("/invitations/:id/accept", handler.AcceptInvitation)
			}

			req := httptest.NewRequest(http.MethodPost, "/invitations/"+tt.invitationID+"/accept", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamInvitationHandler_DeclineInvitation(t *testing.T) {
	userID := primitive.NewObjectID()
	invitationID := primitive.NewObjectID()

	tests := []struct {
		name           string
		userID         string
		invitationID   string
		mockSetup      func(*mocks.MockTeamInvitationService, *mocks.MockUserService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "successful decline invitation",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com", Name: "Test User"}, nil
				}
				m.DeclineInvitationFunc = func(ctx context.Context, iID primitive.ObjectID, email string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "invitation declined", data["message"])
			},
		},
		{
			name:           "missing user ID",
			userID:         "",
			invitationID:   invitationID.Hex(),
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid invitation ID format",
			userID:         userID.Hex(),
			invitationID:   "invalid-id",
			mockSetup:      func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "user not found",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return nil, errors.New("user not found")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:         "invitation not found",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com"}, nil
				}
				m.DeclineInvitationFunc = func(ctx context.Context, iID primitive.ObjectID, email string) error {
					return apperrors.ErrInvitationNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "email mismatch",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "wrong@example.com"}, nil
				}
				m.DeclineInvitationFunc = func(ctx context.Context, iID primitive.ObjectID, email string) error {
					return apperrors.ErrInvitationEmailMismatch
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:         "internal server error",
			userID:       userID.Hex(),
			invitationID: invitationID.Hex(),
			mockSetup: func(m *mocks.MockTeamInvitationService, u *mocks.MockUserService) {
				u.GetUserFunc = func(ctx context.Context, id string) (*models.User, error) {
					return &models.User{ID: userID, Email: "user@example.com"}, nil
				}
				m.DeclineInvitationFunc = func(ctx context.Context, iID primitive.ObjectID, email string) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInvitationService := &mocks.MockTeamInvitationService{}
			mockUserService := &mocks.MockUserService{}
			tt.mockSetup(mockInvitationService, mockUserService)

			handler := NewTeamInvitationHandler(mockInvitationService, mockUserService)

			router := gin.New()
			if tt.userID != "" {
				router.POST("/invitations/:id/decline", setUserID(tt.userID), handler.DeclineInvitation)
			} else {
				router.POST("/invitations/:id/decline", handler.DeclineInvitation)
			}

			req := httptest.NewRequest(http.MethodPost, "/invitations/"+tt.invitationID+"/decline", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
