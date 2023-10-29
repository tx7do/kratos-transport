package aliyun

import (
	"sync"

	aliyun "github.com/aliyunmq/mq-http-go-sdk"

	"github.com/tx7do/kratos-transport/broker"
)

type Subscriber struct {
	sync.RWMutex
	r       *aliyunmqBroker
	topic   string
	options broker.SubscribeOptions
	handler broker.Handler
	binder  broker.Binder
	reader  aliyun.MQConsumer
	closed  bool
	done    chan struct{}
}

func (s *Subscriber) Options() broker.SubscribeOptions {
	return s.options
}

func (s *Subscriber) Topic() string {
	return s.topic
}

func (s *Subscriber) Unsubscribe() error {
	var err error
	s.Lock()
	defer s.Unlock()
	s.closed = true
	return err
}
