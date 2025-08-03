package sse

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/gin-gonic/gin"

	"go-notification-sse/internal/infrastructure/hub"
	"go-notification-sse/internal/infrastructure/logger"
)

type ServerSentEventHandler struct {
	hub    *hub.Hub
	logger logger.Logger
}

func NewServerSentEventHandler(hubInstance *hub.Hub, logger logger.Logger) *ServerSentEventHandler {
	return &ServerSentEventHandler{
		hub:    hubInstance,
		logger: logger.WithField("handler", "sse"),
	}
}

// Connect handles SSE connection requests
func (h *ServerSentEventHandler) Connect(c *gin.Context) {
	h.logger.Info("New SSE connection request")

	// Check if hub is running
	isRunning := h.hub.IsRunning()
	h.logger.Infof("Hub running check in SSE handler: %v", isRunning)
	if !isRunning {
		h.logger.Error("Hub is not running")
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Service temporarily unavailable",
		})
		return
	}

	w := c.Writer

	// Generate unique connection ID
	connID := generateConnectionID()

	// Create SSE connection
	conn := hub.NewSSEConnection(c.Request.Context(), connID, w, c.Request, h.logger)

	// Register connection with hub
	if err := h.hub.RegisterConnection(conn); err != nil {
		h.logger.Errorf("Failed to register connection: %v", err)
		_ = conn.Close()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register connection",
		})
		return
	}

	h.logger.Infof("SSE connection %s connected and registered", conn.ID())
	sse.Encode(w, sse.Event{
		Event: "connected",
		Data: map[string]interface{}{
			"connection_id": conn.ID(),
			"timestamp":     time.Now().Format(time.RFC3339),
		},
	})
	w.Flush()

	clientGone := w.CloseNotify()
	// Keep the connection alive until client disconnects
	for {
		select {
		case <-conn.Context().Done():
			h.logger.Infof("client connection context canceled %s", conn.ID())
			return
		case <-clientGone:
			h.logger.Infof("clietn disconnected %s", conn.ID())
			return
		}
	}
}

// SendMessage sends a message to a specific client (for testing/admin purposes)
func (h *ServerSentEventHandler) SendMessage(c *gin.Context) {
	clientID := c.Param("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Client ID is required",
		})
		return
	}

	var messageReq struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&messageReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid message format",
		})
		return
	}

	message := &hub.Message{
		ID:   generateMessageID(),
		Type: messageReq.Type,
		Data: messageReq.Data,
	}

	if err := h.hub.SendToConnection(c.Request.Context(), clientID, message); err != nil {
		h.logger.Errorf("Failed to send message to client %s: %v", clientID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send message",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "sent",
		"client_id":  clientID,
		"message_id": message.ID,
	})
}

// BroadcastMessage broadcasts a message to all connected clients
func (h *ServerSentEventHandler) BroadcastMessage(c *gin.Context) {
	var messageReq struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}

	if err := c.ShouldBindJSON(&messageReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid message format",
		})
		return
	}

	message := &hub.Message{
		ID:   generateMessageID(),
		Type: messageReq.Type,
		Data: messageReq.Data,
	}

	if err := h.hub.Broadcast(c.Request.Context(), message); err != nil {
		h.logger.Errorf("Failed to broadcast message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to broadcast message",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "broadcasted",
		"message_id":  message.ID,
		"connections": h.hub.ConnectionCount(),
	})
}

// GetConnections returns information about connected connections
func (h *ServerSentEventHandler) GetConnections(c *gin.Context) {
	connections := h.hub.GetConnections()
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

// generateConnectionID generates a unique connection ID
func generateConnectionID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("conn-%x", b)
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("msg-%x", b)
}
