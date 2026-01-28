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

// SubscriberMiddleware is broker subscriber middleware.
type SubscriberMiddleware func(Handler) Handler

// ChainSubscriberMiddleware chains multiple SubscriberMiddleware into a single Handler.
func ChainSubscriberMiddleware(h Handler, mws []SubscriberMiddleware) Handler {
	if len(mws) == 0 {
		return h
	}
	for i := len(mws) - 1; i >= 0; i-- {
		if mws[i] == nil {
			continue
		}
		h = mws[i](h)
	}
	return h
}
