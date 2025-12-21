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

// TeamInvitationHandler handles HTTP requests for invitation operations.
type TeamInvitationHandler struct {
	invitationService *service.TeamInvitationService
	userService       *service.UserService
}

// NewTeamInvitationHandler creates a new TeamInvitationHandler.
func NewTeamInvitationHandler(invitationService *service.TeamInvitationService, userService *service.UserService) *TeamInvitationHandler {
	return &TeamInvitationHandler{
		invitationService: invitationService,
		userService:       userService,
	}
}

// CreateInvitation godoc
// @Summary      Create team invitation
// @Description  Invite a user to join a team. Requires owner or admin role.
// @Tags         team-invitations
// @Accept       json
// @Produce      json
// @Param        teamId  path      string                          true  "Team ID"
// @Param        body    body      models.CreateInvitationRequest  true  "Invitation details"
// @Success      201     {object}  response.Response{data=models.TeamInvitation}
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      409     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/invitations [post]
func (h *TeamInvitationHandler) CreateInvitation(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	inviterIDStr := middleware.GetUserID(c)
	inviterID, _ := primitive.ObjectIDFromHex(inviterIDStr)

	var req models.CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	invitation, err := h.invitationService.CreateInvitation(c.Request.Context(), teamID, inviterID, &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrAlreadyMember) {
			response.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrPendingInvitation) {
			response.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrSeatsExceeded) {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Created(c, invitation)
}

// ListTeamInvitations godoc
// @Summary      List team invitations
// @Description  List all pending invitations for a team. Requires owner or admin role.
// @Tags         team-invitations
// @Accept       json
// @Produce      json
// @Param        teamId  path      string  true  "Team ID"
// @Success      200     {object}  response.Response{data=models.InvitationListResponse}
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/invitations [get]
func (h *TeamInvitationHandler) ListTeamInvitations(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	result, err := h.invitationService.ListTeamInvitations(c.Request.Context(), teamID)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// CancelInvitation godoc
// @Summary      Cancel team invitation
// @Description  Cancel a pending invitation. Requires owner or admin role.
// @Tags         team-invitations
// @Accept       json
// @Produce      json
// @Param        teamId  path      string  true  "Team ID"
// @Param        id      path      string  true  "Invitation ID"
// @Success      200     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      404     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Security     BearerAuth
// @Router       /teams/{teamId}/invitations/{id} [delete]
func (h *TeamInvitationHandler) CancelInvitation(c *gin.Context) {
	teamID, exists := middleware.GetTeamID(c)
	if !exists {
		response.BadRequest(c, "team id not found in context")
		return
	}

	invitationID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid invitation id format")
		return
	}

	if err := h.invitationService.CancelInvitation(c.Request.Context(), invitationID, teamID); err != nil {
		if errors.Is(err, apperrors.ErrInvitationNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "invitation cancelled"})
}

// ListMyInvitations godoc
// @Summary      List my invitations
// @Description  List all pending invitations for the authenticated user
// @Tags         invitations
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Response{data=models.MyInvitationListResponse}
// @Failure      401  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Security     BearerAuth
// @Router       /invitations [get]
func (h *TeamInvitationHandler) ListMyInvitations(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	// Get user to get their email
	user, err := h.userService.GetUser(c.Request.Context(), userIDStr)
	if err != nil {
		response.InternalError(c)
		return
	}

	result, err := h.invitationService.ListMyInvitations(c.Request.Context(), user.Email)
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// AcceptInvitation godoc
// @Summary      Accept invitation
// @Description  Accept an invitation to join a team
// @Tags         invitations
// @Accept       json
// @Produce      json
// @Param        id  path      string  true  "Invitation ID"
// @Success      200 {object}  response.Response{data=models.AcceptInvitationResponse}
// @Failure      400 {object}  response.Response
// @Failure      401 {object}  response.Response
// @Failure      403 {object}  response.Response
// @Failure      404 {object}  response.Response
// @Failure      500 {object}  response.Response
// @Security     BearerAuth
// @Router       /invitations/{id}/accept [post]
func (h *TeamInvitationHandler) AcceptInvitation(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	userID, _ := primitive.ObjectIDFromHex(userIDStr)

	invitationID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid invitation id format")
		return
	}

	// Get user to get their email
	user, err := h.userService.GetUser(c.Request.Context(), userIDStr)
	if err != nil {
		response.InternalError(c)
		return
	}

	result, err := h.invitationService.AcceptInvitation(c.Request.Context(), invitationID, userID, user.Email)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvitationNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrInvitationEmailMismatch) {
			response.Forbidden(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrInvitationExpired) {
			response.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrSeatsExceeded) {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// DeclineInvitation godoc
// @Summary      Decline invitation
// @Description  Decline an invitation to join a team
// @Tags         invitations
// @Accept       json
// @Produce      json
// @Param        id  path      string  true  "Invitation ID"
// @Success      200 {object}  response.Response
// @Failure      400 {object}  response.Response
// @Failure      401 {object}  response.Response
// @Failure      403 {object}  response.Response
// @Failure      404 {object}  response.Response
// @Failure      500 {object}  response.Response
// @Security     BearerAuth
// @Router       /invitations/{id}/decline [post]
func (h *TeamInvitationHandler) DeclineInvitation(c *gin.Context) {
	userIDStr := middleware.GetUserID(c)
	if userIDStr == "" {
		response.Unauthorized(c, "user not authenticated")
		return
	}

	invitationID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid invitation id format")
		return
	}

	// Get user to get their email
	user, err := h.userService.GetUser(c.Request.Context(), userIDStr)
	if err != nil {
		response.InternalError(c)
		return
	}

	if err := h.invitationService.DeclineInvitation(c.Request.Context(), invitationID, user.Email); err != nil {
		if errors.Is(err, apperrors.ErrInvitationNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrInvitationEmailMismatch) {
			response.Forbidden(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "invitation declined"})
}
