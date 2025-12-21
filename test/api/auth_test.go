//go:build api

package api

import (
	"net/http"
	"testing"

	"gin-sample/internal/models"
	"gin-sample/test/api/testserver"
	"gin-sample/test/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegister tests the POST /api/v1/auth/register endpoint.
func TestRegister(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	t.Run("success - creates new user and returns tokens", func(t *testing.T) {
		req := models.CreateUserRequest{
			Name:     "Test User",
			Email:    "test@example.com",
			Password: "password123",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/register", req)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		// Verify response contains expected fields
		accessToken, ok := resp.Data["accessToken"].(string)
		assert.True(t, ok, "accessToken should be a string")
		assert.NotEmpty(t, accessToken)

		refreshToken, ok := resp.Data["refreshToken"].(string)
		assert.True(t, ok, "refreshToken should be a string")
		assert.NotEmpty(t, refreshToken)

		expiresIn, ok := resp.Data["expiresIn"].(float64)
		assert.True(t, ok, "expiresIn should be a number")
		assert.Greater(t, expiresIn, float64(0))

		user, ok := resp.Data["user"].(map[string]interface{})
		require.True(t, ok, "user should be an object")
		assert.Equal(t, "test@example.com", user["email"])
		assert.Equal(t, "Test User", user["name"])
		assert.NotEmpty(t, user["id"])
	})

	t.Run("error - missing required fields", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		req := map[string]string{
			"email": "test@example.com",
			// missing name and password
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/register", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.False(t, resp.Success)
	})

	t.Run("error - invalid email format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		req := models.CreateUserRequest{
			Name:     "Test User",
			Email:    "invalid-email",
			Password: "password123",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/register", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - password too short", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		req := models.CreateUserRequest{
			Name:     "Test User",
			Email:    "test@example.com",
			Password: "123", // too short, min is 6
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/register", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - name too short", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		req := models.CreateUserRequest{
			Name:     "X", // too short, min is 2
			Email:    "test@example.com",
			Password: "password123",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/register", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - duplicate email", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Register first user
		req := models.CreateUserRequest{
			Name:     "Test User",
			Email:    "duplicate@example.com",
			Password: "password123",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/register", req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Try to register with same email
		req2 := models.CreateUserRequest{
			Name:     "Another User",
			Email:    "duplicate@example.com",
			Password: "password456",
		}

		w2 := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/register", req2)

		assert.Equal(t, http.StatusConflict, w2.Code)

		resp := testutil.ParseAPIResponse(t, w2)
		assert.False(t, resp.Success)
	})
}

// TestLogin tests the POST /api/v1/auth/login endpoint.
func TestLogin(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	// Create a test user first
	authHelper := testserver.NewAuthHelper(testServer)
	authHelper.RegisterUser(t, "Login Test User", "logintest@example.com", "password123")

	t.Run("success - returns tokens for valid credentials", func(t *testing.T) {
		req := models.LoginRequest{
			Email:    "logintest@example.com",
			Password: "password123",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/login", req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		// Verify response contains expected fields
		accessToken, ok := resp.Data["accessToken"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, accessToken)

		refreshToken, ok := resp.Data["refreshToken"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, refreshToken)

		user, ok := resp.Data["user"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "logintest@example.com", user["email"])
	})

	t.Run("error - invalid email", func(t *testing.T) {
		req := models.LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/login", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.False(t, resp.Success)
	})

	t.Run("error - wrong password", func(t *testing.T) {
		req := models.LoginRequest{
			Email:    "logintest@example.com",
			Password: "wrongpassword",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/login", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error - missing email", func(t *testing.T) {
		req := map[string]string{
			"password": "password123",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/login", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - missing password", func(t *testing.T) {
		req := map[string]string{
			"email": "logintest@example.com",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/login", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - invalid email format", func(t *testing.T) {
		req := models.LoginRequest{
			Email:    "not-an-email",
			Password: "password123",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/login", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestRefresh tests the POST /api/v1/auth/refresh endpoint.
func TestRefresh(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	// Create a test user and get tokens
	authHelper := testserver.NewAuthHelper(testServer)
	authHelper.RegisterUser(t, "Refresh Test User", "refreshtest@example.com", "password123")
	loginData := authHelper.Login(t, "refreshtest@example.com", "password123")

	refreshToken, ok := loginData["refreshToken"].(string)
	require.True(t, ok, "refreshToken should be a string")

	t.Run("success - returns new access token", func(t *testing.T) {
		req := models.RefreshRequest{
			RefreshToken: refreshToken,
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/refresh", req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		// Verify response contains new access token
		newAccessToken, ok := resp.Data["accessToken"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, newAccessToken)

		expiresIn, ok := resp.Data["expiresIn"].(float64)
		assert.True(t, ok)
		assert.Greater(t, expiresIn, float64(0))
	})

	t.Run("error - invalid refresh token", func(t *testing.T) {
		req := models.RefreshRequest{
			RefreshToken: "invalid-refresh-token",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/refresh", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.False(t, resp.Success)
	})

	t.Run("error - missing refresh token", func(t *testing.T) {
		req := map[string]string{}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/refresh", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - empty refresh token", func(t *testing.T) {
		req := models.RefreshRequest{
			RefreshToken: "",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/refresh", req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestLogout tests the POST /api/v1/auth/logout endpoint.
func TestLogout(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	// Create a test user and get tokens
	authHelper := testserver.NewAuthHelper(testServer)
	authHelper.RegisterUser(t, "Logout Test User", "logouttest@example.com", "password123")
	loginData := authHelper.Login(t, "logouttest@example.com", "password123")

	accessToken, ok := loginData["accessToken"].(string)
	require.True(t, ok, "accessToken should be a string")

	refreshToken, ok := loginData["refreshToken"].(string)
	require.True(t, ok, "refreshToken should be a string")

	t.Run("success - invalidates refresh token", func(t *testing.T) {
		req := models.LogoutRequest{
			RefreshToken: refreshToken,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/logout", accessToken, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify the refresh token is now invalid
		refreshReq := models.RefreshRequest{
			RefreshToken: refreshToken,
		}
		w2 := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/refresh", refreshReq)
		assert.Equal(t, http.StatusUnauthorized, w2.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Get fresh tokens
		authHelper.RegisterUser(t, "Logout Test User 2", "logouttest2@example.com", "password123")
		loginData2 := authHelper.Login(t, "logouttest2@example.com", "password123")
		refreshToken2, _ := loginData2["refreshToken"].(string)

		req := models.LogoutRequest{
			RefreshToken: refreshToken2,
		}

		// Make request without auth token
		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/logout", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("error - missing refresh token in body", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		authHelper.RegisterUser(t, "Logout Test User 3", "logouttest3@example.com", "password123")
		loginData3 := authHelper.Login(t, "logouttest3@example.com", "password123")
		accessToken3, _ := loginData3["accessToken"].(string)

		req := map[string]string{}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/logout", accessToken3, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - invalid access token", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		authHelper.RegisterUser(t, "Logout Test User 4", "logouttest4@example.com", "password123")
		loginData4 := authHelper.Login(t, "logouttest4@example.com", "password123")
		refreshToken4, _ := loginData4["refreshToken"].(string)

		req := models.LogoutRequest{
			RefreshToken: refreshToken4,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/auth/logout", "invalid-token", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestAuthTokenValidity tests that access tokens work correctly with protected endpoints.
func TestAuthTokenValidity(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	authHelper.RegisterUser(t, "Token Test User", "tokentest@example.com", "password123")
	loginData := authHelper.Login(t, "tokentest@example.com", "password123")

	accessToken, _ := loginData["accessToken"].(string)
	user, _ := loginData["user"].(map[string]interface{})
	userID, _ := user["id"].(string)

	t.Run("valid token allows access to protected endpoint", func(t *testing.T) {
		// Try to access user's own profile
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/"+userID, accessToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid token denies access to protected endpoint", func(t *testing.T) {
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/"+userID, "invalid-token", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing token denies access to protected endpoint", func(t *testing.T) {
		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/"+userID, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
