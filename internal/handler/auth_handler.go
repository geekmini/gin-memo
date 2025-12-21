// Package handler contains HTTP handlers for the API.
package handler

import (
	"errors"
	"net/http"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/service"
	"gin-sample/pkg/response"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles HTTP requests for authentication operations.
type AuthHandler struct {
	service service.AuthServicer
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service service.AuthServicer) *AuthHandler {
	return &AuthHandler{service: service}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with email, password, and name
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateUserRequest  true  "User registration details"
// @Success      201      {object}  response.Response{data=models.AuthResponse}
// @Failure      400      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.CreateUserRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			response.Conflict(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Created(c, result)
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user and return access token and refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.LoginRequest  true  "User credentials"
// @Success      200      {object}  response.Response{data=models.AuthResponse}
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidCredentials) {
			response.Unauthorized(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// Refresh godoc
// @Summary      Refresh access token
// @Description  Exchange a refresh token for a new access token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.RefreshRequest  true  "Refresh token"
// @Success      200      {object}  response.Response{data=models.RefreshResponse}
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req models.RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.Refresh(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrInvalidRefreshToken) {
			response.Unauthorized(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, result)
}

// Logout godoc
// @Summary      User logout
// @Description  Invalidate the refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.LogoutRequest  true  "Refresh token to invalidate"
// @Success      204      "No Content"
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Security     BearerAuth
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var req models.LogoutRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.Logout(c.Request.Context(), &req); err != nil {
		response.InternalError(c)
		return
	}

	c.Status(http.StatusNoContent)
}
