package kafka

import (
	"sync"

	kafkaGo "github.com/segmentio/kafka-go"

	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	k       *kafkaBroker
	topic   string
	options broker.SubscribeOptions
	handler broker.Handler
	reader  *kafkaGo.Reader
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
	if s.reader != nil {
		err = s.reader.Close()
	}
	s.closed = true

	if s.k != nil && s.k.subscribers != nil && removeFromManager {
		_ = s.k.subscribers.RemoveOnly(s.topic)
	}

	return err
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}
