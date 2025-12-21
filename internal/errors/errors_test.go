package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrUserNotFound", ErrUserNotFound, "user not found"},
		{"ErrUserAlreadyExists", ErrUserAlreadyExists, "user with this email already exists"},
		{"ErrInvalidCredentials", ErrInvalidCredentials, "invalid email or password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestAuthErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrUnauthorized", ErrUnauthorized, "unauthorized"},
		{"ErrInvalidToken", ErrInvalidToken, "invalid token"},
		{"ErrTokenExpired", ErrTokenExpired, "token expired"},
		{"ErrInvalidRefreshToken", ErrInvalidRefreshToken, "invalid or expired refresh token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestVoiceMemoErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrVoiceMemoNotFound", ErrVoiceMemoNotFound, "voice memo not found"},
		{"ErrVoiceMemoUnauthorized", ErrVoiceMemoUnauthorized, "you can only delete your own voice memos"},
		{"ErrVoiceMemoInvalidStatus", ErrVoiceMemoInvalidStatus, "invalid voice memo status transition"},
		{"ErrTranscriptionQueueFull", ErrTranscriptionQueueFull, "transcription queue is full, please try again later"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestTeamErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrTeamNotFound", ErrTeamNotFound, "team not found"},
		{"ErrTeamSlugTaken", ErrTeamSlugTaken, "team slug is already taken"},
		{"ErrTeamLimitReached", ErrTeamLimitReached, "free users can only create 1 team"},
		{"ErrNotTeamMember", ErrNotTeamMember, "you are not a member of this team"},
		{"ErrInsufficientPermissions", ErrInsufficientPermissions, "insufficient permissions"},
		{"ErrOwnerCannotLeave", ErrOwnerCannotLeave, "owner must transfer ownership before leaving"},
		{"ErrCannotRemoveOwner", ErrCannotRemoveOwner, "cannot remove team owner"},
		{"ErrCannotRemoveSelf", ErrCannotRemoveSelf, "cannot remove yourself, use leave endpoint"},
		{"ErrCannotChangeOwnerRole", ErrCannotChangeOwnerRole, "cannot change owner role, use transfer"},
		{"ErrSeatsExceeded", ErrSeatsExceeded, "team seats limit exceeded"},
		{"ErrInvalidRole", ErrInvalidRole, "invalid role, must be admin or member"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestInvitationErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrInvitationNotFound", ErrInvitationNotFound, "invitation not found"},
		{"ErrInvitationExpired", ErrInvitationExpired, "invitation has expired"},
		{"ErrInvitationEmailMismatch", ErrInvitationEmailMismatch, "invitation email does not match your account"},
		{"ErrAlreadyMember", ErrAlreadyMember, "user is already a team member"},
		{"ErrPendingInvitation", ErrPendingInvitation, "invitation already pending for this email"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestErrorsIsComparison(t *testing.T) {
	// Test that errors.Is works correctly with our sentinel errors
	tests := []struct {
		name   string
		target error
		err    error
		want   bool
	}{
		{"same error", ErrUserNotFound, ErrUserNotFound, true},
		{"different error", ErrUserNotFound, ErrUserAlreadyExists, false},
		{"wrapped error", ErrUserNotFound, errors.New("wrapped: " + ErrUserNotFound.Error()), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errors.Is(tt.err, tt.target)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestAllErrorsAreUnique(t *testing.T) {
	allErrors := []error{
		// User errors
		ErrUserNotFound,
		ErrUserAlreadyExists,
		ErrInvalidCredentials,
		// Auth errors
		ErrUnauthorized,
		ErrInvalidToken,
		ErrTokenExpired,
		ErrInvalidRefreshToken,
		// Voice memo errors
		ErrVoiceMemoNotFound,
		ErrVoiceMemoUnauthorized,
		ErrVoiceMemoInvalidStatus,
		ErrTranscriptionQueueFull,
		// Team errors
		ErrTeamNotFound,
		ErrTeamSlugTaken,
		ErrTeamLimitReached,
		ErrNotTeamMember,
		ErrInsufficientPermissions,
		ErrOwnerCannotLeave,
		ErrCannotRemoveOwner,
		ErrCannotRemoveSelf,
		ErrCannotChangeOwnerRole,
		ErrSeatsExceeded,
		ErrInvalidRole,
		// Invitation errors
		ErrInvitationNotFound,
		ErrInvitationExpired,
		ErrInvitationEmailMismatch,
		ErrAlreadyMember,
		ErrPendingInvitation,
	}

	// Check that all error messages are unique
	seen := make(map[string]bool)
	for _, err := range allErrors {
		msg := err.Error()
		if seen[msg] {
			t.Errorf("duplicate error message found: %s", msg)
		}
		seen[msg] = true
	}
}
