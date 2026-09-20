package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	kProto "github.com/go-kratos/kratos/v2/encoding/proto"
	"google.golang.org/protobuf/proto"

	natsGo "github.com/nats-io/nats.go"

	"github.com/tx7do/kratos-transport/broker"
)

///////////////////////////////////////////////////////////////////////////////
/// JetStream Subscribe
///////////////////////////////////////////////////////////////////////////////

func (b *jetStreamBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	b.RLock()
	if b.js == nil {
		b.RUnlock()
		return nil, errors.New("not connected")
	}
	b.RUnlock()

	options := broker.NewSubscribeOptions(opts...)

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
	}

	// Build JetStream subscribe options
	subOpts := buildSubOpts(options)

	manualAck := options.Context.Value(subManualAckKey{}) != nil

	jsSub := &subscriber{
		remover: b,
		s:       nil,
		options: options,
	}

	// Message handler callback
	fn := func(msg *natsGo.Msg) {
		var errSub error

		m := &broker.Message{
			Headers: natsHeaderToMap(msg.Header),
			Body:    nil,
			Msg:     msg,
		}

		pub := &publication{t: msg.Subject, m: m}

		// Use context.Background() as base to isolate each message's span tree
		ctx, span := b.startConsumerSpan(context.Background(), msg)

		eh := b.options.ErrorHandler

		if binder != nil {
			if b.options.Codec.Name() == kProto.Name {
				if pm, pmOK := binder().(proto.Message); pmOK {
					m.Body = pm
				} else {
					// binder 未返回 proto.Message：退回原始字节，避免断言 panic
					m.Body = msg.Data
				}
			} else {
				m.Body = binder()
			}

			if errSub = broker.Unmarshal(b.options.Codec, msg.Data, &m.Body); errSub != nil {
				pub.err = errSub
				LogErrorf("unmarshal message failed: %v", errSub)
				if eh != nil {
					_ = eh(b.options.Context, pub)
				}
				_ = msg.Nak()
				b.finishConsumerSpan(ctx, span, errSub)
				return
			}
		} else {
			m.Body = msg.Data
		}

		if errSub = handler(ctx, pub); errSub != nil {
			pub.err = errSub
			LogErrorf("handle message failed: %v", errSub)
			if eh != nil {
				_ = eh(b.options.Context, pub)
			}
			_ = msg.Nak()
			b.finishConsumerSpan(ctx, span, errSub)
			return
		}

		if options.AutoAck && !manualAck {
			if errSub = pub.Ack(); errSub != nil {
				LogErrorf("unable to ack msg: %v", errSub)
			}
		}

		b.finishConsumerSpan(ctx, span, errSub)
	}

	var sub *natsGo.Subscription
	var err error

	// Check if pull-based subscription is requested
	isPull := options.Context.Value(subPullKey{}) != nil

	b.RLock()
	if isPull {
		// Pull subscribe
		durableName := ""
		if v, ok := options.Context.Value(subDurableKey{}).(string); ok && v != "" {
			durableName = v
		}
		sub, err = b.js.PullSubscribe(topic, durableName, subOpts...)
		if err != nil {
			b.RUnlock()
			return nil, fmt.Errorf("failed to create pull subscription: %w", err)
		}

		// Start pull loop in background
		batchSize := 10
		if v, ok := options.Context.Value(subPullBatchSizeKey{}).(int); ok && v > 0 {
			batchSize = v
		}
		go b.pullLoop(sub, fn, batchSize, jsSub)
	} else {
		// Push subscribe
		if len(options.Queue) > 0 {
			sub, err = b.js.QueueSubscribe(topic, options.Queue, fn, subOpts...)
		} else {
			sub, err = b.js.Subscribe(topic, fn, subOpts...)
		}
	}
	b.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", topic, err)
	}

	jsSub.s = sub

	b.subscribers.Add(topic, jsSub)

	LogInfof("subscribed to JetStream subject: %s (pull=%v)", topic, isPull)

	return jsSub, nil
}

///////////////////////////////////////////////////////////////////////////////
/// Pull Subscribe Loop
///////////////////////////////////////////////////////////////////////////////

func (b *jetStreamBroker) pullLoop(sub *natsGo.Subscription, handler func(*natsGo.Msg), batchSize int, jsSub *subscriber) {
	for {
		if jsSub.IsClosed() {
			return
		}

		msgs, err := sub.Fetch(batchSize)
		if err != nil {
			if errors.Is(err, natsGo.ErrTimeout) {
				continue
			}
			if errors.Is(err, natsGo.ErrConnectionClosed) {
				// 连接断开时 Fetch 立即失败，退避避免热循环
				select {
				case <-jsSub.options.Context.Done():
					return
				case <-time.After(time.Second):
				}
				continue
			}
			if jsSub.IsClosed() {
				return
			}
			LogErrorf("pull fetch error: %v", err)
			continue
		}

		for _, msg := range msgs {
			handler(msg)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
/// Subscribe Options Builder
///////////////////////////////////////////////////////////////////////////////

func buildSubOpts(options broker.SubscribeOptions) []natsGo.SubOpt {
	var subOpts []natsGo.SubOpt

	if v, ok := options.Context.Value(subDurableKey{}).(string); ok && v != "" {
		subOpts = append(subOpts, natsGo.Durable(v))
	}
	if options.Context.Value(subDeliverAllKey{}) != nil {
		subOpts = append(subOpts, natsGo.DeliverAll())
	}
	if options.Context.Value(subDeliverLastKey{}) != nil {
		subOpts = append(subOpts, natsGo.DeliverLast())
	}
	if options.Context.Value(subDeliverNewKey{}) != nil {
		subOpts = append(subOpts, natsGo.DeliverNew())
	}
	if v, ok := options.Context.Value(subStartSequenceKey{}).(uint64); ok {
		subOpts = append(subOpts, natsGo.StartSequence(v))
	}
	if v, ok := options.Context.Value(subStartTimeKey{}).(time.Time); ok && !v.IsZero() {
		subOpts = append(subOpts, natsGo.StartTime(v))
	}
	if v, ok := options.Context.Value(subAckWaitKey{}).(time.Duration); ok && v > 0 {
		subOpts = append(subOpts, natsGo.AckWait(v))
	}
	if v, ok := options.Context.Value(subMaxAckPendingKey{}).(int); ok && v > 0 {
		subOpts = append(subOpts, natsGo.MaxAckPending(v))
	}
	if v, ok := options.Context.Value(subBindStreamKey{}).(string); ok && v != "" {
		subOpts = append(subOpts, natsGo.BindStream(v))
	}
	if options.Context.Value(subReplayInstantKey{}) != nil {
		subOpts = append(subOpts, natsGo.ReplayInstant())
	}
	if v, ok := options.Context.Value(subDescriptionKey{}).(string); ok && v != "" {
		subOpts = append(subOpts, natsGo.Description(v))
	}
	if options.Context.Value(subManualAckKey{}) != nil {
		subOpts = append(subOpts, natsGo.ManualAck())
	}
	if v, ok := options.Context.Value(subRawOptsKey{}).([]natsGo.SubOpt); ok {
		subOpts = append(subOpts, v...)
	}

	return subOpts
}
