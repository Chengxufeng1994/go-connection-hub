package hub

import "context"

// Connection represents any type of connection (SSE, WebSocket, etc.)
type Connection interface {
	ID() string
	Type() string
	Send(ctx context.Context, message *Message) error
	Close() error
	IsClosed() bool
	Context() context.Context
}

// Message represents a message to be sent through connections
type Message struct {
	ID      string            `json:"id"`
	Type    string            `json:"type"`
	Data    interface{}       `json:"data"`
	Headers map[string]string `json:"headers,omitempty"`
}

