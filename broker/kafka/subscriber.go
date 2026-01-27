package kafka

import (
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"sync"
	"time"

	kafkaGo "github.com/segmentio/kafka-go"

	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultDeadlineMin = 50 * time.Millisecond
	defaultDeadlineMax = 500 * time.Millisecond
	defaultErrorMin    = 500 * time.Millisecond
	defaultErrorMax    = 5 * time.Second
	defaultMaxAttempts = 6
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

	// 退避配置
	deadlineMin time.Duration
	deadlineMax time.Duration
	errorMin    time.Duration
	errorMax    time.Duration
	maxAttempts int
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

		deadlineMin: defaultDeadlineMin,
		deadlineMax: defaultDeadlineMax,
		errorMin:    defaultErrorMin,
		errorMax:    defaultErrorMax,
		maxAttempts: defaultMaxAttempts,
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
	err := s.closeLocked()
	s.Unlock()

	if s.b != nil && s.b.subscribers != nil && removeFromManager {
		_ = s.b.subscribers.RemoveOnly(s.topic)
	}

	return err
}

func (s *subscriber) close() error {
	s.Lock()
	defer s.Unlock()
	return s.closeLocked()
}

func (s *subscriber) closeLocked() error {
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

	// 退避参数
	deadlineAttempt := 0
	errorAttempt := 0

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
				// 先判断是否需要立即退出
				if isFatalError(s, err) {
					LogErrorf("fatal FetchMessage error, exiting: %v", err)
					return
				}

				if errors.Is(err, context.DeadlineExceeded) {
					// 对短超时使用较小的退避并带抖动
					backoffSleep(deadlineAttempt, s.deadlineMin, s.deadlineMax)
					if deadlineAttempt < 6 {
						deadlineAttempt++
					}
					continue
				}

				LogErrorf("FetchMessage error: %s", err.Error())
				backoffSleep(errorAttempt, s.errorMin, s.errorMax)
				if errorAttempt < 6 {
					errorAttempt++
				}
				continue
			}

			// 成功读取，重置退避计数
			deadlineAttempt = 0
			errorAttempt = 0

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
	// 退避参数
	errorAttempt := 0

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
				// 立即退出的情况
				if isFatalError(s, err) {
					LogErrorf("fatal FetchMessage error, exiting: %v", err)
					return
				}

				LogErrorf("FetchMessage error: %s", err.Error())
				backoffSleep(errorAttempt, s.errorMin, s.errorMax)
				if errorAttempt < 6 {
					errorAttempt++
				}
				continue
			}

			// 成功读取，重置退避计数
			errorAttempt = 0

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

var randSrc = rand.New(rand.NewSource(time.Now().UnixNano()))

// 简单的指数退避（带抖动），attempt 从 0 开始
func backoffSleep(attempt int, min, max time.Duration) {
	if min <= 0 {
		min = 50 * time.Millisecond
	}
	if max < min {
		max = min
	}
	// 指数增长（以 2 为基数），限制最大值
	d := min << uint(attempt)
	if d < min {
		d = min
	}
	if d > max {
		d = max
	}
	// 添加抖动，范围 [d/2, d)
	jitter := time.Duration(randSrc.Int63n(int64(d/2 + 1)))
	sleepDur := d/2 + jitter
	time.Sleep(sleepDur)
}

// helper: 判断是否为不可恢复/致命错误，需立即退出
func isFatalError(s *subscriber, err error) bool {
	if err == nil {
		return false
	}
	// EOF 表示读取流结束，直接退出
	if errors.Is(err, io.EOF) {
		return true
	}
	// 如果外部上下文已被取消/截止，立即退出
	if errors.Is(err, context.Canceled) || (s != nil && s.options.Context != nil && s.options.Context.Err() != nil) {
		return true
	}
	// 如果订阅者已被关闭，退出
	if s != nil && s.IsClosed() {
		return true
	}
	// net.Error 非临时错误视为致命（例如永久连接失败）
	var ne net.Error
	if errors.As(err, &ne) {
		if !ne.Temporary() {
			return true
		}
	}
	// 如果错误实现了 Temporary() bool（例如某些 kafka 库错误），且为非临时，则致命
	type temporary interface {
		Temporary() bool
	}
	if te, ok := err.(temporary); ok {
		if !te.Temporary() {
			return true
		}
	}
	return false
}
