package broker

import (
	"context"
)

// Event defines the message event interface
type Event interface {
	// Topic returns the topic of the event
	Topic() string

	// Message returns the message associated with the event
	Message() *Message
	// RawMessage returns the original/raw message
	RawMessage() any

	// Ack acknowledges the message
	Ack() error

	// Error returns any error associated with the event
	Error() error
}

// Handler defines the handler invoked by subscribers
type Handler func(ctx context.Context, evt Event) error

// MiddlewareFunc defines the middleware for handlers
type MiddlewareFunc func(Handler) Handler
