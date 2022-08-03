package kafka

import (
	kafkago "github.com/segmentio/kafka-go"
	"github.com/tx7do/kratos-transport/broker"
	"sync"
)

type subscriber struct {
	k       *kafkaBroker
	topic   string
	opts    broker.SubscribeOptions
	handler broker.Handler
	reader  *kafkago.Reader
	closed  bool
	done    chan struct{}
	sync.RWMutex
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	var err error
	s.Lock()
	defer s.Unlock()
	s.closed = true
	return err
}
