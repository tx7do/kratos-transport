package nats

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	kProto "github.com/go-kratos/kratos/v2/encoding/proto"
	"google.golang.org/protobuf/proto"

	natsGo "github.com/nats-io/nats.go"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultAddr = "nats://127.0.0.1:4222"
)

const (
	TracerMessageSystemKey = "nats"
	SpanNameProducer       = "nats-producer"
	SpanNameConsumer       = "nats-consumer"
)

type natsBroker struct {
	sync.Once
	sync.RWMutex

	connected bool

	options broker.Options

	conn     *natsGo.Conn
	natsOpts natsGo.Options

	subscribers *broker.SubscriberSyncMap

	drain   bool
	closeCh chan error

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &natsBroker{
		options:     options,
		subscribers: broker.NewSubscriberSyncMap(),
	}

	return b
}

func (b *natsBroker) Address() string {
	if b.conn != nil && b.conn.IsConnected() {
		return b.conn.ConnectedUrl()
	}

	if len(b.options.Addrs) > 0 {
		return b.options.Addrs[0]
	}

	return defaultAddr
}

func (b *natsBroker) Name() string {
	return "NATS"
}

func (b *natsBroker) Options() broker.Options {
	return b.options
}

func (b *natsBroker) Init(opts ...broker.Option) error {
	b.setOption(opts...)

	if len(b.options.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, SpanNameProducer, b.options.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, SpanNameConsumer, b.options.Tracings...)
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
		cAddrs = []string{natsGo.DefaultURL}
	}
	return cAddrs
}

func (b *natsBroker) setOption(opts ...broker.Option) {
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
	b.setAddrs(b.options.Addrs)

	if b.options.Context.Value(drainConnectionKey{}) != nil {
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

	status := natsGo.CLOSED
	if b.conn != nil {
		status = b.conn.Status()
	}

	switch status {
	case natsGo.CONNECTED, natsGo.RECONNECTING, natsGo.CONNECTING:
		b.connected = true
		return nil
	default: // DISCONNECTED or CLOSED or DRAINING
		opts := b.natsOpts
		opts.Servers = b.options.Addrs
		opts.Secure = b.options.Secure
		opts.TLSConfig = b.options.TLSConfig

		if b.options.TLSConfig != nil {
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
		if b.conn != nil {
			_ = b.conn.Drain()
		}
		b.closeCh <- nil
	}

	b.subscribers.Clear()

	if b.conn != nil {
		b.conn.Close()
	}

	b.connected = false

	return nil
}

func (b *natsBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *natsBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *natsBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	b.RLock()
	defer b.RUnlock()

	if b.conn == nil {
		return errors.New("not connected")
	}

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	m := natsGo.NewMsg(topic)
	m.Data = msg.BodyBytes()

	if headers, ok := options.Context.Value(headersKey{}).(map[string][]string); ok {
		for k, v := range headers {
			for _, vv := range v {
				m.Header.Add(k, vv)
			}
		}
	}

	var span trace.Span
	ctx, span = b.startProducerSpan(options.Context, m)

	err := b.conn.PublishMsg(m)

	b.finishProducerSpan(ctx, span, err)

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

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
	}

	subs := &subscriber{
		n:       b,
		s:       nil,
		options: options,
	}

	fn := func(msg *natsGo.Msg) {
		var errSub error

		m := &broker.Message{
			Headers: natsHeaderToMap(msg.Header),
			Body:    nil,
			Msg:     msg,
		}

		pub := &publication{t: msg.Subject, m: m}

		ctx, span := b.startConsumerSpan(options.Context, msg)

		eh := b.options.ErrorHandler

		if binder != nil {
			if b.options.Codec.Name() == kProto.Name {
				m.Body = binder().(proto.Message)
			} else {
				m.Body = binder()
			}

			if errSub = broker.Unmarshal(b.options.Codec, msg.Data, &m.Body); errSub != nil {
				pub.err = errSub
				LogErrorf("unmarshal message failed: %v", errSub)
				if eh != nil {
					_ = eh(b.options.Context, pub)
				}

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

			b.finishConsumerSpan(ctx, span, errSub)
			return
		}

		if options.AutoAck {
			if errSub = pub.Ack(); errSub != nil {
				LogErrorf("unable to commit msg: %v", errSub)
			}
		}

		b.finishConsumerSpan(ctx, span, errSub)
	}

	var sub *natsGo.Subscription
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

	b.subscribers.Add(topic, subs)

	return subs, nil
}

func (b *natsBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return nil, err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.request(ctx, topic, sendMsg, opts...)
}

func (b *natsBroker) request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	b.RLock()
	defer b.RUnlock()

	if b.conn == nil {
		return nil, errors.New("not connected")
	}

	options := broker.RequestOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	m := natsGo.NewMsg(topic)
	m.Data = msg.BodyBytes()

	var timeout = time.Second * 2
	timeout, _ = options.Context.Value(requestTimeoutKey{}).(time.Duration)

	if headers, ok := options.Context.Value(headersKey{}).(map[string][]string); ok {
		for k, v := range headers {
			for _, vv := range v {
				m.Header.Add(k, vv)
			}
		}
	}
	var span trace.Span
	ctx, span = b.startProducerSpan(options.Context, m)

	res, err := b.conn.RequestMsg(m, timeout)

	b.finishProducerSpan(ctx, span, err)

	return broker.NewMessage(res, broker.WithMsg(res)), err
}

func (b *natsBroker) onClose(_ *natsGo.Conn) {
	b.closeCh <- nil
}

func (b *natsBroker) onAsyncError(_ *natsGo.Conn, _ *natsGo.Subscription, err error) {
	if errors.Is(err, natsGo.ErrDrainTimeout) {
		b.closeCh <- err
	}
}

func (b *natsBroker) onDisconnectedError(_ *natsGo.Conn, err error) {
	b.closeCh <- err
}

func (b *natsBroker) startProducerSpan(ctx context.Context, msg *natsGo.Msg) (context.Context, trace.Span) {
	if b.producerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(TracerMessageSystemKey),
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

func (b *natsBroker) finishProducerSpan(ctx context.Context, span trace.Span, err error) {
	if b.producerTracer == nil {
		return
	}

	b.producerTracer.End(ctx, span, err)
}

func (b *natsBroker) startConsumerSpan(ctx context.Context, msg *natsGo.Msg) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewMessageCarrier(msg)

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(TracerMessageSystemKey),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Subject),
		semConv.MessagingOperationReceive,
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *natsBroker) finishConsumerSpan(ctx context.Context, span trace.Span, err error) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(ctx, span, err)
}
