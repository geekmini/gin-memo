// Package errors provides custom error types for the application.
package errors

import "errors"

// User errors
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// Auth errors
var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenExpired        = errors.New("token expired")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
)

// Voice memo errors
var (
	ErrVoiceMemoNotFound      = errors.New("voice memo not found")
	ErrVoiceMemoUnauthorized  = errors.New("you can only delete your own voice memos")
	ErrVoiceMemoInvalidStatus = errors.New("invalid voice memo status transition")
	ErrTranscriptionQueueFull = errors.New("transcription queue is full, please try again later")
)

// Team errors
var (
	ErrTeamNotFound            = errors.New("team not found")
	ErrTeamSlugTaken           = errors.New("team slug is already taken")
	ErrTeamLimitReached        = errors.New("free users can only create 1 team")
	ErrNotTeamMember           = errors.New("you are not a member of this team")
	ErrInsufficientPermissions = errors.New("insufficient permissions")
	ErrOwnerCannotLeave        = errors.New("owner must transfer ownership before leaving")
	ErrCannotRemoveOwner       = errors.New("cannot remove team owner")
	ErrCannotRemoveSelf        = errors.New("cannot remove yourself, use leave endpoint")
	ErrCannotChangeOwnerRole   = errors.New("cannot change owner role, use transfer")
	ErrSeatsExceeded           = errors.New("team seats limit exceeded")
	ErrInvalidRole             = errors.New("invalid role, must be admin or member")
)

// Invitation errors
var (
	ErrInvitationNotFound      = errors.New("invitation not found")
	ErrInvitationExpired       = errors.New("invitation has expired")
	ErrInvitationEmailMismatch = errors.New("invitation email does not match your account")
	ErrAlreadyMember           = errors.New("user is already a team member")
	ErrPendingInvitation       = errors.New("invitation already pending for this email")
)
