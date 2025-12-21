package middleware

import (
	"gin-sample/internal/authz"
	"gin-sample/pkg/response"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Context keys for storing team data
const (
	TeamIDKey   = "teamID"
	TeamRoleKey = "teamRole"
)

// TeamAuthz returns a middleware that checks team authorization.
// It validates that the user is a member of the team and has permission for the action.
func TeamAuthz(authorizer authz.Authorizer, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user ID from context (set by Auth middleware)
		userIDStr := GetUserID(c)
		if userIDStr == "" {
			response.Unauthorized(c, "user not authenticated")
			c.Abort()
			return
		}

		userID, err := primitive.ObjectIDFromHex(userIDStr)
		if err != nil {
			response.Unauthorized(c, "invalid user id format")
			c.Abort()
			return
		}

		// Get team ID from path parameter
		teamIDStr := c.Param("teamId")
		if teamIDStr == "" {
			response.BadRequest(c, "team id is required")
			c.Abort()
			return
		}

		teamID, err := primitive.ObjectIDFromHex(teamIDStr)
		if err != nil {
			response.BadRequest(c, "invalid team id format")
			c.Abort()
			return
		}

		// Check authorization
		allowed, err := authorizer.CanPerform(c.Request.Context(), userID, teamID, action)
		if err != nil {
			response.InternalError(c)
			c.Abort()
			return
		}

		if !allowed {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}

		// Get and store user's role in the team
		role, _ := authorizer.GetUserRole(c.Request.Context(), userID, teamID)

		// Store team ID and role in context for handlers
		c.Set(TeamIDKey, teamID)
		c.Set(TeamRoleKey, role)

		c.Next()
	}
}

// TeamMember returns a middleware that only checks team membership (any role).
func TeamMember(authorizer authz.Authorizer) gin.HandlerFunc {
	return TeamAuthz(authorizer, authz.ActionTeamView)
}

// GetTeamID retrieves the team ID from the context.
func GetTeamID(c *gin.Context) (primitive.ObjectID, bool) {
	teamID, exists := c.Get(TeamIDKey)
	if !exists {
		return primitive.NilObjectID, false
	}
	return teamID.(primitive.ObjectID), true
}

// GetTeamRole retrieves the user's team role from the context.
func GetTeamRole(c *gin.Context) string {
	role, exists := c.Get(TeamRoleKey)
	if !exists {
		return ""
	}
	return role.(string)
}
