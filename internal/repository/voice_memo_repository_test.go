package repository

import (
	"context"
	"testing"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestNewVoiceMemoRepository(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)

	assert.NotNil(t, repo)
}

func TestVoiceMemoRepository_Create(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("creates memo with generated ID", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Test Memo",
			AudioFileKey: "voice-memos/test.mp3",
			Status:       models.StatusPendingUpload,
		}

		err := repo.Create(ctx, memo)

		require.NoError(t, err)
		assert.False(t, memo.ID.IsZero())
		assert.NotZero(t, memo.CreatedAt)
		assert.NotZero(t, memo.UpdatedAt)
		assert.Equal(t, 0, memo.Version)
	})

	t.Run("preserves existing ID if set", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		existingID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			ID:           existingID,
			UserID:       primitive.NewObjectID(),
			Title:        "Preset ID Memo",
			AudioFileKey: "voice-memos/preset.mp3",
			Status:       models.StatusPendingUpload,
		}

		err := repo.Create(ctx, memo)

		require.NoError(t, err)
		assert.Equal(t, existingID, memo.ID)
	})
}

func TestVoiceMemoRepository_FindByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("finds existing memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Find Me",
			AudioFileKey: "voice-memos/findme.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		found, err := repo.FindByID(ctx, memo.ID)

		require.NoError(t, err)
		assert.Equal(t, memo.ID, found.ID)
		assert.Equal(t, "Find Me", found.Title)
	})

	t.Run("returns error for non-existent memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		found, err := repo.FindByID(ctx, primitive.NewObjectID())

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("excludes soft-deleted memos", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Deleted Memo",
			AudioFileKey: "voice-memos/deleted.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))
		require.NoError(t, repo.SoftDeleteByID(ctx, memo.ID))

		found, err := repo.FindByID(ctx, memo.ID)

		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}

func TestVoiceMemoRepository_FindByUserID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("returns paginated memos for user", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		userID := primitive.NewObjectID()

		// Create 5 memos for user
		for i := 0; i < 5; i++ {
			memo := &models.VoiceMemo{
				UserID:       userID,
				Title:        "Memo " + string(rune('A'+i)),
				AudioFileKey: "voice-memos/memo" + string(rune('a'+i)) + ".mp3",
				Status:       models.StatusReady,
			}
			require.NoError(t, repo.Create(ctx, memo))
		}

		memos, total, err := repo.FindByUserID(ctx, userID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 5, total)
		assert.Len(t, memos, 5)
	})

	t.Run("excludes team memos", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		userID := primitive.NewObjectID()
		teamID := primitive.NewObjectID()

		// Create private memo
		privateMemo := &models.VoiceMemo{
			UserID:       userID,
			Title:        "Private Memo",
			AudioFileKey: "voice-memos/private.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, privateMemo))

		// Create team memo
		teamMemo := &models.VoiceMemo{
			UserID:       userID,
			TeamID:       &teamID,
			Title:        "Team Memo",
			AudioFileKey: "voice-memos/team.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, teamMemo))

		memos, total, err := repo.FindByUserID(ctx, userID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, memos, 1)
		assert.Equal(t, "Private Memo", memos[0].Title)
	})

	t.Run("excludes soft-deleted memos", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		userID := primitive.NewObjectID()

		memo1 := &models.VoiceMemo{
			UserID:       userID,
			Title:        "Active Memo",
			AudioFileKey: "voice-memos/active.mp3",
			Status:       models.StatusReady,
		}
		memo2 := &models.VoiceMemo{
			UserID:       userID,
			Title:        "Deleted Memo",
			AudioFileKey: "voice-memos/deleted.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo1))
		require.NoError(t, repo.Create(ctx, memo2))
		require.NoError(t, repo.SoftDeleteByID(ctx, memo2.ID))

		memos, total, err := repo.FindByUserID(ctx, userID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, memos, 1)
	})

	t.Run("paginates correctly", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		userID := primitive.NewObjectID()

		for i := 0; i < 10; i++ {
			memo := &models.VoiceMemo{
				UserID:       userID,
				Title:        "Memo " + string(rune('A'+i)),
				AudioFileKey: "voice-memos/memo" + string(rune('a'+i)) + ".mp3",
				Status:       models.StatusReady,
			}
			require.NoError(t, repo.Create(ctx, memo))
		}

		// Page 1
		page1, total, err := repo.FindByUserID(ctx, userID, 1, 3)
		require.NoError(t, err)
		assert.Equal(t, 10, total)
		assert.Len(t, page1, 3)

		// Page 2
		page2, _, err := repo.FindByUserID(ctx, userID, 2, 3)
		require.NoError(t, err)
		assert.Len(t, page2, 3)

		// Page 4 (partial)
		page4, _, err := repo.FindByUserID(ctx, userID, 4, 3)
		require.NoError(t, err)
		assert.Len(t, page4, 1)
	})

	t.Run("returns empty slice when no memos", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memos, total, err := repo.FindByUserID(ctx, primitive.NewObjectID(), 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.NotNil(t, memos)
		assert.Len(t, memos, 0)
	})
}

func TestVoiceMemoRepository_FindByTeamID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("returns memos for team", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()

		for i := 0; i < 3; i++ {
			memo := &models.VoiceMemo{
				UserID:       primitive.NewObjectID(),
				TeamID:       &teamID,
				Title:        "Team Memo " + string(rune('A'+i)),
				AudioFileKey: "voice-memos/team" + string(rune('a'+i)) + ".mp3",
				Status:       models.StatusReady,
			}
			require.NoError(t, repo.Create(ctx, memo))
		}

		memos, total, err := repo.FindByTeamID(ctx, teamID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, memos, 3)
	})

	t.Run("excludes soft-deleted memos", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()

		memo1 := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &teamID,
			Title:        "Active",
			AudioFileKey: "voice-memos/active.mp3",
			Status:       models.StatusReady,
		}
		memo2 := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &teamID,
			Title:        "Deleted",
			AudioFileKey: "voice-memos/deleted.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo1))
		require.NoError(t, repo.Create(ctx, memo2))
		require.NoError(t, repo.SoftDeleteByID(ctx, memo2.ID))

		memos, total, err := repo.FindByTeamID(ctx, teamID, 1, 10)

		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, memos, 1)
	})
}

func TestVoiceMemoRepository_UpdateStatus(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("updates status", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Status Test",
			AudioFileKey: "voice-memos/status.mp3",
			Status:       models.StatusPendingUpload,
		}
		require.NoError(t, repo.Create(ctx, memo))

		err := repo.UpdateStatus(ctx, memo.ID, models.StatusTranscribing)

		require.NoError(t, err)

		found, err := repo.FindByID(ctx, memo.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusTranscribing, found.Status)
		assert.Equal(t, 1, found.Version)
	})

	t.Run("returns error for non-existent memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		err := repo.UpdateStatus(ctx, primitive.NewObjectID(), models.StatusReady)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}

func TestVoiceMemoRepository_UpdateStatusConditional(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("updates when status matches", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Conditional Test",
			AudioFileKey: "voice-memos/conditional.mp3",
			Status:       models.StatusTranscribing,
		}
		require.NoError(t, repo.Create(ctx, memo))

		err := repo.UpdateStatusConditional(ctx, memo.ID, models.StatusTranscribing, models.StatusPendingUpload)

		require.NoError(t, err)

		found, err := repo.FindByID(ctx, memo.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusPendingUpload, found.Status)
	})

	t.Run("silently succeeds when status doesn't match", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Already Changed",
			AudioFileKey: "voice-memos/changed.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		// Try to update from pending_upload to transcribing, but status is already ready
		err := repo.UpdateStatusConditional(ctx, memo.ID, models.StatusPendingUpload, models.StatusTranscribing)

		require.NoError(t, err) // Should succeed silently

		// Verify status unchanged
		found, err := repo.FindByID(ctx, memo.ID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusReady, found.Status)
	})
}

func TestVoiceMemoRepository_UpdateStatusWithOwnership(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("updates when user owns memo and status matches", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		userID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			UserID:       userID,
			Title:        "Owned Memo",
			AudioFileKey: "voice-memos/owned.mp3",
			Status:       models.StatusFailed,
		}
		require.NoError(t, repo.Create(ctx, memo))

		updated, err := repo.UpdateStatusWithOwnership(ctx, memo.ID, userID, models.StatusFailed, models.StatusTranscribing)

		require.NoError(t, err)
		assert.Equal(t, models.StatusTranscribing, updated.Status)
	})

	t.Run("returns error when user doesn't own memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		ownerID := primitive.NewObjectID()
		otherUserID := primitive.NewObjectID()

		memo := &models.VoiceMemo{
			UserID:       ownerID,
			Title:        "Not Yours",
			AudioFileKey: "voice-memos/notyours.mp3",
			Status:       models.StatusFailed,
		}
		require.NoError(t, repo.Create(ctx, memo))

		updated, err := repo.UpdateStatusWithOwnership(ctx, memo.ID, otherUserID, models.StatusFailed, models.StatusTranscribing)

		assert.Nil(t, updated)
		assert.Equal(t, apperrors.ErrVoiceMemoUnauthorized, err)
	})

	t.Run("returns error when status doesn't match", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		userID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			UserID:       userID,
			Title:        "Wrong Status",
			AudioFileKey: "voice-memos/wrongstatus.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		updated, err := repo.UpdateStatusWithOwnership(ctx, memo.ID, userID, models.StatusFailed, models.StatusTranscribing)

		assert.Nil(t, updated)
		assert.Equal(t, apperrors.ErrVoiceMemoInvalidStatus, err)
	})

	t.Run("returns error for non-existent memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		updated, err := repo.UpdateStatusWithOwnership(ctx, primitive.NewObjectID(), primitive.NewObjectID(), models.StatusFailed, models.StatusTranscribing)

		assert.Nil(t, updated)
		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}

func TestVoiceMemoRepository_UpdateStatusWithTeam(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("updates when memo belongs to team and status matches", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &teamID,
			Title:        "Team Memo",
			AudioFileKey: "voice-memos/team.mp3",
			Status:       models.StatusFailed,
		}
		require.NoError(t, repo.Create(ctx, memo))

		updated, err := repo.UpdateStatusWithTeam(ctx, memo.ID, teamID, models.StatusFailed, models.StatusTranscribing)

		require.NoError(t, err)
		assert.Equal(t, models.StatusTranscribing, updated.Status)
	})

	t.Run("returns error when memo doesn't belong to team", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()
		otherTeamID := primitive.NewObjectID()

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &teamID,
			Title:        "Wrong Team",
			AudioFileKey: "voice-memos/wrongteam.mp3",
			Status:       models.StatusFailed,
		}
		require.NoError(t, repo.Create(ctx, memo))

		updated, err := repo.UpdateStatusWithTeam(ctx, memo.ID, otherTeamID, models.StatusFailed, models.StatusTranscribing)

		assert.Nil(t, updated)
		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("returns error when status doesn't match", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &teamID,
			Title:        "Wrong Status",
			AudioFileKey: "voice-memos/wrongstatus.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		updated, err := repo.UpdateStatusWithTeam(ctx, memo.ID, teamID, models.StatusFailed, models.StatusTranscribing)

		assert.Nil(t, updated)
		assert.Equal(t, apperrors.ErrVoiceMemoInvalidStatus, err)
	})
}

func TestVoiceMemoRepository_UpdateTranscriptionAndStatus(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("updates transcription and status", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Transcribe Me",
			AudioFileKey: "voice-memos/transcribe.mp3",
			Status:       models.StatusTranscribing,
		}
		require.NoError(t, repo.Create(ctx, memo))

		err := repo.UpdateTranscriptionAndStatus(ctx, memo.ID, "This is the transcription", models.StatusReady)

		require.NoError(t, err)

		found, err := repo.FindByID(ctx, memo.ID)
		require.NoError(t, err)
		assert.Equal(t, "This is the transcription", found.Transcription)
		assert.Equal(t, models.StatusReady, found.Status)
	})

	t.Run("returns error for non-existent memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		err := repo.UpdateTranscriptionAndStatus(ctx, primitive.NewObjectID(), "text", models.StatusReady)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}

func TestVoiceMemoRepository_SoftDeleteByID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("soft deletes memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Delete Me",
			AudioFileKey: "voice-memos/deleteme.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		err := repo.SoftDeleteByID(ctx, memo.ID)

		require.NoError(t, err)

		// Verify not found
		found, err := repo.FindByID(ctx, memo.ID)
		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("returns error for non-existent memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		err := repo.SoftDeleteByID(ctx, primitive.NewObjectID())

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("returns error for already deleted memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			Title:        "Already Deleted",
			AudioFileKey: "voice-memos/alreadydeleted.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))
		require.NoError(t, repo.SoftDeleteByID(ctx, memo.ID))

		err := repo.SoftDeleteByID(ctx, memo.ID)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}

func TestVoiceMemoRepository_SoftDeleteWithOwnership(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes when user owns memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		userID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			UserID:       userID,
			Title:        "My Memo",
			AudioFileKey: "voice-memos/mymemo.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		err := repo.SoftDeleteWithOwnership(ctx, memo.ID, userID)

		require.NoError(t, err)

		found, err := repo.FindByID(ctx, memo.ID)
		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("returns error when user doesn't own memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		ownerID := primitive.NewObjectID()
		otherUserID := primitive.NewObjectID()

		memo := &models.VoiceMemo{
			UserID:       ownerID,
			Title:        "Not Your Memo",
			AudioFileKey: "voice-memos/notyours.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		err := repo.SoftDeleteWithOwnership(ctx, memo.ID, otherUserID)

		assert.Equal(t, apperrors.ErrVoiceMemoUnauthorized, err)

		// Verify memo still exists
		found, err := repo.FindByID(ctx, memo.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
	})

	t.Run("idempotent for already deleted memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		userID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			UserID:       userID,
			Title:        "Already Gone",
			AudioFileKey: "voice-memos/gone.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))
		require.NoError(t, repo.SoftDeleteWithOwnership(ctx, memo.ID, userID))

		// Delete again
		err := repo.SoftDeleteWithOwnership(ctx, memo.ID, userID)

		assert.NoError(t, err) // Should succeed (idempotent)
	})

	t.Run("returns error for non-existent memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		err := repo.SoftDeleteWithOwnership(ctx, primitive.NewObjectID(), primitive.NewObjectID())

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})
}

func TestVoiceMemoRepository_SoftDeleteWithTeam(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("deletes when memo belongs to team", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &teamID,
			Title:        "Team Memo",
			AudioFileKey: "voice-memos/team.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		err := repo.SoftDeleteWithTeam(ctx, memo.ID, teamID)

		require.NoError(t, err)

		found, err := repo.FindByID(ctx, memo.ID)
		assert.Nil(t, found)
		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)
	})

	t.Run("returns error when memo doesn't belong to team", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()
		otherTeamID := primitive.NewObjectID()

		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &teamID,
			Title:        "Wrong Team",
			AudioFileKey: "voice-memos/wrongteam.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))

		err := repo.SoftDeleteWithTeam(ctx, memo.ID, otherTeamID)

		assert.Equal(t, apperrors.ErrVoiceMemoNotFound, err)

		// Verify memo still exists
		found, err := repo.FindByID(ctx, memo.ID)
		require.NoError(t, err)
		assert.NotNil(t, found)
	})

	t.Run("idempotent for already deleted memo", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()
		memo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &teamID,
			Title:        "Already Gone",
			AudioFileKey: "voice-memos/gone.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, memo))
		require.NoError(t, repo.SoftDeleteWithTeam(ctx, memo.ID, teamID))

		err := repo.SoftDeleteWithTeam(ctx, memo.ID, teamID)

		assert.NoError(t, err) // Idempotent
	})
}

func TestVoiceMemoRepository_SoftDeleteByTeamID(t *testing.T) {
	tdb := SetupTestDB(t)
	defer tdb.Cleanup(t)

	repo := NewVoiceMemoRepository(tdb.Database)
	ctx := context.Background()

	t.Run("soft deletes all team memos", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		teamID := primitive.NewObjectID()
		otherTeamID := primitive.NewObjectID()

		// Create memos for team
		for i := 0; i < 3; i++ {
			memo := &models.VoiceMemo{
				UserID:       primitive.NewObjectID(),
				TeamID:       &teamID,
				Title:        "Team Memo " + string(rune('A'+i)),
				AudioFileKey: "voice-memos/team" + string(rune('a'+i)) + ".mp3",
				Status:       models.StatusReady,
			}
			require.NoError(t, repo.Create(ctx, memo))
		}

		// Create memo for other team
		otherMemo := &models.VoiceMemo{
			UserID:       primitive.NewObjectID(),
			TeamID:       &otherTeamID,
			Title:        "Other Team Memo",
			AudioFileKey: "voice-memos/otherteam.mp3",
			Status:       models.StatusReady,
		}
		require.NoError(t, repo.Create(ctx, otherMemo))

		err := repo.SoftDeleteByTeamID(ctx, teamID)

		require.NoError(t, err)

		// Verify team memos deleted
		memos, total, err := repo.FindByTeamID(ctx, teamID, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 0, total)
		assert.Len(t, memos, 0)

		// Verify other team memo still exists
		otherMemos, otherTotal, err := repo.FindByTeamID(ctx, otherTeamID, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, 1, otherTotal)
		assert.Len(t, otherMemos, 1)
	})

	t.Run("succeeds when team has no memos", func(t *testing.T) {
		tdb.ClearCollection(t, "voice_memos")

		err := repo.SoftDeleteByTeamID(ctx, primitive.NewObjectID())

		assert.NoError(t, err)
	})
}
