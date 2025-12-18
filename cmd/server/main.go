package main

import (
	"fmt"
	"net/http"

	"gin-sample/internal/config"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration from .env
	cfg := config.Load()

	// Set Gin mode (debug/release)
	gin.SetMode(cfg.GinMode)

	// Create a new Gin router with default middleware (logger & recovery)
	r := gin.Default()

	// Define a health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// Start the server on configured port
	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	r.Run(addr)
}
