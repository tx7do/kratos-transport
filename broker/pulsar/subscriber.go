package pulsar

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/tx7do/kratos-transport/broker"
	"sync"
)

type subscriber struct {
	r       *pulsarBroker
	topic   string
	opts    broker.SubscribeOptions
	handler broker.Handler
	reader  pulsar.Consumer
	closed  bool
	channel chan pulsar.ConsumerMessage
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
	s.Lock()
	defer s.Unlock()

	close(s.channel)

	err := s.reader.Unsubscribe()
	s.reader.Close()

	s.closed = true
	return err
}
