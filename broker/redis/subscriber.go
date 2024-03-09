package redis

import (
	"errors"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	b *redisBroker

	topic  string
	done   chan error
	closed bool

	handler broker.Handler
	binder  broker.Binder

	options broker.SubscribeOptions

	conn *redis.PubSubConn
}

func (s *subscriber) onStart() error {
	return nil
}

func (s *subscriber) onMessage(channel string, data []byte) error {
	var m broker.Message

	if s.binder != nil {
		m.Body = s.binder()
	} else {
		m.Body = data
	}

	p := publication{
		topic:   channel,
		message: &m,
	}

	if p.err = broker.Unmarshal(s.b.options.Codec, data, &m.Body); p.err != nil {
		//log.Error("[redis]", err)
		return p.err
	}

	if p.err = s.handler(s.options.Context, &p); p.err != nil {
		return p.err
	}

	if s.options.AutoAck {
		if p.err = p.Ack(); p.err != nil {
			return p.err
		}
	}

	return nil
}

func (s *subscriber) ping() error {
	if s.conn == nil {
		return errors.New("cannot ping")
	}

	if err := s.conn.Ping(""); err != nil {
		return err
	}
	return nil
}

func (s *subscriber) recv() {
	defer func(conn *redis.PubSubConn) {
		err := conn.Close()
		if err != nil {
			log.Error("[redis] close pubsub connection error: ", err)
		}
	}(s.conn)

	s.done = make(chan error, 1)

	ticker := time.NewTicker(DefaultHealthCheckPeriod)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				if err := s.ping(); err != nil {
					s.done <- err
					return
				}
			case <-s.options.Context.Done():
				s.done <- nil
				return
			}
		}
	}()

	_ = s.ping()

	for {
		switch x := s.conn.Receive().(type) {
		case error:
			log.Errorf("[redis] recv error: %s\n", x.Error())
			s.done <- x
			return

		case redis.Message:
			if err := s.onMessage(x.Channel, x.Data); err != nil {
				s.done <- err
				break
			}

		case redis.Subscription:
			switch x.Count {
			case 0:
				s.done <- nil
				return
			}

		case redis.Pong:
			log.Debug("[redis] pong")
		}
	}
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

	s.closed = true

	var err error
	if s.conn != nil {
		err = s.conn.Unsubscribe()
	}

	if s.b != nil && s.b.subscribers != nil && removeFromManager {
		_ = s.b.subscribers.RemoveOnly(s.topic)
	}

	return err
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}
