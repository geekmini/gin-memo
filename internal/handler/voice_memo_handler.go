package handler

import (
	"strconv"

	"gin-sample/internal/service"
	"gin-sample/pkg/response"

	"github.com/gin-gonic/gin"
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
