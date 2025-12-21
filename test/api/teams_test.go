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

// TestCreateTeam tests the POST /api/v1/teams endpoint.
func TestCreateTeam(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)

	t.Run("success - creates team with required fields", func(t *testing.T) {
		userData, token := authHelper.CreateAuthenticatedUser(t, "Team Owner", "teamowner@example.com", "password123")
		userID := testserver.GetIDFromResponse(t, userData)

		req := models.CreateTeamRequest{
			Name: "Test Team",
			Slug: "test-team",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", token, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		assert.Equal(t, "Test Team", resp.Data["name"])
		assert.Equal(t, "test-team", resp.Data["slug"])
		assert.Equal(t, userID, resp.Data["ownerId"])
		assert.NotEmpty(t, resp.Data["id"])
	})

	t.Run("success - creates team with all optional fields", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Full Team Owner", "fullteam@example.com", "password123")

		req := models.CreateTeamRequest{
			Name:        "Full Team",
			Slug:        "full-team",
			Description: "A team with all fields",
			LogoURL:     "https://example.com/logo.png",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", token, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "A team with all fields", resp.Data["description"])
		assert.Equal(t, "https://example.com/logo.png", resp.Data["logoUrl"])
	})

	t.Run("error - missing required name", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "No Name", "noname@example.com", "password123")

		req := map[string]interface{}{
			"slug": "no-name-team",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - missing required slug", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "No Slug", "noslug@example.com", "password123")

		req := map[string]interface{}{
			"name": "No Slug Team",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - name too short", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Short Name", "shortname@example.com", "password123")

		req := models.CreateTeamRequest{
			Name: "X", // min is 2
			Slug: "short-name-team",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - duplicate slug", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Use two different users because free users can only create 1 team
		_, token1 := authHelper.CreateAuthenticatedUser(t, "First User", "firstuser@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Second User", "seconduser@example.com", "password123")

		req1 := models.CreateTeamRequest{
			Name: "First Team",
			Slug: "duplicate-slug",
		}

		w1 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", token1, req1)
		require.Equal(t, http.StatusCreated, w1.Code)

		// Second user tries to create team with same slug
		req2 := models.CreateTeamRequest{
			Name: "Second Team",
			Slug: "duplicate-slug",
		}

		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", token2, req2)

		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("error - invalid slug format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Invalid Slug", "invalidslug@example.com", "password123")

		req := models.CreateTeamRequest{
			Name: "Invalid Slug Team",
			Slug: "Invalid Slug!", // invalid characters
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		req := models.CreateTeamRequest{
			Name: "Unauthorized Team",
			Slug: "unauthorized-team",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestListTeams tests the GET /api/v1/teams endpoint.
func TestListTeams(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - returns empty list when no teams", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "No Teams", "noteams@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams", token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		assert.True(t, ok)
		assert.Empty(t, items)

		pagination, ok := resp.Data["pagination"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, float64(0), pagination["totalItems"])
	})

	t.Run("success - returns user's teams", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Team User", "teamuser@example.com", "password123")

		// Create 1 team (free users limited to 1 team)
		teamHelper.CreateTeam(t, token, "Team A")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams", token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, items, 1)
	})

	t.Run("success - user only sees teams they belong to", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "User One", "user1@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "User Two", "user2@example.com", "password123")

		// User 1 creates 1 team (free users limited to 1 team)
		teamHelper.CreateTeam(t, token1, "User1 Team")

		// User 2 creates 1 team
		teamHelper.CreateTeam(t, token2, "User2 Team")

		// User 1 should see only their 1 team
		w1 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams", token1, nil)
		resp1 := testutil.ParseAPIResponse(t, w1)
		items1, _ := resp1.Data["items"].([]interface{})
		assert.Len(t, items1, 1)

		// User 2 should see only their 1 team
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams", token2, nil)
		resp2 := testutil.ParseAPIResponse(t, w2)
		items2, _ := resp2.Data["items"].([]interface{})
		assert.Len(t, items2, 1)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestGetTeam tests the GET /api/v1/teams/:teamId endpoint.
func TestGetTeam(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - gets team as owner", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Team Owner", "teamowner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token, "Get Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID, token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "Get Team", resp.Data["name"])
		assert.Equal(t, teamID, resp.Data["id"])
	})

	t.Run("error - non-member cannot access team", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Private Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Non-member tries to access
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID, token2, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - team not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "User", "user@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+nonExistentID, token, nil)

		assert.Equal(t, http.StatusForbidden, w.Code) // Forbidden because user is not a member of this team
	})

	t.Run("error - invalid team ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "User", "user2@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/invalid-id", token, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestUpdateTeam tests the PUT /api/v1/teams/:teamId endpoint.
func TestUpdateTeam(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - owner updates team name", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token, "Original Name")
		teamID := testserver.GetIDFromResponse(t, teamData)

		newName := "Updated Name"
		req := models.UpdateTeamRequest{
			Name: &newName,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID, token, req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "Updated Name", resp.Data["name"])
	})

	t.Run("success - owner updates all fields", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token, "Original")
		teamID := testserver.GetIDFromResponse(t, teamData)

		newName := "New Name"
		newSlug := "new-slug"
		newDesc := "New Description"
		newLogo := "https://example.com/new-logo.png"

		req := models.UpdateTeamRequest{
			Name:        &newName,
			Slug:        &newSlug,
			Description: &newDesc,
			LogoURL:     &newLogo,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID, token, req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "New Name", resp.Data["name"])
		assert.Equal(t, "new-slug", resp.Data["slug"])
		assert.Equal(t, "New Description", resp.Data["description"])
		assert.Equal(t, "https://example.com/new-logo.png", resp.Data["logoUrl"])
	})

	t.Run("error - duplicate slug", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Use two different users because free users can only create 1 team
		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner1", "owner1@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Owner2", "owner2@example.com", "password123")

		teamHelper.CreateTeam(t, token1, "First Team")
		teamData2 := teamHelper.CreateTeam(t, token2, "Second Team")
		teamID2 := testserver.GetIDFromResponse(t, teamData2)

		// Try to update second team's slug to first team's slug
		existingSlug := "first-team"
		req := models.UpdateTeamRequest{
			Slug: &existingSlug,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID2, token2, req)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("error - non-member cannot update team", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Private Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		newName := "Hacked Name"
		req := models.UpdateTeamRequest{
			Name: &newName,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID, token2, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()
		newName := "New Name"
		req := models.UpdateTeamRequest{
			Name: &newName,
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestDeleteTeam tests the DELETE /api/v1/teams/:teamId endpoint.
func TestDeleteTeam(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - owner deletes team", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token, "Delete Me")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID, token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "deleted")

		// Verify team is no longer in list
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams", token, nil)
		listResp := testutil.ParseAPIResponse(t, w2)
		items, _ := listResp.Data["items"].([]interface{})
		assert.Empty(t, items)
	})

	t.Run("error - non-owner cannot delete team", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Non-owner", "nonowner@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Protected Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Non-member tries to delete
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID, token2, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestTransferOwnership tests the POST /api/v1/teams/:teamId/transfer endpoint.
func TestTransferOwnership(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("error - new owner must be team member", func(t *testing.T) {
		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		userData2, _ := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")
		nonMemberID := testserver.GetIDFromResponse(t, userData2)

		teamData := teamHelper.CreateTeam(t, token1, "Transfer Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.TransferOwnershipRequest{
			NewOwnerID: nonMemberID,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/transfer", token1, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - non-owner cannot transfer", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		userData2, token2 := authHelper.CreateAuthenticatedUser(t, "Non-owner", "nonowner@example.com", "password123")
		user2ID := testserver.GetIDFromResponse(t, userData2)

		teamData := teamHelper.CreateTeam(t, token1, "Transfer Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.TransferOwnershipRequest{
			NewOwnerID: user2ID,
		}

		// Non-owner (who is also not a member) tries to transfer
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/transfer", token2, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - missing new owner ID", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token, "Transfer Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := map[string]interface{}{}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/transfer", token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - invalid new owner ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token, "Transfer Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.TransferOwnershipRequest{
			NewOwnerID: "invalid-id",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/transfer", token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()
		req := models.TransferOwnershipRequest{
			NewOwnerID: primitive.NewObjectID().Hex(),
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/transfer", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
