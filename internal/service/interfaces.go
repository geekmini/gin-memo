// Package service contains business logic for the application.
package service

import (
	"context"

	"gin-sample/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthServicer defines the interface for authentication operations.
type AuthServicer interface {
	Register(ctx context.Context, req *models.CreateUserRequest) (*models.AuthResponse, error)
	Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error)
	Refresh(ctx context.Context, req *models.RefreshRequest) (*models.RefreshResponse, error)
	Logout(ctx context.Context, req *models.LogoutRequest) error
	LogoutAll(ctx context.Context, userID primitive.ObjectID) error
}

// UserServicer defines the interface for user operations.
type UserServicer interface {
	GetUser(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	GetAllUsers(ctx context.Context) ([]models.User, error)
	UpdateUser(ctx context.Context, id primitive.ObjectID, req *models.UpdateUserRequest) (*models.User, error)
	DeleteUser(ctx context.Context, id primitive.ObjectID) error
}

// TeamServicer defines the interface for team operations.
type TeamServicer interface {
	CreateTeam(ctx context.Context, userID primitive.ObjectID, req *models.CreateTeamRequest) (*models.Team, error)
	ListTeams(ctx context.Context, userID primitive.ObjectID, page, limit int) (*models.TeamListResponse, error)
	GetTeam(ctx context.Context, teamID primitive.ObjectID) (*models.Team, error)
	UpdateTeam(ctx context.Context, teamID primitive.ObjectID, req *models.UpdateTeamRequest) (*models.Team, error)
	DeleteTeam(ctx context.Context, teamID primitive.ObjectID) error
	TransferOwnership(ctx context.Context, teamID, currentOwnerID, newOwnerID primitive.ObjectID) error
}

// TeamMemberServicer defines the interface for team member operations.
type TeamMemberServicer interface {
	ListMembers(ctx context.Context, teamID primitive.ObjectID) (*models.TeamMemberListResponse, error)
	RemoveMember(ctx context.Context, teamID, targetUserID, requestingUserID primitive.ObjectID) error
	UpdateRole(ctx context.Context, teamID, targetUserID, requestingUserID primitive.ObjectID, newRole string) error
	LeaveTeam(ctx context.Context, teamID, userID primitive.ObjectID) error
	GetMember(ctx context.Context, teamID, userID primitive.ObjectID) (*models.TeamMember, error)
}

// TeamInvitationServicer defines the interface for invitation operations.
type TeamInvitationServicer interface {
	CreateInvitation(ctx context.Context, teamID, inviterID primitive.ObjectID, req *models.CreateInvitationRequest) (*models.TeamInvitation, error)
	ListTeamInvitations(ctx context.Context, teamID primitive.ObjectID) (*models.InvitationListResponse, error)
	CancelInvitation(ctx context.Context, invitationID, teamID primitive.ObjectID) error
	ListMyInvitations(ctx context.Context, userEmail string) (*models.MyInvitationListResponse, error)
	AcceptInvitation(ctx context.Context, invitationID, userID primitive.ObjectID, userEmail string) (*models.AcceptInvitationResponse, error)
	DeclineInvitation(ctx context.Context, invitationID primitive.ObjectID, userEmail string) error
}

// VoiceMemoServicer defines the interface for voice memo operations.
type VoiceMemoServicer interface {
	// Private voice memo operations
	ListByUserID(ctx context.Context, userID string, page, limit int) (*models.VoiceMemoListResponse, error)
	CreateVoiceMemo(ctx context.Context, userID primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error)
	GetVoiceMemo(ctx context.Context, memoID primitive.ObjectID) (*models.VoiceMemo, error)
	DeleteVoiceMemo(ctx context.Context, memoID, userID primitive.ObjectID) error
	ConfirmUpload(ctx context.Context, memoID, userID primitive.ObjectID) error
	RetryTranscription(ctx context.Context, memoID, userID primitive.ObjectID) error

	// Team voice memo operations
	ListByTeamID(ctx context.Context, teamID string, page, limit int) (*models.VoiceMemoListResponse, error)
	CreateTeamVoiceMemo(ctx context.Context, userID, teamID primitive.ObjectID, req *models.CreateVoiceMemoRequest) (*models.CreateVoiceMemoResponse, error)
	DeleteTeamVoiceMemo(ctx context.Context, memoID, teamID primitive.ObjectID) error
	ConfirmTeamUpload(ctx context.Context, memoID, teamID primitive.ObjectID) error
	RetryTeamTranscription(ctx context.Context, memoID, teamID primitive.ObjectID) error
}

// Ensure concrete types implement interfaces
var (
	_ AuthServicer           = (*AuthService)(nil)
	_ UserServicer           = (*UserService)(nil)
	_ TeamServicer           = (*TeamService)(nil)
	_ TeamMemberServicer     = (*TeamMemberService)(nil)
	_ TeamInvitationServicer = (*TeamInvitationService)(nil)
	_ VoiceMemoServicer      = (*VoiceMemoService)(nil)
)
