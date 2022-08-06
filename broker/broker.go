package broker

import (
	"context"
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
