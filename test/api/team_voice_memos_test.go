//go:build api

package api

import (
	"context"
	"net/http"
	"testing"
	"time"

	"gin-sample/internal/models"
	"gin-sample/test/api/testserver"
	"gin-sample/test/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestCreateTeamVoiceMemo tests the POST /api/v1/teams/:teamId/voice-memos endpoint.
func TestCreateTeamVoiceMemo(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - owner creates team voice memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Voice Memo Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.CreateVoiceMemoRequest{
			Title:       "Team Memo",
			Duration:    120,
			FileSize:    1048576,
			AudioFormat: "mp3",
			Tags:        []string{"meeting", "notes"},
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		memo, ok := resp.Data["memo"].(map[string]interface{})
		require.True(t, ok, "memo should be an object")
		assert.Equal(t, "Team Memo", memo["title"])
		assert.Equal(t, teamID, memo["teamId"])
		assert.NotEmpty(t, memo["id"])

		uploadURL, ok := resp.Data["uploadUrl"].(string)
		assert.True(t, ok, "uploadUrl should be a string")
		assert.NotEmpty(t, uploadURL)
	})

	t.Run("success - admin creates team voice memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		adminData, adminToken := authHelper.CreateAuthenticatedUser(t, "Admin", "admin@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Admin Memo Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)
		adminOID := testserver.GetObjectIDFromResponse(t, adminData)

		// Add admin to team
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   adminOID,
			Role:     models.RoleAdmin,
			JoinedAt: time.Now(),
		})

		req := models.CreateVoiceMemoRequest{
			Title:       "Admin's Team Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "wav",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", adminToken, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("success - member creates team voice memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		memberData, memberToken := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Member Memo Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		// Add member to team
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		req := models.CreateVoiceMemoRequest{
			Title:       "Member's Team Memo",
			Duration:    90,
			FileSize:    768000,
			AudioFormat: "mp3",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", memberToken, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("error - non-member cannot create team memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		_, nonMemberToken := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Private Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.CreateVoiceMemoRequest{
			Title:       "Unauthorized Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", nonMemberToken, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - invalid team ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "User", "user@example.com", "password123")

		req := models.CreateVoiceMemoRequest{
			Title:       "Invalid Team",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/invalid-id/voice-memos", token, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - missing required fields", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Missing Fields Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := map[string]interface{}{
			"audioFormat": "mp3",
			// missing title, fileSize
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()

		req := models.CreateVoiceMemoRequest{
			Title:       "Unauthorized",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestListTeamVoiceMemos tests the GET /api/v1/teams/:teamId/voice-memos endpoint.
func TestListTeamVoiceMemos(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - returns empty list when no memos", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Empty Memos Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, nil)

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

	t.Run("success - returns team memos", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Memos Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create multiple team memos
		for i := 0; i < 3; i++ {
			req := models.CreateVoiceMemoRequest{
				Title:       "Team Memo",
				Duration:    60 * (i + 1),
				FileSize:    512000,
				AudioFormat: "mp3",
			}
			w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, req)
			require.Equal(t, http.StatusCreated, w.Code)
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, items, 3)
	})

	t.Run("success - all team members can view memos", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		memberData, memberToken := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Shared Memos Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		// Add member to team
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		// Owner creates a memo
		req := models.CreateVoiceMemoRequest{
			Title:       "Owner's Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Member can see the memo
		w = testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos", memberToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		items, _ := resp.Data["items"].([]interface{})
		assert.Len(t, items, 1)
	})

	t.Run("success - pagination works", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Paginated Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create 5 memos
		for i := 0; i < 5; i++ {
			req := models.CreateVoiceMemoRequest{
				Title:       "Paginated Memo",
				Duration:    60,
				FileSize:    512000,
				AudioFormat: "mp3",
			}
			w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, req)
			require.Equal(t, http.StatusCreated, w.Code)
		}

		// Get page 1 with limit 2
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos?page=1&limit=2", ownerToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		items, _ := resp.Data["items"].([]interface{})
		assert.Len(t, items, 2)

		pagination, _ := resp.Data["pagination"].(map[string]interface{})
		assert.Equal(t, float64(5), pagination["totalItems"])
		assert.Equal(t, float64(3), pagination["totalPages"])
	})

	t.Run("error - non-member cannot list team memos", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner5@example.com", "password123")
		_, nonMemberToken := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Private Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos", nonMemberToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestGetTeamVoiceMemo tests the GET /api/v1/teams/:teamId/voice-memos/:id endpoint.
func TestGetTeamVoiceMemo(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - owner gets team memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Get Memo Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Get This Memo",
			Duration:    120,
			FileSize:    1048576,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Get the memo
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, ownerToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, "Get This Memo", resp.Data["title"])
		assert.Equal(t, memoID, resp.Data["id"])
	})

	t.Run("success - member gets team memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		memberData, memberToken := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Member Get Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		// Add member to team
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		// Owner creates a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Owner's Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Member gets the memo
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, memberToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("error - non-member cannot get team memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		_, nonMemberToken := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Private Get Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Owner creates a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Private Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Non-member tries to get memo
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, nonMemberToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - memo not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Not Found Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos/"+nonExistentID, ownerToken, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - invalid memo ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner5@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Invalid ID Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos/invalid-id", ownerToken, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()
		memoID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestDeleteTeamVoiceMemo tests the DELETE /api/v1/teams/:teamId/voice-memos/:id endpoint.
func TestDeleteTeamVoiceMemo(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - owner deletes team memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Delete Memo Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Delete This Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Delete the memo
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, ownerToken, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify memo is deleted
		getW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, ownerToken, nil)
		assert.Equal(t, http.StatusNotFound, getW.Code)
	})

	t.Run("success - admin deletes team memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		adminData, adminToken := authHelper.CreateAuthenticatedUser(t, "Admin", "admin@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Admin Delete Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)
		adminOID := testserver.GetObjectIDFromResponse(t, adminData)

		// Add admin to team
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   adminOID,
			Role:     models.RoleAdmin,
			JoinedAt: time.Now(),
		})

		// Owner creates a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Admin Deletes This",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Admin deletes the memo
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, adminToken, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("success - member can delete team memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		memberData, memberToken := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Member Delete Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		// Add member to team
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		// Owner creates a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Deletable Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Member deletes the memo (allowed per authorization rules)
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, memberToken, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("error - non-member cannot delete team memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		_, nonMemberToken := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Private Delete Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Owner creates a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Private Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Non-member tries to delete
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, nonMemberToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - memo not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner5@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Not Found Delete Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/voice-memos/"+nonExistentID, ownerToken, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()
		memoID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestConfirmTeamUpload tests the POST /api/v1/teams/:teamId/voice-memos/:id/confirm-upload endpoint.
func TestConfirmTeamUpload(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - confirms team upload", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Confirm Upload Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Confirm Upload Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)
		uploadURL := createResp.Data["uploadUrl"].(string)

		// Upload test audio
		uploadTestAudio(t, uploadURL)

		// Confirm upload
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/confirm-upload", ownerToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "transcription started")
	})

	t.Run("error - cannot confirm twice", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Double Confirm Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Double Confirm Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)
		uploadURL := createResp.Data["uploadUrl"].(string)

		uploadTestAudio(t, uploadURL)

		// First confirm
		w1 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/confirm-upload", ownerToken, nil)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second confirm should fail
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/confirm-upload", ownerToken, nil)
		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("error - non-member cannot confirm upload", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		_, nonMemberToken := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Private Confirm Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Owner creates a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Private Confirm Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Non-member tries to confirm
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/confirm-upload", nonMemberToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - memo not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Not Found Confirm Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+nonExistentID+"/confirm-upload", ownerToken, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()
		memoID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/confirm-upload", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestRetryTeamTranscription tests the POST /api/v1/teams/:teamId/voice-memos/:id/retry-transcription endpoint.
func TestRetryTeamTranscription(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - retries failed transcription", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Retry Success Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Failed Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)
		uploadURL := createResp.Data["uploadUrl"].(string)

		// Upload test audio
		uploadTestAudio(t, uploadURL)

		// Set memo status to failed directly via repository
		memoOID, err := primitive.ObjectIDFromHex(memoID)
		require.NoError(t, err)
		err = testServer.VoiceMemoRepo.UpdateStatus(context.Background(), memoOID, models.StatusFailed)
		require.NoError(t, err)

		// Retry transcription
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/retry-transcription", ownerToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "transcription retry started")

		// Verify memo status changed to transcribing
		updatedMemo, err := testServer.VoiceMemoRepo.FindByID(context.Background(), memoOID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusTranscribing, updatedMemo.Status)
	})

	t.Run("error - cannot retry for non-failed memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Retry Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create a memo (status is pending_upload)
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Retry Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Try to retry (should fail - not in failed state)
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/retry-transcription", ownerToken, nil)

		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("error - non-member cannot retry transcription", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		_, nonMemberToken := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Private Retry Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Owner creates a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Private Retry Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Non-member tries to retry
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/retry-transcription", nonMemberToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - memo not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Not Found Retry Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+nonExistentID+"/retry-transcription", ownerToken, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()
		memoID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/retry-transcription", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestTeamVoiceMemoWorkflow tests the complete team voice memo workflow.
func TestTeamVoiceMemoWorkflow(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("full workflow - create, upload, confirm, transcribe", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Workflow Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// 1. Create voice memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Workflow Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)
		uploadURL := createResp.Data["uploadUrl"].(string)
		assert.Equal(t, string(models.StatusPendingUpload), memo["status"])

		// 2. Upload audio
		uploadTestAudio(t, uploadURL)

		// 3. Confirm upload
		confirmW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID+"/confirm-upload", ownerToken, nil)
		assert.Equal(t, http.StatusOK, confirmW.Code)

		// 4. Start transcription processor
		ctx := context.Background()
		testServer.StartTranscriptionProcessor(ctx)

		// 5. Wait for transcription to complete
		time.Sleep(500 * time.Millisecond)
		testServer.StopTranscriptionProcessor()

		// 6. Verify memo status is ready
		memoOID, _ := primitive.ObjectIDFromHex(memoID)
		updatedMemo, err := testServer.VoiceMemoRepo.FindByID(ctx, memoOID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusReady, updatedMemo.Status)
		assert.NotEmpty(t, updatedMemo.Transcription)
	})

	t.Run("team members can view each other's memos", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		memberData, memberToken := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, ownerToken, "Shared Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		// Add member to team
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		// Owner creates a memo
		createReq := models.CreateVoiceMemoRequest{
			Title:       "Owner's Memo",
			Duration:    60,
			FileSize:    512000,
			AudioFormat: "mp3",
		}
		createW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/voice-memos", ownerToken, createReq)
		require.Equal(t, http.StatusCreated, createW.Code)

		createResp := testutil.ParseAPIResponse(t, createW)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Member can list and get the memo
		listW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos", memberToken, nil)
		assert.Equal(t, http.StatusOK, listW.Code)

		getW := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/voice-memos/"+memoID, memberToken, nil)
		assert.Equal(t, http.StatusOK, getW.Code)
	})
}
