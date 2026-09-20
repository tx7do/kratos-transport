package pulsar

import (
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	r       *pulsarBroker
	topic   string
	options broker.SubscribeOptions
	handler broker.Handler
	reader  pulsar.Consumer
	closed  bool
	channel chan pulsar.ConsumerMessage
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

	if s.closed {
		return nil
	}

	s.closed = true

	// 先停掉消费者再关 channel，避免客户端投递 goroutine 向已关闭的 channel 发送
	var err error
	if s.reader != nil {
		err = s.reader.Unsubscribe()
		s.reader.Close()
	}

	close(s.channel)

	if s.r != nil && s.r.subscribers != nil && removeFromManager {
		_ = s.r.subscribers.RemoveOnly(s.topic)
	}

	return err
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}
