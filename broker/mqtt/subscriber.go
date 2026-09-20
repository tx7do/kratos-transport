package mqtt

import (
	"errors"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	options broker.SubscribeOptions
	m       *mqttBroker

	closed bool
	topic  string
	qos    byte

	callback paho.MessageHandler
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

	if s.m != nil && s.m.client != nil {
		token := s.m.client.Unsubscribe(s.topic)
		// 等待命令完成后再读错误，否则可能在错误未设置时漏报失败
		if !token.WaitTimeout(10 * time.Second) {
			err = errors.New("mqtt unsubscribe timeout")
		} else {
			err = token.Error()
		}
	}

	s.closed = true

	if s.m != nil && s.m.subscribers != nil && removeFromManager {
		_ = s.m.subscribers.RemoveOnly(s.topic)
	}

	return err
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}
