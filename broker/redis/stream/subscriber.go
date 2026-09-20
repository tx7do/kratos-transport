package stream

import (
	"errors"
	"sync"
	"time"

	"github.com/tx7do/kratos-transport/broker"
	redisOption "github.com/tx7do/kratos-transport/broker/redis/option"
)

type subscriber struct {
	sync.RWMutex

	b *streamBroker

	topic    string
	group    string
	consumer string

	blockTime time.Duration
	count     int

	handler broker.Handler
	binder  broker.Binder

	options broker.SubscribeOptions

	closed bool
}

func (s *subscriber) onMessage(msgID string, data []byte) error {
	var m broker.Message

	if s.binder != nil {
		m.Body = s.binder()

		if err := broker.Unmarshal(s.b.options.Codec, data, &m.Body); err != nil {
			// 毒消息：通知 ErrorHandler（消息滞留 PEL，等待人工/后续恢复机制处理）
			redisOption.LogErrorf("unmarshal message failed: %v", err)
			if eh := s.b.options.ErrorHandler; eh != nil {
				mm := broker.Message{Body: nil}
				p := publication{topic: s.topic, message: &mm, err: err}
				_ = eh(s.options.Context, &p)
			}
			return err
		}
	} else {
		m.Body = data
	}

	p := publication{
		topic:   s.topic,
		group:   s.group,
		msgID:   msgID,
		message: &m,
		pool:    s.b.pool,
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

		err := s.receiveLoop()

		if err == nil || s.IsClosed() {
			return
		}

		redisOption.LogErrorf("stream recv error: %s, reconnecting in %v...", err.Error(), reconnectDelay)

		select {
		case <-time.After(reconnectDelay):
		case <-s.options.Context.Done():
			return
		}

		// 重连后确保消费组存在
		if s.b.pool != nil {
			if reErr := s.b.ensureGroup(s.topic, s.group); reErr != nil {
				redisOption.LogWarnf("re-ensure group: %v", reErr)
			}
		}

		if reconnectDelay < maxReconnectDelay {
			reconnectDelay *= 2
		}
	}
}

// receiveLoop 使用 XREADGROUP 持续消费消息
func (s *subscriber) receiveLoop() error {
	for {
		if s.IsClosed() {
			return nil
		}

		select {
		case <-s.options.Context.Done():
			return nil
		default:
		}

		// XREADGROUP GROUP group consumer BLOCK timeout COUNT count STREAMS stream >
		if s.b.pool == nil {
			return errors.New("redis-stream: broker disconnected")
		}
		conn := s.b.pool.Get()

		reply, err := conn.Do("XREADGROUP",
			"GROUP", s.group, s.consumer,
			"BLOCK", int(s.blockTime.Milliseconds()),
			"COUNT", s.count,
			"STREAMS", s.topic, ">",
		)

		if err != nil {
			_ = conn.Close()
			return err
		}

		_ = conn.Close()

		// reply 格式: [][]any{[streamName, [ [id, [field, value, ...]], ... ]}
		if reply == nil {
			// 超时无消息，继续
			continue
		}

		streams, ok := reply.([]any)
		if !ok || len(streams) == 0 {
			continue
		}

		for _, streamEntry := range streams {
			streamData, ok := streamEntry.([]any)
			if !ok || len(streamData) < 2 {
				continue
			}

			messages, ok := streamData[1].([]any)
			if !ok {
				continue
			}

			for _, msgEntry := range messages {
				if s.IsClosed() {
					return nil
				}

				msgData, ok := msgEntry.([]any)
				if !ok || len(msgData) < 2 {
					continue
				}

				msgID, _ := msgData[0].(string)
				fields, _ := msgData[1].([]any)

				// 提取 body 字段
				body := s.extractField(fields, "body")
				if body == nil {
					continue
				}

				data, ok := body.([]byte)
				if !ok {
					continue
				}

				if err := s.onMessage(msgID, data); err != nil {
					redisOption.LogErrorf("onMessage error [stream=%s id=%s]: %s", s.topic, msgID, err.Error())
				}
			}
		}
	}
}

// extractField 从 Redis Stream 消息的 field-value 对中提取指定字段
func (s *subscriber) extractField(fields []any, key string) any {
	for i := 0; i < len(fields)-1; i += 2 {
		k, ok := fields[i].([]byte)
		if ok && string(k) == key {
			return fields[i+1]
		}
	}
	return nil
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

	if s.b != nil && s.b.subscribers != nil && removeFromManager {
		_ = s.b.subscribers.RemoveOnly(s.topic)
	}

	return nil
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}
