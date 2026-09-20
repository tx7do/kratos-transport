package rabbitmq

import (
	"context"
	"errors"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	TracerMessageSystemKey = "rabbitmq"
	SpanNameProducer       = "rabbitmq-producer"
	SpanNameConsumer       = "rabbitmq-consumer"

	ProtocolVersion = "0.9.1"
	Protocol        = "AMQP"
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

	if l := broker.LoggerFromOptions(&options); l != nil {
		SetLogger(l)
	}

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
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, SpanNameProducer, b.options.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, SpanNameConsumer, b.options.Tracings...)
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
	// 注意：b.wg 从未 Add（旧实现的每消息计数已移除），
	// 此处不再等待投递 goroutine——它们的退出由 deliveries channel 关闭保证

	return ret
}

func (b *rabbitBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return broker.GenericRequest(ctx, b, topic, msg, opts...)
}

func (b *rabbitBroker) Publish(ctx context.Context, routingKey string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, routingKey, msg, opts...)
}

func (b *rabbitBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *rabbitBroker) publish(ctx context.Context, routingKey string, msg *broker.Message, opts ...broker.PublishOption) error {
	if b.conn == nil {
		return errors.New("connection is nil")
	}

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	rMsg := amqp.Publishing{
		Body:    msg.BodyBytes(),
		Headers: amqp.Table{},
	}

	for k, v := range msg.Headers {
		rMsg.Headers[k] = v
	}

	if value, ok := options.Context.Value(deliveryModeKey{}).(uint8); ok {
		rMsg.DeliveryMode = value
	}

	if value, ok := options.Context.Value(priorityKey{}).(uint8); ok {
		rMsg.Priority = value
	}

	if value, ok := options.Context.Value(contentTypeKey{}).(string); ok {
		rMsg.ContentType = value
	}

	if value, ok := options.Context.Value(contentEncodingKey{}).(string); ok {
		rMsg.ContentEncoding = value
	}

	if value, ok := options.Context.Value(correlationIDKey{}).(string); ok {
		rMsg.CorrelationId = value
	}

	if value, ok := options.Context.Value(replyToKey{}).(string); ok {
		rMsg.ReplyTo = value
	}

	if value, ok := options.Context.Value(expirationKey{}).(string); ok {
		rMsg.Expiration = value
	}

	if value, ok := options.Context.Value(messageIDKey{}).(string); ok {
		rMsg.MessageId = value
	}

	if value, ok := options.Context.Value(timestampKey{}).(time.Time); ok {
		rMsg.Timestamp = value
	}

	if value, ok := options.Context.Value(messageTypeKey{}).(string); ok {
		rMsg.Type = value
	}

	if value, ok := options.Context.Value(userIDKey{}).(string); ok {
		rMsg.UserId = value
	}

	if value, ok := options.Context.Value(appIDKey{}).(string); ok {
		rMsg.AppId = value
	}

	if headers, ok := options.Context.Value(publishHeadersKey{}).(map[string]any); ok {
		for k, v := range headers {
			rMsg.Headers[k] = v
		}
	}

	// determine target exchange
	var exchangeName string
	if val, ok := options.Context.Value(publishExchangeKey{}).(string); ok && val != "" {
		exchangeName = val
	} else {
		exchangeName = b.conn.defaultExchangeName
	}

	if val, ok := options.Context.Value(publishDeclareQueueKey{}).(*DeclarePublishQueueInfo); ok {
		if val.Durable {
			val.AutoDelete = false
		}
		if err := b.conn.DeclarePublishQueue(exchangeName, val.Queue, routingKey, val.BindArguments, val.QueueArguments, val.Durable, val.AutoDelete); err != nil {
			return err
		}
	}

	var span trace.Span
	ctx, span = b.startProducerSpan(options.Context, routingKey, &rMsg)

	var mandatory bool
	if val, ok := options.Context.Value(mandatoryKey{}).(bool); ok {
		mandatory = val
	}

	err := b.conn.Publish(ctx, exchangeName, routingKey, mandatory, rMsg)

	b.finishProducerSpan(ctx, span, routingKey, err)

	return err
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

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
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

			if p.err = broker.Unmarshal(b.options.Codec, msg.Body, &m.Body); p.err != nil {
				// 反序列化失败：跳过 handler（避免拿到零值 Body 继续处理），
				// 走统一的错误/Nack 路径
				LogErrorf("unmarshal message failed: %v", p.err)
				if eh := b.options.ErrorHandler; eh != nil {
					_ = eh(ctx, p)
				}
				if !options.AutoAck {
					_ = msg.Nack(false, requeueOnError)
				}
				b.finishConsumerSpan(ctx, span, p.err)
				return
			}
		} else {
			m.Body = msg.Body
		}

		p.err = handler(ctx, p)
		if p.err != nil {
			if eh := b.options.ErrorHandler; eh != nil {
				_ = eh(ctx, p)
			}
		}
		if p.err == nil && ackSuccess && !options.AutoAck {
			_ = msg.Ack(false)
		} else if p.err != nil && !options.AutoAck {
			_ = msg.Nack(false, requeueOnError)
		}

		b.finishConsumerSpan(ctx, span, p.err)
	}

	// determine target exchange for subscription
	var subscribeExchangeName string
	if val, ok := options.Context.Value(subscribeExchangeKey{}).(string); ok && val != "" {
		subscribeExchangeName = val
	} else {
		subscribeExchangeName = b.conn.defaultExchangeName
	}

	sub := &subscriber{
		topic:        routingKey,
		options:      options,
		r:            b,
		exchangeName: subscribeExchangeName,
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

	if val, ok := options.Context.Value(subscribeBindArgsKey{}).(map[string]any); ok {
		sub.headers = val
	}

	if val, ok := options.Context.Value(subscribeQueueArgsKey{}).(map[string]any); ok {
		sub.queueArgs = val
	}

	if old := b.subscribers.Get(routingKey); old != nil {
		// 同主题重复订阅：先退订旧订阅，避免旧订阅继续消费（泄漏 + 重复消费）
		if uerr := old.Unsubscribe(false); uerr != nil {
			LogWarnf("unsubscribe old subscriber for topic %q failed: %v", routingKey, uerr)
		}
	}
	b.subscribers.Add(routingKey, sub)

	// 同步声明队列并绑定到 exchange，确保 Subscribe 返回后即可安全发布
	if err := sub.declareAndBind(); err != nil {
		b.subscribers.RemoveOnly(routingKey)
		return nil, err
	}

	// resubscribe 负责消费和断线重连（同一消费者，无双消费者风险）
	go sub.resubscribe()

	return sub, nil
}

func (b *rabbitBroker) startProducerSpan(ctx context.Context, routingKey string, msg *amqp.Publishing) (context.Context, trace.Span) {
	if b.producerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(TracerMessageSystemKey),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(routingKey),
		semConv.MessagingMessageIDKey.String(msg.MessageId),
		semConv.MessagingProtocolKey.String(Protocol),
		semConv.MessagingProtocolVersionKey.String(ProtocolVersion),
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	if span != nil {
		otel.GetTextMapPropagator().Inject(ctx, carrier)
	}

	return ctx, span
}

func (b *rabbitBroker) finishProducerSpan(ctx context.Context, span trace.Span, routingKey string, err error) {
	if b.producerTracer == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingRabbitmqRoutingKeyKey.String(routingKey),
	}

	b.producerTracer.End(ctx, span, err, attrs...)
}

func (b *rabbitBroker) startConsumerSpan(ctx context.Context, queueName string, msg *amqp.Delivery) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(TracerMessageSystemKey),
		semConv.MessagingDestinationKindQueue,
		semConv.MessagingDestinationKey.String(queueName),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(msg.MessageId),
		semConv.MessagingRabbitmqRoutingKeyKey.String(msg.RoutingKey),
		semConv.MessagingProtocolKey.String(Protocol),
		semConv.MessagingProtocolVersionKey.String(ProtocolVersion),
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *rabbitBroker) finishConsumerSpan(ctx context.Context, span trace.Span, err error) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(ctx, span, err)
}
