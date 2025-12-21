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

func TestNewTeamMemberHandler(t *testing.T) {
	mockService := &mocks.MockTeamMemberService{}
	handler := NewTeamMemberHandler(mockService)

	assert.NotNil(t, handler)
	assert.Equal(t, mockService, handler.service)
}

func TestTeamMemberHandler_ListMembers(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	now := time.Now()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		mockSetup      func(*mocks.MockTeamMemberService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful list members",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.ListMembersFunc = func(ctx context.Context, tID primitive.ObjectID) (*models.TeamMemberListResponse, error) {
					return &models.TeamMemberListResponse{
						Items: []models.TeamMemberWithUser{
							{
								ID:       primitive.NewObjectID(),
								TeamID:   teamID,
								UserID:   userID,
								Role:     models.RoleOwner,
								JoinedAt: now,
								User:     &models.UserSummary{ID: userID, Email: "owner@example.com", Name: "Owner"},
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
			mockSetup:      func(m *mocks.MockTeamMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.ListMembersFunc = func(ctx context.Context, tID primitive.ObjectID) (*models.TeamMemberListResponse, error) {
					return nil, errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamMemberService{}
			tt.mockSetup(mockService)

			handler := NewTeamMemberHandler(mockService)

			router := gin.New()
			if tt.teamID != nil {
				router.GET("/teams/:teamId/members", setTeamID(*tt.teamID), handler.ListMembers)
			} else {
				router.GET("/teams/:teamId/members", handler.ListMembers)
			}

			req := httptest.NewRequest(http.MethodGet, "/teams/"+teamID.Hex()+"/members", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamMemberHandler_RemoveMember(t *testing.T) {
	teamID := primitive.NewObjectID()
	requestingUserID := primitive.NewObjectID()
	targetUserID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		userID         string
		targetUserID   string
		mockSetup      func(*mocks.MockTeamMemberService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "successful remove member",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.RemoveMemberFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "member removed successfully", data["message"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			userID:         requestingUserID.Hex(),
			targetUserID:   targetUserID.Hex(),
			mockSetup:      func(m *mocks.MockTeamMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid target user ID format",
			teamID:         &teamID,
			userID:         requestingUserID.Hex(),
			targetUserID:   "invalid-id",
			mockSetup:      func(m *mocks.MockTeamMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "target user not a team member",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.RemoveMemberFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID) error {
					return apperrors.ErrNotTeamMember
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "cannot remove owner",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.RemoveMemberFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID) error {
					return apperrors.ErrCannotRemoveOwner
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "cannot remove self",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: requestingUserID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.RemoveMemberFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID) error {
					return apperrors.ErrCannotRemoveSelf
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "insufficient permissions",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.RemoveMemberFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID) error {
					return apperrors.ErrInsufficientPermissions
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:         "internal server error",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.RemoveMemberFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamMemberService{}
			tt.mockSetup(mockService)

			handler := NewTeamMemberHandler(mockService)

			router := gin.New()
			handlers := []gin.HandlerFunc{}
			if tt.teamID != nil {
				handlers = append(handlers, setTeamID(*tt.teamID))
			}
			if tt.userID != "" {
				handlers = append(handlers, setUserID(tt.userID))
			}
			handlers = append(handlers, handler.RemoveMember)
			router.DELETE("/teams/:teamId/members/:userId", handlers...)

			req := httptest.NewRequest(http.MethodDelete, "/teams/"+teamID.Hex()+"/members/"+tt.targetUserID, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestTeamMemberHandler_UpdateRole(t *testing.T) {
	teamID := primitive.NewObjectID()
	requestingUserID := primitive.NewObjectID()
	targetUserID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		userID         string
		targetUserID   string
		body           interface{}
		mockSetup      func(*mocks.MockTeamMemberService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "successful update role",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			body:         models.UpdateRoleRequest{Role: models.RoleAdmin},
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.UpdateRoleFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID, newRole string) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "role updated successfully", data["message"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			userID:         requestingUserID.Hex(),
			targetUserID:   targetUserID.Hex(),
			body:           models.UpdateRoleRequest{Role: models.RoleAdmin},
			mockSetup:      func(m *mocks.MockTeamMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid target user ID format",
			teamID:         &teamID,
			userID:         requestingUserID.Hex(),
			targetUserID:   "invalid-id",
			body:           models.UpdateRoleRequest{Role: models.RoleAdmin},
			mockSetup:      func(m *mocks.MockTeamMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid JSON body",
			teamID:         &teamID,
			userID:         requestingUserID.Hex(),
			targetUserID:   targetUserID.Hex(),
			body:           "invalid json",
			mockSetup:      func(m *mocks.MockTeamMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "target user not a team member",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			body:         models.UpdateRoleRequest{Role: models.RoleAdmin},
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.UpdateRoleFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID, newRole string) error {
					return apperrors.ErrNotTeamMember
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:         "cannot change owner role",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			body:         models.UpdateRoleRequest{Role: models.RoleAdmin},
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.UpdateRoleFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID, newRole string) error {
					return apperrors.ErrCannotChangeOwnerRole
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid role",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			body:         models.UpdateRoleRequest{Role: models.RoleAdmin},
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.UpdateRoleFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID, newRole string) error {
					return apperrors.ErrInvalidRole
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "insufficient permissions",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			body:         models.UpdateRoleRequest{Role: models.RoleAdmin},
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.UpdateRoleFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID, newRole string) error {
					return apperrors.ErrInsufficientPermissions
				}
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:         "internal server error",
			teamID:       &teamID,
			userID:       requestingUserID.Hex(),
			targetUserID: targetUserID.Hex(),
			body:         models.UpdateRoleRequest{Role: models.RoleAdmin},
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.UpdateRoleFunc = func(ctx context.Context, tID, target, requester primitive.ObjectID, newRole string) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamMemberService{}
			tt.mockSetup(mockService)

			handler := NewTeamMemberHandler(mockService)

			router := gin.New()
			handlers := []gin.HandlerFunc{}
			if tt.teamID != nil {
				handlers = append(handlers, setTeamID(*tt.teamID))
			}
			if tt.userID != "" {
				handlers = append(handlers, setUserID(tt.userID))
			}
			handlers = append(handlers, handler.UpdateRole)
			router.PUT("/teams/:teamId/members/:userId/role", handlers...)

			var body []byte
			switch v := tt.body.(type) {
			case string:
				body = []byte(v)
			default:
				body, _ = json.Marshal(v)
			}

			req := httptest.NewRequest(http.MethodPut, "/teams/"+teamID.Hex()+"/members/"+tt.targetUserID+"/role", bytes.NewBuffer(body))
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

func TestTeamMemberHandler_LeaveTeam(t *testing.T) {
	teamID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	tests := []struct {
		name           string
		teamID         *primitive.ObjectID
		userID         string
		mockSetup      func(*mocks.MockTeamMemberService)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "successful leave team",
			teamID: &teamID,
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.LeaveTeamFunc = func(ctx context.Context, tID, uID primitive.ObjectID) error {
					return nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				data := resp["data"].(map[string]interface{})
				assert.Equal(t, "left team successfully", data["message"])
			},
		},
		{
			name:           "missing team ID in context",
			teamID:         nil,
			userID:         userID.Hex(),
			mockSetup:      func(m *mocks.MockTeamMemberService) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "not a team member",
			teamID: &teamID,
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.LeaveTeamFunc = func(ctx context.Context, tID, uID primitive.ObjectID) error {
					return apperrors.ErrNotTeamMember
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:   "owner cannot leave",
			teamID: &teamID,
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.LeaveTeamFunc = func(ctx context.Context, tID, uID primitive.ObjectID) error {
					return apperrors.ErrOwnerCannotLeave
				}
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "internal server error",
			teamID: &teamID,
			userID: userID.Hex(),
			mockSetup: func(m *mocks.MockTeamMemberService) {
				m.LeaveTeamFunc = func(ctx context.Context, tID, uID primitive.ObjectID) error {
					return errors.New("database error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mocks.MockTeamMemberService{}
			tt.mockSetup(mockService)

			handler := NewTeamMemberHandler(mockService)

			router := gin.New()
			handlers := []gin.HandlerFunc{}
			if tt.teamID != nil {
				handlers = append(handlers, setTeamID(*tt.teamID))
			}
			if tt.userID != "" {
				handlers = append(handlers, func(c *gin.Context) {
					c.Set(middleware.UserIDKey, tt.userID)
					c.Next()
				})
			}
			handlers = append(handlers, handler.LeaveTeam)
			router.POST("/teams/:teamId/leave", handlers...)

			req := httptest.NewRequest(http.MethodPost, "/teams/"+teamID.Hex()+"/leave", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
