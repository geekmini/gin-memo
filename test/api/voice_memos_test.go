//go:build api

package api

import (
	"bytes"
	"context"
	"io"
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

// TestCreateVoiceMemo tests the POST /api/v1/voice-memos endpoint.
func TestCreateVoiceMemo(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	_, accessToken := authHelper.CreateAuthenticatedUser(t, "Voice Memo User", "voicememo@example.com", "password123")

	t.Run("success - creates voice memo and returns upload URL", func(t *testing.T) {
		req := models.CreateVoiceMemoRequest{
			Title:       "Test Memo",
			Duration:    120,
			FileSize:    1048576, // 1MB
			AudioFormat: "mp3",
			Tags:        []string{"test", "demo"},
			IsFavorite:  false,
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", accessToken, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		// Verify memo fields
		memo, ok := resp.Data["memo"].(map[string]interface{})
		require.True(t, ok, "memo should be an object")
		assert.Equal(t, "Test Memo", memo["title"])
		assert.Equal(t, float64(120), memo["duration"])
		assert.Equal(t, "mp3", memo["audioFormat"])
		assert.Equal(t, string(models.StatusPendingUpload), memo["status"])
		assert.NotEmpty(t, memo["id"])

		// Verify upload URL is returned
		uploadURL, ok := resp.Data["uploadUrl"].(string)
		assert.True(t, ok, "uploadUrl should be a string")
		assert.NotEmpty(t, uploadURL)
		assert.Contains(t, uploadURL, "http") // Should be a valid URL
	})

	t.Run("success - creates memo without optional fields", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Minimal User", "minimal@example.com", "password123")

		req := models.CreateVoiceMemoRequest{
			Title:       "Minimal Memo",
			FileSize:    500000,
			AudioFormat: "wav",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", token, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
	})

	t.Run("error - missing required title", func(t *testing.T) {
		req := map[string]interface{}{
			"fileSize":    1048576,
			"audioFormat": "mp3",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", accessToken, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - missing required fileSize", func(t *testing.T) {
		req := map[string]interface{}{
			"title":       "No Size",
			"audioFormat": "mp3",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", accessToken, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - invalid audio format", func(t *testing.T) {
		req := models.CreateVoiceMemoRequest{
			Title:       "Invalid Format",
			FileSize:    1048576,
			AudioFormat: "invalid",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", accessToken, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - file size exceeds maximum", func(t *testing.T) {
		req := models.CreateVoiceMemoRequest{
			Title:       "Too Big",
			FileSize:    200000000, // 200MB, max is 100MB
			AudioFormat: "mp3",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", accessToken, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - title too long", func(t *testing.T) {
		longTitle := make([]byte, 250)
		for i := range longTitle {
			longTitle[i] = 'a'
		}

		req := models.CreateVoiceMemoRequest{
			Title:       string(longTitle), // max is 200
			FileSize:    1048576,
			AudioFormat: "mp3",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", accessToken, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		req := models.CreateVoiceMemoRequest{
			Title:       "Unauthorized",
			FileSize:    1048576,
			AudioFormat: "mp3",
		}

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestListVoiceMemos tests the GET /api/v1/voice-memos endpoint.
func TestListVoiceMemos(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	voiceMemoHelper := testserver.NewVoiceMemoHelper(testServer)

	_, accessToken := authHelper.CreateAuthenticatedUser(t, "List User", "listuser@example.com", "password123")

	t.Run("success - returns empty list when no memos", func(t *testing.T) {
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/voice-memos", accessToken, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		require.NotNil(t, resp.Data)

		items, ok := resp.Data["items"].([]interface{})
		assert.True(t, ok)
		assert.Empty(t, items)

		pagination, ok := resp.Data["pagination"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(0), pagination["totalItems"])
	})

	t.Run("success - returns user's memos", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Multi Memo User", "multimemo@example.com", "password123")

		// Create multiple memos
		voiceMemoHelper.CreateVoiceMemo(t, token, "Memo 1", 60)
		voiceMemoHelper.CreateVoiceMemo(t, token, "Memo 2", 120)
		voiceMemoHelper.CreateVoiceMemo(t, token, "Memo 3", 180)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/voice-memos", token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, ok := resp.Data["items"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, items, 3)

		pagination, ok := resp.Data["pagination"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, float64(3), pagination["totalItems"])
	})

	t.Run("success - pagination works", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Paginated User", "paginated@example.com", "password123")

		// Create 5 memos
		for i := 1; i <= 5; i++ {
			voiceMemoHelper.CreateVoiceMemo(t, token, "Memo", i*60)
		}

		// Get page 1 with limit 2
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/voice-memos?page=1&limit=2", token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)

		items, _ := resp.Data["items"].([]interface{})
		assert.Len(t, items, 2)

		pagination, _ := resp.Data["pagination"].(map[string]interface{})
		assert.Equal(t, float64(1), pagination["page"])
		assert.Equal(t, float64(2), pagination["limit"])
		assert.Equal(t, float64(5), pagination["totalItems"])
		assert.Equal(t, float64(3), pagination["totalPages"])
	})

	t.Run("success - different users see only their own memos", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "User One", "user1@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "User Two", "user2@example.com", "password123")

		// User 1 creates 2 memos
		voiceMemoHelper.CreateVoiceMemo(t, token1, "User1 Memo A", 60)
		voiceMemoHelper.CreateVoiceMemo(t, token1, "User1 Memo B", 120)

		// User 2 creates 1 memo
		voiceMemoHelper.CreateVoiceMemo(t, token2, "User2 Memo", 90)

		// User 1 should see 2 memos
		w1 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/voice-memos", token1, nil)
		resp1 := testutil.ParseAPIResponse(t, w1)
		items1, _ := resp1.Data["items"].([]interface{})
		assert.Len(t, items1, 2)

		// User 2 should see 1 memo
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/voice-memos", token2, nil)
		resp2 := testutil.ParseAPIResponse(t, w2)
		items2, _ := resp2.Data["items"].([]interface{})
		assert.Len(t, items2, 1)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/api/v1/voice-memos", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestDeleteVoiceMemo tests the DELETE /api/v1/voice-memos/:id endpoint.
func TestDeleteVoiceMemo(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	voiceMemoHelper := testserver.NewVoiceMemoHelper(testServer)

	t.Run("success - deletes own memo", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Delete User", "deleteuser@example.com", "password123")
		memoData := voiceMemoHelper.CreateVoiceMemo(t, token, "To Delete", 60)

		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/voice-memos/"+memoID, token, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify memo is no longer in list
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodGet, "/api/v1/voice-memos", token, nil)
		resp := testutil.ParseAPIResponse(t, w2)
		items, _ := resp.Data["items"].([]interface{})
		assert.Empty(t, items)
	})

	t.Run("success - idempotent delete (already deleted)", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Idempotent User", "idempotent@example.com", "password123")
		memoData := voiceMemoHelper.CreateVoiceMemo(t, token, "Double Delete", 60)

		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// First delete
		w1 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/voice-memos/"+memoID, token, nil)
		assert.Equal(t, http.StatusNoContent, w1.Code)

		// Second delete should also succeed (idempotent)
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/voice-memos/"+memoID, token, nil)
		assert.Equal(t, http.StatusNoContent, w2.Code)
	})

	t.Run("error - cannot delete another user's memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Other User", "other@example.com", "password123")

		memoData := voiceMemoHelper.CreateVoiceMemo(t, token1, "Owner's Memo", 60)
		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// User 2 tries to delete User 1's memo
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/voice-memos/"+memoID, token2, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - memo not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Not Found User", "notfound@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/voice-memos/"+nonExistentID, token, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - invalid memo ID format", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Invalid ID User", "invalidid@example.com", "password123")

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodDelete, "/api/v1/voice-memos/invalid-id", token, nil)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Unauth User", "unauthuser@example.com", "password123")
		memoData := voiceMemoHelper.CreateVoiceMemo(t, token, "Unauth Delete", 60)
		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		w := testutil.MakeRequest(t, testServer.Router, http.MethodDelete, "/api/v1/voice-memos/"+memoID, nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestConfirmUpload tests the POST /api/v1/voice-memos/:id/confirm-upload endpoint.
func TestConfirmUpload(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	voiceMemoHelper := testserver.NewVoiceMemoHelper(testServer)

	t.Run("success - confirms upload and triggers transcription", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Confirm User", "confirm@example.com", "password123")
		memoData := voiceMemoHelper.CreateVoiceMemo(t, token, "Confirm Memo", 60)

		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Simulate uploading a file to MinIO
		uploadURL := memoData["uploadUrl"].(string)
		uploadTestAudio(t, uploadURL)

		// Confirm upload
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+memoID+"/confirm-upload", token, nil)

		assert.Equal(t, http.StatusOK, w.Code)

		resp := testutil.ParseAPIResponse(t, w)
		assert.True(t, resp.Success)
		assert.Contains(t, resp.Data["message"], "transcription started")
	})

	t.Run("error - cannot confirm twice (invalid status)", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Double Confirm", "doubleconfirm@example.com", "password123")
		memoData := voiceMemoHelper.CreateVoiceMemo(t, token, "Double Memo", 60)

		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		uploadURL := memoData["uploadUrl"].(string)
		uploadTestAudio(t, uploadURL)

		// First confirm
		w1 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+memoID+"/confirm-upload", token, nil)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Second confirm should fail (status changed)
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+memoID+"/confirm-upload", token, nil)
		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("error - cannot confirm another user's memo", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token1 := authHelper.CreateAuthenticatedUser(t, "Owner", "owner2@example.com", "password123")
		_, token2 := authHelper.CreateAuthenticatedUser(t, "Other", "other2@example.com", "password123")

		memoData := voiceMemoHelper.CreateVoiceMemo(t, token1, "Owner's Memo", 60)
		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+memoID+"/confirm-upload", token2, nil)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("error - memo not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Not Found", "notfound2@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+nonExistentID+"/confirm-upload", token, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Unauth", "unauth2@example.com", "password123")
		memoData := voiceMemoHelper.CreateVoiceMemo(t, token, "Unauth Memo", 60)
		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+memoID+"/confirm-upload", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestRetryTranscription tests the POST /api/v1/voice-memos/:id/retry-transcription endpoint.
func TestRetryTranscription(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)

	t.Run("error - cannot retry transcription for non-failed memo", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Retry User", "retry@example.com", "password123")

		// Create memo (status is pending_upload)
		req := models.CreateVoiceMemoRequest{
			Title:       "Retry Memo",
			Duration:    60,
			FileSize:    1048576,
			AudioFormat: "mp3",
		}

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos", token, req)
		require.Equal(t, http.StatusCreated, w.Code)

		createResp := testutil.ParseAPIResponse(t, w)
		memo, _ := createResp.Data["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// Try to retry (should fail - not in failed state)
		w2 := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+memoID+"/retry-transcription", token, nil)

		assert.Equal(t, http.StatusConflict, w2.Code)
	})

	t.Run("error - memo not found", func(t *testing.T) {
		testServer.CleanupBetweenTests(t)

		_, token := authHelper.CreateAuthenticatedUser(t, "Not Found", "notfound3@example.com", "password123")
		nonExistentID := primitive.NewObjectID().Hex()

		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+nonExistentID+"/retry-transcription", token, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("error - unauthorized without token", func(t *testing.T) {
		memoID := primitive.NewObjectID().Hex()

		w := testutil.MakeRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+memoID+"/retry-transcription", nil)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// TestActualFileUpload tests actual file upload to MinIO through pre-signed URLs.
func TestActualFileUpload(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	voiceMemoHelper := testserver.NewVoiceMemoHelper(testServer)

	t.Run("success - upload and download actual file via pre-signed URLs", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Upload User", "upload@example.com", "password123")
		memoData := voiceMemoHelper.CreateVoiceMemo(t, token, "Upload Test", 60)

		// Get the upload URL
		uploadURL := memoData["uploadUrl"].(string)
		require.NotEmpty(t, uploadURL)

		// Upload actual test audio content
		testContent := []byte("fake audio content for testing purposes")
		uploadTestAudioWithContent(t, uploadURL, testContent)

		// Verify file exists in MinIO
		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)

		// The file should exist in MinIO
		ctx := context.Background()
		memoOID, _ := primitive.ObjectIDFromHex(memoID)
		storedMemo, err := testServer.VoiceMemoRepo.FindByID(ctx, memoOID)
		require.NoError(t, err)

		// Check file exists
		exists := testServer.MinIO.ObjectExists(ctx, storedMemo.AudioFileKey)
		assert.True(t, exists, "uploaded file should exist in MinIO")
	})
}

// TestTranscriptionWorkflow tests the full transcription workflow.
func TestTranscriptionWorkflow(t *testing.T) {
	testServer.CleanupBetweenTests(t)

	authHelper := testserver.NewAuthHelper(testServer)
	voiceMemoHelper := testserver.NewVoiceMemoHelper(testServer)

	t.Run("full workflow - create, upload, confirm, transcribe", func(t *testing.T) {
		_, token := authHelper.CreateAuthenticatedUser(t, "Workflow User", "workflow@example.com", "password123")

		// 1. Create voice memo
		memoData := voiceMemoHelper.CreateVoiceMemo(t, token, "Workflow Memo", 60)
		memo, _ := memoData["memo"].(map[string]interface{})
		memoID := memo["id"].(string)
		assert.Equal(t, string(models.StatusPendingUpload), memo["status"])

		// 2. Upload audio
		uploadURL := memoData["uploadUrl"].(string)
		uploadTestAudio(t, uploadURL)

		// 3. Confirm upload
		w := testutil.MakeAuthRequest(t, testServer.Router, http.MethodPost, "/api/v1/voice-memos/"+memoID+"/confirm-upload", token, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		// 4. Start transcription processor
		ctx := context.Background()
		testServer.StartTranscriptionProcessor(ctx)

		// 5. Wait for transcription to complete (mock service is fast)
		time.Sleep(500 * time.Millisecond)
		testServer.StopTranscriptionProcessor()

		// 6. Verify memo status is ready
		memoOID, _ := primitive.ObjectIDFromHex(memoID)
		updatedMemo, err := testServer.VoiceMemoRepo.FindByID(ctx, memoOID)
		require.NoError(t, err)
		assert.Equal(t, models.StatusReady, updatedMemo.Status)
		assert.NotEmpty(t, updatedMemo.Transcription)
	})
}

// uploadTestAudio uploads test audio content to the given pre-signed URL.
func uploadTestAudio(t *testing.T, uploadURL string) {
	t.Helper()
	uploadTestAudioWithContent(t, uploadURL, []byte("test audio content"))
}

// uploadTestAudioWithContent uploads specific content to the given pre-signed URL.
func uploadTestAudioWithContent(t *testing.T, uploadURL string, content []byte) {
	t.Helper()

	req, err := http.NewRequest(http.MethodPut, uploadURL, bytes.NewReader(content))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "audio/mpeg")
	req.ContentLength = int64(len(content))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}
}
