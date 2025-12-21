// Package mocks provides mock implementations of service interfaces for testing.
package mocks

import (
	"context"

	"gin-sample/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockAuthService is a mock implementation of AuthServicer.
type MockAuthService struct {
	RegisterFunc func(ctx context.Context, req *models.CreateUserRequest) (*models.AuthResponse, error)
	LoginFunc    func(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error)
	RefreshFunc  func(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error)
	LogoutFunc   func(ctx context.Context, req *models.LogoutRequest) error
}

func (m *MockAuthService) Register(ctx context.Context, req *models.CreateUserRequest) (*models.AuthResponse, error) {
	if m.RegisterFunc != nil {
		return m.RegisterFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockAuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockAuthService) Refresh(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error) {
	if m.RefreshFunc != nil {
		return m.RefreshFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockAuthService) Logout(ctx context.Context, req *models.LogoutRequest) error {
	if m.LogoutFunc != nil {
		return m.LogoutFunc(ctx, req)
	}
	return nil
}

// MockUserService is a mock implementation of UserServicer.
type MockUserService struct {
	GetUserFunc     func(ctx context.Context, id string) (*models.User, error)
	GetAllUsersFunc func(ctx context.Context) ([]models.User, error)
	UpdateUserFunc  func(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error)
	DeleteUserFunc  func(ctx context.Context, id string) error
}

func (m *MockUserService) GetUser(ctx context.Context, id string) (*models.User, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUserService) GetAllUsers(ctx context.Context) ([]models.User, error) {
	if m.GetAllUsersFunc != nil {
		return m.GetAllUsersFunc(ctx)
	}
	return nil, nil
}

func (m *MockUserService) UpdateUser(ctx context.Context, id string, req *models.UpdateUserRequest) (*models.User, error) {
	if m.UpdateUserFunc != nil {
		return m.UpdateUserFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *MockUserService) DeleteUser(ctx context.Context, id string) error {
	if m.DeleteUserFunc != nil {
		return m.DeleteUserFunc(ctx, id)
	}
	return nil
}

// MockTeamService is a mock implementation of TeamServicer.
type MockTeamService struct {
	CreateTeamFunc        func(ctx context.Context, userID primitive.ObjectID, req *models.CreateTeamRequest) (*models.Team, error)
	ListTeamsFunc         func(ctx context.Context, userID primitive.ObjectID, page, limit int) (*models.TeamListResponse, error)
	GetTeamFunc           func(ctx context.Context, teamID primitive.ObjectID) (*models.Team, error)
	UpdateTeamFunc        func(ctx context.Context, teamID primitive.ObjectID, req *models.UpdateTeamRequest) (*models.Team, error)
	DeleteTeamFunc        func(ctx context.Context, teamID primitive.ObjectID) error
	TransferOwnershipFunc func(ctx context.Context, teamID, currentOwnerID, newOwnerID primitive.ObjectID) error
}

func (m *MockTeamService) CreateTeam(ctx context.Context, userID primitive.ObjectID, req *models.CreateTeamRequest) (*models.Team, error) {
	if m.CreateTeamFunc != nil {
		return m.CreateTeamFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *MockTeamService) ListTeams(ctx context.Context, userID primitive.ObjectID, page, limit int) (*models.TeamListResponse, error) {
	if m.ListTeamsFunc != nil {
		return m.ListTeamsFunc(ctx, userID, page, limit)
	}
	return nil, nil
}

func (m *MockTeamService) GetTeam(ctx context.Context, teamID primitive.ObjectID) (*models.Team, error) {
	if m.GetTeamFunc != nil {
		return m.GetTeamFunc(ctx, teamID)
	}
	return nil, nil
}

func (m *MockTeamService) UpdateTeam(ctx context.Context, teamID primitive.ObjectID, req *models.UpdateTeamRequest) (*models.Team, error) {
	if m.UpdateTeamFunc != nil {
		return m.UpdateTeamFunc(ctx, teamID, req)
	}
	return nil, nil
}

func (m *MockTeamService) DeleteTeam(ctx context.Context, teamID primitive.ObjectID) error {
	if m.DeleteTeamFunc != nil {
		return m.DeleteTeamFunc(ctx, teamID)
	}
	return nil
}

func (m *MockTeamService) TransferOwnership(ctx context.Context, teamID, currentOwnerID, newOwnerID primitive.ObjectID) error {
	if m.TransferOwnershipFunc != nil {
		return m.TransferOwnershipFunc(ctx, teamID, currentOwnerID, newOwnerID)
	}
	return nil
}

// MockTeamMemberService is a mock implementation of TeamMemberServicer.
type MockTeamMemberService struct {
	ListMembersFunc  func(ctx context.Context, teamID primitive.ObjectID) (*models.TeamMemberListResponse, error)
	RemoveMemberFunc func(ctx context.Context, teamID, targetUserID, requestingUserID primitive.ObjectID) error
	UpdateRoleFunc   func(ctx context.Context, teamID, targetUserID, requestingUserID primitive.ObjectID, newRole string) error
	LeaveTeamFunc    func(ctx context.Context, teamID, userID primitive.ObjectID) error
	GetMemberFunc    func(ctx context.Context, teamID, userID primitive.ObjectID) (*models.TeamMember, error)
}

func (m *MockTeamMemberService) ListMembers(ctx context.Context, teamID primitive.ObjectID) (*models.TeamMemberListResponse, error) {
	if m.ListMembersFunc != nil {
		return m.ListMembersFunc(ctx, teamID)
	}
	return nil, nil
}

func (m *MockTeamMemberService) RemoveMember(ctx context.Context, teamID, targetUserID, requestingUserID primitive.ObjectID) error {
	if m.RemoveMemberFunc != nil {
		return m.RemoveMemberFunc(ctx, teamID, targetUserID, requestingUserID)
	}
	return nil
}

func (m *MockTeamMemberService) UpdateRole(ctx context.Context, teamID, targetUserID, requestingUserID primitive.ObjectID, newRole string) error {
	if m.UpdateRoleFunc != nil {
		return m.UpdateRoleFunc(ctx, teamID, targetUserID, requestingUserID, newRole)
	}
	return nil
}

func (m *MockTeamMemberService) LeaveTeam(ctx context.Context, teamID, userID primitive.ObjectID) error {
	if m.LeaveTeamFunc != nil {
		return m.LeaveTeamFunc(ctx, teamID, userID)
	}
	return nil
}

func (m *MockTeamMemberService) GetMember(ctx context.Context, teamID, userID primitive.ObjectID) (*models.TeamMember, error) {
	if m.GetMemberFunc != nil {
		return m.GetMemberFunc(ctx, teamID, userID)
	}
	return nil, nil
}

// MockTeamInvitationService is a mock implementation of TeamInvitationServicer.
type MockTeamInvitationService struct {
	CreateInvitationFunc    func(ctx context.Context, teamID, inviterID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error)
	ListTeamInvitationsFunc func(ctx context.Context, teamID primitive.ObjectID) (*models.InvitationListResponse, error)
	CancelInvitationFunc    func(ctx context.Context, invitationID, teamID primitive.ObjectID) error
	ListMyInvitationsFunc   func(ctx context.Context, userEmail string) (*models.MyInvitationListResponse, error)
	AcceptInvitationFunc    func(ctx context.Context, invitationID, userID primitive.ObjectID, userEmail string) (*models.AcceptInvitationResponse, error)
	DeclineInvitationFunc   func(ctx context.Context, invitationID primitive.ObjectID, userEmail string) error
}

func (m *MockTeamInvitationService) CreateInvitation(ctx context.Context, teamID, inviterID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error) {
	if m.CreateInvitationFunc != nil {
		return m.CreateInvitationFunc(ctx, teamID, inviterID, req)
	}
	return nil, nil
}

func (m *MockTeamInvitationService) ListTeamInvitations(ctx context.Context, teamID primitive.ObjectID) (*models.InvitationListResponse, error) {
	if m.ListTeamInvitationsFunc != nil {
		return m.ListTeamInvitationsFunc(ctx, teamID)
	}
	return nil, nil
}

func (m *MockTeamInvitationService) CancelInvitation(ctx context.Context, invitationID, teamID primitive.ObjectID) error {
	if m.CancelInvitationFunc != nil {
		return m.CancelInvitationFunc(ctx, invitationID, teamID)
	}
	return nil
}

func (m *MockTeamInvitationService) ListMyInvitations(ctx context.Context, userEmail string) (*models.MyInvitationListResponse, error) {
	if m.ListMyInvitationsFunc != nil {
		return m.ListMyInvitationsFunc(ctx, userEmail)
	}
	return nil, nil
}

func (m *MockTeamInvitationService) AcceptInvitation(ctx context.Context, invitationID, userID primitive.ObjectID, userEmail string) (*models.AcceptInvitationResponse, error) {
	if m.AcceptInvitationFunc != nil {
		return m.AcceptInvitationFunc(ctx, invitationID, userID, userEmail)
	}
	return nil, nil
}

func (m *MockTeamInvitationService) DeclineInvitation(ctx context.Context, invitationID primitive.ObjectID, userEmail string) error {
	if m.DeclineInvitationFunc != nil {
		return m.DeclineInvitationFunc(ctx, invitationID, userEmail)
	}
	return nil
}

// MockVoiceMemoService is a mock implementation of VoiceMemoServicer.
type MockVoiceMemoService struct {
	ListByUserIDFunc           func(ctx context.Context, userID string, page, limit int) (*models.VoiceMemoListResponse, error)
	CreateVoiceMemoFunc        func(ctx context.Context, userID primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error)
	GetVoiceMemoFunc           func(ctx context.Context, memoID primitive.ObjectID) (*models.VoiceMemo, error)
	DeleteVoiceMemoFunc        func(ctx context.Context, memoID, userID primitive.ObjectID) error
	ConfirmUploadFunc          func(ctx context.Context, memoID, userID primitive.ObjectID) error
	RetryTranscriptionFunc     func(ctx context.Context, memoID, userID primitive.ObjectID) error
	ListByTeamIDFunc           func(ctx context.Context, teamID string, page, limit int) (*models.VoiceMemoListResponse, error)
	CreateTeamVoiceMemoFunc    func(ctx context.Context, userID, teamID primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error)
	DeleteTeamVoiceMemoFunc    func(ctx context.Context, memoID, teamID primitive.ObjectID) error
	ConfirmTeamUploadFunc      func(ctx context.Context, memoID, teamID primitive.ObjectID) error
	RetryTeamTranscriptionFunc func(ctx context.Context, memoID, teamID primitive.ObjectID) error
}

func (m *MockVoiceMemoService) ListByUserID(ctx context.Context, userID string, page, limit int) (*models.VoiceMemoListResponse, error) {
	if m.ListByUserIDFunc != nil {
		return m.ListByUserIDFunc(ctx, userID, page, limit)
	}
	return nil, nil
}

func (m *MockVoiceMemoService) CreateVoiceMemo(ctx context.Context, userID primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error) {
	if m.CreateVoiceMemoFunc != nil {
		return m.CreateVoiceMemoFunc(ctx, userID, req)
	}
	return nil, nil
}

func (m *MockVoiceMemoService) GetVoiceMemo(ctx context.Context, memoID primitive.ObjectID) (*models.VoiceMemo, error) {
	if m.GetVoiceMemoFunc != nil {
		return m.GetVoiceMemoFunc(ctx, memoID)
	}
	return nil, nil
}

func (m *MockVoiceMemoService) DeleteVoiceMemo(ctx context.Context, memoID, userID primitive.ObjectID) error {
	if m.DeleteVoiceMemoFunc != nil {
		return m.DeleteVoiceMemoFunc(ctx, memoID, userID)
	}
	return nil
}

func (m *MockVoiceMemoService) ConfirmUpload(ctx context.Context, memoID, userID primitive.ObjectID) error {
	if m.ConfirmUploadFunc != nil {
		return m.ConfirmUploadFunc(ctx, memoID, userID)
	}
	return nil
}

func (m *MockVoiceMemoService) RetryTranscription(ctx context.Context, memoID, userID primitive.ObjectID) error {
	if m.RetryTranscriptionFunc != nil {
		return m.RetryTranscriptionFunc(ctx, memoID, userID)
	}
	return nil
}

func (m *MockVoiceMemoService) ListByTeamID(ctx context.Context, teamID string, page, limit int) (*models.VoiceMemoListResponse, error) {
	if m.ListByTeamIDFunc != nil {
		return m.ListByTeamIDFunc(ctx, teamID, page, limit)
	}
	return nil, nil
}

func (m *MockVoiceMemoService) CreateTeamVoiceMemo(ctx context.Context, userID, teamID primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error) {
	if m.CreateTeamVoiceMemoFunc != nil {
		return m.CreateTeamVoiceMemoFunc(ctx, userID, teamID, req)
	}
	return nil, nil
}

func (m *MockVoiceMemoService) DeleteTeamVoiceMemo(ctx context.Context, memoID, teamID primitive.ObjectID) error {
	if m.DeleteTeamVoiceMemoFunc != nil {
		return m.DeleteTeamVoiceMemoFunc(ctx, memoID, teamID)
	}
	return nil
}

func (m *MockVoiceMemoService) ConfirmTeamUpload(ctx context.Context, memoID, teamID primitive.ObjectID) error {
	if m.ConfirmTeamUploadFunc != nil {
		return m.ConfirmTeamUploadFunc(ctx, memoID, teamID)
	}
	return nil
}

func (m *MockVoiceMemoService) RetryTeamTranscription(ctx context.Context, memoID, teamID primitive.ObjectID) error {
	if m.RetryTeamTranscriptionFunc != nil {
		return m.RetryTeamTranscriptionFunc(ctx, memoID, teamID)
	}
	return nil
}
