package sse

import (
	"github.com/gin-gonic/gin"

	"go-notification-sse/internal/infrastructure/hub"
	"go-notification-sse/internal/infrastructure/logger"
)

func InitSSERouter(logger logger.Logger, hubInstance *hub.Hub, rg *gin.RouterGroup) {
	sseHandler := NewServerSentEventHandler(hubInstance, logger)

	// SSE connection endpoint
	sseGroup := rg.Group("/sse")
	sseGroup.GET("", SSEHeadersMiddleware(), sseHandler.Connect)

	// Broadcasting API endpoints
	apiGroup := rg.Group("/api/v1/sse")
	apiGroup.GET("/connections", sseHandler.GetConnections)
	apiGroup.POST("/broadcast", sseHandler.BroadcastMessage)
	apiGroup.POST("/send/:clientId", sseHandler.SendMessage)
}
