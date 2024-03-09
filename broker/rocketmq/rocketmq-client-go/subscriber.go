package rocketmqClientGo

import (
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	r       *rocketmqBroker
	topic   string
	options broker.SubscribeOptions
	handler broker.Handler
	reader  rocketmq.PushConsumer
	closed  bool
	done    chan struct{}
}

func (s *subscriber) Options() broker.SubscribeOptions {
	s.RLock()
	defer s.RUnlock()

	return s.options
}

func (s *subscriber) Topic() string {
	s.RLock()
	defer s.RUnlock()

	return s.topic
}

func (s *subscriber) Unsubscribe(removeFromManager bool) error {
	s.Lock()
	defer s.Unlock()

	var err error
	err = s.reader.Unsubscribe(s.topic)

	s.closed = true

	if removeFromManager {

	}

	return err
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}
