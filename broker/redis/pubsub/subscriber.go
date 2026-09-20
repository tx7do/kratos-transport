package pubsub

import (
	"errors"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
	redisOption "github.com/tx7do/kratos-transport/broker/redis/option"
)

type subscriber struct {
	sync.RWMutex

	b *pubsubBroker

	topic  string
	closed bool

	handler broker.Handler
	binder  broker.Binder

	options broker.SubscribeOptions

	conn *redis.PubSubConn
}

func (s *subscriber) onMessage(channel string, data []byte) error {
	var m broker.Message

	if s.binder != nil {
		m.Body = s.binder()

		if err := broker.Unmarshal(s.b.options.Codec, data, &m.Body); err != nil {
			// 毒消息：通知 ErrorHandler（PUBLISH 无重投语义，只能丢弃）
			redisOption.LogErrorf("unmarshal message failed: %v", err)
			p := publication{topic: channel, message: &m, err: err}
			if eh := s.b.options.ErrorHandler; eh != nil {
				_ = eh(s.options.Context, &p)
			}
			return nil
		}
	} else {
		m.Body = data
	}

	p := publication{
		topic:   channel,
		message: &m,
	}

	if p.err = s.handler(s.options.Context, &p); p.err != nil {
		if eh := s.b.options.ErrorHandler; eh != nil {
			_ = eh(s.options.Context, &p)
		}
		return p.err
	}

	if s.options.AutoAck {
		if p.err = p.Ack(); p.err != nil {
			return p.err
		}
	}

	return nil
}

func (s *subscriber) recv() {
	reconnectDelay := 1 * time.Second
	maxReconnectDelay := 30 * time.Second

	for {
		if s.IsClosed() {
			return
		}

		// 确保有连接
		s.RLock()
		hasConn := s.conn != nil
		s.RUnlock()

		if !hasConn {
			if !s.reconnect() {
				redisOption.LogErrorf("reconnect failed, retrying in %v...", reconnectDelay)
				select {
				case <-time.After(reconnectDelay):
				case <-s.options.Context.Done():
					return
				}
				if reconnectDelay < maxReconnectDelay {
					reconnectDelay *= 2
				}
				continue
			}
			reconnectDelay = 1 * time.Second
		}

		// 运行接收循环
		err := s.receiveLoop()

		// 关闭当前连接
		s.Lock()
		if s.conn != nil {
			_ = s.conn.Close()
			s.conn = nil
		}
		s.Unlock()

		if err == nil || s.IsClosed() {
			return
		}

		redisOption.LogErrorf("recv error: %s, reconnecting in %v...", err.Error(), reconnectDelay)

		select {
		case <-time.After(reconnectDelay):
		case <-s.options.Context.Done():
			return
		}

		if reconnectDelay < maxReconnectDelay {
			reconnectDelay *= 2
		}
	}
}

func (s *subscriber) receiveLoop() error {
	stopPing := make(chan struct{})
	defer close(stopPing)

	pingErr := make(chan error, 1)
	ticker := time.NewTicker(redisOption.DefaultHealthCheckPeriod)
	defer ticker.Stop()

	// 健康检查协程
	go func() {
		for {
			select {
			case <-stopPing:
				return
			case <-ticker.C:
				s.RLock()
				addr := s.b.Address()
				pool := s.b.pool
				s.RUnlock()
				if pool == nil {
					return
				}
				// 用独立连接做健康检查：PubSubConn 的 Ping 内部会 Receive，
				// 与接收循环并发会偷走 PUBLISH 响应帧（消息丢失 + 数据竞争）
				checkConn, err := redis.DialURL(addr,
					redis.DialConnectTimeout(redisOption.DefaultConnectTimeout),
					redis.DialReadTimeout(3*time.Second),
				)
				if err != nil {
					pingErr <- err
					return
				}
				_, err = checkConn.Do("PING")
				_ = checkConn.Close()
				if err != nil {
					pingErr <- err
					return
				}
			case <-s.options.Context.Done():
				pingErr <- nil
				return
			}
		}
	}()

	for {
		select {
		case err := <-pingErr:
			return err
		default:
		}

		s.RLock()
		conn := s.conn
		s.RUnlock()
		if conn == nil {
			return errors.New("connection is nil")
		}

		switch x := conn.Receive().(type) {
		case error:
			return x

		case redis.Message:
			if err := s.onMessage(x.Channel, x.Data); err != nil {
				redisOption.LogErrorf("onMessage error: %s", err.Error())
			}

		case redis.Subscription:
			if x.Count == 0 {
				return nil
			}

		case redis.Pong:
			// 心跳 PONG 帧是正常保活行为，静默处理，避免周期性日志噪音（issue #133）
		}
	}
}

func (s *subscriber) reconnect() bool {
	s.Lock()
	defer s.Unlock()

	if s.b.pool == nil {
		return false
	}

	conn := &redis.PubSubConn{Conn: s.b.pool.Get()}
	if conn.Conn == nil {
		redisOption.LogError("failed to get connection from pool")
		return false
	}

	if err := conn.Subscribe(s.topic); err != nil {
		_ = conn.Close()
		redisOption.LogErrorf("resubscribe error: %s", err.Error())
		return false
	}

	s.conn = conn
	redisOption.LogInfof("reconnected and resubscribed to topic: %s", s.topic)
	return true
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
