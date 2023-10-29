package rabbitmq

import (
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	r *rabbitBroker

	options broker.SubscribeOptions
	topic   string
	ch      *rabbitChannel

	queueArgs map[string]interface{}
	fn        func(msg amqp.Delivery)
	headers   map[string]interface{}

	durableQueue bool
	autoDelete   bool
	closed       bool
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

func (s *subscriber) Unsubscribe() error {
	s.Lock()
	defer s.Unlock()

	s.closed = true

	var err error
	if s.ch != nil {
		err = s.ch.Close()
	}

	if s.r != nil && s.r.subscribers != nil {
		_ = s.r.subscribers.Remove(s.topic)
	}

	return err
}

func (s *subscriber) resubscribe() {
	minResubscribeDelay := defaultMinResubscribeDelay
	maxResubscribeDelay := defaultMaxResubscribeDelay
	expFactor := defaultExpFactor
	reSubscribeDelay := defaultResubscribeDelay

	for {
		closed := s.IsClosed()
		if closed {
			// we are unsubscribed, showdown routine
			return
		}

		select {
		case <-s.r.conn.close:
			return
		case <-s.r.conn.waitConnection:
		}

		s.r.mtx.Lock()
		if !s.r.conn.connected {
			s.r.mtx.Unlock()
			continue
		}

		ch, sub, err := s.r.conn.Consume(
			s.options.Queue,
			s.topic,
			s.headers,
			s.queueArgs,
			s.options.AutoAck,
			s.durableQueue,
			s.autoDelete,
		)

		s.r.mtx.Unlock()
		switch err {
		case nil:
			reSubscribeDelay = minResubscribeDelay
			s.Lock()
			s.ch = ch
			s.Unlock()
		default:
			if reSubscribeDelay > maxResubscribeDelay {
				reSubscribeDelay = maxResubscribeDelay
			}
			time.Sleep(reSubscribeDelay)
			reSubscribeDelay *= expFactor
			continue
		}
		for d := range sub {
			s.r.wg.Add(1)
			s.fn(d)
			s.r.wg.Done()
		}
	}
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}
