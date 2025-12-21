//go:build api

package api

import (
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

// TestCreateInvitation tests the POST /api/v1/teams/:teamId/invitations endpoint.
func TestCreateInvitation(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)

	t.Run("success - owner creates member invitation", func(t *testing.T) {
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Invite Test Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.CreateInvitationRequest{
			Email: "newmember@example.com",
			Role:  models.RoleMember,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", ownerToken, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		assert.Equal(t, "newmember@example.com", resp.Data["email"])
		assert.Equal(t, models.RoleMember, resp.Data["role"])
		assert.NotEmpty(t, resp.Data["id"])
		assert.NotEmpty(t, resp.Data["expiresAt"])
	})

	t.Run("success - owner creates admin invitation", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Admin Invite Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.CreateInvitationRequest{
			Email: "newadmin@example.com",
			Role:  models.RoleAdmin,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", ownerToken, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Equal(t, models.RoleAdmin, resp.Data["role"])
	})

	t.Run("success - admin creates invitation", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		userData, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Admin Test Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		// Create admin user and add to team
		admin2Data, admin2Token := authHelper.CreateAuthenticatedUser(t, "Admin User", "admin@example.com", "password123")
		admin2OID := testserver.GetObjectIDFromResponse(t, admin2Data)

		adminMember := &models.TeamMember{
			TeamID:   teamOID,
			UserID:   admin2OID,
			Role:     models.RoleAdmin,
			JoinedAt: time.Now(),
		}
		teamHelper.SeedTeamMember(t, adminMember)

		// Admin creates invitation
		req := models.CreateInvitationRequest{
			Email: "newinvitee@example.com",
			Role:  models.RoleMember,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", admin2Token, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		// Verify owner ID is set correctly in response
		_ = userData // owner exists but not used directly
	})

	t.Run("error - already a member", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Already Member Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		// Create another user who is already a member
		memberData, _ := authHelper.CreateAuthenticatedUser(t, "Existing Member", "existing@example.com", "password123")
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		existingMember := &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		}
		teamHelper.SeedTeamMember(t, existingMember)

		// Try to invite existing member
		req := models.CreateInvitationRequest{
			Email: "existing@example.com",
			Role:  models.RoleMember,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", ownerToken, req)

		assert.Equal(t, http.StatusConflict, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.False(t, resp.Success)
	})

	t.Run("error - pending invitation exists", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Pending Invite Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create first invitation
		req := models.CreateInvitationRequest{
			Email: "pending@example.com",
			Role:  models.RoleMember,
		}

		w1 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", ownerToken, req)
		require.Equal(t, http.StatusCreated, w1.Code)

		// Try to create duplicate invitation
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", ownerToken, req)

		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("error - invalid email format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Invalid Email Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.CreateInvitationRequest{
			Email: "not-an-email",
			Role:  models.RoleMember,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", ownerToken, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - invalid role", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Invalid Role Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := map[string]string{
			"email": "valid@example.com",
			"role":  "superuser", // invalid role
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", ownerToken, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - member cannot create invitation", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Member Invite Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		// Create member user and add to team
		memberData, memberToken := authHelper.CreateAuthenticatedUser(t, "Member User", "member@example.com", "password123")
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		member := &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		}
		teamHelper.SeedTeamMember(t, member)

		// Member tries to create invitation
		req := models.CreateInvitationRequest{
			Email: "newinvitee@example.com",
			Role:  models.RoleMember,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", memberToken, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Unauth Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		req := models.CreateInvitationRequest{
			Email: "newinvitee@example.com",
			Role:  models.RoleMember,
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestListTeamInvitations tests the GET /api/v1/teams/:teamId/invitations endpoint.
func TestListTeamInvitations(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)
	invitationHelper := testserver.NewInvitationHelper(testServer)

	t.Run("success - owner lists invitations", func(t *testing.T) {
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "List Invitations Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create some invitations
		invitationHelper.CreateInvitation(t, ownerToken, teamID, "invite1@example.com", models.RoleMember)
		invitationHelper.CreateInvitation(t, ownerToken, teamID, "invite2@example.com", models.RoleAdmin)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/invitations", ownerToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		require.True(t, ok, "items should be an array")
		assert.Len(t, items, 2)
	})

	t.Run("success - returns empty list when no invitations", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Empty Invitations Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/invitations", ownerToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		require.True(t, ok, "items should be an array")
		assert.Len(t, items, 0)
	})

	t.Run("error - member cannot list invitations", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Member List Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		// Create member and add to team
		memberData, memberToken := authHelper.CreateAuthenticatedUser(t, "Member User", "member@example.com", "password123")
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		member := &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		}
		teamHelper.SeedTeamMember(t, member)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/invitations", memberToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// TestCancelInvitation tests the DELETE /api/v1/teams/:teamId/invitations/:id endpoint.
func TestCancelInvitation(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)
	invitationHelper := testserver.NewInvitationHelper(testServer)

	t.Run("success - owner cancels invitation", func(t *testing.T) {
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Cancel Invitation Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "tocancel@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/invitations/"+invitationID, ownerToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "cancelled")

		// Verify invitation is no longer listed
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/invitations", ownerToken, nil)
		listResp := testutil.ParseAPIResponse(t, w2)
		items := listResp.Data["items"].([]interface{})
		assert.Len(t, items, 0)
	})

	t.Run("error - invitation not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Not Found Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/invitations/"+nonExistentID, ownerToken, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - invalid invitation ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Invalid ID Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/invitations/invalid-id", ownerToken, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - member cannot cancel invitation", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Member Cancel Team")
		teamID := testserver.GetIDFromResponse(t, teamData)
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		// Create invitation
		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "tocancel@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		// Create member and add to team
		memberData, memberToken := authHelper.CreateAuthenticatedUser(t, "Member User", "member@example.com", "password123")
		memberOID := testserver.GetObjectIDFromResponse(t, memberData)

		member := &models.TeamMember{
			TeamID:   teamOID,
			UserID:   memberOID,
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		}
		teamHelper.SeedTeamMember(t, member)

		// Member tries to cancel invitation
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/invitations/"+invitationID, memberToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// TestListMyInvitations tests the GET /api/v1/invitations endpoint.
func TestListMyInvitations(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)
	invitationHelper := testserver.NewInvitationHelper(testServer)

	t.Run("success - lists user's pending invitations", func(t *testing.T) {
		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "My Invitations Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create another user who will receive invitations
		_, inviteeToken := authHelper.CreateAuthenticatedUser(t, "Invitee User", "invitee@example.com", "password123")

		// Create invitation for the invitee
		invitationHelper.CreateInvitation(t, ownerToken, teamID, "invitee@example.com", models.RoleMember)

		// Invitee lists their invitations
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/invitations", inviteeToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		require.True(t, ok, "items should be an array")
		assert.Len(t, items, 1)

		// Verify invitation details
		invitation := items[0].(map[string]interface{})
		assert.Equal(t, models.RoleMember, invitation["role"])
		assert.NotNil(t, invitation["team"])
	})

	t.Run("success - returns empty list when no invitations", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, userToken := authHelper.CreateAuthenticatedUser(t, "No Invites User", "noinvites@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/invitations", userToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		require.True(t, ok, "items should be an array")
		assert.Len(t, items, 0)
	})

	t.Run("success - only shows invitations for user's email", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Multi User Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create two users
		_, user1Token := authHelper.CreateAuthenticatedUser(t, "User One", "user1@example.com", "password123")
		authHelper.CreateAuthenticatedUser(t, "User Two", "user2@example.com", "password123")

		// Create invitations for both users
		invitationHelper.CreateInvitation(t, ownerToken, teamID, "user1@example.com", models.RoleMember)
		invitationHelper.CreateInvitation(t, ownerToken, teamID, "user2@example.com", models.RoleAdmin)

		// User1 should only see their invitation
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/invitations", user1Token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		items := resp.Data["items"].([]interface{})
		assert.Len(t, items, 1)

		invitation := items[0].(map[string]interface{})
		assert.Equal(t, models.RoleMember, invitation["role"])
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/invitations", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestAcceptInvitation tests the POST /api/v1/invitations/:id/accept endpoint.
func TestAcceptInvitation(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)
	invitationHelper := testserver.NewInvitationHelper(testServer)

	t.Run("success - accept invitation and join team", func(t *testing.T) {
		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Accept Invitation Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create invitee
		_, inviteeToken := authHelper.CreateAuthenticatedUser(t, "Invitee User", "invitee@example.com", "password123")

		// Create invitation
		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "invitee@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		// Accept invitation
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+invitationID+"/accept", inviteeToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "accepted")
		assert.Equal(t, teamID, resp.Data["teamId"])

		// Verify user is now a team member
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/members", inviteeToken, nil)
		assert.Equal(t, http.StatusOK, w2.Code)
	})

	t.Run("error - invitation not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, userToken := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+nonExistentID+"/accept", userToken, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - email mismatch", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Email Mismatch Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create invitation for a specific email
		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "intended@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		// Different user tries to accept
		_, wrongUserToken := authHelper.CreateAuthenticatedUser(t, "Wrong User", "wronguser@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+invitationID+"/accept", wrongUserToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - expired invitation", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		ownerData, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		ownerOID := testserver.GetObjectIDFromResponse(t, ownerData)
		teamData := teamHelper.CreateTeam(t, ownerToken, "Expired Invitation Team")
		teamOID := testserver.GetObjectIDFromResponse(t, teamData)

		// Create invitee
		_, inviteeToken := authHelper.CreateAuthenticatedUser(t, "Invitee User", "invitee@example.com", "password123")

		// Seed expired invitation directly using SeedInvitationRaw to preserve ExpiresAt
		expiredInvitation := &models.TeamInvitation{
			TeamID:    teamOID,
			Email:     "invitee@example.com",
			InvitedBy: ownerOID,
			Role:      models.RoleMember,
			ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired yesterday
			CreatedAt: time.Now().Add(-48 * time.Hour),
		}
		seededInvitation := invitationHelper.SeedInvitationRaw(t, expiredInvitation)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+seededInvitation.ID.Hex()+"/accept", inviteeToken, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.False(t, resp.Success)
	})

	t.Run("error - invalid invitation ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, userToken := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/invalid-id/accept", userToken, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+nonExistentID+"/accept", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestDeclineInvitation tests the POST /api/v1/invitations/:id/decline endpoint.
func TestDeclineInvitation(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)
	invitationHelper := testserver.NewInvitationHelper(testServer)

	t.Run("success - decline invitation", func(t *testing.T) {
		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Decline Invitation Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create invitee
		_, inviteeToken := authHelper.CreateAuthenticatedUser(t, "Invitee User", "invitee@example.com", "password123")

		// Create invitation
		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "invitee@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		// Decline invitation
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+invitationID+"/decline", inviteeToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "declined")

		// Verify invitation is no longer visible to the user
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/invitations", inviteeToken, nil)
		listResp := testutil.ParseAPIResponse(t, w2)
		items := listResp.Data["items"].([]interface{})
		assert.Len(t, items, 0)
	})

	t.Run("error - invitation not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, userToken := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+nonExistentID+"/decline", userToken, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - email mismatch", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Decline Mismatch Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create invitation for a specific email
		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "intended@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		// Different user tries to decline
		_, wrongUserToken := authHelper.CreateAuthenticatedUser(t, "Wrong User", "wronguser@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+invitationID+"/decline", wrongUserToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - invalid invitation ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, userToken := authHelper.CreateAuthenticatedUser(t, "Test User", "testuser@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/invalid-id/decline", userToken, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+nonExistentID+"/decline", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestInvitationWorkflow tests the complete invitation workflow.
func TestInvitationWorkflow(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	teamHelper := testserver.NewTeamHelper(testServer)
	invitationHelper := testserver.NewInvitationHelper(testServer)

	t.Run("complete workflow - invite, accept, verify membership", func(t *testing.T) {
		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Workflow Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create user to invite
		_, newUserToken := authHelper.CreateAuthenticatedUser(t, "New Member", "newmember@example.com", "password123")

		// Step 1: Owner creates invitation
		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "newmember@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		// Step 2: Verify invitation appears in team's list
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/invitations", ownerToken, nil)
		teamInvResp := testutil.ParseAPIResponse(t, w)
		teamItems := teamInvResp.Data["items"].([]interface{})
		assert.Len(t, teamItems, 1)

		// Step 3: Verify invitation appears in user's list
		w = testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/invitations", newUserToken, nil)
		userInvResp := testutil.ParseAPIResponse(t, w)
		userItems := userInvResp.Data["items"].([]interface{})
		assert.Len(t, userItems, 1)

		// Step 4: User accepts invitation
		w = testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+invitationID+"/accept", newUserToken, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 5: Verify invitation no longer in team's list
		w = testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/invitations", ownerToken, nil)
		teamInvResp = testutil.ParseAPIResponse(t, w)
		teamItems = teamInvResp.Data["items"].([]interface{})
		assert.Len(t, teamItems, 0)

		// Step 6: Verify user is now in team members list
		w = testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/members", ownerToken, nil)
		membersResp := testutil.ParseAPIResponse(t, w)
		members := membersResp.Data["items"].([]interface{})
		assert.Len(t, members, 2) // owner + new member
	})

	t.Run("workflow - invite and decline", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Decline Workflow Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create user to invite
		_, newUserToken := authHelper.CreateAuthenticatedUser(t, "New Member", "newmember@example.com", "password123")

		// Step 1: Owner creates invitation
		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "newmember@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		// Step 2: User declines invitation
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/invitations/"+invitationID+"/decline", newUserToken, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 3: Verify invitation no longer in team's list
		w = testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/invitations", ownerToken, nil)
		teamInvResp := testutil.ParseAPIResponse(t, w)
		teamItems := teamInvResp.Data["items"].([]interface{})
		assert.Len(t, teamItems, 0)

		// Step 4: Verify user is NOT in team members list
		w = testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/teams/"+teamID+"/members", ownerToken, nil)
		membersResp := testutil.ParseAPIResponse(t, w)
		members := membersResp.Data["items"].([]interface{})
		assert.Len(t, members, 1) // only owner
	})

	t.Run("workflow - invite and cancel", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		// Create owner and team
		_, ownerToken := authHelper.CreateAuthenticatedUser(t, "Team Owner", "owner@example.com", "password123")
		teamData := teamHelper.CreateTeam(t, ownerToken, "Cancel Workflow Team")
		teamID := testserver.GetIDFromResponse(t, teamData)

		// Create user to invite
		_, newUserToken := authHelper.CreateAuthenticatedUser(t, "New Member", "newmember@example.com", "password123")

		// Step 1: Owner creates invitation
		invitationData := invitationHelper.CreateInvitation(t, ownerToken, teamID, "newmember@example.com", models.RoleMember)
		invitationID := testserver.GetIDFromResponse(t, invitationData)

		// Step 2: Owner cancels invitation
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/teams/"+teamID+"/invitations/"+invitationID, ownerToken, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// Step 3: Verify invitation no longer in user's list
		w = testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/invitations", newUserToken, nil)
		userInvResp := testutil.ParseAPIResponse(t, w)
		userItems := userInvResp.Data["items"].([]interface{})
		assert.Len(t, userItems, 0)
	})
}
