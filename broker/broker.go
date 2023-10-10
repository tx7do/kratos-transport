package broker

import (
	"context"
	"fmt"
)

type Any interface{}

type Handler func(context.Context, Event) error

type Binder func() Any

type Headers map[string]string

type Message struct {
	Headers Headers
	Body    Any
}

func (m Message) GetHeaders() Headers {
	return m.Headers
}

func (m Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

type Event interface {
	Topic() string
	Message() *Message
	RawMessage() interface{}
	Ack() error
	Error() error
}

type Subscriber interface {
	Options() SubscribeOptions
	Topic() string
	Unsubscribe() error
}

type Broker interface {
	Name() string
	Options() Options
	Address() string

	Init(...Option) error

	Connect() error
	Disconnect() error

	Publish(topic string, msg Any, opts ...PublishOption) error

	Subscribe(topic string, handler Handler, binder Binder, opts ...SubscribeOption) (Subscriber, error)
}

func Subscribe[T any](broker Broker, topic string, handler func(context.Context, string, Headers, *T) error, opts ...SubscribeOption) (Subscriber, error) {
	return broker.Subscribe(
		topic,
		func(ctx context.Context, event Event) error {
			switch t := event.Message().Body.(type) {
			case *T:
				if err := handler(ctx, event.Topic(), event.Message().Headers, t); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported type: %T", t)
			}
			return nil
		},
		func() Any {
			var t T
			return &t
		},
		opts...,
	)
}
