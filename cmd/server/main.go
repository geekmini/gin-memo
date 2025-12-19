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

	// JWT Manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry)

	// Repository layer
	userRepo := repository.NewUserRepository(mongoDB.Database)

	// Service layer
	userService := service.NewUserService(userRepo, redisCache, jwtManager)

	// Handler layer
	userHandler := handler.NewUserHandler(userService)

	// Router
	r := router.Setup(userHandler, jwtManager)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
