// Package handler contains HTTP handlers for the API.
package handler

import (
	"errors"

	apperrors "gin-sample/internal/errors"
	"gin-sample/internal/models"
	"gin-sample/internal/service"
	"gin-sample/pkg/response"

	"github.com/gin-gonic/gin"
)

// UserHandler handles HTTP requests for user operations.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with email, password, and name
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateUserRequest  true  "User registration details"
// @Success      201      {object}  response.Response{data=models.User}
// @Failure      400      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req models.CreateUserRequest

	// Bind JSON body to struct and validate
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Call service
	user, err := h.service.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			response.Conflict(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Created(c, user)
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user and return JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.LoginRequest  true  "User credentials"
// @Success      200      {object}  response.Response{data=models.LoginResponse}
// @Failure      400      {object}  response.Response
// @Failure      401      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Router       /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
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

// GetUser godoc
// @Summary      Get user by ID
// @Description  Retrieve a single user by their ID
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  response.Response{data=models.User}
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Security     BearerAuth
// @Router       /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")

	user, err := h.service.GetUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, user)
}

// GetAllUsers godoc
// @Summary      List all users
// @Description  Retrieve a list of all users
// @Tags         users
// @Accept       json
// @Produce      json
// @Success      200  {object}  response.Response{data=[]models.User}
// @Failure      500  {object}  response.Response
// @Security     BearerAuth
// @Router       /users [get]
func (h *UserHandler) GetAllUsers(c *gin.Context) {
	users, err := h.service.GetAllUsers(c.Request.Context())
	if err != nil {
		response.InternalError(c)
		return
	}

	response.Success(c, users)
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Update a user's email and/or name
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id       path      string                    true  "User ID"
// @Param        request  body      models.UpdateUserRequest  true  "Fields to update"
// @Success      200      {object}  response.Response{data=models.User}
// @Failure      400      {object}  response.Response
// @Failure      404      {object}  response.Response
// @Failure      409      {object}  response.Response
// @Failure      500      {object}  response.Response
// @Security     BearerAuth
// @Router       /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.UpdateUser(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		if errors.Is(err, apperrors.ErrUserAlreadyExists) {
			response.Conflict(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, user)
}

// DeleteUser godoc
// @Summary      Delete user
// @Description  Remove a user from the system
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "User ID"
// @Success      200  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Failure      500  {object}  response.Response
// @Security     BearerAuth
// @Router       /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	err := h.service.DeleteUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrUserNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c)
		return
	}

	response.Success(c, gin.H{"message": "user deleted"})
}
