package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gin-sample/pkg/auth"
	"gin-sample/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuth(t *testing.T) {
	jwtManager := auth.NewJWTManager("testsecret", 15*time.Minute)
	authMiddleware := Auth(jwtManager)

	t.Run("allows request with valid token", func(t *testing.T) {
		userID := "507f1f77bcf86cd799439011"
		token, _ := jwtManager.GenerateToken(userID)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		var capturedUserID string
		handler := func(c *gin.Context) {
			capturedUserID = GetUserID(c)
			c.Status(http.StatusOK)
		}

		authMiddleware(c)
		if !c.IsAborted() {
			handler(c)
		}

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, userID, capturedUserID)
	})

	t.Run("rejects request without authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		authMiddleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request with invalid header format - no Bearer prefix", func(t *testing.T) {
		token, _ := jwtManager.GenerateToken("user123")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", token) // Missing "Bearer " prefix

		authMiddleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request with invalid header format - wrong prefix", func(t *testing.T) {
		token, _ := jwtManager.GenerateToken("user123")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Basic "+token)

		authMiddleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request with invalid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer invalid.token.here")

		authMiddleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request with expired token", func(t *testing.T) {
		shortManager := auth.NewJWTManager("testsecret", 1*time.Millisecond)
		token, _ := shortManager.GenerateToken("user123")
		time.Sleep(10 * time.Millisecond)

		shortAuthMiddleware := Auth(shortManager)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		shortAuthMiddleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects request with token signed by different secret", func(t *testing.T) {
		otherManager := auth.NewJWTManager("differentsecret", 15*time.Minute)
		token, _ := otherManager.GenerateToken("user123")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		authMiddleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})

	t.Run("rejects empty bearer token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request.Header.Set("Authorization", "Bearer ")

		authMiddleware(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.True(t, c.IsAborted())
	})
}

func TestGetUserID(t *testing.T) {
	t.Run("returns user ID when set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		expectedUserID := "507f1f77bcf86cd799439011"
		c.Set(UserIDKey, expectedUserID)

		userID := GetUserID(c)

		assert.Equal(t, expectedUserID, userID)
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		userID := GetUserID(c)

		assert.Empty(t, userID)
	})
}

func TestAuthMiddleware_Integration(t *testing.T) {
	jwtManager := auth.NewJWTManager("testsecret", 15*time.Minute)

	router := gin.New()
	router.Use(Auth(jwtManager))
	router.GET("/protected", func(c *gin.Context) {
		userID := GetUserID(c)
		response.Success(c, gin.H{"userId": userID})
	})

	t.Run("full request with valid token", func(t *testing.T) {
		userID := "507f1f77bcf86cd799439011"
		token, _ := jwtManager.GenerateToken(userID)

		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), userID)
	})

	t.Run("full request without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestUserIDKey(t *testing.T) {
	t.Run("constant has expected value", func(t *testing.T) {
		require.Equal(t, "userID", UserIDKey)
	})
}
