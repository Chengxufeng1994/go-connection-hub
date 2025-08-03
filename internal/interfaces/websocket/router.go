package websocket

import (
	"go-notification-sse/internal/infrastructure/hub"
	"go-notification-sse/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
)

// InitWebSocketRouter initializes WebSocket routes
func InitWebSocketRouter(logger logger.Logger, hubInstance *hub.Hub, rg *gin.RouterGroup) {
	wsHandler := NewWebSocketHandler(hubInstance, logger)

	// WebSocket connection endpoint
	wsGroup := rg.Group("/ws")
	wsGroup.GET("", wsHandler.Connect)

	// WebSocket API endpoints (only connection info, no broadcast/send)
	apiGroup := rg.Group("/api/v1/ws")
	apiGroup.GET("/connections", wsHandler.GetConnections)
}
