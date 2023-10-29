package broker

import "context"

type Event interface {
	Topic() string

	Message() *Message
	RawMessage() interface{}

	Ack() error

	Error() error
}

type Handler func(ctx context.Context, evt Event) error
