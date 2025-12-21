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

// TeamHandler handles HTTP requests for team operations.
type TeamHandler struct {
	service *service.TeamService
}

// NewTeamHandler creates a new TeamHandler.
func NewTeamHandler(service *service.TeamService) *TeamHandler {
	return &TeamHandler{service: service}
}

// CreateTeam godoc
// @Summary      Create a new team
// @Description  Create a new team. The authenticated user becomes the owner.
// @Tags         teams
// @Accept       json
// @Produce      json
// @Param        body  body      models.CreateTeamRequest  true  "Team details"
// @Success      201   {object}  response.Response{data=models.Team}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      403   {object}  response.Response
// @Failure      409   {object}  response.Response
// @Failure      500   {object}  response.Response
// @Security     BearerAuth
// @Router       /teams [post]
func (h *TeamHandler) CreateTeam(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		response.Unauthorized(c, "invalid user id format")
		return
	}

	var req models.CreateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	team, err := h.service.CreateTeam(c.Request.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrTeamLimitReached) {
			response.Forbidden(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrTeamSlugTaken) {
			response.Conflict(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Created(c, team)
}

// ListTeams godoc
// @Summary      List user's teams
// @Description  Retrieve paginated list of teams the authenticated user belongs to
// @Tags         teams
// @Accept       json
// @Produce      json
// @Param        page   query     int  false  "Page number (default: 1)"
// @Param        limit  query     int  false  "Items per page (default: 10, max: 10)"
// @Success      200    {object}  response.Response{data=models.TeamListResponse}
// @Failure      401    {object}  response.Response
// @Failure      500    {object}  response.Response
// @Security     BearerAuth
// @Router       /teams [get]
func (h *TeamHandler) ListTeams(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		response.Unauthorized(c, "invalid user id format")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.service.ListTeams(c.Request.Context(), userID, page, limit)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// GetTeam godoc
// @Summary      Get team details
// @Description  Retrieve details of a specific team
// @Tags         teams
// @Accept       json
// @Produce      json
// @Param        teamId  path      string  true  "Team ID"
// @Success      200     {object}  response.Response{data=models.Team}
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      404     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId} [get]
func (h *TeamHandler) GetTeam(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	team, err := h.service.GetTeam(c.Request.Context(), teamID)
	if err != nil {
		if errors.Is(err, apperrors.ErrTeamNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, team)
}

// UpdateTeam godoc
// @Summary      Update team
// @Description  Update team details. Requires owner or admin role.
// @Tags         teams
// @Accept       json
// @Produce      json
// @Param        teamId  path      string                    true  "Team ID"
// @Param        body    body      models.UpdateTeamRequest  true  "Team update details"
// @Success      200     {object}  response.Response{data=models.Team}
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      404     {object}  response.Response
// @Failure      409     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId} [put]
func (h *TeamHandler) UpdateTeam(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	var req models.UpdateTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	team, err := h.service.UpdateTeam(c.Request.Context(), teamID, &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrTeamNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrTeamSlugTaken) {
			response.Conflict(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, team)
}

// DeleteTeam godoc
// @Summary      Delete team
// @Description  Soft delete a team and all its data. Requires owner role.
// @Tags         teams
// @Accept       json
// @Produce      json
// @Param        teamId  path      string  true  "Team ID"
// @Success      200     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      404     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId} [delete]
func (h *TeamHandler) DeleteTeam(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	if err := h.service.DeleteTeam(c.Request.Context(), teamID); err != nil {
		if errors.Is(err, apperrors.ErrTeamNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "team deleted successfully"})
}

// TransferOwnership godoc
// @Summary      Transfer team ownership
// @Description  Transfer team ownership to another member. Requires owner role.
// @Tags         teams
// @Accept       json
// @Produce      json
// @Param        teamId  path      string                           true  "Team ID"
// @Param        body    body      models.TransferOwnershipRequest  true  "New owner details"
// @Success      200     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      404     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/transfer [post]
func (h *TeamHandler) TransferOwnership(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	userIDStr := middleware.GetUserID(c)
	currentOwnerID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		response.Unauthorized(c, "invalid user id format")
		return
	}

	var req models.TransferOwnershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	newOwnerID, err := primitive.ObjectIDFromHex(req.NewOwnerID)
	if err != nil {
		response.BadRequest(c, "invalid new owner id format")
		return
	}

	if err := h.service.TransferOwnership(c.Request.Context(), teamID, currentOwnerID, newOwnerID); err != nil {
		if errors.Is(err, apperrors.ErrNotTeamMember) {
			response.NotFound(c, "new owner must be a team member")
			return
		}
		if errors.Is(err, apperrors.ErrTeamNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "ownership transferred successfully"})
}
