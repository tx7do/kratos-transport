package nats

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	natsGo "github.com/nats-io/nats.go"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	jsTracerMessageSystemKey = "nats-jetstream"
	jsSpanNameProducer       = "nats-jetstream-producer"
	jsSpanNameConsumer       = "nats-jetstream-consumer"
)

///////////////////////////////////////////////////////////////////////////////
/// JetStream Broker
///////////////////////////////////////////////////////////////////////////////

type jetStreamBroker struct {
	sync.Once
	sync.RWMutex

	connected bool

	options broker.Options

	conn     *natsGo.Conn
	natsOpts natsGo.Options
	js       natsGo.JetStreamContext

	drain   bool
	closeCh chan error

	subscribers *broker.SubscriberSyncMap

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

// NewJetStreamBroker creates a new NATS JetStream broker.
func NewJetStreamBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	return &jetStreamBroker{
		options:     options,
		subscribers: broker.NewSubscriberSyncMap(),
	}
}

func (b *jetStreamBroker) Name() string {
	return "NATS-JetStream"
}

func (b *jetStreamBroker) Options() broker.Options {
	return b.options
}

func (b *jetStreamBroker) Address() string {
	if b.conn != nil && b.conn.IsConnected() {
		return b.conn.ConnectedUrl()
	}

	if len(b.options.Addrs) > 0 {
		return b.options.Addrs[0]
	}

	return defaultAddr
}

func (b *jetStreamBroker) Init(opts ...broker.Option) error {
	b.setOption(opts...)

	if len(b.options.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, jsSpanNameProducer, b.options.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, jsSpanNameConsumer, b.options.Tracings...)
	}

	return nil
}

func (b *jetStreamBroker) Connect() error {
	b.Lock()
	defer b.Unlock()

	if b.connected {
		return nil
	}

	status := natsGo.CLOSED
	if b.conn != nil {
		status = b.conn.Status()
	}

	switch status {
	case natsGo.CONNECTED, natsGo.RECONNECTING, natsGo.CONNECTING:
		b.connected = true
		return nil
	default:
		opts := b.natsOpts
		opts.Servers = b.options.Addrs
		opts.Secure = b.options.Secure
		opts.TLSConfig = b.options.TLSConfig

		if b.options.TLSConfig != nil {
			opts.Secure = true
		}

		c, err := opts.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect to NATS server: %w", err)
		}
		b.conn = c

		// Create JetStream context
		jsOpts := b.extractJSContextOpts()
		js, err := c.JetStream(jsOpts...)
		if err != nil {
			c.Close()
			return fmt.Errorf("failed to create JetStream context: %w", err)
		}
		b.js = js
		b.connected = true

		LogInfof("connected to NATS JetStream at %s", b.Address())
		return nil
	}
}

func (b *jetStreamBroker) Disconnect() error {
	b.Lock()
	defer b.Unlock()

	if b.drain {
		if b.conn != nil {
			_ = b.conn.Drain()
		}
		if b.closeCh != nil {
			b.closeCh <- nil
		}
	}

	b.subscribers.Clear()

	if b.conn != nil {
		b.conn.Close()
	}

	b.connected = false

	return nil
}

// JetStreamContext returns the underlying JetStream context for advanced usage.
func (b *jetStreamBroker) JetStreamContext() natsGo.JetStreamContext {
	return b.js
}

// Conn returns the underlying NATS connection.
func (b *jetStreamBroker) Conn() *natsGo.Conn {
	return b.conn
}

///////////////////////////////////////////////////////////////////////////////
/// Internal helpers
///////////////////////////////////////////////////////////////////////////////

func (b *jetStreamBroker) setAddrs(addrs []string) []string {
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
		cAddrs = []string{natsGo.DefaultURL}
	}
	return cAddrs
}

func (b *jetStreamBroker) setOption(opts ...broker.Option) {
	for _, o := range opts {
		o(&b.options)
	}

	b.Once.Do(func() {
		b.natsOpts = natsGo.GetDefaultOptions()
	})

	if value, ok := b.options.Context.Value(optionsKey{}).(natsGo.Options); ok {
		b.natsOpts = value
	}

	if len(b.options.Addrs) == 0 {
		b.options.Addrs = b.natsOpts.Servers
	}

	if !b.options.Secure {
		b.options.Secure = b.natsOpts.Secure
	}

	if b.options.TLSConfig == nil {
		b.options.TLSConfig = b.natsOpts.TLSConfig
	}
	b.options.Addrs = b.setAddrs(b.options.Addrs)

	if b.options.Context.Value(drainConnectionKey{}) != nil {
		b.drain = true
		// 有缓冲：closeCh 没有读取方，无缓冲会让回调与持锁的 Disconnect 永久阻塞
		b.closeCh = make(chan error, 8)
		b.natsOpts.ClosedCB = b.onClose
		b.natsOpts.AsyncErrorCB = b.onAsyncError
		b.natsOpts.DisconnectedErrCB = b.onDisconnectedError
	}
}

func (b *jetStreamBroker) extractJSContextOpts() []natsGo.JSOpt {
	if v, ok := b.options.Context.Value(jetStreamContextOptsKey{}).([]natsGo.JSOpt); ok {
		return v
	}
	return nil
}

func (b *jetStreamBroker) onClose(_ *natsGo.Conn) {
	b.closeCh <- nil
}

func (b *jetStreamBroker) onAsyncError(_ *natsGo.Conn, _ *natsGo.Subscription, err error) {
	if errors.Is(err, natsGo.ErrDrainTimeout) {
		b.closeCh <- err
	}
}

func (b *jetStreamBroker) onDisconnectedError(_ *natsGo.Conn, err error) {
	b.closeCh <- err
}

///////////////////////////////////////////////////////////////////////////////
/// Tracing
///////////////////////////////////////////////////////////////////////////////

func (b *jetStreamBroker) startProducerSpan(ctx context.Context, msg *natsGo.Msg) (context.Context, trace.Span) {
	if b.producerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(jsTracerMessageSystemKey),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Subject),
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	if span != nil {
		otel.GetTextMapPropagator().Inject(ctx, carrier)
	}

	return ctx, span
}

func (b *jetStreamBroker) finishProducerSpan(ctx context.Context, span trace.Span, err error) {
	if b.producerTracer == nil {
		return
	}

	b.producerTracer.End(ctx, span, err)
}

func (b *jetStreamBroker) startConsumerSpan(ctx context.Context, msg *natsGo.Msg) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(msg)

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(jsTracerMessageSystemKey),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Subject),
		semConv.MessagingOperationReceive,
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *jetStreamBroker) finishConsumerSpan(ctx context.Context, span trace.Span, err error) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(ctx, span, err)
}
