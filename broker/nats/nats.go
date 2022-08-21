package nats

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	NATS "github.com/nats-io/nats.go"
	"github.com/tx7do/kratos-transport/broker"
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

	span := b.startProducerSpan(m)

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

		span := b.startConsumerSpan(msg)

		eh := b.opts.ErrorHandler

		if binder != nil {
			m.Body = binder()
		}

		if err := broker.Unmarshal(b.opts.Codec, msg.Data, m.Body); err != nil {
			pub.err = err
			log.Error(err)
			if eh != nil {
				_ = eh(b.opts.Context, pub)
			}
			return
		}

		if err := handler(b.opts.Context, pub); err != nil {
			pub.err = err
			log.Error(err)
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

func (b *natsBroker) startProducerSpan(msg *NATS.Msg) trace.Span {
	if b.opts.Tracer.Tracer == nil {
		return nil
	}

	carrier := NewMessageCarrier(msg)
	ctx := b.opts.Tracer.Propagators.Extract(b.opts.Context, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("nats"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Subject),
	}
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindProducer),
	}
	ctx, span := b.opts.Tracer.Tracer.Start(ctx, "nats.produce", opts...)

	b.opts.Tracer.Propagators.Inject(ctx, carrier)

	return span
}

func (b *natsBroker) finishProducerSpan(span trace.Span, err error) {
	if span == nil {
		return
	}

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}

func (b *natsBroker) startConsumerSpan(msg *NATS.Msg) trace.Span {
	if b.opts.Tracer.Tracer == nil {
		return nil
	}

	carrier := NewMessageCarrier(msg)
	ctx := b.opts.Tracer.Propagators.Extract(b.opts.Context, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("nats"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Subject),
		semConv.MessagingOperationReceive,
	}
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
	}
	newCtx, span := b.opts.Tracer.Tracer.Start(ctx, "nats.consume", opts...)

	b.opts.Tracer.Propagators.Inject(newCtx, carrier)

	return span
}

func (b *natsBroker) finishConsumerSpan(span trace.Span) {
	if span == nil {
		return
	}

	span.End()
}
