package nsq

import (
	"sync"

	"github.com/nsqio/go-nsq"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	topic string

	n       *nsqBroker
	options broker.SubscribeOptions

	consumer    *nsq.Consumer
	handlerFunc nsq.HandlerFunc

	// needsHandler: Subscribe 先于 Connect 登记时为 true，Connect 阶段补建 handler
	needsHandler bool
	binder       broker.Binder
	handler      broker.Handler
	config       *nsq.Config

	concurrency int
	closed      bool
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

	if s.consumer != nil {
		s.consumer.Stop()
	}

	s.closed = true

	if s.n != nil && s.n.subscribers != nil && removeFromManager {
		_ = s.n.subscribers.RemoveOnly(s.topic)
	}

	return nil
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}

// buildHandler 为延迟登记的订阅构造消息处理闭包（与 Subscribe 同步路径一致）
func (s *subscriber) buildHandler(cm *nsq.Consumer) {
	options := s.options
	b := s.n
	binder := s.binder
	handler := s.handler

	h := nsq.HandlerFunc(func(nm *nsq.Message) error {
		if !options.AutoAck {
			nm.DisableAutoResponse()
		}

		var m broker.Message
		var errSub error

		if binder != nil {
			m.Body = binder()

			if errSub = broker.Unmarshal(b.options.Codec, nm.Body, &m.Body); errSub != nil {
				// 毒消息：通知 ErrorHandler 并 Finish，避免无限 Requeue 后被丢弃且无感知
				LogErrorf("unmarshal message failed: %v", errSub)
				p := &publication{topic: s.topic, nsqMsg: nm, msg: &m, err: errSub}
				if eh := b.options.ErrorHandler; eh != nil {
					_ = eh(b.options.Context, p)
				}
				nm.Finish()
				return nil
			}
		} else {
			m.Body = nm.Body
		}

		p := &publication{topic: s.topic, nsqMsg: nm, msg: &m}

		if errSub = handler(b.options.Context, p); errSub != nil {
			p.err = errSub
			if eh := b.options.ErrorHandler; eh != nil {
				_ = eh(b.options.Context, p)
			}
			// 处理失败即终止重投：显式 FIN 后必须返回 nil——
			// 返回 err 会让 go-nsq 对已 FIN 的消息再发 REQ（协议错误）
			if errFinish := p.Ack(); errFinish != nil {
				LogErrorf("unable to commit msg: %v", errFinish)
			}
			return nil
		}

		return nil
	})

	s.handlerFunc = h
}
