package handler

import (
	"errors"
	"strconv"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/middleware"
	"gin-sample/internal/models"
	"gin-sample/internal/service"
	"gin-sample/pkg/response"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// VoiceMemoHandler handles HTTP requests for voice memo operations.
type VoiceMemoHandler struct {
	service *service.VoiceMemoService
}

// NewVoiceMemoHandler creates a new VoiceMemoHandler.
func NewVoiceMemoHandler(service *service.VoiceMemoService) *VoiceMemoHandler {
	return &VoiceMemoHandler{service: service}
}

// ListVoiceMemos godoc
// @Summary      List user's voice memos
// @Description  Retrieve paginated list of the authenticated user's voice memos, sorted by newest first
// @Tags         voice-memos
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number (default: 1)"
// @Param        limit  query     int  false  "Items per page (default: 10, max: 10)"
// @Success      200    {object}  response.Response{data=models.VoiceMemoListResponse}
// @Failure      401    {object}  response.Response
// @Failure      500    {object}  response.Response
// @Security     BearerAuth
// @Router       /voice-memos [get]
func (h *VoiceMemoHandler) ListVoiceMemos(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	// Get memos from service
	result, err := h.service.ListByUserID(c.Request.Context(), userID.(string), page, limit)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// DeleteVoiceMemo godoc
// @Summary      Soft delete voice memo
// @Description  Mark a voice memo as deleted. User can only delete their own memos. Idempotent - returns 204 even if already deleted.
// @Tags         voice-memos
// @Param        id   path      string  true  "Voice Memo ID"
// @Success      204  "No Content"
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Security     BearerAuth
// @Router       /voice-memos/{id} [delete]
func (h *VoiceMemoHandler) DeleteVoiceMemo(c *gin.Context) {
	// Validate and parse memo ID from path
	memoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid voice memo id format")
		return
	}

	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	// Validate and parse user ID
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		response.Unauthorized(c, "invalid user id format")
		return
	}

	// Call service to delete (atomic operation with ownership check)
	err = h.service.DeleteVoiceMemo(c.Request.Context(), memoID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrVoiceMemoNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrVoiceMemoUnauthorized) {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

// ListTeamVoiceMemos godoc
// @Summary      List team voice memos
// @Description  Retrieve paginated list of a team's voice memos, sorted by newest first
// @Tags         team-voice-memos
// @Accept       json
// @Produce      json
// @Param        teamId path      string  true   "Team ID"
// @Param        page   query     int     false  "Page number (default: 1)"
// @Param        limit  query     int     false  "Items per page (default: 10, max: 10)"
// @Success      200    {object}  response.Response{data=models.VoiceMemoListResponse}
// @Failure      400    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Failure      403    {object}  response.Response
// @Failure      500    {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/voice-memos [get]
func (h *VoiceMemoHandler) ListTeamVoiceMemos(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.service.ListByTeamID(c.Request.Context(), teamID.Hex(), page, limit)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// GetTeamVoiceMemo godoc
// @Summary      Get team voice memo
// @Description  Retrieve a specific voice memo from a team
// @Tags         team-voice-memos
// @Accept       json
// @Produce      json
// @Param        teamId path      string  true  "Team ID"
// @Param        id     path      string  true  "Voice Memo ID"
// @Success      200    {object}  response.Response{data=models.VoiceMemo}
// @Failure      400    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Failure      403    {object}  response.Response
// @Failure      404    {object}  response.Response
// @Failure      500    {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/voice-memos/{id} [get]
func (h *VoiceMemoHandler) GetTeamVoiceMemo(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	memoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid voice memo id format")
		return
	}

	memo, err := h.service.GetVoiceMemo(c.Request.Context(), memoID)
	if err != nil {
		if errors.Is(err, apperrors.ErrVoiceMemoNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	// Verify memo belongs to this team
	if memo.TeamID == nil || *memo.TeamID != teamID {
		response.NotFound(c, "voice memo not found")
		return
	}

	response.Success(c, memo)
}

// DeleteTeamVoiceMemo godoc
// @Summary      Delete team voice memo
// @Description  Soft delete a voice memo from a team. Idempotent - returns 204 even if already deleted.
// @Tags         team-voice-memos
// @Param        teamId path      string  true  "Team ID"
// @Param        id     path      string  true  "Voice Memo ID"
// @Success      204    "No Content"
// @Failure      400    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Failure      403    {object}  response.Response
// @Failure      404    {object}  response.Response
// @Failure      500    {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/voice-memos/{id} [delete]
func (h *VoiceMemoHandler) DeleteTeamVoiceMemo(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	memoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid voice memo id format")
		return
	}

	// Call service to delete (atomic operation with team check)
	if err := h.service.DeleteTeamVoiceMemo(c.Request.Context(), memoID, teamID); err != nil {
		if errors.Is(err, apperrors.ErrVoiceMemoNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

// CreateVoiceMemo godoc
// @Summary      Create voice memo
// @Description  Create a new private voice memo and get a pre-signed URL for audio upload
// @Tags         voice-memos
// @Accept       json
// @Produce      json
// @Param        request body      models.CreateVoiceMemoRequest  true  "Voice memo details"
// @Success      201     {object}  response.Response{data=models.CreateVoiceMemoResponse}
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /voice-memos [post]
func (h *VoiceMemoHandler) CreateVoiceMemo(c *gin.Context) {
	// Get user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		response.Unauthorized(c, "invalid user id format")
		return
	}

	// Bind and validate request
	var req models.CreateVoiceMemoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Create memo via service
	result, err := h.service.CreateVoiceMemo(c.Request.Context(), userID, &req)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Created(c, result)
}

// CreateTeamVoiceMemo godoc
// @Summary      Create team voice memo
// @Description  Create a new team voice memo and get a pre-signed URL for audio upload
// @Tags         team-voice-memos
// @Accept       json
// @Produce      json
// @Param        teamId  path      string                         true  "Team ID"
// @Param        request body      models.CreateVoiceMemoRequest  true  "Voice memo details"
// @Success      201     {object}  response.Response{data=models.CreateVoiceMemoResponse}
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/voice-memos [post]
func (h *VoiceMemoHandler) CreateTeamVoiceMemo(c *gin.Context) {
	// Get team ID from middleware context
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		response.Unauthorized(c, "invalid user id format")
		return
	}

	// Bind and validate request
	var req models.CreateVoiceMemoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Create memo via service
	result, err := h.service.CreateTeamVoiceMemo(c.Request.Context(), userID, teamID, &req)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Created(c, result)
}

// ConfirmUpload godoc
// @Summary      Confirm audio upload
// @Description  Confirm that audio has been uploaded to S3 and trigger transcription
// @Tags         voice-memos
// @Produce      json
// @Param        id   path      string  true  "Voice Memo ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response  "Invalid status transition"
// @Failure      503  {object}  response.Response  "Transcription queue full"
// @Failure      500  {object}  response.Response
// @Security     BearerAuth
// @Router       /voice-memos/{id}/confirm-upload [post]
func (h *VoiceMemoHandler) ConfirmUpload(c *gin.Context) {
	// Parse memo ID
	memoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid voice memo id format")
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		response.Unauthorized(c, "invalid user id format")
		return
	}

	// Confirm upload via service
	err = h.service.ConfirmUpload(c.Request.Context(), memoID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrVoiceMemoNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrVoiceMemoUnauthorized) {
			response.Forbidden(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrVoiceMemoInvalidStatus) {
			response.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrTranscriptionQueueFull) {
			response.Error(c, 503, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "upload confirmed, transcription started"})
}

// ConfirmTeamUpload godoc
// @Summary      Confirm team audio upload
// @Description  Confirm that audio has been uploaded to S3 and trigger transcription for a team memo
// @Tags         team-voice-memos
// @Produce      json
// @Param        teamId path      string  true  "Team ID"
// @Param        id     path      string  true  "Voice Memo ID"
// @Success      200    {object}  response.Response
// @Failure      400    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Failure      403    {object}  response.Response
// @Failure      404    {object}  response.Response
// @Failure      409    {object}  response.Response  "Invalid status transition"
// @Failure      503    {object}  response.Response  "Transcription queue full"
// @Failure      500    {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/voice-memos/{id}/confirm-upload [post]
func (h *VoiceMemoHandler) ConfirmTeamUpload(c *gin.Context) {
	// Get team ID from middleware context
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	// Parse memo ID
	memoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid voice memo id format")
		return
	}

	// Confirm upload via service
	err = h.service.ConfirmTeamUpload(c.Request.Context(), memoID, teamID)
	if err != nil {
		if errors.Is(err, apperrors.ErrVoiceMemoNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrVoiceMemoInvalidStatus) {
			response.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrTranscriptionQueueFull) {
			response.Error(c, 503, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "upload confirmed, transcription started"})
}

// RetryTranscription godoc
// @Summary      Retry transcription
// @Description  Retry transcription for a failed voice memo
// @Tags         voice-memos
// @Produce      json
// @Param        id   path      string  true  "Voice Memo ID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      403  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      409  {object}  response.Response  "Invalid status - memo is not in failed state"
// @Failure      503  {object}  response.Response  "Transcription queue full"
// @Failure      500  {object}  response.Response
// @Security     BearerAuth
// @Router       /voice-memos/{id}/retry-transcription [post]
func (h *VoiceMemoHandler) RetryTranscription(c *gin.Context) {
	// Parse memo ID
	memoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid voice memo id format")
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		response.Unauthorized(c, "invalid user id format")
		return
	}

	// Retry transcription via service
	err = h.service.RetryTranscription(c.Request.Context(), memoID, userID)
	if err != nil {
		if errors.Is(err, apperrors.ErrVoiceMemoNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrVoiceMemoUnauthorized) {
			response.Forbidden(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrVoiceMemoInvalidStatus) {
			response.Conflict(c, "memo is not in failed state")
			return
		}
		if errors.Is(err, apperrors.ErrTranscriptionQueueFull) {
			response.Error(c, 503, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "transcription retry started"})
}

// RetryTeamTranscription godoc
// @Summary      Retry team transcription
// @Description  Retry transcription for a failed team voice memo
// @Tags         team-voice-memos
// @Produce      json
// @Param        teamId path      string  true  "Team ID"
// @Param        id     path      string  true  "Voice Memo ID"
// @Success      200    {object}  response.Response
// @Failure      400    {object}  response.Response
// @Failure      401    {object}  response.Response
// @Failure      403    {object}  response.Response
// @Failure      404    {object}  response.Response
// @Failure      409    {object}  response.Response  "Invalid status - memo is not in failed state"
// @Failure      503    {object}  response.Response  "Transcription queue full"
// @Failure      500    {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/voice-memos/{id}/retry-transcription [post]
func (h *VoiceMemoHandler) RetryTeamTranscription(c *gin.Context) {
	// Get team ID from middleware context
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	// Parse memo ID
	memoID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid voice memo id format")
		return
	}

	// Retry transcription via service
	err = h.service.RetryTeamTranscription(c.Request.Context(), memoID, teamID)
	if err != nil {
		if errors.Is(err, apperrors.ErrVoiceMemoNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrVoiceMemoInvalidStatus) {
			response.Conflict(c, "memo is not in failed state")
			return
		}
		if errors.Is(err, apperrors.ErrTranscriptionQueueFull) {
			response.Error(c, 503, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "transcription retry started"})
}
