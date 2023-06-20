package rabbitmq

import (
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultMinResubscribeDelay = 100 * time.Millisecond
	defaultMaxResubscribeDelay = 30 * time.Second
	defaultExpFactor           = time.Duration(2)
	defaultResubscribeDelay    = defaultMinResubscribeDelay
)

type subscriber struct {
	mtx          sync.Mutex
	mayRun       bool
	opts         broker.SubscribeOptions
	topic        string
	ch           *rabbitChannel
	durableQueue bool
	autoDelete   bool
	queueArgs    map[string]interface{}
	r            *rabbitBroker
	fn           func(msg amqp.Delivery)
	headers      map[string]interface{}
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.mayRun = false
	if s.ch != nil {
		return s.ch.Close()
	}
	return nil
}

func (s *subscriber) resubscribe() {
	minResubscribeDelay := defaultMinResubscribeDelay
	maxResubscribeDelay := defaultMaxResubscribeDelay
	expFactor := defaultExpFactor
	reSubscribeDelay := defaultResubscribeDelay

	for {
		s.mtx.Lock()
		mayRun := s.mayRun
		s.mtx.Unlock()
		if !mayRun {
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
			s.opts.Queue,
			s.topic,
			s.headers,
			s.queueArgs,
			s.opts.AutoAck,
			s.durableQueue,
			s.autoDelete,
		)

		s.r.mtx.Unlock()
		switch err {
		case nil:
			reSubscribeDelay = minResubscribeDelay
			s.mtx.Lock()
			s.ch = ch
			s.mtx.Unlock()
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
