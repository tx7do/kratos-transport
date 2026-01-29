package stomp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	stompV3 "github.com/go-stomp/stomp/v3"
	frameV3 "github.com/go-stomp/stomp/v3/frame"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"
)

const (
	defaultAddr = "stomp://127.0.0.1:61613"
)

type stompBroker struct {
	options  broker.Options
	endpoint *url.URL

	stompConn *stompV3.Conn

	subscribers *broker.SubscriberSyncMap

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &stompBroker{
		options:     options,
		subscribers: broker.NewSubscriberSyncMap(),
	}

	return b
}

func (b *stompBroker) Name() string {
	return "stomp"
}

func (b *stompBroker) defaults() {
	WithConnectTimeout(30 * time.Second)(&b.options)
	WithVirtualHost("/")(&b.options)
}

func (b *stompBroker) Options() broker.Options {
	if b.options.Context == nil {
		b.options.Context = context.Background()
	}
	return b.options
}

func (b *stompBroker) Address() string {
	if len(b.options.Addrs) > 0 {
		return b.options.Addrs[0]
	}
	return ""
}

func (b *stompBroker) Init(opts ...broker.Option) error {
	b.defaults()

	b.options.Apply(opts...)

	var cAddrs []string
	for _, addr := range b.options.Addrs {
		if len(addr) == 0 {
			continue
		}
		addr = refitUrl(addr, false)
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{defaultAddr}
	}
	b.options.Addrs = cAddrs

	if len(b.options.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "stomp-producer", b.options.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "stomp-consumer", b.options.Tracings...)
	}

	return nil
}

func (b *stompBroker) Connect() error {
	uri, err := url.Parse(b.Address())
	if err != nil {
		return err
	}

	if !isSchema(uri.Scheme) {
		return fmt.Errorf("expected stomp:// protocol but was %s", uri.Scheme)
	}

	var connectTimeOut time.Duration
	if v, ok := b.options.Context.Value(connectTimeoutKey{}).(time.Duration); ok {
		connectTimeOut = v
	}

	var netConn net.Conn
	if connectTimeOut > 0 {
		netConn, err = net.DialTimeout("tcp", uri.Host, connectTimeOut)
	} else {
		netConn, err = net.Dial("tcp", uri.Host)
	}
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", uri.Host, err)
	}

	var stompOpts []func(*stompV3.Conn) error
	if uri.User != nil && uri.User.Username() != "" {
		password, _ := uri.User.Password()
		stompOpts = append(stompOpts, stompV3.ConnOpt.Login(uri.User.Username(), password))
	}
	if v, ok := b.options.Context.Value(authKey{}).(*authRecord); ok {
		stompOpts = append(stompOpts, stompV3.ConnOpt.Login(v.username, v.password))
	}
	if headers, ok := b.options.Context.Value(connectHeaderKey{}).(map[string]string); ok {
		for k, v := range headers {
			stompOpts = append(stompOpts, stompV3.ConnOpt.Header(k, v))
		}
	}
	if host, ok := b.options.Context.Value(vHostKey{}).(string); ok {
		LogInfof("Adding host: %s", host)
		stompOpts = append(stompOpts, stompV3.ConnOpt.Host(host))
	}
	if v, ok := b.options.Context.Value(heartBeatKey{}).(*heartbeatTimeout); ok {
		stompOpts = append(stompOpts, stompV3.ConnOpt.HeartBeat(v.sendTimeout, v.recvTimeout))
	}
	if v, ok := b.options.Context.Value(heartBeatErrorKey{}).(time.Duration); ok {
		stompOpts = append(stompOpts, stompV3.ConnOpt.HeartBeatError(v))
	}
	if v, ok := b.options.Context.Value(msgSendTimeoutKey{}).(time.Duration); ok {
		stompOpts = append(stompOpts, stompV3.ConnOpt.MsgSendTimeout(v))
	}
	if v, ok := b.options.Context.Value(rcvReceiptTimeoutKey{}).(time.Duration); ok {
		stompOpts = append(stompOpts, stompV3.ConnOpt.RcvReceiptTimeout(v))
	}

	b.stompConn, err = stompV3.Connect(netConn, stompOpts...)
	if err != nil {
		_ = netConn.Close()
		return fmt.Errorf("failed to connect to %s: %v", uri.Host, err)
	}

	return nil
}

func (b *stompBroker) Disconnect() error {
	var err error

	if b.stompConn != nil {
		err = b.stompConn.Disconnect()
	}

	b.subscribers.Clear()

	return err
}

func (b *stompBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return nil, errors.New("not implemented")
}

func (b *stompBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *stompBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	if b.stompConn == nil {
		return errors.New("not connected")
	}

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	stompOpt := make([]func(*frameV3.Frame) error, 0)

	span := b.startProducerSpan(options.Context, topic, &stompOpt)

	for k, v := range msg.Headers {
		stompOpt = append(stompOpt, stompV3.SendOpt.Header(k, v))
	}

	if headers, ok := options.Context.Value(headerKey{}).(map[string]string); ok {
		for k, v := range headers {
			stompOpt = append(stompOpt, stompV3.SendOpt.Header(k, v))
		}
	}
	if withReceipt, ok := options.Context.Value(receiptKey{}).(bool); ok && withReceipt {
		stompOpt = append(stompOpt, stompV3.SendOpt.Receipt)
	}
	if withoutContentLength, ok := options.Context.Value(suppressContentLengthKey{}).(bool); ok && withoutContentLength {
		stompOpt = append(stompOpt, stompV3.SendOpt.NoContentLength)
	}

	err := b.stompConn.Send(topic, "", msg.BodyBytes(), stompOpt...)

	b.finishProducerSpan(span, err)

	return err
}

func (b *stompBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if b.stompConn == nil {
		return nil, errors.New("not connected")
	}

	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}

	stompOpt := make([]func(*frameV3.Frame) error, 0, len(opts))

	if durableQueue, ok := options.Context.Value(durableQueueKey{}).(bool); ok && durableQueue {
		stompOpt = append(stompOpt, stompV3.SubscribeOpt.Header("persistent", "true"))
	}

	if headers, ok := options.Context.Value(subscribeHeaderKey{}).(map[string]string); ok && len(headers) > 0 {
		for k, v := range headers {
			stompOpt = append(stompOpt, stompV3.SubscribeOpt.Header(k, v))
		}
	}

	var ackSuccess bool
	if bVal, ok := options.Context.Value(ackSuccessKey{}).(bool); ok && bVal {
		options.AutoAck = false
		ackSuccess = true
	}

	var ackMode stompV3.AckMode
	if options.AutoAck {
		ackMode = stompV3.AckAuto
	} else {
		ackMode = stompV3.AckClientIndividual
	}

	sub, err := b.stompConn.Subscribe(topic, ackMode, stompOpt...)
	if err != nil {
		return nil, err
	}

	go func() {
		for msg := range sub.C {
			go func(msg *stompV3.Message) {
				m := &broker.Message{
					Headers: stompHeaderToMap(msg.Header),
				}

				p := &publication{msg: msg, m: m, topic: topic, broker: b}

				ctx, span := b.startConsumerSpan(options.Context, msg)

				if binder != nil {
					m.Body = binder()

					if err = broker.Unmarshal(b.options.Codec, msg.Body, &m.Body); err != nil {
						p.err = err
						LogError(err)
						b.finishConsumerSpan(span, p.err)
						return
					}
				} else {
					m.Body = msg.Body
				}

				if err = handler(ctx, p); p.err != nil {
					p.err = err
					b.finishConsumerSpan(span, p.err)
					return
				}

				if options.AutoAck || ackSuccess {
					err = msg.Conn.Ack(msg)
					p.err = err
				}

				b.finishConsumerSpan(span, err)
			}(msg)
		}
	}()

	subs := &subscriber{
		b:       b,
		sub:     sub,
		topic:   topic,
		options: options,
	}

	b.subscribers.Add(topic, subs)

	return subs, nil
}

func (b *stompBroker) startProducerSpan(ctx context.Context, topic string, msg *[]func(*frameV3.Frame) error) trace.Span {
	if b.producerTracer == nil {
		return nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("stomp"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(topic),
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (b *stompBroker) finishProducerSpan(span trace.Span, err error) {
	if b.producerTracer == nil {
		return
	}

	b.producerTracer.End(context.Background(), span, err)
}

func (b *stompBroker) startConsumerSpan(ctx context.Context, msg *stompV3.Message) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("stomp"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Destination),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(msg.Header.Get("message-id")),
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *stompBroker) finishConsumerSpan(span trace.Span, err error) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(context.Background(), span, err)
}
