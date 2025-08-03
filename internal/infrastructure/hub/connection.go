package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go-notification-sse/internal/infrastructure/logger"

	"github.com/gorilla/websocket"
)

// SSEConnection implements the Connection interface for Server-Sent Events
type SSEConnection struct {
	id      string
	writer  http.ResponseWriter
	request *http.Request

	ctx    context.Context
	cancel context.CancelFunc

	closed   bool
	closedMu sync.RWMutex

	logger logger.Logger

	// Keep-alive mechanism
	lastActivity time.Time
	activityMu   sync.RWMutex
}

// NewSSEConnection creates a new SSE connection
func NewSSEConnection(
	ctx context.Context,
	id string,
	w http.ResponseWriter,
	r *http.Request,
	logger logger.Logger,
) *SSEConnection {
	rctx, cancel := context.WithCancel(ctx)

	conn := &SSEConnection{
		id:           id,
		writer:       w,
		request:      r,
		ctx:          rctx,
		cancel:       cancel,
		logger:       logger.WithField("connection_id", id),
		lastActivity: time.Now(),
	}

	// Set up proper SSE headers
	conn.setupSSEHeaders()

	// Start keep-alive mechanism
	go conn.keepAlive()

	return conn
}

// ID returns unique connection identifier
func (c *SSEConnection) ID() string {
	return c.id
}

// Type returns the connection type
func (c *SSEConnection) Type() string {
	return "sse"
}

// Send sends a message to this connection via SSE
func (c *SSEConnection) Send(ctx context.Context, message *Message) error {
	if c.IsClosed() {
		return fmt.Errorf("client is closed")
	}

	c.updateActivity()

	// Create SSE formatted message
	sseMessage, err := c.formatSSEMessage(message)
	if err != nil {
		return fmt.Errorf("failed to format SSE message: %w", err)
	}

	// Write message to client with timeout
	done := make(chan error, 1)
	go func() {
		_, err := c.writer.Write(sseMessage)
		if err != nil {
			done <- err
			return
		}

		// Flush the data to ensure it's sent immediately
		if flusher, ok := c.writer.(http.Flusher); ok {
			flusher.Flush()
		}

		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			c.logger.Errorf("Failed to write message: %v", err)
			c.Close()
			return err
		}
		return nil

	case <-ctx.Done():
		c.logger.Warn("Send operation cancelled")
		return ctx.Err()

	case <-time.After(10 * time.Second):
		c.logger.Warn("Send operation timed out")
		c.Close()
		return fmt.Errorf("send timeout")
	}
}

// Close gracefully closes the connection
func (c *SSEConnection) Close() error {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.cancel()

	c.logger.Info("SSE connection closed")
	return nil
}

// IsClosed returns true if connection is closed
func (c *SSEConnection) IsClosed() bool {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()
	return c.closed
}

// Context returns the connection's context (for cancellation)
func (c *SSEConnection) Context() context.Context {
	return c.ctx
}

// setupSSEHeaders sets up the proper headers for SSE connection
func (c *SSEConnection) setupSSEHeaders() {
	c.writer.Header().Set("Content-Type", "text/event-stream")
	c.writer.Header().Set("Cache-Control", "no-cache")
	c.writer.Header().Set("Connection", "keep-alive")
	c.writer.Header().Set("X-Accel-Buffering", "no") // For nginx
	c.writer.Header().Set("Access-Control-Allow-Origin", "*")
	c.writer.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
}

// formatSSEMessage formats a message according to SSE specification
func (c *SSEConnection) formatSSEMessage(message *Message) ([]byte, error) {
	var result []byte

	// Add message ID if present
	if message.ID != "" {
		result = append(result, fmt.Sprintf("id: %s\n", message.ID)...)
	}

	// Add event type if present
	if message.Type != "" {
		result = append(result, fmt.Sprintf("event: %s\n", message.Type)...)
	}

	// Add data (JSON encode if necessary)
	var data string
	switch v := message.Data.(type) {
	case string:
		data = v
	case []byte:
		data = string(v)
	default:
		jsonData, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal data: %w", err)
		}
		data = string(jsonData)
	}

	// Split multi-line data
	lines := splitLines(data)
	for _, line := range lines {
		result = append(result, fmt.Sprintf("data: %s\n", line)...)
	}

	// End with double newline
	result = append(result, "\n"...)

	return result, nil
}

// keepAlive sends periodic keep-alive messages to maintain connection
func (c *SSEConnection) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if c.IsClosed() {
				return
			}

			// Check if client is still active (last activity within 5 minutes)
			c.activityMu.RLock()
			lastActivity := c.lastActivity
			c.activityMu.RUnlock()

			if time.Since(lastActivity) > 5*time.Minute {
				c.logger.Info("Connection inactive for too long, closing connection")
				c.Close()
				return
			}

			// Send keep-alive message
			keepAliveMsg := &Message{
				ID:   fmt.Sprintf("keepalive-%d", time.Now().Unix()),
				Type: "keepalive",
				Data: map[string]interface{}{
					"timestamp": time.Now().Unix(),
					"message":   "connection alive",
				},
			}

			if err := c.Send(context.Background(), keepAliveMsg); err != nil {
				c.logger.Errorf("Failed to send keep-alive: %v", err)
				c.Close()
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// updateActivity updates the last activity timestamp
func (c *SSEConnection) updateActivity() {
	c.activityMu.Lock()
	c.lastActivity = time.Now()
	c.activityMu.Unlock()
}

// splitLines splits a string into lines for SSE data field formatting
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}

	var lines []string
	current := ""

	for _, char := range s {
		if char == '\n' || char == '\r' {
			lines = append(lines, current)
			current = ""
			continue
		}
		current += string(char)
	}

	// Add the last line if it's not empty or if the string doesn't end with a newline
	if current != "" || len(lines) == 0 {
		lines = append(lines, current)
	}

	return lines
}

// WebSocketConnection implements the Connection interface for WebSocket connections
type WebSocketConnection struct {
	id   string
	conn *websocket.Conn

	ctx    context.Context
	cancel context.CancelFunc

	closed   bool
	closedMu sync.RWMutex

	logger logger.Logger

	// Message sending channel
	send chan *Message

	// Keep-alive mechanism
	lastActivity time.Time
	activityMu   sync.RWMutex

	// Write timeout for WebSocket operations
	writeTimeout time.Duration

	// Pong timeout for connection health
	pongTimeout time.Duration
}

// NewWebSocketConnection creates a new WebSocket connection
func NewWebSocketConnection(
	id string,
	conn *websocket.Conn,
	logger logger.Logger,
) *WebSocketConnection {
	ctx, cancel := context.WithCancel(context.Background())

	wsConn := &WebSocketConnection{
		id:           id,
		conn:         conn,
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger.WithField("connection_id", id),
		send:         make(chan *Message, 256),
		lastActivity: time.Now(),
		writeTimeout: 10 * time.Second,
		pongTimeout:  60 * time.Second,
	}

	// Set up WebSocket connection settings
	wsConn.setupWebSocket()

	// Start background routines
	go wsConn.writePump()
	go wsConn.readPump()

	return wsConn
}

// ID returns unique connection identifier
func (c *WebSocketConnection) ID() string {
	return c.id
}

// Type returns the connection type
func (c *WebSocketConnection) Type() string {
	return "websocket"
}

// Send sends a message to this WebSocket connection
func (c *WebSocketConnection) Send(ctx context.Context, message *Message) error {
	if c.IsClosed() {
		return fmt.Errorf("WebSocket connection is closed")
	}

	select {
	case c.send <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-c.ctx.Done():
		return fmt.Errorf("connection closed")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send timeout")
	}
}

// Close gracefully closes the WebSocket connection
func (c *WebSocketConnection) Close() error {
	c.closedMu.Lock()
	defer c.closedMu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	c.cancel()

	// Close the send channel
	close(c.send)

	// Send close message and close WebSocket connection
	c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	c.conn.WriteMessage(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
	)
	c.conn.Close()

	c.logger.Info("WebSocket connection closed")
	return nil
}

// IsClosed returns true if connection is closed
func (c *WebSocketConnection) IsClosed() bool {
	c.closedMu.RLock()
	defer c.closedMu.RUnlock()
	return c.closed
}

// Context returns the connection's context (for cancellation)
func (c *WebSocketConnection) Context() context.Context {
	return c.ctx
}

// setupWebSocket configures WebSocket connection settings
func (c *WebSocketConnection) setupWebSocket() {
	// Set read deadline and pong handler for keep-alive
	c.conn.SetReadDeadline(time.Now().Add(c.pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		c.updateActivity()
		c.conn.SetReadDeadline(time.Now().Add(c.pongTimeout))
		return nil
	})
}

// writePump handles sending messages to the WebSocket connection
func (c *WebSocketConnection) writePump() {
	ticker := time.NewTicker(
		54 * time.Second,
	) // Send ping every 54 seconds (less than pong timeout)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))

			if !ok {
				// The send channel was closed
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send the message as JSON
			if err := c.conn.WriteJSON(message); err != nil {
				c.logger.Errorf("Failed to write message: %v", err)
				return
			}

			c.updateActivity()

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Errorf("Failed to send ping: %v", err)
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// readPump handles reading messages from the WebSocket connection
func (c *WebSocketConnection) readPump() {
	defer func() {
		c.Close()
	}()

	for {
		// Read message from client
		messageType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				c.logger.Errorf("WebSocket error: %v", err)
			}
			break
		}

		c.updateActivity()

		// Handle different message types
		switch messageType {
		case websocket.TextMessage:
			c.logger.Debugf("Received text message: %s", string(data))
			// Echo the message back for demonstration
			// In a real application, you would process the message here
			response := &Message{
				ID:   fmt.Sprintf("echo-%d", time.Now().Unix()),
				Type: "echo",
				Data: map[string]interface{}{
					"original":  string(data),
					"timestamp": time.Now().Unix(),
				},
			}

			// Send echo response
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := c.Send(ctx, response); err != nil {
				c.logger.Errorf("Failed to send echo response: %v", err)
			}
			cancel()

		case websocket.BinaryMessage:
			c.logger.Debugf("Received binary message of length: %d", len(data))
			// Handle binary messages if needed

		case websocket.CloseMessage:
			c.logger.Info("Received close message from client")
			return
		}
	}
}

// updateActivity updates the last activity timestamp
func (c *WebSocketConnection) updateActivity() {
	c.activityMu.Lock()
	c.lastActivity = time.Now()
	c.activityMu.Unlock()
}

