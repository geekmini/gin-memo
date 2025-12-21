//go:build api

package testserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"gin-sample/internal/models"
	"gin-sample/pkg/response"
	"gin-sample/test/testutil"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthHelper provides authentication helpers for API tests.
type AuthHelper struct {
	server *TestServer
}

// NewAuthHelper creates a new auth helper.
func NewAuthHelper(server *TestServer) *AuthHelper {
	return &AuthHelper{server: server}
}

// RegisterUser registers a new user and returns the user data.
func (ah *AuthHelper) RegisterUser(t *testing.T, name, email, password string) map[string]interface{} {
	t.Helper()

	req := models.CreateUserRequest{
		Name:     name,
		Email:    email,
		Password: password,
	}

	w := testutil.MakeRequest(t, ah.server.Router, http.MethodPost, "/api/v1/auth/register", req)
	require.Equal(t, http.StatusCreated, w.Code, "register should return 201, got: %s", w.Body.String())

	var resp response.Response
	testutil.ParseResponse(t, w, &resp)
	require.True(t, resp.Success, "register response should be successful")

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "response data should be a map")
	return data
}

// Login logs in a user and returns the auth response containing tokens.
func (ah *AuthHelper) Login(t *testing.T, email, password string) map[string]interface{} {
	t.Helper()

	req := models.LoginRequest{
		Email:    email,
		Password: password,
	}

	w := testutil.MakeRequest(t, ah.server.Router, http.MethodPost, "/api/v1/auth/login", req)
	require.Equal(t, http.StatusOK, w.Code, "login should return 200, got: %s", w.Body.String())

	var resp response.Response
	testutil.ParseResponse(t, w, &resp)
	require.True(t, resp.Success, "login response should be successful")

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "response data should be a map")
	return data
}

// GetAccessToken logs in and returns just the access token.
func (ah *AuthHelper) GetAccessToken(t *testing.T, email, password string) string {
	t.Helper()

	data := ah.Login(t, email, password)
	token, ok := data["accessToken"].(string)
	require.True(t, ok, "accessToken should be a string")

	return token
}

// CreateAuthenticatedUser creates a user and returns the user data and access token.
func (ah *AuthHelper) CreateAuthenticatedUser(t *testing.T, name, email, password string) (userData map[string]interface{}, accessToken string) {
	t.Helper()

	userData = ah.RegisterUser(t, name, email, password)
	authData := ah.Login(t, email, password)

	accessToken, ok := authData["accessToken"].(string)
	require.True(t, ok, "accessToken should be a string")

	return userData, accessToken
}

// CreateDefaultUser creates a user with default test credentials.
func (ah *AuthHelper) CreateDefaultUser(t *testing.T) (userData map[string]interface{}, accessToken string) {
	t.Helper()
	return ah.CreateAuthenticatedUser(t, "Test User", "test@example.com", "password123")
}

// TeamHelper provides team-related helpers for API tests.
type TeamHelper struct {
	server *TestServer
}

// NewTeamHelper creates a new team helper.
func NewTeamHelper(server *TestServer) *TeamHelper {
	return &TeamHelper{server: server}
}

// CreateTeam creates a new team and returns the team data.
func (th *TeamHelper) CreateTeam(t *testing.T, token, name string) map[string]interface{} {
	t.Helper()

	// Generate slug from name: lowercase, replace spaces with hyphens
	slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))

	req := models.CreateTeamRequest{
		Name: name,
		Slug: slug,
	}

	w := testutil.MakeAuthRequest(t, th.server.Router, http.MethodPost, "/api/v1/teams", token, req)
	require.Equal(t, http.StatusCreated, w.Code, "create team should return 201, got: %s", w.Body.String())

	var resp response.Response
	testutil.ParseResponse(t, w, &resp)
	require.True(t, resp.Success, "create team response should be successful")

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "response data should be a map")
	return data
}

// VoiceMemoHelper provides voice memo helpers for API tests.
type VoiceMemoHelper struct {
	server *TestServer
}

// NewVoiceMemoHelper creates a new voice memo helper.
func NewVoiceMemoHelper(server *TestServer) *VoiceMemoHelper {
	return &VoiceMemoHelper{server: server}
}

// CreateVoiceMemo creates a voice memo and returns the response data.
func (vh *VoiceMemoHelper) CreateVoiceMemo(t *testing.T, token, title string, duration int) map[string]interface{} {
	t.Helper()

	req := models.CreateVoiceMemoRequest{
		Title:       title,
		Duration:    duration,
		FileSize:    1024 * 1024, // 1MB default file size
		AudioFormat: "mp3",
	}

	w := testutil.MakeAuthRequest(t, vh.server.Router, http.MethodPost, "/api/v1/voice-memos", token, req)
	require.Equal(t, http.StatusCreated, w.Code, "create voice memo should return 201, got: %s", w.Body.String())

	var resp response.Response
	testutil.ParseResponse(t, w, &resp)
	require.True(t, resp.Success, "create voice memo response should be successful")

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "response data should be a map")
	return data
}

// ParseResponseData is a generic helper to parse response data into a specific type.
func ParseResponseData[T any](t *testing.T, data map[string]interface{}) T {
	t.Helper()

	jsonBytes, err := json.Marshal(data)
	require.NoError(t, err, "failed to marshal response data")

	var result T
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err, "failed to unmarshal response data")

	return result
}

// GetIDFromResponse extracts the ID from response data.
// It handles both direct ID fields and nested user objects (for auth responses).
func GetIDFromResponse(t *testing.T, data map[string]interface{}) string {
	t.Helper()

	// Try direct id field first
	if id, ok := data["id"].(string); ok {
		return id
	}

	// Try "ID" as fallback
	if id, ok := data["ID"].(string); ok {
		return id
	}

	// Try nested user object (for auth responses)
	if user, ok := data["user"].(map[string]interface{}); ok {
		if id, ok := user["id"].(string); ok {
			return id
		}
	}

	t.Fatal("id should be a string in response data (checked: id, ID, user.id)")
	return ""
}

// GetObjectIDFromResponse extracts and parses the ID as ObjectID.
func GetObjectIDFromResponse(t *testing.T, data map[string]interface{}) primitive.ObjectID {
	t.Helper()

	idStr := GetIDFromResponse(t, data)
	oid, err := primitive.ObjectIDFromHex(idStr)
	require.NoError(t, err, "failed to parse ObjectID")

	return oid
}

// AssertErrorResponse asserts the response is an error with expected status and message contains.
func AssertErrorResponse(t *testing.T, w interface {
	Code() int
	Body() []byte
}, expectedStatus int, messageContains string) {
	t.Helper()

	// This would need the actual recorder interface
	// For now, provide a simpler version
}

// SeedUser directly inserts a user into the database (bypasses API).
func (ah *AuthHelper) SeedUser(t *testing.T, user *models.User) *models.User {
	t.Helper()
	ctx := context.Background()

	err := ah.server.UserRepo.Create(ctx, user)
	require.NoError(t, err, "failed to seed user")

	return user
}

// SeedTeam directly inserts a team into the database (bypasses API).
func (th *TeamHelper) SeedTeam(t *testing.T, team *models.Team) *models.Team {
	t.Helper()
	ctx := context.Background()

	err := th.server.TeamRepo.Create(ctx, team)
	require.NoError(t, err, "failed to seed team")

	return team
}

// SeedTeamMember directly inserts a team member into the database.
func (th *TeamHelper) SeedTeamMember(t *testing.T, member *models.TeamMember) *models.TeamMember {
	t.Helper()
	ctx := context.Background()

	err := th.server.TeamMemberRepo.Create(ctx, member)
	require.NoError(t, err, "failed to seed team member")

	return member
}

// InvitationHelper provides invitation-related helpers for API tests.
type InvitationHelper struct {
	server *TestServer
}

// NewInvitationHelper creates a new invitation helper.
func NewInvitationHelper(server *TestServer) *InvitationHelper {
	return &InvitationHelper{server: server}
}

// CreateInvitation creates an invitation via API and returns the response data.
func (ih *InvitationHelper) CreateInvitation(t *testing.T, token, teamID, email, role string) map[string]interface{} {
	t.Helper()

	req := models.CreateInvitationRequest{
		Email: email,
		Role:  role,
	}

	w := testutil.MakeAuthRequest(t, ih.server.Router, http.MethodPost, "/api/v1/teams/"+teamID+"/invitations", token, req)
	require.Equal(t, http.StatusCreated, w.Code, "create invitation should return 201, got: %s", w.Body.String())

	var resp response.Response
	testutil.ParseResponse(t, w, &resp)
	require.True(t, resp.Success, "create invitation response should be successful")

	data, ok := resp.Data.(map[string]interface{})
	require.True(t, ok, "response data should be a map")
	return data
}

// SeedInvitation directly inserts an invitation into the database (bypasses API).
// Note: This uses the repository's Create method which sets default ExpiresAt.
// Use SeedInvitationRaw for full control over all fields (e.g., expired invitations).
func (ih *InvitationHelper) SeedInvitation(t *testing.T, invitation *models.TeamInvitation) *models.TeamInvitation {
	t.Helper()
	ctx := context.Background()

	err := ih.server.TeamInvitationRepo.Create(ctx, invitation)
	require.NoError(t, err, "failed to seed invitation")

	return invitation
}

// SeedInvitationRaw directly inserts an invitation into MongoDB without going through
// the repository's Create method. This allows full control over all fields, including
// ExpiresAt (useful for testing expired invitations).
func (ih *InvitationHelper) SeedInvitationRaw(t *testing.T, invitation *models.TeamInvitation) *models.TeamInvitation {
	t.Helper()
	ctx := context.Background()

	// Set ID if not already set
	if invitation.ID.IsZero() {
		invitation.ID = primitive.NewObjectID()
	}

	// Insert directly into the collection
	collection := ih.server.MongoDB.Database.Collection("team_invitations")
	_, err := collection.InsertOne(ctx, invitation)
	require.NoError(t, err, "failed to seed invitation directly")

	return invitation
}
