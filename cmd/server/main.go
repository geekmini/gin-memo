package main

import (
	"fmt"
	"log"

	"gin-sample/internal/config"
	"gin-sample/internal/database"
	"gin-sample/internal/handler"
	"gin-sample/internal/repository"
	"gin-sample/internal/router"
	"gin-sample/internal/service"
	"gin-sample/pkg/auth"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()
	log.Println("Configuration loaded")

	// Set Gin mode
	gin.SetMode(cfg.GinMode)

	// Database
	mongoDB := database.NewMongoDB(cfg.MongoURI, cfg.MongoDatabase)
	defer mongoDB.Close()

	// JWT Manager
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTExpiry)

	// Repository layer
	userRepo := repository.NewUserRepository(mongoDB.Database)

	// Service layer
	userService := service.NewUserService(userRepo, jwtManager)

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
