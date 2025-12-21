package handler

import (
	"errors"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/middleware"
	"gin-sample/internal/models"
	"gin-sample/internal/service"
	"gin-sample/pkg/response"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TeamMemberHandler handles HTTP requests for team member operations.
type TeamMemberHandler struct {
	service service.TeamMemberServicer
}

// NewTeamMemberHandler creates a new TeamMemberHandler.
func NewTeamMemberHandler(service service.TeamMemberServicer) *TeamMemberHandler {
	return &TeamMemberHandler{service: service}
}

// ListMembers godoc
// @Summary      List team members
// @Description  Retrieve all members of a team with their details
// @Tags         team-members
// @Accept       json
// @Produce      json
// @Param        teamId  path      string  true  "Team ID"
// @Success      200     {object}  response.Response{data=models.TeamMemberListResponse}
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/members [get]
func (h *TeamMemberHandler) ListMembers(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	result, err := h.service.ListMembers(c.Request.Context(), teamID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// RemoveMember godoc
// @Summary      Remove team member
// @Description  Remove a member from the team. Requires owner or admin role.
// @Tags         team-members
// @Accept       json
// @Produce      json
// @Param        teamId  path      string  true  "Team ID"
// @Param        userId  path      string  true  "User ID to remove"
// @Success      200     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      404     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/members/{userId} [delete]
func (h *TeamMemberHandler) RemoveMember(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	targetUserID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "invalid user id format")
		return
	}

	requestingUserIDStr := middleware.GetUserID(c)
	requestingUserID, _ := primitive.ObjectIDFromHex(requestingUserIDStr)

	if err := h.service.RemoveMember(c.Request.Context(), teamID, targetUserID, requestingUserID); err != nil {
		if errors.Is(err, apperrors.ErrNotTeamMember) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrCannotRemoveOwner) {
			response.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrCannotRemoveSelf) {
			response.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrInsufficientPermissions) {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "member removed successfully"})
}

// UpdateRole godoc
// @Summary      Update member role
// @Description  Update a member's role in the team. Requires owner or admin role.
// @Tags         team-members
// @Accept       json
// @Produce      json
// @Param        teamId  path      string                    true  "Team ID"
// @Param        userId  path      string                    true  "User ID"
// @Param        body    body      models.UpdateRoleRequest  true  "New role"
// @Success      200     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      404     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/members/{userId}/role [put]
func (h *TeamMemberHandler) UpdateRole(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	targetUserID, err := primitive.ObjectIDFromHex(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "invalid user id format")
		return
	}

	requestingUserIDStr := middleware.GetUserID(c)
	requestingUserID, _ := primitive.ObjectIDFromHex(requestingUserIDStr)

	var req models.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.UpdateRole(c.Request.Context(), teamID, targetUserID, requestingUserID, req.Role); err != nil {
		if errors.Is(err, apperrors.ErrNotTeamMember) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrCannotChangeOwnerRole) {
			response.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrInvalidRole) {
			response.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrInsufficientPermissions) {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "role updated successfully"})
}

// LeaveTeam godoc
// @Summary      Leave team
// @Description  Leave a team. Owner cannot leave without transferring ownership.
// @Tags         team-members
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
// @Router       /teams/{teamId}/leave [post]
func (h *TeamMemberHandler) LeaveTeam(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	userIDStr := middleware.GetUserID(c)
	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	if err := h.service.LeaveTeam(c.Request.Context(), teamID, userID); err != nil {
		if errors.Is(err, apperrors.ErrNotTeamMember) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrOwnerCannotLeave) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "left team successfully"})
}
