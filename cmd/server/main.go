package main

import (
	"fmt"
	"log"

	"gin-sample/internal/authz"
	"gin-sample/internal/cache"
	"gin-sample/internal/config"
	"gin-sample/internal/database"
	"gin-sample/internal/handler"
	"gin-sample/internal/repository"
	"gin-sample/internal/router"
	"gin-sample/internal/service"
	"gin-sample/internal/storage"
	"gin-sample/internal/validator"
	"gin-sample/pkg/auth"

	"github.com/gin-gonic/gin"
)

// @title           Gin Sample API
// @version         1.0
// @description     A REST API for user management built with Gin, MongoDB, and Redis.

// @contact.name    API Support
// @contact.email   support@example.com

// @host            localhost:8080
// @BasePath        /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

func main() {
	// Load configuration
	cfg := config.Load()
	log.Println("Configuration loaded")

	// Register custom validators
	validator.RegisterCustomValidators()

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Database
	mongoDB := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase)
	defer mongoDB.Close()

	// Redis Cache
	redisCache := cache.NewRedis(cfg.RedisURI)
	defer redisCache.Close()

	// S3 Storage
	s3Client := storage.NewS3Client(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3UseSSL)

	// JWT Manager
	jwtManager := auth.NewJWTManager(cfg.AccessTokenSecret, cfg.AccessTokenExpiry)

	// Repository layer
	userRepo := repository.NewUserRepository(mongoDB.Database)
	refreshTokenRepo := repository.NewRefreshTokenRepository(mongoDB.Database)
	voiceMemoRepo := repository.NewVoiceMemoRepository(mongoDB.Database)
	teamRepo := repository.NewTeamRepository(mongoDB.Database)
	teamMemberRepo := repository.NewTeamMemberRepository(mongoDB.Database)
	teamInvitationRepo := repository.NewTeamInvitationRepository(mongoDB.Database)

	// Authorization
	authorizer := authz.NewLocalAuthorizer(teamMemberRepo)

	// Service layer
	authService := service.NewAuthService(userRepo, refreshTokenRepo, redisCache, jwtManager, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)
	userService := service.NewUserService(userRepo, redisCache)
	voiceMemoService := service.NewVoiceMemoService(voiceMemoRepo, s3Client)
	teamService := service.NewTeamService(teamRepo, teamMemberRepo, teamInvitationRepo, voiceMemoRepo)
	teamMemberService := service.NewTeamMemberService(teamMemberRepo, userRepo, teamRepo)
	teamInvitationService := service.NewTeamInvitationService(teamInvitationRepo, teamMemberRepo, teamRepo, userRepo)

	// Handler layer
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	voiceMemoHandler := handler.NewVoiceMemoHandler(voiceMemoService)
	teamHandler := handler.NewTeamHandler(teamService)
	teamMemberHandler := handler.NewTeamMemberHandler(teamMemberService)
	invitationHandler := handler.NewTeamInvitationHandler(teamInvitationService, userService)

	// Router
	r := router.Setup(&router.Config{
		AuthHandler:       authHandler,
		UserHandler:       userHandler,
		VoiceMemoHandler:  voiceMemoHandler,
		TeamHandler:       teamHandler,
		TeamMemberHandler: teamMemberHandler,
		InvitationHandler: invitationHandler,
		JWTManager:        jwtManager,
		Authorizer:        authorizer,
	})

	// Start server
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
