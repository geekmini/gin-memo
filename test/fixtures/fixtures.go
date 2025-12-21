// Package fixtures provides test data builders for unit and integration tests.
package fixtures

import (
	"fmt"
	"time"

	"gin-sample/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ===== User Fixtures =====

// UserBuilder provides fluent API for building test users.
type UserBuilder struct {
	user models.User
}

// NewUser creates a new UserBuilder with sensible defaults.
func NewUser() *UserBuilder {
	return &UserBuilder{
		user: models.User{
			ID:        primitive.NewObjectID(),
			Name:      "Test User",
			Email:     fmt.Sprintf("test-%s@example.com", primitive.NewObjectID().Hex()[:8]),
			Password:  "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // "password123" hashed
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}

func (b *UserBuilder) WithID(id primitive.ObjectID) *UserBuilder {
	b.user.ID = id
	return b
}

func (b *UserBuilder) WithName(name string) *UserBuilder {
	b.user.Name = name
	return b
}

func (b *UserBuilder) WithEmail(email string) *UserBuilder {
	b.user.Email = email
	return b
}

func (b *UserBuilder) WithPassword(password string) *UserBuilder {
	b.user.Password = password
	return b
}

func (b *UserBuilder) Build() models.User {
	return b.user
}

func (b *UserBuilder) BuildPtr() *models.User {
	return &b.user
}

// ===== Team Fixtures =====

// TeamBuilder provides fluent API for building test teams.
type TeamBuilder struct {
	team models.Team
}

// NewTeam creates a new TeamBuilder with sensible defaults.
func NewTeam() *TeamBuilder {
	ownerID := primitive.NewObjectID()
	return &TeamBuilder{
		team: models.Team{
			ID:            primitive.NewObjectID(),
			Name:          "Test Team",
			Slug:          fmt.Sprintf("test-team-%s", primitive.NewObjectID().Hex()[:8]),
			Description:   "A test team",
			LogoURL:       "",
			OwnerID:       ownerID,
			Seats:         10,
			RetentionDays: 30,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			DeletedAt:     nil,
		},
	}
}

func (b *TeamBuilder) WithID(id primitive.ObjectID) *TeamBuilder {
	b.team.ID = id
	return b
}

func (b *TeamBuilder) WithName(name string) *TeamBuilder {
	b.team.Name = name
	return b
}

func (b *TeamBuilder) WithSlug(slug string) *TeamBuilder {
	b.team.Slug = slug
	return b
}

func (b *TeamBuilder) WithOwnerID(ownerID primitive.ObjectID) *TeamBuilder {
	b.team.OwnerID = ownerID
	return b
}

func (b *TeamBuilder) WithSeats(seats int) *TeamBuilder {
	b.team.Seats = seats
	return b
}

func (b *TeamBuilder) Deleted() *TeamBuilder {
	now := time.Now()
	b.team.DeletedAt = &now
	return b
}

func (b *TeamBuilder) Build() models.Team {
	return b.team
}

func (b *TeamBuilder) BuildPtr() *models.Team {
	return &b.team
}

// ===== TeamMember Fixtures =====

// TeamMemberBuilder provides fluent API for building test team members.
type TeamMemberBuilder struct {
	member models.TeamMember
}

// NewTeamMember creates a new TeamMemberBuilder with sensible defaults.
func NewTeamMember() *TeamMemberBuilder {
	return &TeamMemberBuilder{
		member: models.TeamMember{
			ID:       primitive.NewObjectID(),
			TeamID:   primitive.NewObjectID(),
			UserID:   primitive.NewObjectID(),
			Role:     models.RoleMember,
			JoinedAt: time.Now(),
		},
	}
}

func (b *TeamMemberBuilder) WithID(id primitive.ObjectID) *TeamMemberBuilder {
	b.member.ID = id
	return b
}

func (b *TeamMemberBuilder) WithTeamID(teamID primitive.ObjectID) *TeamMemberBuilder {
	b.member.TeamID = teamID
	return b
}

func (b *TeamMemberBuilder) WithUserID(userID primitive.ObjectID) *TeamMemberBuilder {
	b.member.UserID = userID
	return b
}

func (b *TeamMemberBuilder) WithRole(role string) *TeamMemberBuilder {
	b.member.Role = role
	return b
}

func (b *TeamMemberBuilder) AsOwner() *TeamMemberBuilder {
	b.member.Role = models.RoleOwner
	return b
}

func (b *TeamMemberBuilder) AsAdmin() *TeamMemberBuilder {
	b.member.Role = models.RoleAdmin
	return b
}

func (b *TeamMemberBuilder) AsMember() *TeamMemberBuilder {
	b.member.Role = models.RoleMember
	return b
}

func (b *TeamMemberBuilder) Build() models.TeamMember {
	return b.member
}

func (b *TeamMemberBuilder) BuildPtr() *models.TeamMember {
	return &b.member
}

// ===== TeamInvitation Fixtures =====

// TeamInvitationBuilder provides fluent API for building test team invitations.
type TeamInvitationBuilder struct {
	invitation models.TeamInvitation
}

// NewTeamInvitation creates a new TeamInvitationBuilder with sensible defaults.
func NewTeamInvitation() *TeamInvitationBuilder {
	return &TeamInvitationBuilder{
		invitation: models.TeamInvitation{
			ID:        primitive.NewObjectID(),
			TeamID:    primitive.NewObjectID(),
			Email:     fmt.Sprintf("invited-%s@example.com", primitive.NewObjectID().Hex()[:8]),
			InvitedBy: primitive.NewObjectID(),
			Role:      models.RoleMember,
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days from now
			CreatedAt: time.Now(),
		},
	}
}

func (b *TeamInvitationBuilder) WithID(id primitive.ObjectID) *TeamInvitationBuilder {
	b.invitation.ID = id
	return b
}

func (b *TeamInvitationBuilder) WithTeamID(teamID primitive.ObjectID) *TeamInvitationBuilder {
	b.invitation.TeamID = teamID
	return b
}

func (b *TeamInvitationBuilder) WithEmail(email string) *TeamInvitationBuilder {
	b.invitation.Email = email
	return b
}

func (b *TeamInvitationBuilder) WithInvitedBy(userID primitive.ObjectID) *TeamInvitationBuilder {
	b.invitation.InvitedBy = userID
	return b
}

func (b *TeamInvitationBuilder) WithRole(role string) *TeamInvitationBuilder {
	b.invitation.Role = role
	return b
}

func (b *TeamInvitationBuilder) Expired() *TeamInvitationBuilder {
	b.invitation.ExpiresAt = time.Now().Add(-24 * time.Hour) // Expired 1 day ago
	return b
}

func (b *TeamInvitationBuilder) Build() models.TeamInvitation {
	return b.invitation
}

func (b *TeamInvitationBuilder) BuildPtr() *models.TeamInvitation {
	return &b.invitation
}

// ===== VoiceMemo Fixtures =====

// VoiceMemoBuilder provides fluent API for building test voice memos.
type VoiceMemoBuilder struct {
	memo models.VoiceMemo
}

// NewVoiceMemo creates a new VoiceMemoBuilder with sensible defaults.
func NewVoiceMemo() *VoiceMemoBuilder {
	return &VoiceMemoBuilder{
		memo: models.VoiceMemo{
			ID:            primitive.NewObjectID(),
			UserID:        primitive.NewObjectID(),
			TeamID:        nil, // Private memo by default
			Title:         "Test Voice Memo",
			Transcription: "This is a test transcription.",
			AudioFileKey:  fmt.Sprintf("voice-memos/%s.mp3", primitive.NewObjectID().Hex()),
			AudioFileURL:  "", // Set dynamically
			Duration:      120,
			FileSize:      1048576, // 1MB
			AudioFormat:   "mp3",
			Tags:          []string{"test"},
			IsFavorite:    false,
			Status:        models.StatusReady,
			Version:       1,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			DeletedAt:     nil,
		},
	}
}

func (b *VoiceMemoBuilder) WithID(id primitive.ObjectID) *VoiceMemoBuilder {
	b.memo.ID = id
	return b
}

func (b *VoiceMemoBuilder) WithUserID(userID primitive.ObjectID) *VoiceMemoBuilder {
	b.memo.UserID = userID
	return b
}

func (b *VoiceMemoBuilder) WithTeamID(teamID primitive.ObjectID) *VoiceMemoBuilder {
	b.memo.TeamID = &teamID
	return b
}

func (b *VoiceMemoBuilder) Private() *VoiceMemoBuilder {
	b.memo.TeamID = nil
	return b
}

func (b *VoiceMemoBuilder) WithTitle(title string) *VoiceMemoBuilder {
	b.memo.Title = title
	return b
}

func (b *VoiceMemoBuilder) WithTranscription(text string) *VoiceMemoBuilder {
	b.memo.Transcription = text
	return b
}

func (b *VoiceMemoBuilder) WithStatus(status models.VoiceMemoStatus) *VoiceMemoBuilder {
	b.memo.Status = status
	return b
}

func (b *VoiceMemoBuilder) PendingUpload() *VoiceMemoBuilder {
	b.memo.Status = models.StatusPendingUpload
	return b
}

func (b *VoiceMemoBuilder) Transcribing() *VoiceMemoBuilder {
	b.memo.Status = models.StatusTranscribing
	return b
}

func (b *VoiceMemoBuilder) Ready() *VoiceMemoBuilder {
	b.memo.Status = models.StatusReady
	return b
}

func (b *VoiceMemoBuilder) Failed() *VoiceMemoBuilder {
	b.memo.Status = models.StatusFailed
	return b
}

func (b *VoiceMemoBuilder) WithVersion(version int) *VoiceMemoBuilder {
	b.memo.Version = version
	return b
}

func (b *VoiceMemoBuilder) WithTags(tags []string) *VoiceMemoBuilder {
	b.memo.Tags = tags
	return b
}

func (b *VoiceMemoBuilder) Favorite() *VoiceMemoBuilder {
	b.memo.IsFavorite = true
	return b
}

func (b *VoiceMemoBuilder) Deleted() *VoiceMemoBuilder {
	now := time.Now()
	b.memo.DeletedAt = &now
	return b
}

func (b *VoiceMemoBuilder) Build() models.VoiceMemo {
	return b.memo
}

func (b *VoiceMemoBuilder) BuildPtr() *models.VoiceMemo {
	return &b.memo
}

// ===== RefreshToken Fixtures =====

// RefreshTokenBuilder provides fluent API for building test refresh tokens.
type RefreshTokenBuilder struct {
	token models.RefreshToken
}

// NewRefreshToken creates a new RefreshTokenBuilder with sensible defaults.
func NewRefreshToken() *RefreshTokenBuilder {
	return &RefreshTokenBuilder{
		token: models.RefreshToken{
			ID:        primitive.NewObjectID(),
			Token:     fmt.Sprintf("rf_%s", primitive.NewObjectID().Hex()),
			UserID:    primitive.NewObjectID(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days from now
			CreatedAt: time.Now(),
		},
	}
}

func (b *RefreshTokenBuilder) WithID(id primitive.ObjectID) *RefreshTokenBuilder {
	b.token.ID = id
	return b
}

func (b *RefreshTokenBuilder) WithToken(token string) *RefreshTokenBuilder {
	b.token.Token = token
	return b
}

func (b *RefreshTokenBuilder) WithUserID(userID primitive.ObjectID) *RefreshTokenBuilder {
	b.token.UserID = userID
	return b
}

func (b *RefreshTokenBuilder) Expired() *RefreshTokenBuilder {
	b.token.ExpiresAt = time.Now().Add(-24 * time.Hour) // Expired 1 day ago
	return b
}

func (b *RefreshTokenBuilder) Build() models.RefreshToken {
	return b.token
}

func (b *RefreshTokenBuilder) BuildPtr() *models.RefreshToken {
	return &b.token
}
