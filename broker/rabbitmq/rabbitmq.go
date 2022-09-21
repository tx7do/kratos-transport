package rabbitmq

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/streadway/amqp"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

type rabbitBroker struct {
	mtx sync.Mutex
	wg  sync.WaitGroup

	conn *rabbitConnection
	opts broker.Options

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &rabbitBroker{
		opts: options,
	}

	return b
}

func (b *rabbitBroker) Name() string {
	return "rabbitmq"
}

func (b *rabbitBroker) Options() broker.Options {
	return b.opts
}

func (b *rabbitBroker) Address() string {
	if len(b.opts.Addrs) > 0 {
		return b.opts.Addrs[0]
	}
	return ""
}

func (b *rabbitBroker) Init(opts ...broker.Option) error {
	b.opts.Apply(opts...)

	var addrs []string
	for _, addr := range b.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		if !hasUrlPrefix(addr) {
			addr = "amqp://" + addr
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		addrs = []string{DefaultRabbitURL}
	}
	b.opts.Addrs = addrs

	if len(b.opts.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "rabbitmq-consumer", b.opts.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "rabbitmq-consumer", b.opts.Tracings...)
	}

	return nil
}

func (b *rabbitBroker) Connect() error {
	if b.conn == nil {
		b.conn = newRabbitMQConnection(b.opts)
	}

	conf := DefaultAmqpConfig

	if auth, ok := b.opts.Context.Value(externalAuthKey{}).(ExternalAuthentication); ok {
		conf.SASL = []amqp.Authentication{&auth}
	}

	conf.TLSClientConfig = b.opts.TLSConfig

	return b.conn.Connect(b.opts.Secure, &conf)
}

func (b *rabbitBroker) Disconnect() error {
	if b.conn == nil {
		return errors.New("connection is nil")
	}
	ret := b.conn.Close()
	b.wg.Wait()
	return ret
}

func (b *rabbitBroker) Publish(routingKey string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.opts.Codec, msg)
	if err != nil {
		return err
	}

	return b.publish(routingKey, buf, opts...)
}

func (b *rabbitBroker) publish(routingKey string, buf []byte, opts ...broker.PublishOption) error {
	if b.conn == nil {
		return errors.New("connection is nil")
	}

	options := broker.PublishOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	msg := amqp.Publishing{
		Body:    buf,
		Headers: amqp.Table{},
	}

	if value, ok := options.Context.Value(deliveryModeKey{}).(uint8); ok {
		msg.DeliveryMode = value
	}

	if value, ok := options.Context.Value(priorityKey{}).(uint8); ok {
		msg.Priority = value
	}

	if value, ok := options.Context.Value(contentTypeKey{}).(string); ok {
		msg.ContentType = value
	}

	if value, ok := options.Context.Value(contentEncodingKey{}).(string); ok {
		msg.ContentEncoding = value
	}

	if value, ok := options.Context.Value(correlationIDKey{}).(string); ok {
		msg.CorrelationId = value
	}

	if value, ok := options.Context.Value(replyToKey{}).(string); ok {
		msg.ReplyTo = value
	}

	if value, ok := options.Context.Value(expirationKey{}).(string); ok {
		msg.Expiration = value
	}

	if value, ok := options.Context.Value(messageIDKey{}).(string); ok {
		msg.MessageId = value
	}

	if value, ok := options.Context.Value(timestampKey{}).(time.Time); ok {
		msg.Timestamp = value
	}

	if value, ok := options.Context.Value(messageTypeKey{}).(string); ok {
		msg.Type = value
	}

	if value, ok := options.Context.Value(userIDKey{}).(string); ok {
		msg.UserId = value
	}

	if value, ok := options.Context.Value(appIDKey{}).(string); ok {
		msg.AppId = value
	}

	if headers, ok := options.Context.Value(publishHeadersKey{}).(map[string]interface{}); ok {
		for k, v := range headers {
			msg.Headers[k] = v
		}
	}

	if val, ok := options.Context.Value(publishDeclareQueueKey{}).(*DeclarePublishQueueInfo); ok {
		if val.Durable {
			val.AutoDelete = false
		}
		if err := b.conn.DeclarePublishQueue(val.Queue, routingKey, val.BindArguments, val.QueueArguments, val.Durable, val.AutoDelete); err != nil {
			return err
		}
	}

	span := b.startProducerSpan(options.Context, routingKey, &msg)

	err := b.conn.Publish(b.conn.exchange.Name, routingKey, msg)

	b.finishProducerSpan(span, routingKey, err)

	return nil
}

func (b *rabbitBroker) Subscribe(routingKey string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if b.conn == nil {
		return nil, errors.New("not connected")
	}

	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}

	var requeueOnError = false
	if val, ok := options.Context.Value(requeueOnErrorKey{}).(bool); ok {
		requeueOnError = val
	}

	var ackSuccess = false
	if val, ok := options.Context.Value(ackSuccessKey{}).(bool); ok && val {
		options.AutoAck = false
		ackSuccess = true
	}

	fn := func(msg amqp.Delivery) {
		m := &broker.Message{
			Headers: rabbitHeaderToMap(msg.Headers),
			Body:    nil,
		}

		ctx, span := b.startConsumerSpan(options.Context, options.Queue, &msg)

		p := &publication{d: msg, m: m, t: msg.RoutingKey}

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
		if p.err == nil && ackSuccess && !options.AutoAck {
			_ = msg.Ack(false)
		} else if p.err != nil && !options.AutoAck {
			_ = msg.Nack(false, requeueOnError)
		}

		b.finishConsumerSpan(span, p.err)
	}

	sub := &subscriber{
		topic:        routingKey,
		opts:         options,
		mayRun:       true,
		r:            b,
		durableQueue: true,
		autoDelete:   false,
		fn:           fn,
		headers:      nil,
		queueArgs:    nil,
	}

	if val, ok := options.Context.Value(durableQueueKey{}).(bool); ok {
		sub.durableQueue = val
		sub.autoDelete = false
	}

	if val, ok := options.Context.Value(autoDeleteQueueKey{}).(bool); ok {
		sub.autoDelete = val
		sub.durableQueue = false
	}

	if val, ok := options.Context.Value(subscribeBindArgsKey{}).(map[string]interface{}); ok {
		sub.headers = val
	}

	if val, ok := options.Context.Value(subscribeQueueArgsKey{}).(map[string]interface{}); ok {
		sub.queueArgs = val
	}

	go sub.resubscribe()

	return sub, nil
}

func (b *rabbitBroker) startProducerSpan(ctx context.Context, routingKey string, msg *amqp.Publishing) trace.Span {
	if b.producerTracer == nil {
		return nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("rabbitmq"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(routingKey),
		semConv.MessagingMessageIDKey.String(msg.MessageId),
		semConv.MessagingProtocolKey.String("AMQP"),
		semConv.MessagingProtocolVersionKey.String("0.9.1"),
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (b *rabbitBroker) finishProducerSpan(span trace.Span, routingKey string, err error) {
	if b.producerTracer == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingRabbitmqRoutingKeyKey.String(routingKey),
	}

	b.producerTracer.End(context.Background(), span, err, attrs...)
}

func (b *rabbitBroker) startConsumerSpan(ctx context.Context, queueName string, msg *amqp.Delivery) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("rabbitmq"),
		semConv.MessagingDestinationKindQueue,
		semConv.MessagingDestinationKey.String(queueName),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(msg.MessageId),
		semConv.MessagingRabbitmqRoutingKeyKey.String(msg.RoutingKey),
		semConv.MessagingProtocolKey.String("AMQP"),
		semConv.MessagingProtocolVersionKey.String("0.9.1"),
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *rabbitBroker) finishConsumerSpan(span trace.Span, err error) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(context.Background(), span, err)
}
