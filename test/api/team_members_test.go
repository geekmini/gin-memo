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

// TestListMembers tests the GET /api/v1/teams/:teamId/members endpoint.
func TestListMembers(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - returns members list with owner", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token, "Test Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/members", token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, items, 1) // Only owner

		member := items[0].(map[string]interface{})
		assert.Equal(t, models.RoleOwner, member["role"])
	})

	t.Run("success - returns multiple members", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData1, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		userData2, _ := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Team With Members")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		// Add second user as member directly via repo
		user1OID := testserver.GetObjectIDFromResponse(t, userData1)
		user2OID := testserver.GetObjectIDFromResponse(t, userData2)

		_ = user1OID // Owner already added

		teamMember := &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		}
		teamHelper.SeedTeamMember(t, teamMember)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/members", token1, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, _ := resp.Data["items"].([]interface{})
		assert.Len(t, items, 2)
	})

	t.Run("error - non-member cannot list members", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Private Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/members", token2, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/members", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestRemoveMember tests the DELETE /api/v1/teams/:teamId/members/:userId endpoint.
func TestRemoveMember(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - owner removes member", func(t *testing.T) {
		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		userData2, _ := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Remove Member Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		user2ID := testserver.GetIDFromResponse(t, userData2)
		user2OID := testserver.GetObjectIDFromResponse(t, userData2)

		// Add second user as member
		teamMember := &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		}
		teamHelper.SeedTeamMember(t, teamMember)

		// Owner removes member
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/members/"+user2ID, token1, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "removed")

		// Verify member is removed
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/members", token1, nil)
		listResp := testutil.ParseAPIResponse(t, w2)
		items, _ := listResp.Data["items"].([]interface{})
		assert.Len(t, items, 1) // Only owner remains
	})

	t.Run("error - cannot remove owner", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData1, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token1, "Cannot Remove Owner")
		teamID := testserver.GetIDFromResponse(t, teamData)
		user1ID := testserver.GetIDFromResponse(t, userData1)

		// Try to remove owner (self)
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/members/"+user1ID, token1, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - regular member cannot remove others", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData1, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		userData2, token2 := authHelper.CreateAuthenticatedUser(t, "Member1", "member1@example.com", "password123")
		userData3, _ := authHelper.CreateAuthenticatedUser(t, "Member2", "member2@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Member Cannot Remove")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		_ = userData1 // Owner created with team

		user2OID := testserver.GetObjectIDFromResponse(t, userData2)
		user3ID := testserver.GetIDFromResponse(t, userData3)
		user3OID := testserver.GetObjectIDFromResponse(t, userData3)

		// Add both users as members
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user3OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		// Member 1 tries to remove Member 2
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/members/"+user3ID, token2, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - member not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token, "Member Not Found")
		teamID := testserver.GetIDFromResponse(t, teamData)

		nonExistentUserID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/members/"+nonExistentUserID, token, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - non-member cannot remove", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner5@example.com", "password123")
		userData2, token2 := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Non-member Cannot Remove")
		teamID := testserver.GetIDFromResponse(t, teamData)
		user2ID := testserver.GetIDFromResponse(t, userData2)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/members/"+user2ID, token2, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()
		userID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/members/"+userID, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestUpdateRole tests the PUT /api/v1/teams/:teamId/members/:userId/role endpoint.
func TestUpdateRole(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - owner promotes member to admin", func(t *testing.T) {
		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		userData2, _ := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Update Role Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		user2ID := testserver.GetIDFromResponse(t, userData2)
		user2OID := testserver.GetObjectIDFromResponse(t, userData2)

		// Add member
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		req := models.UpdateRoleRequest{
			Role: models.RoleAdmin,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID+"/members/"+user2ID+"/role", token1, req)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "updated")

		// Verify role changed
		ctx := context.Background()
		updatedMember, err := testServer.TeamMemberRepo.FindByTeamAndUser(ctx, teamOID, user2OID)
		require.NoError(t, err)
		assert.Equal(t, models.RoleAdmin, updatedMember.Role)
	})

	t.Run("success - owner demotes admin to member", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		userData2, _ := authHelper.CreateAuthenticatedUser(t, "Admin", "admin@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Demote Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		user2ID := testserver.GetIDFromResponse(t, userData2)
		user2OID := testserver.GetObjectIDFromResponse(t, userData2)

		// Add admin
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleAdmin,
			JoinedAt: time.Now(),
		})

		req := models.UpdateRoleRequest{
			Role: models.RoleMember,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID+"/members/"+user2ID+"/role", token1, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("error - cannot change owner role", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		userData1, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token1, "Cannot Change Owner")
		teamID := testserver.GetIDFromResponse(t, teamData)
		user1ID := testserver.GetIDFromResponse(t, userData1)

		req := models.UpdateRoleRequest{
			Role: models.RoleMember,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID+"/members/"+user1ID+"/role", token1, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - invalid role", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner4@example.com", "password123")
		userData2, _ := authHelper.CreateAuthenticatedUser(t, "Member", "member2@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Invalid Role Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		user2ID := testserver.GetIDFromResponse(t, userData2)
		user2OID := testserver.GetObjectIDFromResponse(t, userData2)

		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		req := map[string]string{
			"role": "invalid-role",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID+"/members/"+user2ID+"/role", token1, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - member cannot update roles", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner5@example.com", "password123")
		userData2, token2 := authHelper.CreateAuthenticatedUser(t, "Member1", "member3@example.com", "password123")
		userData3, _ := authHelper.CreateAuthenticatedUser(t, "Member2", "member4@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Member Cannot Update")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		user2OID := testserver.GetObjectIDFromResponse(t, userData2)
		user3ID := testserver.GetIDFromResponse(t, userData3)
		user3OID := testserver.GetObjectIDFromResponse(t, userData3)

		// Add both as members
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user3OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		req := models.UpdateRoleRequest{
			Role: models.RoleAdmin,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID+"/members/"+user3ID+"/role", token2, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()
		userID := primitive.NewObjectID().Hex()

		req := models.UpdateRoleRequest{
			Role: models.RoleAdmin,
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID+"/members/"+userID+"/role", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestLeaveTeam tests the POST /api/v1/teams/:teamId/leave endpoint.
func TestLeaveTeam(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - member leaves team", func(t *testing.T) {
		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		userData2, token2 := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Leave Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		user2OID := testserver.GetObjectIDFromResponse(t, userData2)

		// Add member
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		// Member leaves
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/leave", token2, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "left team")

		// Verify member is removed - accessing team should fail
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID, token2, nil)
		assert.Equal(t, http.StatusForbidden, w2.Code)
	})

	t.Run("error - owner cannot leave", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, token1, "Owner Cannot Leave")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/leave", token1, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.False(t, resp.Success)
	})

	t.Run("error - non-member cannot leave", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner3@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Non-member", "nonmember@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Non-member Cannot Leave")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/leave", token2, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		teamID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/leave", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestAdminManagesMember tests that admins can manage regular members.
func TestAdminManagesMember(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("admin can remove member", func(t *testing.T) {
		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		userData2, token2 := authHelper.CreateAuthenticatedUser(t, "Admin", "admin@example.com", "password123")
		userData3, _ := authHelper.CreateAuthenticatedUser(t, "Member", "member@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Admin Removes Member")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		user2OID := testserver.GetObjectIDFromResponse(t, userData2)
		user3ID := testserver.GetIDFromResponse(t, userData3)
		user3OID := testserver.GetObjectIDFromResponse(t, userData3)

		// Add admin
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleAdmin,
			JoinedAt: time.Now(),
		})

		// Add member
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user3OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		// Admin removes member
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/members/"+user3ID, token2, nil)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("admin can update member role", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		userData2, token2 := authHelper.CreateAuthenticatedUser(t, "Admin", "admin2@example.com", "password123")
		userData3, _ := authHelper.CreateAuthenticatedUser(t, "Member", "member2@example.com", "password123")

		teamData := teamHelper.CreateTeam(t, token1, "Admin Updates Role")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		user2OID := testserver.GetObjectIDFromResponse(t, userData2)
		user3ID := testserver.GetIDFromResponse(t, userData3)
		user3OID := testserver.GetObjectIDFromResponse(t, userData3)

		// Add admin and member
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user2OID,
			Role:     models.RoleAdmin,
			JoinedAt: time.Now(),
		})
		teamHelper.SeedTeamMember(t, &models.TeamMember{
			TeamID:   teamOID,
			UserID:   user3OID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		})

		req := models.UpdateRoleRequest{
			Role: models.RoleAdmin,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPut, "/api/v1/teams/"+teamID+"/members/"+user3ID+"/role", token2, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
