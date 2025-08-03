package websocket

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"go-notification-sse/internal/infrastructure/hub"
	"go-notification-sse/internal/infrastructure/logger"
)

// WebSocketHandler handles WebSocket connections and messages
type WebSocketHandler struct {
	hub      *hub.Hub
	logger   logger.Logger
	upgrader websocket.Upgrader
}

// NewWebSocketHandler creates a new WebSocket handler instance
func NewWebSocketHandler(hubInstance *hub.Hub, logger logger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hubInstance,
		logger: logger.WithField("handler", "websocket"),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin for development
				// In production, you should implement proper origin checking
				return true
			},
		},
	}
}

// Connect handles WebSocket connection upgrade requests
func (h *WebSocketHandler) Connect(c *gin.Context) {
	h.logger.Info("New WebSocket connection request")

	// Check if hub is running
	if !h.hub.IsRunning() {
		h.logger.Error("Hub is not running")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Service temporarily unavailable",
		})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf("Failed to upgrade connection: %v", err)
		return
	}

	// Generate unique connection ID
	connID := generateWebSocketConnectionID()

	// Create WebSocket connection
	wsConn := hub.NewWebSocketConnection(connID, conn, h.logger)

	// Register connection with hub
	if err := h.hub.RegisterConnection(wsConn); err != nil {
		h.logger.Errorf("Failed to register WebSocket connection: %v", err)
		wsConn.Close()
		return
	}

	h.logger.Infof("WebSocket connection %s connected and registered", wsConn.ID())

	// Keep the connection alive until client disconnects
	<-wsConn.Context().Done()
	h.logger.Infof("WebSocket connection %s disconnected", wsConn.ID())
}

// GetConnections returns information about WebSocket connections
func (h *WebSocketHandler) GetConnections(c *gin.Context) {
	connections := h.hub.GetConnectionsByType("websocket")
	connectionInfo := make([]gin.H, len(connections))

	for i, conn := range connections {
		connectionInfo[i] = gin.H{
			"id":     conn.ID(),
			"type":   conn.Type(),
			"closed": conn.IsClosed(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"total_connections": len(connections),
		"connections":       connectionInfo,
		"hub_running":       h.hub.IsRunning(),
	})
}

// generateWebSocketConnectionID generates a unique WebSocket connection ID
func generateWebSocketConnectionID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("ws-%x", b)
}
