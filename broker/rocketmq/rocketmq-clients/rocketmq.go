package rocketmqClients

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	rmqClient "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"
	"github.com/tx7do/kratos-transport/tracing"
)

const (
	TracerMessageSystemKey = "rocketmq"
	SpanNameProducer       = "rocketmq-producer"
	SpanNameConsumer       = "rocketmq-consumer"
)

type rocketmqBroker struct {
	sync.RWMutex

	options     broker.Options
	enableTrace bool

	nameServers   []string
	nameServerUrl string

	credentials rocketmqOption.Credentials

	retryCount int

	instanceName string
	groupName    string
	namespace    string

	connected bool

	producers   map[string]rmqClient.Producer
	consumer    rmqClient.SimpleConsumer
	subscribers *broker.SubscriberSyncMap

	subscriptionExpressions map[string]*rmqClient.FilterExpression
	awaitDuration           time.Duration
	maxMessageNum           int32
	invisibleDuration       time.Duration
	receiveInterval         time.Duration

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	rocketmqOptions := broker.NewOptionsAndApply(opts...)

	return &rocketmqBroker{
		options:           rocketmqOptions,
		retryCount:        2,
		awaitDuration:     defaultAwaitDuration,
		maxMessageNum:     defaultMaxMessageNum,
		invisibleDuration: defaultInvisibleDuration,
		receiveInterval:   defaultReceiveInterval,
		producers:         make(map[string]rmqClient.Producer),
		subscribers:       broker.NewSubscriberSyncMap(),
		credentials:       rocketmqOption.Credentials{},
	}
}

func (b *rocketmqBroker) Name() string {
	return "rocketmqV5"
}

func (b *rocketmqBroker) Address() string {
	if len(b.nameServers) > 0 {
		return b.nameServers[0]
	} else if b.nameServerUrl != "" {
		return b.nameServerUrl
	}
	return rocketmqOption.DefaultAddr
}

func (b *rocketmqBroker) Options() broker.Options {
	return b.options
}

func (b *rocketmqBroker) Init(opts ...broker.Option) error {
	b.options.Apply(opts...)

	// init logger
	rmqClient.ResetLogger()
	_ = os.Setenv(rmqClient.ENABLE_CONSOLE_APPENDER, "true")
	_ = os.Setenv(rmqClient.CLIENT_LOG_LEVEL, "info")

	if v, ok := b.options.Context.Value(rocketmqOption.NameServersKey{}).([]string); ok {
		b.nameServers = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.NameServerUrlKey{}).(string); ok {
		b.nameServerUrl = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.AccessKey{}).(string); ok {
		b.credentials.AccessKey = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.SecretKey{}).(string); ok {
		b.credentials.AccessSecret = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.SecurityTokenKey{}).(string); ok {
		b.credentials.SecurityToken = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.CredentialsKey{}).(*rocketmqOption.Credentials); ok {
		b.credentials = *v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.RetryCountKey{}).(int); ok {
		b.retryCount = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.NamespaceKey{}).(string); ok {
		b.namespace = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.InstanceNameKey{}).(string); ok {
		b.instanceName = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.GroupNameKey{}).(string); ok {
		b.groupName = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.EnableTraceKey{}).(bool); ok {
		b.enableTrace = v
	}

	if v, ok := b.options.Context.Value(rocketmqOption.SubscriptionExpressionsKey{}).(map[string]*rmqClient.FilterExpression); ok {
		b.subscriptionExpressions = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.AwaitDurationKey{}).(time.Duration); ok {
		b.awaitDuration = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.MaxMessageNumKey{}).(int32); ok {
		b.maxMessageNum = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.InvisibleDurationKey{}).(time.Duration); ok {
		b.invisibleDuration = v
	}
	if v, ok := b.options.Context.Value(rocketmqOption.ReceiveIntervalKey{}).(time.Duration); ok {
		b.receiveInterval = v
	}

	if v, ok := b.options.Context.Value(rocketmqOption.LoggerLevelKey{}).(log.Level); ok {
		var strLevel string
		switch v {
		case log.LevelDebug:
			strLevel = "debug"
		case log.LevelWarn:
			strLevel = "warn"
		case log.LevelError:
			strLevel = "error"
		case log.LevelInfo:
			strLevel = "info"
		default:
			panic("unhandled default case")
		}
		_ = os.Setenv(rmqClient.CLIENT_LOG_LEVEL, strLevel)
	}

	if len(b.options.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, SpanNameProducer, b.options.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, SpanNameConsumer, b.options.Tracings...)
	}

	return nil
}

func (b *rocketmqBroker) Connect() error {
	b.RLock()
	if b.connected {
		b.RUnlock()
		return nil
	}
	b.RUnlock()

	p, err := b.createProducer()
	if err != nil {
		return err
	}

	if p != nil {
		_ = p.GracefulStop()
	}

	b.Lock()
	b.connected = true
	b.Unlock()

	return err
}

func (b *rocketmqBroker) Disconnect() error {
	b.RLock()
	if !b.connected {
		b.RUnlock()
		return nil
	}
	b.RUnlock()

	b.Lock()
	defer b.Unlock()
	for _, p := range b.producers {
		if err := p.GracefulStop(); err != nil {
			return err
		}
	}
	b.producers = make(map[string]rmqClient.Producer)

	b.connected = false
	return nil
}

func (b *rocketmqBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return nil, errors.New("not implemented")
}

func (b *rocketmqBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *rocketmqBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *rocketmqBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	rocketmqOptions := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&rocketmqOptions)
	}

	b.Lock()
	producer, ok := b.producers[topic]
	if !ok {
		var err error
		producer, err = b.createProducer()
		if err != nil {
			b.Unlock()
			return err
		}

		b.producers[topic] = producer
	}
	b.Unlock()

	rMsg := &rmqClient.Message{
		Topic: topic,
		Body:  msg.BodyBytes(),
	}
	for k, v := range msg.Headers {
		rMsg.AddProperty(k, v)
	}

	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.PropertiesKey{}).(map[string]string); ok {
		for pk, pv := range v {
			rMsg.AddProperty(pk, pv)
		}
	}
	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.TagsKey{}).(string); ok {
		rMsg.SetTag(v)
	}
	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.KeysKey{}).([]string); ok {
		rMsg.SetKeys(v...)
	}
	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.DeliveryTimestampKey{}).(time.Time); ok {
		rMsg.SetDelayTimestamp(v)
	}
	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.MessageGroupKey{}).(string); ok {
		rMsg.SetMessageGroup(v)
	}

	var sendAsync bool
	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.SendAsyncKey{}).(bool); ok {
		sendAsync = v
	}

	var sendWithTransaction bool
	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.SendWithTransactionKey{}).(bool); ok {
		sendWithTransaction = v
	}

	var err error
	if sendWithTransaction {
		err = b.doSendTransaction(rocketmqOptions.Context, producer, rMsg)
	} else if sendAsync {
		err = b.doSendAsync(rocketmqOptions.Context, producer, rMsg)
	} else {
		err = b.doSend(rocketmqOptions.Context, producer, rMsg)
	}

	return err
}

func (b *rocketmqBroker) doSend(ctx context.Context, producer rmqClient.Producer, rMsg *rmqClient.Message) error {
	var span trace.Span
	ctx, span = b.startProducerSpan(ctx, rMsg, false)

	var err error
	var receipts []*rmqClient.SendReceipt
	receipts, err = producer.Send(b.options.Context, rMsg)
	if err != nil {
		LogErrorf("send message error: %s\n", err)
		b.finishProducerSpan(ctx, span, nil, err)
	} else {
		b.finishProducerSpan(ctx, span, receipts[0], nil)
	}
	return err
}

func (b *rocketmqBroker) doSendAsync(ctx context.Context, producer rmqClient.Producer, rMsg *rmqClient.Message) error {
	var span trace.Span
	ctx, span = b.startProducerSpan(ctx, rMsg, false)

	producer.SendAsync(ctx, rMsg, func(ctx context.Context, receipts []*rmqClient.SendReceipt, err error) {
		if err != nil {
			LogErrorf("send async message error: %s\n", err)
			b.finishProducerSpan(ctx, span, nil, err)
		} else {
			b.finishProducerSpan(ctx, span, receipts[0], nil)
		}
	})

	return nil
}

func (b *rocketmqBroker) doSendTransaction(ctx context.Context, producer rmqClient.Producer, rMsg *rmqClient.Message) error {
	var span trace.Span
	ctx, span = b.startProducerSpan(ctx, rMsg, true)

	transaction := producer.BeginTransaction()

	var err error
	var receipts []*rmqClient.SendReceipt
	if receipts, err = producer.SendWithTransaction(ctx, rMsg, transaction); err != nil {
		return err
	}

	if err = transaction.Commit(); err != nil {
		LogErrorf("send transaction message error: %s\n", err)
		b.finishProducerSpan(ctx, span, nil, err)
		return err
	}

	b.finishProducerSpan(ctx, span, receipts[0], nil)

	return nil
}

func (b *rocketmqBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	rocketmqOptions := &broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
		Queue:   b.groupName,
	}
	for _, o := range opts {
		o(rocketmqOptions)
	}

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
	}

	if b.consumer == nil {
		c, err := b.createConsumer(rocketmqOptions)
		if err != nil {
			return nil, err
		}
		b.consumer = c
	}

	sub := &subscriber{
		r:       b,
		options: *rocketmqOptions,
		topic:   topic,
		handler: handler,
		binder:  binder,
		reader:  b.consumer,
		done:    make(chan error),
	}

	var filterExpression *rmqClient.FilterExpression
	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.SubscriptionFilterExpressionKey{}).(*rmqClient.FilterExpression); ok {
		filterExpression = v
	} else {
		filterExpression = rmqClient.SUB_ALL
	}

	if err := b.consumer.Subscribe(topic, filterExpression); err != nil {
		return nil, err
	}

	b.subscribers.Add(topic, sub)

	return sub, nil
}

func (b *rocketmqBroker) makeConfig() *rmqClient.Config {
	return &rmqClient.Config{
		Endpoint:      b.nameServers[0],
		NameSpace:     b.namespace,
		ConsumerGroup: b.groupName,
		Credentials: &credentials.SessionCredentials{
			AccessKey:     b.credentials.AccessKey,
			AccessSecret:  b.credentials.AccessSecret,
			SecurityToken: b.credentials.SecurityToken,
		},
	}
}

func (b *rocketmqBroker) createProducer(topic ...string) (producer rmqClient.Producer, err error) {
	cfg := b.makeConfig()

	var opts []rmqClient.ProducerOption
	if len(topic) > 0 {
		opts = append(opts, rmqClient.WithTopics(topic...))
	}

	// create producer
	if producer, err = rmqClient.NewProducer(cfg, opts...); err != nil {
		return nil, err
	}

	// start producer
	if err = producer.Start(); err != nil {
		return nil, err
	}

	return
}

func (b *rocketmqBroker) createConsumer(options *broker.SubscribeOptions) (consumer rmqClient.SimpleConsumer, err error) {
	cfg := b.makeConfig()

	var opts []rmqClient.SimpleConsumerOption
	opts = append(opts, rmqClient.WithSimpleAwaitDuration(b.awaitDuration))
	if len(b.subscriptionExpressions) > 0 {
		opts = append(opts, rmqClient.WithSimpleSubscriptionExpressions(b.subscriptionExpressions))
	}

	// create consumer
	if consumer, err = rmqClient.NewSimpleConsumer(cfg, opts...); err != nil {
		return nil, err
	}

	// start consumer
	if err = consumer.Start(); err != nil {
		return nil, err
	}

	go b.run()

	return
}

func (b *rocketmqBroker) run() {
	ctx := b.options.Context
	for {
		//fmt.Println("start receive message")

		if b.consumer == nil {
			return
		}

		var err error

		// receive the message
		var messages []*rmqClient.MessageView
		if messages, err = b.consumer.Receive(ctx, b.maxMessageNum, b.invisibleDuration); err != nil {
			continue
		}

		// process message
		for _, mv := range messages {
			if mv == nil {
				LogErrorf("received message is nil")
				continue
			}

			newCtx, span := b.startConsumerSpan(ctx, mv)

			sub := b.subscribers.Get(mv.GetTopic())
			if sub == nil {
				err = errors.New(fmt.Sprintf("[%s] subscriber not found", mv.GetTopic()))
				LogErrorf(err.Error())
				b.finishConsumerSpan(newCtx, span, err)
				continue
			}

			aSub := sub.(*subscriber)

			if err = aSub.onMessage(newCtx, mv); err != nil {
				LogErrorf("[%s] subscriber not found", mv.GetTopic())
				b.finishConsumerSpan(newCtx, span, err)
				continue
			}

			b.finishConsumerSpan(newCtx, span, nil)
		}

		time.Sleep(b.receiveInterval)
	}
}

func (b *rocketmqBroker) startProducerSpan(ctx context.Context, msg *rmqClient.Message, transaction bool) (context.Context, trace.Span) {
	if b.producerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingProtocolKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_MESSAGING_PROTOCOL),
		semConv.MessagingProtocolVersionKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_MESSAGING_PROTOCOL_VERSION),
		semConv.MessagingRocketmqNamespaceKey.String(b.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(b.groupName),
		semConv.MessagingRocketmqClientIDKey.String(b.instanceName),
		semConv.MessagingDestinationKindTopic,

		semConv.MessagingOperationKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_SEND_OPERATION),
		semConv.MessagingDestinationKey.String(msg.Topic),
	}

	if msg.GetDeliveryTimestamp() != nil {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTypeDelay)
	} else if msg.GetMessageGroup() != nil && strings.ToLower(*msg.GetMessageGroup()) == "fifo" {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTypeFifo)
	} else if transaction {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTypeTransaction)
	} else {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTypeNormal)
	}

	if msg.GetTag() != nil {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTagKey.String(*msg.GetTag()))
	}
	if len(msg.GetKeys()) > 0 {
		attrs = append(attrs, semConv.MessagingRocketmqMessageKeysKey.StringSlice(msg.GetKeys()))
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	if span != nil {
		otel.GetTextMapPropagator().Inject(ctx, carrier)
	}

	return ctx, span
}

func (b *rocketmqBroker) finishProducerSpan(ctx context.Context, span trace.Span, receipt *rmqClient.SendReceipt, err error) {
	if b.producerTracer == nil {
		return
	}

	var messageId string
	if receipt != nil {
		messageId = receipt.MessageID
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingMessageIDKey.String(messageId),
		semConv.MessagingRocketmqNamespaceKey.String(b.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(b.groupName),
	}

	b.producerTracer.End(ctx, span, err, attrs...)
}

func (b *rocketmqBroker) startConsumerSpan(ctx context.Context, msg *rmqClient.MessageView) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingProtocolKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_MESSAGING_PROTOCOL),
		semConv.MessagingProtocolVersionKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_MESSAGING_PROTOCOL_VERSION),
		semConv.MessagingRocketmqNamespaceKey.String(b.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(b.groupName),
		semConv.MessagingRocketmqClientIDKey.String(b.instanceName),
		semConv.MessagingDestinationKindTopic,

		semConv.MessagingOperationKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_RECEIVE_OPERATION),
		semConv.MessagingDestinationKey.String(msg.GetTopic()),
		semConv.MessagingMessageIDKey.String(msg.GetMessageId()),
	}

	if msg.GetTag() != nil {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTagKey.String(*msg.GetTag()))
	}
	if len(msg.GetKeys()) > 0 {
		attrs = append(attrs, semConv.MessagingRocketmqMessageKeysKey.StringSlice(msg.GetKeys()))
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *rocketmqBroker) finishConsumerSpan(ctx context.Context, span trace.Span, err error) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(ctx, span, err)
}
