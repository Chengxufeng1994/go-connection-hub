package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go-notification-sse/internal/infrastructure/hub"
	"go-notification-sse/internal/infrastructure/logger"
)

type ChatHandler struct {
	hub    *hub.Hub
	logger logger.Logger
}

type ChatMessageRequest struct {
	Username  string `json:"username" binding:"required"`
	Message   string `json:"message" binding:"required"`
	Timestamp string `json:"timestamp"`
}

type ChatMessageResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func NewChatHandler(hubInstance *hub.Hub, logger logger.Logger) *ChatHandler {
	return &ChatHandler{
		hub:    hubInstance,
		logger: logger.WithField("handler", "chat"),
	}
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	var req ChatMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Invalid request format: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid message format",
		})
		return
	}

	// Parse timestamp or use current time
	var timestamp time.Time
	if req.Timestamp != "" {
		if t, err := time.Parse(time.RFC3339, req.Timestamp); err == nil {
			timestamp = t
		} else {
			timestamp = time.Now()
		}
	} else {
		timestamp = time.Now()
	}

	// Create chat message
	messageID := generateMessageID()
	chatMessage := ChatMessageResponse{
		ID:        messageID,
		Username:  req.Username,
		Message:   req.Message,
		Timestamp: timestamp,
	}

	// Create hub message
	hubMessage := &hub.Message{
		ID:   messageID,
		Type: "chat_message",
		Data: chatMessage,
	}

	// Broadcast to all connected clients
	if err := h.hub.Broadcast(c.Request.Context(), hubMessage); err != nil {
		h.logger.Errorf("Failed to broadcast message: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send message",
		})
		return
	}

	h.logger.Infof("Chat message sent by %s to %d connections", req.Username, h.hub.ConnectionCount())

	c.JSON(http.StatusOK, gin.H{
		"status":      "sent",
		"message_id":  messageID,
		"connections": h.hub.ConnectionCount(),
	})
}

func generateMessageID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}