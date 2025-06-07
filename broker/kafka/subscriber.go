package kafka

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"

	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	sync.RWMutex

	b *kafkaBroker

	topic string

	options broker.SubscribeOptions
	handler broker.Handler
	binder  broker.Binder

	reader *kafkaGo.Reader

	closed bool
	done   chan struct{}

	batchSize     int
	batchInterval time.Duration
}

func newSubscriber(
	b *kafkaBroker,
	topic string,
	options broker.SubscribeOptions,
	readerConfig kafkaGo.ReaderConfig,
	handler broker.Handler,
	binder broker.Binder,
) *subscriber {
	sub := &subscriber{
		b:       b,
		options: options,
		topic:   topic,
		handler: handler,
		binder:  binder,
		reader:  kafkaGo.NewReader(readerConfig),
		done:    make(chan struct{}),
	}
	return sub
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

	err := s.close()

	if s.b != nil && s.b.subscribers != nil && removeFromManager {
		_ = s.b.subscribers.RemoveOnly(s.topic)
	}

	return err
}

func (s *subscriber) close() error {
	s.Lock()
	defer s.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	var err error
	if s.reader != nil {
		err = s.reader.Close()
	}

	if s.done != nil {
		close(s.done)
		s.done = nil
	}

	return err
}

func (s *subscriber) IsClosed() bool {
	s.RLock()
	defer s.RUnlock()

	return s.closed
}

func (s *subscriber) isBatchMode() bool {
	return s.batchSize > 0 || s.batchInterval > 0
}

func (s *subscriber) run() {
	if s.isBatchMode() {
		s.processBatchMessage()
	} else {
		s.processSingleMessage()
	}
}

func (s *subscriber) processBatchMessage() {
	messageBuffer := make([]kafkaGo.Message, 0, s.batchSize)

	ticker := time.NewTicker(s.batchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.options.Context.Done():
			_ = s.close()
			return

		case <-s.done:
			_ = s.close()
			return

		case <-ticker.C:
			// 定时触发批量处理
			if len(messageBuffer) > 0 {
				s.handleBatchMessage(messageBuffer)
				messageBuffer = make([]kafkaGo.Message, 0, s.batchSize)
			}

		default:
			// 设置读取超时，避免阻塞时间过长
			ctx, cancel := context.WithTimeout(s.options.Context, 1*time.Second)
			m, err := s.reader.FetchMessage(ctx)
			cancel()

			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}

				if errors.Is(err, context.DeadlineExceeded) {
					// 超时，继续循环
					continue
				}
				LogErrorf("FetchMessage error: %s", err.Error())
				time.Sleep(1 * time.Second)
				continue
			}

			// 将消息添加到缓冲区
			messageBuffer = append(messageBuffer, m)

			// 当缓冲区达到批处理大小时处理消息
			if len(messageBuffer) >= s.batchSize {
				s.handleBatchMessage(messageBuffer)
				messageBuffer = make([]kafkaGo.Message, 0, s.batchSize)
			}
		}
	}
}

func (s *subscriber) processSingleMessage() {
	for {
		select {
		case <-s.options.Context.Done():
			_ = s.close()
			return

		case <-s.done:
			_ = s.close()
			return

		default:
			km, err := s.reader.FetchMessage(s.options.Context)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}

				LogErrorf("FetchMessage error: %s", err.Error())
				continue
			}

			if done := s.handleMessage(km); done {
			}
		}
	}
}

func (s *subscriber) handleBatchMessage(messages []kafkaGo.Message) {
	for _, km := range messages {
		if done := s.handleMessage(km); done {
			//return true
		}
	}
}

func (s *subscriber) handleMessage(km kafkaGo.Message) bool {
	var err error

	ctx, span := s.b.startConsumerSpan(s.options.Context, &km)

	bm := &broker.Message{
		Headers:   kafkaHeaderToMap(km.Headers),
		Body:      nil,
		Partition: km.Partition,
		Offset:    km.Offset,
	}

	if s.binder != nil {
		bm.Body = s.binder()

		if err = broker.Unmarshal(s.b.options.Codec, km.Value, &bm.Body); err != nil {
			LogErrorf("unmarshal message failed: %v", err)
			s.b.finishConsumerSpan(span, err)
			return true
		}
	} else {
		bm.Body = km.Value
	}

	pub := newPublication(s.options.Context, s.reader, km, bm)

	if err = s.handler(ctx, pub); err != nil {
		LogErrorf("handle message failed: %v", err)
		s.b.finishConsumerSpan(span, err)
		return true
	}

	if s.options.AutoAck {
		if err = pub.Ack(); err != nil {
			LogErrorf("unable to commit km: %v", err)
			s.b.finishConsumerSpan(span, err)
			return true
		}
	}

	s.b.finishConsumerSpan(span, err)

	return false
}
