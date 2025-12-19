// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"strings"

	"gin-sample/pkg/auth"
	"gin-sample/pkg/response"

	"github.com/gin-gonic/gin"
)

// Context keys for storing user data
const (
	UserIDKey = "userID"
)

// Auth returns a middleware that validates JWT tokens.
func Auth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "invalid authorization header format")
			c.Abort()
			return
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		// Store user ID in context for handlers to use
		c.Set(UserIDKey, claims.UserID)

		// Continue to next handler
		c.Next()
	}
}

// GetUserID retrieves the user ID from the context.
// Returns empty string if not found.
func GetUserID(c *gin.Context) string {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return ""
	}
	return userID.(string)
}
