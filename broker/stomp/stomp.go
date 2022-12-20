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

	"github.com/go-kratos/kratos/v2/log"

	"github.com/go-stomp/stomp/v3/frame"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"
)

const (
	defaultAddr = "stomp://127.0.0.1:61613"
)

type stompBroker struct {
	opts     broker.Options
	endpoint *url.URL

	stompConn *stomp.Conn

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &stompBroker{
		opts: options,
	}

	return b
}

func (b *stompBroker) Name() string {
	return "stomp"
}

func (b *stompBroker) defaults() {
	WithConnectTimeout(30 * time.Second)(&b.opts)
	WithVirtualHost("/")(&b.opts)
}

func (b *stompBroker) Options() broker.Options {
	if b.opts.Context == nil {
		b.opts.Context = context.Background()
	}
	return b.opts
}

func (b *stompBroker) Address() string {
	if len(b.opts.Addrs) > 0 {
		return b.opts.Addrs[0]
	}
	return ""
}

func (b *stompBroker) Init(opts ...broker.Option) error {
	b.defaults()

	b.opts.Apply(opts...)

	var cAddrs []string
	for _, addr := range b.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		addr = refitUrl(addr, false)
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{defaultAddr}
	}
	b.opts.Addrs = cAddrs

	if len(b.opts.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "stomp-producer", b.opts.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "stomp-consumer", b.opts.Tracings...)
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
	if v, ok := b.opts.Context.Value(connectTimeoutKey{}).(time.Duration); ok {
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

	var stompOpts []func(*stomp.Conn) error
	if uri.User != nil && uri.User.Username() != "" {
		password, _ := uri.User.Password()
		stompOpts = append(stompOpts, stomp.ConnOpt.Login(uri.User.Username(), password))
	}
	if v, ok := b.opts.Context.Value(authKey{}).(*authRecord); ok {
		stompOpts = append(stompOpts, stomp.ConnOpt.Login(v.username, v.password))
	}
	if headers, ok := b.opts.Context.Value(connectHeaderKey{}).(map[string]string); ok {
		for k, v := range headers {
			stompOpts = append(stompOpts, stomp.ConnOpt.Header(k, v))
		}
	}
	if host, ok := b.opts.Context.Value(vHostKey{}).(string); ok {
		log.Infof("Adding host: %s", host)
		stompOpts = append(stompOpts, stomp.ConnOpt.Host(host))
	}
	if v, ok := b.opts.Context.Value(heartBeatKey{}).(*heartbeatTimeout); ok {
		stompOpts = append(stompOpts, stomp.ConnOpt.HeartBeat(v.sendTimeout, v.recvTimeout))
	}
	if v, ok := b.opts.Context.Value(heartBeatErrorKey{}).(time.Duration); ok {
		stompOpts = append(stompOpts, stomp.ConnOpt.HeartBeatError(v))
	}
	if v, ok := b.opts.Context.Value(msgSendTimeoutKey{}).(time.Duration); ok {
		stompOpts = append(stompOpts, stomp.ConnOpt.MsgSendTimeout(v))
	}
	if v, ok := b.opts.Context.Value(rcvReceiptTimeoutKey{}).(time.Duration); ok {
		stompOpts = append(stompOpts, stomp.ConnOpt.RcvReceiptTimeout(v))
	}

	b.stompConn, err = stomp.Connect(netConn, stompOpts...)
	if err != nil {
		_ = netConn.Close()
		return fmt.Errorf("failed to connect to %s: %v", uri.Host, err)
	}

	return nil
}

func (b *stompBroker) Disconnect() error {
	return b.stompConn.Disconnect()
}

func (b *stompBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.opts.Codec, msg)
	if err != nil {
		return err
	}

	return b.publish(topic, buf, opts...)
}

func (b *stompBroker) publish(topic string, msg []byte, opts ...broker.PublishOption) error {
	if b.stompConn == nil {
		return errors.New("not connected")
	}

	options := broker.PublishOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	stompOpt := make([]func(*frame.Frame) error, 0, 0)

	span := b.startProducerSpan(options.Context, topic, &stompOpt)

	if headers, ok := options.Context.Value(headerKey{}).(map[string]string); ok {
		for k, v := range headers {
			stompOpt = append(stompOpt, stomp.SendOpt.Header(k, v))
		}
	}
	if withReceipt, ok := options.Context.Value(receiptKey{}).(bool); ok && withReceipt {
		stompOpt = append(stompOpt, stomp.SendOpt.Receipt)
	}
	if withoutContentLength, ok := options.Context.Value(suppressContentLengthKey{}).(bool); ok && withoutContentLength {
		stompOpt = append(stompOpt, stomp.SendOpt.NoContentLength)
	}

	err := b.stompConn.Send(topic, "", msg, stompOpt...)

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

	stompOpt := make([]func(*frame.Frame) error, 0, len(opts))

	if durableQueue, ok := options.Context.Value(durableQueueKey{}).(bool); ok && durableQueue {
		stompOpt = append(stompOpt, stomp.SubscribeOpt.Header("persistent", "true"))
	}

	if headers, ok := options.Context.Value(subscribeHeaderKey{}).(map[string]string); ok && len(headers) > 0 {
		for k, v := range headers {
			stompOpt = append(stompOpt, stomp.SubscribeOpt.Header(k, v))
		}
	}

	var ackSuccess bool
	if bVal, ok := options.Context.Value(ackSuccessKey{}).(bool); ok && bVal {
		options.AutoAck = false
		ackSuccess = true
	}

	var ackMode stomp.AckMode
	if options.AutoAck {
		ackMode = stomp.AckAuto
	} else {
		ackMode = stomp.AckClientIndividual
	}

	sub, err := b.stompConn.Subscribe(topic, ackMode, stompOpt...)
	if err != nil {
		return nil, err
	}

	go func() {
		for msg := range sub.C {
			go func(msg *stomp.Message) {
				m := &broker.Message{
					Headers: stompHeaderToMap(msg.Header),
				}

				p := &publication{msg: msg, m: m, topic: topic, broker: b}

				ctx, span := b.startConsumerSpan(options.Context, msg)

				if binder != nil {
					m.Body = binder()
				} else {
					m.Body = msg.Body
				}

				if err := broker.Unmarshal(b.opts.Codec, msg.Body, &m.Body); err != nil {
					p.err = err
					log.Error(err)
				}

				p.err = handler(ctx, p)
				if p.err == nil && !options.AutoAck && ackSuccess {
					_ = msg.Conn.Ack(msg)
				}

				b.finishConsumerSpan(span)
			}(msg)
		}
	}()

	return &subscriber{sub: sub, topic: topic, opts: options}, nil
}

func (b *stompBroker) startProducerSpan(ctx context.Context, topic string, msg *[]func(*frame.Frame) error) trace.Span {
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

func (b *stompBroker) startConsumerSpan(ctx context.Context, msg *stomp.Message) (context.Context, trace.Span) {
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

func (b *stompBroker) finishConsumerSpan(span trace.Span) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(context.Background(), span, nil)
}
