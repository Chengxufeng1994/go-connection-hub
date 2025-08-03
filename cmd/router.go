package main

import (
	"go-notification-sse/internal/infrastructure/hub"
	"go-notification-sse/internal/infrastructure/logger"
	"go-notification-sse/internal/interfaces/rest/v1/handler"
	"go-notification-sse/internal/interfaces/sse"
	"go-notification-sse/internal/interfaces/websocket"
	"net/http"

	"github.com/gin-gonic/gin"
)

func InitRouter(hubInstance *hub.Hub, log logger.Logger) http.Handler {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	rootGroup := router.Group("")

	// Simple debug endpoint
	rootGroup.GET("/debug", func(c *gin.Context) {
		log.Info("Debug endpoint hit!")
		c.JSON(http.StatusOK, gin.H{"debug": "working"})
	})

	// Health check endpoint
	rootGroup.GET("/hub/status", func(c *gin.Context) {
		isRunning := hubInstance.IsRunning()
		log.Infof(
			"Hub status check - Running: %v, Connections: %d",
			isRunning,
			hubInstance.ConnectionCount(),
		)
		c.JSON(http.StatusOK, gin.H{
			"status":      "healthy",
			"hub_running": isRunning,
			"connections": hubInstance.ConnectionCount(),
		})
	})

	// Chat API endpoints
	chatHandler := handler.NewChatHandler(hubInstance, log)
	apiGroup := rootGroup.Group("/api")
	{
		apiGroup.POST("/messages", chatHandler.SendMessage)
	}

	sse.InitSSERouter(log, hubInstance, rootGroup)
	websocket.InitWebSocketRouter(log, hubInstance, rootGroup)

	return router
}
