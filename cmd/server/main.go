package main

import (
	"fmt"
	"log"

	"gin-sample/internal/cache"
	"gin-sample/internal/config"
	"gin-sample/internal/database"
	"gin-sample/internal/handler"
	"gin-sample/internal/repository"
	"gin-sample/internal/router"
	"gin-sample/internal/service"
	"gin-sample/internal/storage"
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
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry)

	// Repository layer
	userRepo := repository.NewUserRepository(mongoDB.Database)
	voiceMemoRepo := repository.NewVoiceMemoRepository(mongoDB.Database)

	// Service layer
	userService := service.NewUserService(userRepo, redisCache, jwtManager)
	voiceMemoService := service.NewVoiceMemoService(voiceMemoRepo, s3Client)

	// Handler layer
	userHandler := handler.NewUserHandler(userService)
	voiceMemoHandler := handler.NewVoiceMemoHandler(voiceMemoService)

	// Router
	r := router.Setup(userHandler, voiceMemoHandler, jwtManager)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
