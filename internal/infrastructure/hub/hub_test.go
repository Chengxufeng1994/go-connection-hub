package hub

import (
	"context"
	"io"
	"testing"
	"time"

	"go-notification-sse/internal/infrastructure/logger"
)

func TestHub_StartStop(t *testing.T) {
	logger := &mockLogger{}
	hub := New(logger)

	ctx := context.Background()

	// Test starting hub
	err := hub.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start hub: %v", err)
	}

	if !hub.IsRunning() {
		t.Error("Hub should be running after start")
	}

	// Test stopping hub
	err = hub.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop hub: %v", err)
	}

	if hub.IsRunning() {
		t.Error("Hub should not be running after stop")
	}
}

func TestHub_ConnectionManagement(t *testing.T) {
	logger := &mockLogger{}
	hub := New(logger)

	ctx := context.Background()
	hub.Start(ctx)
	defer hub.Stop(ctx)

	// Initially no connections
	if hub.ConnectionCount() != 0 {
		t.Errorf("Expected 0 connections, got %d", hub.ConnectionCount())
	}

	// Create a mock connection
	conn := &mockConnection{
		id:  "test-conn-1",
		ctx: ctx,
	}

	// Register connection
	err := hub.RegisterConnection(conn)
	if err != nil {
		t.Fatalf("Failed to register connection: %v", err)
	}

	// Give some time for registration to process
	time.Sleep(100 * time.Millisecond)

	if hub.ConnectionCount() != 1 {
		t.Errorf("Expected 1 connection, got %d", hub.ConnectionCount())
	}

	// Get connection
	retrievedConn, exists := hub.GetConnection("test-conn-1")
	if !exists {
		t.Error("Connection should exist")
	}
	if retrievedConn.ID() != "test-conn-1" {
		t.Errorf("Expected connection ID 'test-conn-1', got '%s'", retrievedConn.ID())
	}

	// Unregister connection
	err = hub.UnregisterConnection("test-conn-1")
	if err != nil {
		t.Fatalf("Failed to unregister connection: %v", err)
	}

	// Give some time for unregistration to process
	time.Sleep(100 * time.Millisecond)

	if hub.ConnectionCount() != 0 {
		t.Errorf("Expected 0 connections after unregistration, got %d", hub.ConnectionCount())
	}
}

func TestHub_Broadcasting(t *testing.T) {
	logger := &mockLogger{}
	hub := New(logger)

	ctx := context.Background()
	hub.Start(ctx)
	defer hub.Stop(ctx)

	// Create mock connections
	conn1 := &mockConnection{
		id:  "conn-1",
		ctx: ctx,
	}
	conn2 := &mockConnection{
		id:  "conn-2",
		ctx: ctx,
	}

	// Register connections
	hub.RegisterConnection(conn1)
	hub.RegisterConnection(conn2)

	// Give time for registration
	time.Sleep(100 * time.Millisecond)

	// Create a test message
	message := &Message{
		ID:   "test-msg-1",
		Type: "test",
		Data: "Hello World",
	}

	// Broadcast message
	err := hub.Broadcast(ctx, message)
	if err != nil {
		t.Fatalf("Failed to broadcast message: %v", err)
	}

	// Give time for broadcast to process
	time.Sleep(100 * time.Millisecond)

	// Check that both connections received the message
	if len(conn1.receivedMessages) != 1 {
		t.Errorf("Connection1 should have received 1 message, got %d", len(conn1.receivedMessages))
	}
	if len(conn2.receivedMessages) != 1 {
		t.Errorf("Connection2 should have received 1 message, got %d", len(conn2.receivedMessages))
	}
}

// Mock implementations for testing

type mockLogger struct{}

func (m *mockLogger) Debug(msg string)                              {}
func (m *mockLogger) Debugf(format string, args ...any)             {}
func (m *mockLogger) Info(msg string)                               {}
func (m *mockLogger) Infof(format string, args ...any)              {}
func (m *mockLogger) Warn(msg string)                               {}
func (m *mockLogger) Warnf(format string, args ...any)              {}
func (m *mockLogger) Error(msg string)                              {}
func (m *mockLogger) Errorf(format string, args ...any)             {}
func (m *mockLogger) Fatal(msg string)                              {}
func (m *mockLogger) Fatalf(format string, args ...any)             {}
func (m *mockLogger) WithField(key string, value any) logger.Logger { return m }
func (m *mockLogger) WithFields(fields logger.Fields) logger.Logger { return m }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger { return m }
func (m *mockLogger) SetLevel(level logger.Level)                   {}
func (m *mockLogger) SetOutput(output io.Writer)                    {}

type mockConnection struct {
	id               string
	ctx              context.Context
	closed           bool
	receivedMessages []*Message
}

func (m *mockConnection) ID() string   { return m.id }
func (m *mockConnection) Type() string { return "mock" }
func (m *mockConnection) Send(ctx context.Context, message *Message) error {
	m.receivedMessages = append(m.receivedMessages, message)
	return nil
}
func (m *mockConnection) Close() error             { m.closed = true; return nil }
func (m *mockConnection) IsClosed() bool           { return m.closed }
func (m *mockConnection) Context() context.Context { return m.ctx }
