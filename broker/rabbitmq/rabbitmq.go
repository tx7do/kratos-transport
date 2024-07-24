package rabbitmq

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

type rabbitBroker struct {
	mtx sync.Mutex
	wg  sync.WaitGroup

	conn    *rabbitConnection
	options broker.Options

	subscribers *broker.SubscriberSyncMap

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &rabbitBroker{
		options:     options,
		subscribers: broker.NewSubscriberSyncMap(),
	}

	return b
}

func (b *rabbitBroker) Name() string {
	return "rabbitmq"
}

func (b *rabbitBroker) Options() broker.Options {
	return b.options
}

func (b *rabbitBroker) Address() string {
	if len(b.options.Addrs) > 0 {
		return b.options.Addrs[0]
	}
	return ""
}

func (b *rabbitBroker) Init(opts ...broker.Option) error {
	b.options.Apply(opts...)

	var addrs []string
	for _, addr := range b.options.Addrs {
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
	b.options.Addrs = addrs

	if len(b.options.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "rabbitmq-producer", b.options.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "rabbitmq-consumer", b.options.Tracings...)
	}

	return nil
}

func (b *rabbitBroker) Connect() error {
	if b.conn == nil {
		b.conn = newRabbitMQConnection(b.options)
	}

	conf := DefaultAmqpConfig

	if auth, ok := b.options.Context.Value(externalAuthKey{}).(ExternalAuthentication); ok {
		conf.SASL = []amqp.Authentication{&auth}
	}

	conf.TLSClientConfig = b.options.TLSConfig

	return b.conn.Connect(b.options.Secure, &conf)
}

func (b *rabbitBroker) Disconnect() error {
	if b.conn == nil {
		return errors.New("connection is nil")
	}

	b.subscribers.Clear()

	ret := b.conn.Close()
	b.wg.Wait()

	return ret
}

func (b *rabbitBroker) Request(ctx context.Context, topic string, msg broker.Any, timeout time.Duration, opts ...broker.PublishOption) (broker.Any, error) {
	return nil, errors.New("not implemented")
}

func (b *rabbitBroker) Publish(ctx context.Context, routingKey string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg)
	if err != nil {
		return err
	}

	return b.publish(ctx, routingKey, buf, opts...)
}

func (b *rabbitBroker) publish(ctx context.Context, routingKey string, buf []byte, opts ...broker.PublishOption) error {
	if b.conn == nil {
		return errors.New("connection is nil")
	}

	options := broker.PublishOptions{
		Context: ctx,
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

	err := b.conn.Publish(ctx, b.conn.exchange.Name, routingKey, msg)

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

		p := &publication{d: msg, message: m, topic: msg.RoutingKey}

		if binder != nil {
			m.Body = binder()
		} else {
			m.Body = msg.Body
		}

		if p.err = broker.Unmarshal(b.options.Codec, msg.Body, &m.Body); p.err != nil {
			log.Errorf("[rabbitmq] unmarshal message failed: %v", p.err)
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
		options:      options,
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

	b.subscribers.Add(routingKey, sub)

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
