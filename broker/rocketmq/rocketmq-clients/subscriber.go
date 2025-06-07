package rocketmqClients

import (
	"context"
	"errors"
	"sync"

	rmqClient "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	r       *rocketmqBroker
	options broker.SubscribeOptions
	handler broker.Handler
	binder  broker.Binder

	topic string

	closed bool
	done   chan error

	reader rmqClient.SimpleConsumer
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

	var err error
	if s.reader != nil {
		err = s.reader.Unsubscribe(s.topic)
	} else {
		err = errors.New("reader is nil")
	}
	s.closed = true

	if removeFromManager {
		_ = s.r.subscribers.RemoveOnly(s.topic)
	}

	return err
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}

func (s *subscriber) onMessage(ctx context.Context, msg *rmqClient.MessageView) error {
	if msg == nil {
		return errors.New("message view is nil")
	}

	outMessage := broker.Message{}

	if s.binder != nil {
		outMessage.Body = s.binder()

		if err := broker.Unmarshal(s.r.options.Codec, msg.GetBody(), &outMessage.Body); err != nil {
			//LogError(err)
			return err
		}
	} else {
		outMessage.Body = msg.GetBody()
	}

	outMessage.Headers = msg.GetProperties()

	p := publication{
		ctx:        ctx,
		topic:      msg.GetTopic(),
		message:    &outMessage,
		reader:     s.reader,
		rmqMessage: msg,
	}

	if p.err = s.handler(ctx, &p); p.err != nil {
		return p.err
	}

	if s.options.AutoAck {
		if p.err = p.Ack(); p.err != nil {
			return p.err
		}
	}

	return nil
}
