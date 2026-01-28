package broker

import (
	"context"
	"fmt"
)

// TypedHandler defines a handler function with a specific message type
type TypedHandler[T any] func(ctx context.Context, topic string, headers Headers, msg *T) error

// TypedToEventHandler converts a TypedHandler to a generic Handler
func TypedToEventHandler[T any](h TypedHandler[T]) Handler {
	return func(ctx context.Context, evt Event) error {
		if evt == nil {
			return fmt.Errorf("evt is nil")
		}
		msg := evt.Message()
		if msg == nil || msg.Body == nil {
			return fmt.Errorf("event message or body is nil")
		}

		var zero T
		expectedType := fmt.Sprintf("%T", &zero)

		switch v := msg.Body.(type) {
		case *T:
			return h(ctx, evt.Topic(), msg.Headers, v)
		case T:
			return h(ctx, evt.Topic(), msg.Headers, &v)
		default:
			return fmt.Errorf("unsupported body type: expected %s, got %T", expectedType, msg.Body)
		}
	}
}
