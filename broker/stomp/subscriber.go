package stomp

import (
	stompV3 "github.com/go-stomp/stomp/v3"

	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	opts  broker.SubscribeOptions
	topic string
	sub   *stompV3.Subscription
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	return s.sub.Unsubscribe()
}
