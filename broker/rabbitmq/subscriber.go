package rabbitmq

import (
	"errors"
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

	exchangeName string
	queueArgs    map[string]any
	fn           func(msg amqp.Delivery)
	headers      map[string]any

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

func (s *subscriber) Unsubscribe(removeFromManager bool) error {
	s.Lock()
	defer s.Unlock()

	s.closed = true

	var err error
	if s.ch != nil {
		err = s.ch.Close()
	}

	if s.r != nil && s.r.subscribers != nil && removeFromManager {
		_ = s.r.subscribers.RemoveOnly(s.topic)
	}

	return err
}

// consumeOnce 同步执行一次队列声明/绑定并开始消费。
// 首次订阅必须同步完成：否则 Subscribe 返回后队列尚未绑定，
// 此时向 exchange 投递的消息会因无匹配绑定而直接丢弃。
func (s *subscriber) consumeOnce() error {
	s.r.mtx.Lock()
	defer s.r.mtx.Unlock()

	if !s.r.conn.connected {
		return errors.New("rabbitmq: not connected")
	}

	ch, sub, err := s.r.conn.Consume(
		s.exchangeName,
		s.options.Queue,
		s.topic,
		s.headers,
		s.queueArgs,
		s.options.AutoAck,
		s.durableQueue,
		s.autoDelete,
	)
	if err != nil {
		return err
	}

	s.Lock()
	s.ch = ch
	s.Unlock()

	go func() {
		// deliveries channel 随连接关闭而关闭，无需逐消息计数
		// （每消息 wg.Add 与 Disconnect 的 wg.Wait 并发违反 WaitGroup 契约）
		for d := range sub {
			s.fn(d)
		}
	}()

	return nil
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

		// 首次消费已由 consumeOnce 同步完成；
		// 这里必须先等【断线事件】再等【重连完成】。
		// 旧实现直接等 waitConnection——Connect 成功时它已被 close，
		// 会立即再 Consume 一次造成双消费者
		select {
		case <-s.r.conn.close:
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
			s.exchangeName,
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
