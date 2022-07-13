package nsq

import (
	"github.com/nsqio/go-nsq"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	topic       string
	opts        broker.SubscribeOptions
	consumer    *nsq.Consumer
	handlerFunc nsq.HandlerFunc
	concurrency int
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	s.consumer.Stop()
	return nil
}
