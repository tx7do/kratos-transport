package nats

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	NATS "github.com/nats-io/nats.go"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultAddr = "nats://127.0.0.1:4222"
)

type natsBroker struct {
	sync.Once
	sync.RWMutex

	connected bool

	opts broker.Options

	conn     *NATS.Conn
	natsOpts NATS.Options

	drain   bool
	closeCh chan error

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &natsBroker{
		opts: options,
	}

	return b
}

func (b *natsBroker) Address() string {
	if b.conn != nil && b.conn.IsConnected() {
		return b.conn.ConnectedUrl()
	}

	if len(b.opts.Addrs) > 0 {
		return b.opts.Addrs[0]
	}

	return defaultAddr
}

func (b *natsBroker) Name() string {
	return "nats"
}

func (b *natsBroker) Options() broker.Options {
	return b.opts
}

func (b *natsBroker) Init(opts ...broker.Option) error {
	b.setOption(opts...)

	if len(b.opts.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "nats-producer", b.opts.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "nats-consumer", b.opts.Tracings...)
	}

	return nil
}

func (b *natsBroker) setAddrs(addrs []string) []string {
	//nolint:prealloc
	var cAddrs []string
	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{NATS.DefaultURL}
	}
	return cAddrs
}

func (b *natsBroker) setOption(opts ...broker.Option) {
	for _, o := range opts {
		o(&b.opts)
	}

	b.Once.Do(func() {
		b.natsOpts = NATS.GetDefaultOptions()
	})

	if value, ok := b.opts.Context.Value(optionsKey{}).(NATS.Options); ok {
		b.natsOpts = value
	}

	if len(b.opts.Addrs) == 0 {
		b.opts.Addrs = b.natsOpts.Servers
	}

	if !b.opts.Secure {
		b.opts.Secure = b.natsOpts.Secure
	}

	if b.opts.TLSConfig == nil {
		b.opts.TLSConfig = b.natsOpts.TLSConfig
	}
	b.setAddrs(b.opts.Addrs)

	if b.opts.Context.Value(drainConnectionKey{}) != nil {
		b.drain = true
		b.closeCh = make(chan error)
		b.natsOpts.ClosedCB = b.onClose
		b.natsOpts.AsyncErrorCB = b.onAsyncError
		b.natsOpts.DisconnectedErrCB = b.onDisconnectedError
	}
}

func (b *natsBroker) Connect() error {
	b.Lock()
	defer b.Unlock()

	if b.connected {
		return nil
	}

	status := NATS.CLOSED
	if b.conn != nil {
		status = b.conn.Status()
	}

	switch status {
	case NATS.CONNECTED, NATS.RECONNECTING, NATS.CONNECTING:
		b.connected = true
		return nil
	default: // DISCONNECTED or CLOSED or DRAINING
		opts := b.natsOpts
		opts.Servers = b.opts.Addrs
		opts.Secure = b.opts.Secure
		opts.TLSConfig = b.opts.TLSConfig

		if b.opts.TLSConfig != nil {
			opts.Secure = true
		}

		c, err := opts.Connect()
		if err != nil {
			return err
		}
		b.conn = c
		b.connected = true
		return nil
	}
}

func (b *natsBroker) Disconnect() error {
	b.Lock()
	defer b.Unlock()

	if b.drain {
		_ = b.conn.Drain()
		b.closeCh <- nil
	}

	b.conn.Close()

	b.connected = false

	return nil
}

func (b *natsBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.opts.Codec, msg)
	if err != nil {
		return err
	}

	return b.publish(topic, buf, opts...)
}

func (b *natsBroker) publish(topic string, buf []byte, opts ...broker.PublishOption) error {
	b.RLock()
	defer b.RUnlock()

	if b.conn == nil {
		return errors.New("not connected")
	}

	options := broker.PublishOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	m := NATS.NewMsg(topic)
	m.Data = buf

	if headers, ok := options.Context.Value(headersKey{}).(map[string][]string); ok {
		for k, v := range headers {
			for _, vv := range v {
				m.Header.Add(k, vv)
			}
		}
	}

	span := b.startProducerSpan(options.Context, m)

	err := b.conn.PublishMsg(m)

	b.finishProducerSpan(span, err)

	return err
}

func (b *natsBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	b.RLock()
	if b.conn == nil {
		b.RUnlock()
		return nil, errors.New("not connected")
	}
	b.RUnlock()

	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}

	subs := &subscriber{s: nil, opts: options}

	fn := func(msg *NATS.Msg) {
		m := &broker.Message{
			Headers: natsHeaderToMap(msg.Header),
			Body:    nil,
		}

		pub := &publication{t: msg.Subject, m: m}

		ctx, span := b.startConsumerSpan(options.Context, msg)

		eh := b.opts.ErrorHandler

		if binder != nil {
			m.Body = binder()
		} else {
			m.Body = msg.Data
		}

		if err := broker.Unmarshal(b.opts.Codec, msg.Data, &m.Body); err != nil {
			pub.err = err
			log.Errorf("[nats]: unmarshal message failed: %v", err)
			if eh != nil {
				_ = eh(b.opts.Context, pub)
			}
			return
		}

		if err := handler(ctx, pub); err != nil {
			pub.err = err
			log.Errorf("[nats]: process message failed: %v", err)
			if eh != nil {
				_ = eh(b.opts.Context, pub)
			}
		}
		if options.AutoAck {
			if err := pub.Ack(); err != nil {
				log.Errorf("[nats]: unable to commit msg: %v", err)
			}
		}

		b.finishConsumerSpan(span)
	}

	var sub *NATS.Subscription
	var err error

	b.RLock()
	if len(options.Queue) > 0 {
		sub, err = b.conn.QueueSubscribe(topic, options.Queue, fn)
	} else {
		sub, err = b.conn.Subscribe(topic, fn)
	}
	b.RUnlock()
	if err != nil {
		return nil, err
	}

	subs.s = sub

	return subs, nil
}

func (b *natsBroker) onClose(_ *NATS.Conn) {
	b.closeCh <- nil
}

func (b *natsBroker) onAsyncError(_ *NATS.Conn, _ *NATS.Subscription, err error) {
	if err == NATS.ErrDrainTimeout {
		b.closeCh <- err
	}
}

func (b *natsBroker) onDisconnectedError(_ *NATS.Conn, err error) {
	b.closeCh <- err
}

func (b *natsBroker) startProducerSpan(ctx context.Context, msg *NATS.Msg) trace.Span {
	if b.producerTracer == nil {
		return nil
	}

	carrier := NewMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("nats"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Subject),
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (b *natsBroker) finishProducerSpan(span trace.Span, err error) {
	if b.producerTracer == nil {
		return
	}

	b.producerTracer.End(context.Background(), span, err)
}

func (b *natsBroker) startConsumerSpan(ctx context.Context, msg *NATS.Msg) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("nats"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Subject),
		semConv.MessagingOperationReceive,
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *natsBroker) finishConsumerSpan(span trace.Span) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(context.Background(), span, nil)
}
