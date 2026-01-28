package broker

import (
	"context"
	"fmt"
)

// Broker defines the message broker interface
type Broker interface {
	// Name returns the broker name
	Name() string

	// Options returns the broker options
	Options() Options

	// Address returns the broker address
	Address() string

	// Init initializes the broker with options
	Init(...Option) error

	// Connect connects to the broker
	Connect() error

	// Disconnect disconnects from the broker
	Disconnect() error

	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, msg any, opts ...PublishOption) error

	// Subscribe subscribes to a topic with a handler and binder
	Subscribe(topic string, handler Handler, binder Binder, opts ...SubscribeOption) (Subscriber, error)

	// Request sends a request message and waits for a response
	Request(ctx context.Context, topic string, msg any, opts ...RequestOption) (any, error)
}

// Subscribe is a helper function to subscribe to a topic with a typed handler
func Subscribe[T any](broker Broker, topic string, handler TypedHandler[T], opts ...SubscribeOption) (Subscriber, error) {
	return broker.Subscribe(
		topic,
		func(ctx context.Context, event Event) error {
			if event == nil || event.Message() == nil || event.Message().Body == nil {
				return fmt.Errorf("event or message body is nil")
			}

			// construct expected type string (pointer to T is expected by handler)
			var zero T
			expectedType := fmt.Sprintf("%T", &zero)

			switch v := event.Message().Body.(type) {
			case *T:
				// already the expected pointer type
				if err := handler(ctx, event.Topic(), event.Message().Headers, v); err != nil {
					return err
				}
			case T:
				// value type: take address of the copy
				if err := handler(ctx, event.Topic(), event.Message().Headers, &v); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported type: expected %s, got %T", expectedType, event.Message().Body)
			}
			return nil
		},
		func() any {
			var t T
			return &t
		},
		opts...,
	)
}
