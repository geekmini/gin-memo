// Package router sets up HTTP routes for the API.
package router

import (
	"net/http"

	_ "gin-sample/swagger" // Import generated swagger docs

	"gin-sample/internal/handler"
	"gin-sample/internal/middleware"
	"gin-sample/pkg/auth"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Setup creates and configures the Gin router.
func Setup(userHandler *handler.UserHandler, voiceMemoHandler *handler.VoiceMemoHandler, jwtManager *auth.JWTManager) *gin.Engine {
	r := gin.Default()

	// Global middleware
	r.Use(middleware.CORS())

	// Swagger docs at /docs
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}

		// User routes (protected)
		users := v1.Group("/users")
		users.Use(middleware.Auth(jwtManager))
		{
			users.GET("", userHandler.GetAllUsers)
			users.GET("/:id", userHandler.GetUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}

		// Voice memo routes (protected)
		voiceMemos := v1.Group("/voice-memos")
		voiceMemos.Use(middleware.Auth(jwtManager))
		{
			voiceMemos.GET("", voiceMemoHandler.ListVoiceMemos)
		}
	}

	return r
}
