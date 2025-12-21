// Package router sets up HTTP routes for the API.
package router

import (
	"net/http"

	_ "gin-sample/swagger" // Import generated swagger docs

	"gin-sample/internal/authz"
	"gin-sample/internal/handler"
	"gin-sample/internal/middleware"
	"gin-sample/pkg/auth"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Config holds all dependencies needed to set up routes.
type Config struct {
	AuthHandler       *handler.AuthHandler
	UserHandler       *handler.UserHandler
	VoiceMemoHandler  *handler.VoiceMemoHandler
	TeamHandler       *handler.TeamHandler
	TeamMemberHandler *handler.TeamMemberHandler
	InvitationHandler *handler.TeamInvitationHandler
	JWTManager        *auth.JWTManager
	Authorizer        authz.Authorizer
}

// Setup creates and configures the Gin router.
func Setup(cfg *Config) *gin.Engine {
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
		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/register", cfg.AuthHandler.Register)
			authRoutes.POST("/login", cfg.AuthHandler.Login)
			authRoutes.POST("/refresh", cfg.AuthHandler.Refresh)
		}

		// Auth routes (protected)
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.Auth(cfg.JWTManager))
		{
			authProtected.POST("/logout", cfg.AuthHandler.Logout)
		}

		// User routes (protected)
		users := v1.Group("/users")
		users.Use(middleware.Auth(cfg.JWTManager))
		{
			users.GET("", cfg.UserHandler.GetAllUsers)
			users.GET("/:id", cfg.UserHandler.GetUser)
			users.PUT("/:id", cfg.UserHandler.UpdateUser)
			users.DELETE("/:id", cfg.UserHandler.DeleteUser)
		}

		// Private voice memo routes (protected)
		voiceMemos := v1.Group("/voice-memos")
		voiceMemos.Use(middleware.Auth(cfg.JWTManager))
		{
			voiceMemos.GET("", cfg.VoiceMemoHandler.ListVoiceMemos)
			voiceMemos.POST("", cfg.VoiceMemoHandler.CreateVoiceMemo)
			voiceMemos.DELETE("/:id", cfg.VoiceMemoHandler.DeleteVoiceMemo)
			voiceMemos.POST("/:id/confirm-upload", cfg.VoiceMemoHandler.ConfirmUpload)
			voiceMemos.POST("/:id/retry-transcription", cfg.VoiceMemoHandler.RetryTranscription)
		}

		// Team routes (protected)
		teams := v1.Group("/teams")
		teams.Use(middleware.Auth(cfg.JWTManager))
		{
			// Team CRUD
			teams.POST("", cfg.TeamHandler.CreateTeam)
			teams.GET("", cfg.TeamHandler.ListTeams)

			// Team routes requiring team membership
			teamWithID := teams.Group("/:teamId")
			{
				// Team details - requires view permission
				teamWithID.GET("", middleware.TeamAuthz(cfg.Authorizer, authz.ActionTeamView), cfg.TeamHandler.GetTeam)
				teamWithID.PUT("", middleware.TeamAuthz(cfg.Authorizer, authz.ActionTeamUpdate), cfg.TeamHandler.UpdateTeam)
				teamWithID.DELETE("", middleware.TeamAuthz(cfg.Authorizer, authz.ActionTeamDelete), cfg.TeamHandler.DeleteTeam)
				teamWithID.POST("/transfer", middleware.TeamAuthz(cfg.Authorizer, authz.ActionTeamTransfer), cfg.TeamHandler.TransferOwnership)

				// Team members
				members := teamWithID.Group("/members")
				{
					members.GET("", middleware.TeamAuthz(cfg.Authorizer, authz.ActionTeamView), cfg.TeamMemberHandler.ListMembers)
					members.DELETE("/:userId", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemberRemove), cfg.TeamMemberHandler.RemoveMember)
					members.PUT("/:userId/role", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemberUpdateRole), cfg.TeamMemberHandler.UpdateRole)
				}
				teamWithID.POST("/leave", middleware.TeamMember(cfg.Authorizer), cfg.TeamMemberHandler.LeaveTeam)

				// Team invitations
				invitations := teamWithID.Group("/invitations")
				{
					invitations.POST("", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemberInvite), cfg.InvitationHandler.CreateInvitation)
					invitations.GET("", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemberInvite), cfg.InvitationHandler.ListTeamInvitations)
					invitations.DELETE("/:id", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemberInvite), cfg.InvitationHandler.CancelInvitation)
				}

				// Team voice memos
				teamMemos := teamWithID.Group("/voice-memos")
				{
					teamMemos.GET("", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemoView), cfg.VoiceMemoHandler.ListTeamVoiceMemos)
					teamMemos.POST("", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemoCreate), cfg.VoiceMemoHandler.CreateTeamVoiceMemo)
					teamMemos.GET("/:id", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemoView), cfg.VoiceMemoHandler.GetTeamVoiceMemo)
					teamMemos.DELETE("/:id", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemoDelete), cfg.VoiceMemoHandler.DeleteTeamVoiceMemo)
					teamMemos.POST("/:id/confirm-upload", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemoCreate), cfg.VoiceMemoHandler.ConfirmTeamUpload)
					teamMemos.POST("/:id/retry-transcription", middleware.TeamAuthz(cfg.Authorizer, authz.ActionMemoCreate), cfg.VoiceMemoHandler.RetryTeamTranscription)
				}
			}
		}

		// User invitations routes (protected)
		invitations := v1.Group("/invitations")
		invitations.Use(middleware.Auth(cfg.JWTManager))
		{
			invitations.GET("", cfg.InvitationHandler.ListMyInvitations)
			invitations.POST("/:id/accept", cfg.InvitationHandler.AcceptInvitation)
			invitations.POST("/:id/decline", cfg.InvitationHandler.DeclineInvitation)
		}
	}

	return r
}
