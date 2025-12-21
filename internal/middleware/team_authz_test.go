package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gin-sample/internal/authz"
	"gin-sample/internal/authz/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/mock/gomock"
)

func TestTeamAuthz(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUserID := primitive.NewObjectID()
	validTeamID := primitive.NewObjectID()

	t.Run("allows request when user has permission", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthz := mocks.NewMockAuthorizer(ctrl)
		mockAuthz.EXPECT().
			CanPerform(gomock.Any(), validUserID, validTeamID, authz.ActionTeamView).
			Return(true, nil)
		mockAuthz.EXPECT().
			GetUserRole(gomock.Any(), validUserID, validTeamID).
			Return("member", nil)

		middleware := TeamAuthz(mockAuthz, authz.ActionTeamView)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/teams/"+validTeamID.Hex(), nil)
		c.Params = gin.Params{{Key: "teamId", Value: validTeamID.Hex()}}
		c.Set(UserIDKey, validUserID.Hex())

		var handlerCalled bool
		handler := func(c *gin.Context) {
			handlerCalled = true
			c.Status(http.StatusOK)
		}

		middleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.True(t, handlerCalled)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify context values set
		teamID, exists := GetTeamID(c)
		assert.True(t, exists)
		assert.Equal(t, validTeamID, teamID)
		assert.Equal(t, "member", GetTeamRole(c))
	})

	t.Run("rejects request when user lacks permission", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthz := mocks.NewMockAuthorizer(ctrl)
		mockAuthz.EXPECT().
			CanPerform(gomock.Any(), validUserID, validTeamID, authz.ActionTeamUpdate).
			Return(false, nil)

		middleware := TeamAuthz(mockAuthz, authz.ActionTeamUpdate)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPut, "/teams/"+validTeamID.Hex(), nil)
		c.Params = gin.Params{{Key: "teamId", Value: validTeamID.Hex()}}
		c.Set(UserIDKey, validUserID.Hex())

		middleware(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request when user not authenticated", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthz := mocks.NewMockAuthorizer(ctrl)
		middleware := TeamAuthz(mockAuthz, authz.ActionTeamView)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/teams/"+validTeamID.Hex(), nil)
		c.Params = gin.Params{{Key: "teamId", Value: validTeamID.Hex()}}
		// UserID not set

		middleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request with invalid user ID format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthz := mocks.NewMockAuthorizer(ctrl)
		middleware := TeamAuthz(mockAuthz, authz.ActionTeamView)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/teams/"+validTeamID.Hex(), nil)
		c.Params = gin.Params{{Key: "teamId", Value: validTeamID.Hex()}}
		c.Set(UserIDKey, "invalid-user-id")

		middleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request when team ID missing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthz := mocks.NewMockAuthorizer(ctrl)
		middleware := TeamAuthz(mockAuthz, authz.ActionTeamView)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/teams/", nil)
		// No teamId param
		c.Set(UserIDKey, validUserID.Hex())

		middleware(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request with invalid team ID format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthz := mocks.NewMockAuthorizer(ctrl)
		middleware := TeamAuthz(mockAuthz, authz.ActionTeamView)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/teams/invalid-team-id", nil)
		c.Params = gin.Params{{Key: "teamId", Value: "invalid-team-id"}}
		c.Set(UserIDKey, validUserID.Hex())

		middleware(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("returns internal error when authorizer fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthz := mocks.NewMockAuthorizer(ctrl)
		mockAuthz.EXPECT().
			CanPerform(gomock.Any(), validUserID, validTeamID, authz.ActionTeamView).
			Return(false, errors.New("database error"))

		middleware := TeamAuthz(mockAuthz, authz.ActionTeamView)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/teams/"+validTeamID.Hex(), nil)
		c.Params = gin.Params{{Key: "teamId", Value: validTeamID.Hex()}}
		c.Set(UserIDKey, validUserID.Hex())

		middleware(c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.True(t, c.IsAborted())
	})
}

func TestTeamMember(t *testing.T) {
	gin.SetMode(gin.TestMode)

	validUserID := primitive.NewObjectID()
	validTeamID := primitive.NewObjectID()

	t.Run("uses ActionTeamView for membership check", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockAuthz := mocks.NewMockAuthorizer(ctrl)
		// Verify it calls CanPerform with ActionTeamView
		mockAuthz.EXPECT().
			CanPerform(gomock.Any(), validUserID, validTeamID, authz.ActionTeamView).
			Return(true, nil)
		mockAuthz.EXPECT().
			GetUserRole(gomock.Any(), validUserID, validTeamID).
			Return("member", nil)

		middleware := TeamMember(mockAuthz)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/teams/"+validTeamID.Hex(), nil)
		c.Params = gin.Params{{Key: "teamId", Value: validTeamID.Hex()}}
		c.Set(UserIDKey, validUserID.Hex())

		middleware(c)

		assert.False(t, c.IsAborted())
	})
}

func TestGetTeamID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns team ID when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		expectedTeamID := primitive.NewObjectID()
		c.Set(TeamIDKey, expectedTeamID)

		teamID, exists := GetTeamID(c)

		assert.True(t, exists)
		assert.Equal(t, expectedTeamID, teamID)
	})

	t.Run("returns false when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		teamID, exists := GetTeamID(c)

		assert.False(t, exists)
		assert.Equal(t, primitive.NilObjectID, teamID)
	})
}

func TestGetTeamRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("returns role when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set(TeamRoleKey, "admin")

		role := GetTeamRole(c)

		assert.Equal(t, "admin", role)
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		role := GetTeamRole(c)

		assert.Empty(t, role)
	})
}

func TestContextKeys(t *testing.T) {
	t.Run("TeamIDKey has expected value", func(t *testing.T) {
		assert.Equal(t, "teamID", TeamIDKey)
	})

	t.Run("TeamRoleKey has expected value", func(t *testing.T) {
		assert.Equal(t, "teamRole", TeamRoleKey)
	})
}
