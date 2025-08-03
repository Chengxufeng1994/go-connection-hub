package hub

import (
	"encoding/json"
	"fmt"
	"time"
)

// MessageType defines common message types
type MessageType string

const (
	MessageTypeConnection   MessageType = "connection"
	MessageTypeNotification MessageType = "notification"
	MessageTypeKeepAlive    MessageType = "keepalive"
	MessageTypeError        MessageType = "error"
	MessageTypeAlert        MessageType = "alert"
	MessageTypeUpdate       MessageType = "update"
	MessageTypeSystem       MessageType = "system"
	MessageTypeBroadcast    MessageType = "broadcast"
)

// MessagePriority defines message priority levels
type MessagePriority string

const (
	PriorityLow    MessagePriority = "low"
	PriorityNormal MessagePriority = "normal"
	PriorityHigh   MessagePriority = "high"
	PriorityCritical MessagePriority = "critical"
)

// MessageBuilder helps build messages with fluent interface
type MessageBuilder struct {
	message *Message
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		message: &Message{
			Headers: make(map[string]string),
		},
	}
}

// WithID sets the message ID
func (mb *MessageBuilder) WithID(id string) *MessageBuilder {
	mb.message.ID = id
	return mb
}

// WithType sets the message type
func (mb *MessageBuilder) WithType(msgType MessageType) *MessageBuilder {
	mb.message.Type = string(msgType)
	return mb
}

// WithData sets the message data
func (mb *MessageBuilder) WithData(data interface{}) *MessageBuilder {
	mb.message.Data = data
	return mb
}

// WithHeader adds a header to the message
func (mb *MessageBuilder) WithHeader(key, value string) *MessageBuilder {
	if mb.message.Headers == nil {
		mb.message.Headers = make(map[string]string)
	}
	mb.message.Headers[key] = value
	return mb
}

// WithPriority sets the message priority
func (mb *MessageBuilder) WithPriority(priority MessagePriority) *MessageBuilder {
	return mb.WithHeader("priority", string(priority))
}

// WithTimestamp adds a timestamp to the message
func (mb *MessageBuilder) WithTimestamp() *MessageBuilder {
	return mb.WithHeader("timestamp", time.Now().UTC().Format(time.RFC3339))
}

// Build returns the constructed message
func (mb *MessageBuilder) Build() *Message {
	// Auto-generate ID if not provided
	if mb.message.ID == "" {
		mb.message.ID = generateMessageID()
	}
	
	// Auto-add timestamp if not present
	if _, exists := mb.message.Headers["timestamp"]; !exists {
		mb.WithTimestamp()
	}
	
	return mb.message
}

// NotificationMessage creates a notification message
func NotificationMessage(title, body string) *Message {
	return NewMessageBuilder().
		WithType(MessageTypeNotification).
		WithData(map[string]interface{}{
			"title": title,
			"body":  body,
		}).
		WithPriority(PriorityNormal).
		Build()
}

// AlertMessage creates an alert message
func AlertMessage(level, message string) *Message {
	return NewMessageBuilder().
		WithType(MessageTypeAlert).
		WithData(map[string]interface{}{
			"level":   level,
			"message": message,
		}).
		WithPriority(PriorityHigh).
		Build()
}

// SystemMessage creates a system message
func SystemMessage(action string, data interface{}) *Message {
	return NewMessageBuilder().
		WithType(MessageTypeSystem).
		WithData(map[string]interface{}{
			"action": action,
			"data":   data,
		}).
		WithPriority(PriorityNormal).
		Build()
}

// UpdateMessage creates an update message
func UpdateMessage(resource string, data interface{}) *Message {
	return NewMessageBuilder().
		WithType(MessageTypeUpdate).
		WithData(map[string]interface{}{
			"resource": resource,
			"data":     data,
		}).
		WithPriority(PriorityNormal).
		Build()
}

// BroadcastMessage creates a broadcast message
func BroadcastMessage(data interface{}) *Message {
	return NewMessageBuilder().
		WithType(MessageTypeBroadcast).
		WithData(data).
		WithPriority(PriorityNormal).
		Build()
}

// ErrorMessage creates an error message
func ErrorMessage(code string, message string, details interface{}) *Message {
	return NewMessageBuilder().
		WithType(MessageTypeError).
		WithData(map[string]interface{}{
			"code":    code,
			"message": message,
			"details": details,
		}).
		WithPriority(PriorityHigh).
		Build()
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// MessageValidator validates messages before sending
type MessageValidator struct{}

// NewMessageValidator creates a new message validator
func NewMessageValidator() *MessageValidator {
	return &MessageValidator{}
}

// Validate validates a message
func (mv *MessageValidator) Validate(message *Message) error {
	if message == nil {
		return fmt.Errorf("message cannot be nil")
	}
	
	if message.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}
	
	if message.Type == "" {
		return fmt.Errorf("message type cannot be empty")
	}
	
	// Validate data can be JSON marshaled
	if message.Data != nil {
		if _, err := json.Marshal(message.Data); err != nil {
			return fmt.Errorf("message data must be JSON serializable: %w", err)
		}
	}
	
	return nil
}

// IsValidMessageType checks if a message type is valid
func IsValidMessageType(msgType string) bool {
	validTypes := []MessageType{
		MessageTypeConnection,
		MessageTypeNotification,
		MessageTypeKeepAlive,
		MessageTypeError,
		MessageTypeAlert,
		MessageTypeUpdate,
		MessageTypeSystem,
		MessageTypeBroadcast,
	}
	
	for _, validType := range validTypes {
		if string(validType) == msgType {
			return true
		}
	}
	return false
}

// GetMessagePriority extracts priority from message headers
func GetMessagePriority(message *Message) MessagePriority {
	if message.Headers == nil {
		return PriorityNormal
	}
	
	priority, exists := message.Headers["priority"]
	if !exists {
		return PriorityNormal
	}
	
	switch MessagePriority(priority) {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityCritical:
		return MessagePriority(priority)
	default:
		return PriorityNormal
	}
}