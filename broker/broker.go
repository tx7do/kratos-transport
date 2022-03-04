package broker

import "github.com/tx7do/kratos-transport/common"

type Handler func(Event) error

type Message struct {
	Header map[string]string
	Body   []byte
}

type Event interface {
	Topic() string
	Message() *Message
	Ack() error
	Error() error
}

type Subscriber interface {
	Options() common.SubscribeOptions
	Topic() string
	Unsubscribe() error
}

type Broker interface {
	Init(...common.Option) error
	Options() common.Options
	Address() string
	Connect() error
	Disconnect() error
	Publish(topic string, m *Message, opts ...common.PublishOption) error
	Subscribe(topic string, h Handler, opts ...common.SubscribeOption) (Subscriber, error)
	String() string
}
