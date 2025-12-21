package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gin-sample/internal/authz"
	"gin-sample/internal/cache"
	"gin-sample/internal/config"
	"gin-sample/internal/database"
	"gin-sample/internal/handler"
	"gin-sample/internal/queue"
	"gin-sample/internal/repository"
	"gin-sample/internal/router"
	"gin-sample/internal/service"
	"gin-sample/internal/storage"
	"gin-sample/internal/transcription"
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

	// Transcription queue and processor
	transcriptionQueue := queue.NewMemoryQueue(100)
	transcriptionService := transcription.NewMockService()

	// Service layer
	authService := service.NewAuthService(userRepo, refreshTokenRepo, redisCache, jwtManager, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)
	userService := service.NewUserService(userRepo, redisCache)
	voiceMemoService := service.NewVoiceMemoService(voiceMemoRepo, s3Client, transcriptionQueue)
	teamService := service.NewTeamService(teamRepo, teamMemberRepo, teamInvitationRepo, voiceMemoRepo)
	teamMemberService := service.NewTeamMemberService(teamMemberRepo, userRepo, teamRepo)
	teamInvitationService := service.NewTeamInvitationService(teamInvitationRepo, teamMemberRepo, teamRepo, userRepo)

	// Transcription processor (uses voiceMemoRepo for updates)
	transcriptionProcessor := queue.NewProcessor(transcriptionQueue, transcriptionService, voiceMemoRepo, 2)

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

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start transcription processor
	transcriptionProcessor.Start(ctx)

	// Create HTTP server for graceful shutdown support
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("Shutdown signal received")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server first (drain connections)
	log.Println("Shutting down HTTP server...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Cancel context to signal processor shutdown
	cancel()

	// Stop transcription processor (waits for workers)
	log.Println("Stopping transcription processor...")
	transcriptionProcessor.Stop()

	log.Println("Server shutdown complete")
}
