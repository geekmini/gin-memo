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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestGetUser tests the GET /api/v1/users/:id endpoint.
func TestGetUser(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	userData, accessToken := authHelper.CreateAuthenticatedUser(t, "Get User Test", "getuser@example.com", "password123")
	userID := testserver.GetIDFromResponse(t, userData)

	t.Run("success - get own profile", func(t *testing.T) {
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/"+userID, accessToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		assert.Equal(t, "getuser@example.com", resp.Data["email"])
		assert.Equal(t, "Get User Test", resp.Data["name"])
		assert.Equal(t, userID, resp.Data["id"])
	})

	t.Run("success - get another user's profile", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create two users
		user1Data, token1 := authHelper.CreateAuthenticatedUser(t, "User One", "user1@example.com", "password123")
		user1ID := testserver.GetIDFromResponse(t, user1Data)

		user2Data, _ := authHelper.CreateAuthenticatedUser(t, "User Two", "user2@example.com", "password123")
		user2ID := testserver.GetIDFromResponse(t, user2Data)

		// User 1 gets User 2's profile
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/"+user2ID, token1, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "User Two", resp.Data["name"])
		assert.NotEqual(t, user1ID, resp.Data["id"])
	})

	t.Run("error - user not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/"+nonExistentID, token, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.False(t, resp.Success)
	})

	t.Run("error - invalid user ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser2@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/invalid-id", token, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData2, _ := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser3@example.com", "password123")
		userID2 := testserver.GetIDFromResponse(t, userData2)

		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/"+userID2, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestGetAllUsers tests the GET /api/v1/users endpoint.
func TestGetAllUsers(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)

	t.Run("success - returns empty list when no users", func(t *testing.T) {
		// Create a user to get a token (this user will be in the list)
		_, token := authHelper.CreateAuthenticatedUser(t, "First User", "firstuser@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users", token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIListResponse(t, w)
		assert.True(t, resp.Success)

		// Should have at least the user we created (data is the array directly)
		assert.GreaterOrEqual(t, len(resp.Data), 1)
	})

	t.Run("success - returns multiple users", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create multiple users
		_, token := authHelper.CreateAuthenticatedUser(t, "User A", "usera@example.com", "password123")
		authHelper.RegisterUser(t, "User B", "userb@example.com", "password123")
		authHelper.RegisterUser(t, "User C", "userc@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users", token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIListResponse(t, w)
		assert.True(t, resp.Success)
		// Should have all 3 users
		assert.GreaterOrEqual(t, len(resp.Data), 3)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/users", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestUpdateUser tests the PUT /api/v1/users/:id endpoint.
func TestUpdateUser(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)

	t.Run("success - update name", func(t *testing.T) {
		userData, token := authHelper.CreateAuthenticatedUser(t, "Original Name", "updatename@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		newName := "Updated Name"
		req := models.UpdateUserRequest{
			Name: &newName,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/users/"+userID, token, req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "Updated Name", resp.Data["name"])
		assert.Equal(t, "updatename@example.com", resp.Data["email"]) // Email unchanged
	})

	t.Run("success - update email", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData, token := authHelper.CreateAuthenticatedUser(t, "Email Update User", "oldemail@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		newEmail := "newemail@example.com"
		req := models.UpdateUserRequest{
			Email: &newEmail,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/users/"+userID, token, req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "newemail@example.com", resp.Data["email"])
		assert.Equal(t, "Email Update User", resp.Data["name"]) // Name unchanged
	})

	t.Run("success - update both name and email", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData, token := authHelper.CreateAuthenticatedUser(t, "Both Update User", "both@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		newName := "New Both Name"
		newEmail := "newboth@example.com"
		req := models.UpdateUserRequest{
			Name:  &newName,
			Email: &newEmail,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/users/"+userID, token, req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "New Both Name", resp.Data["name"])
		assert.Equal(t, "newboth@example.com", resp.Data["email"])
	})

	t.Run("error - duplicate email", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create first user
		authHelper.RegisterUser(t, "First User", "existing@example.com", "password123")

		// Create second user
		userData, token := authHelper.CreateAuthenticatedUser(t, "Second User", "second@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		// Try to update second user's email to first user's email
		existingEmail := "existing@example.com"
		req := models.UpdateUserRequest{
			Email: &existingEmail,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/users/"+userID, token, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("error - invalid email format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData, token := authHelper.CreateAuthenticatedUser(t, "Invalid Email User", "validemail@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		invalidEmail := "not-an-email"
		req := models.UpdateUserRequest{
			Email: &invalidEmail,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/users/"+userID, token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - name too short", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData, token := authHelper.CreateAuthenticatedUser(t, "Short Name User", "shortname@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		shortName := "X" // min is 2
		req := models.UpdateUserRequest{
			Name: &shortName,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/users/"+userID, token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - user not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		newName := "New Name"
		req := models.UpdateUserRequest{
			Name: &newName,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/users/"+nonExistentID, token, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData, _ := authHelper.CreateAuthenticatedUser(t, "Unauth User", "unauthuser@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		newName := "New Name"
		req := models.UpdateUserRequest{
			Name: &newName,
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPut, "/api/v1/users/"+userID, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestDeleteUser tests the DELETE /api/v1/users/:id endpoint.
func TestDeleteUser(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)

	t.Run("success - delete user", func(t *testing.T) {
		userData, token := authHelper.CreateAuthenticatedUser(t, "Delete Me", "deleteme@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/users/"+userID, token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		// Verify user is deleted
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/users/"+userID, token, nil)
		assert.Equal(t, http.StatusNotFound, w2.Code)
	})

	t.Run("error - user not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/users/"+nonExistentID, token, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - invalid user ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser2@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/users/invalid-id", token, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData, _ := authHelper.CreateAuthenticatedUser(t, "Unauth User", "unauthuser@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		w := testutil.MakeRequest(t, testServer.Router, http.MethodDelete, "/api/v1/users/"+userID, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
