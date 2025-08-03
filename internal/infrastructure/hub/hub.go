package hub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-notification-sse/internal/infrastructure/logger"
)

// Hub manages connections without depending on specific interfaces
type Hub struct {
	connections   map[string]Connection
	connectionsMu sync.RWMutex

	running   bool
	runningMu sync.RWMutex

	logger logger.Logger

	// Channels for internal communication
	register   chan Connection
	unregister chan string
	broadcast  chan *Message

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new Hub instance
func New(logger logger.Logger) *Hub {
	return &Hub{
		connections: make(map[string]Connection),
		logger:      logger.WithField("component", "hub"),
		register:    make(chan Connection, 100),
		unregister:  make(chan string, 100),
		broadcast:   make(chan *Message, 1000),
	}
}

// Start starts the hub and begins processing connection events
func (h *Hub) Start(ctx context.Context) error {
	h.runningMu.Lock()
	defer h.runningMu.Unlock()

	if h.running {
		return fmt.Errorf("hub is already running")
	}

	h.ctx, h.cancel = context.WithCancel(ctx)
	h.running = true

	go h.run()

	h.logger.Info("Hub started successfully")
	return nil
}

// Stop gracefully stops the hub and disconnects all connections
func (h *Hub) Stop(ctx context.Context) error {
	h.runningMu.Lock()
	defer h.runningMu.Unlock()

	if !h.running {
		return nil
	}

	h.cancel()

	// Close all connections
	h.connectionsMu.Lock()
	for _, conn := range h.connections {
		if err := conn.Close(); err != nil {
			h.logger.Errorf("Failed to close connection %s: %v", conn.ID(), err)
		}
	}
	h.connections = make(map[string]Connection)
	h.connectionsMu.Unlock()

	h.running = false
	h.logger.Info("Hub stopped successfully")
	return nil
}

// IsRunning returns true if the hub is currently running
func (h *Hub) IsRunning() bool {
	h.runningMu.RLock()
	defer h.runningMu.RUnlock()
	return h.running
}

// RegisterConnection adds a new connection to the hub
func (h *Hub) RegisterConnection(conn Connection) error {
	if !h.IsRunning() {
		return fmt.Errorf("hub is not running")
	}

	select {
	case h.register <- conn:
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("hub is shutting down")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout registering connection")
	}
}

// UnregisterConnection removes a connection from the hub
func (h *Hub) UnregisterConnection(connID string) error {
	if !h.IsRunning() {
		return fmt.Errorf("hub is not running")
	}

	select {
	case h.unregister <- connID:
		return nil
	case <-h.ctx.Done():
		return fmt.Errorf("hub is shutting down")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout unregistering connection")
	}
}

// GetConnection returns a connection by ID
func (h *Hub) GetConnection(connID string) (Connection, bool) {
	h.connectionsMu.RLock()
	defer h.connectionsMu.RUnlock()

	conn, exists := h.connections[connID]
	return conn, exists
}

// GetConnections returns all active connections
func (h *Hub) GetConnections() []Connection {
	h.connectionsMu.RLock()
	defer h.connectionsMu.RUnlock()

	connections := make([]Connection, 0, len(h.connections))
	for _, conn := range h.connections {
		connections = append(connections, conn)
	}
	return connections
}

// GetConnectionsByType returns connections of a specific type
func (h *Hub) GetConnectionsByType(connType string) []Connection {
	h.connectionsMu.RLock()
	defer h.connectionsMu.RUnlock()

	var connections []Connection
	for _, conn := range h.connections {
		if conn.Type() == connType {
			connections = append(connections, conn)
		}
	}
	return connections
}

// ConnectionCount returns the number of active connections
func (h *Hub) ConnectionCount() int {
	h.connectionsMu.RLock()
	defer h.connectionsMu.RUnlock()
	return len(h.connections)
}

// Broadcast sends a message to all connections
func (h *Hub) Broadcast(ctx context.Context, message *Message) error {
	if !h.IsRunning() {
		return fmt.Errorf("hub is not running")
	}

	select {
	case h.broadcast <- message:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	case <-h.ctx.Done():
		return fmt.Errorf("hub is shutting down")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout broadcasting message")
	}
}

// BroadcastToType sends a message to all connections of a specific type
func (h *Hub) BroadcastToType(ctx context.Context, connType string, message *Message) error {
	connections := h.GetConnectionsByType(connType)

	for _, conn := range connections {
		go func(c Connection) {
			if err := c.Send(ctx, message); err != nil {
				h.logger.Errorf("Failed to send message to connection %s: %v", c.ID(), err)
				// Auto-unregister failed connections
				h.UnregisterConnection(c.ID())
			}
		}(conn)
	}

	h.logger.Infof("Broadcasted message to %d connections of type %s", len(connections), connType)
	return nil
}

// SendToConnection sends a message to a specific connection
func (h *Hub) SendToConnection(ctx context.Context, connID string, message *Message) error {
	conn, exists := h.GetConnection(connID)
	if !exists {
		return fmt.Errorf("connection %s not found", connID)
	}

	if err := conn.Send(ctx, message); err != nil {
		h.logger.Errorf("Failed to send message to connection %s: %v", connID, err)
		// Auto-unregister failed connections
		h.UnregisterConnection(connID)
		return err
	}

	return nil
}

// run is the main hub loop that processes connection events
func (h *Hub) run() {
	ticker := time.NewTicker(30 * time.Second) // Cleanup interval
	defer ticker.Stop()

	for {
		select {
		case conn := <-h.register:
			h.handleRegister(conn)

		case connID := <-h.unregister:
			h.handleUnregister(connID)

		case message := <-h.broadcast:
			h.handleBroadcast(message)

		case <-ticker.C:
			h.cleanupClosedConnections()

		case <-h.ctx.Done():
			h.logger.Info("Hub run loop stopped")
			return
		}
	}
}

// handleRegister processes connection registration
func (h *Hub) handleRegister(conn Connection) {
	h.connectionsMu.Lock()
	h.connections[conn.ID()] = conn
	h.connectionsMu.Unlock()

	h.logger.Infof("Connection %s registered (type: %s)", conn.ID(), conn.Type())

	// Monitor connection context for disconnection
	go func() {
		<-conn.Context().Done()
		h.UnregisterConnection(conn.ID())
	}()
}

// handleUnregister processes connection unregistration
func (h *Hub) handleUnregister(connID string) {
	h.connectionsMu.Lock()
	conn, exists := h.connections[connID]
	if exists {
		delete(h.connections, connID)
		conn.Close()
	}
	h.connectionsMu.Unlock()

	if exists {
		h.logger.Infof("Connection %s unregistered", connID)
	}
}

// handleBroadcast processes broadcast messages
func (h *Hub) handleBroadcast(message *Message) {
	connections := h.GetConnections()
	successCount := 0

	for _, conn := range connections {
		go func(c Connection) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := c.Send(ctx, message); err != nil {
				h.logger.Errorf("Failed to send broadcast to connection %s: %v", c.ID(), err)
				h.UnregisterConnection(c.ID())
			} else {
				successCount++
			}
		}(conn)
	}

	h.logger.Infof("Broadcasted message %s to %d connections", message.ID, len(connections))
}

// cleanupClosedConnections removes connections that have been closed
func (h *Hub) cleanupClosedConnections() {
	h.connectionsMu.Lock()
	defer h.connectionsMu.Unlock()

	closedConnections := make([]string, 0)
	for id, conn := range h.connections {
		if conn.IsClosed() {
			closedConnections = append(closedConnections, id)
		}
	}

	for _, id := range closedConnections {
		delete(h.connections, id)
		h.logger.Infof("Cleaned up closed connection %s", id)
	}
}
