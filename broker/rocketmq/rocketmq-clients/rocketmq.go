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

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"
	"github.com/tx7do/kratos-transport/tracing"
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

func (r *rocketmqBroker) Name() string {
	return "rocketmqV5"
}

func (r *rocketmqBroker) Address() string {
	if len(r.nameServers) > 0 {
		return r.nameServers[0]
	} else if r.nameServerUrl != "" {
		return r.nameServerUrl
	}
	return rocketmqOption.DefaultAddr
}

func (r *rocketmqBroker) Options() broker.Options {
	return r.options
}

func (r *rocketmqBroker) Init(opts ...broker.Option) error {
	r.options.Apply(opts...)

	// init logger
	rmqClient.ResetLogger()
	_ = os.Setenv(rmqClient.ENABLE_CONSOLE_APPENDER, "true")
	_ = os.Setenv(rmqClient.CLIENT_LOG_LEVEL, "info")

	if v, ok := r.options.Context.Value(rocketmqOption.NameServersKey{}).([]string); ok {
		r.nameServers = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.NameServerUrlKey{}).(string); ok {
		r.nameServerUrl = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.AccessKey{}).(string); ok {
		r.credentials.AccessKey = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.SecretKey{}).(string); ok {
		r.credentials.AccessSecret = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.SecurityTokenKey{}).(string); ok {
		r.credentials.SecurityToken = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.CredentialsKey{}).(*rocketmqOption.Credentials); ok {
		r.credentials = *v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.RetryCountKey{}).(int); ok {
		r.retryCount = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.NamespaceKey{}).(string); ok {
		r.namespace = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.InstanceNameKey{}).(string); ok {
		r.instanceName = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.GroupNameKey{}).(string); ok {
		r.groupName = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.EnableTraceKey{}).(bool); ok {
		r.enableTrace = v
	}

	if v, ok := r.options.Context.Value(rocketmqOption.SubscriptionExpressionsKey{}).(map[string]*rmqClient.FilterExpression); ok {
		r.subscriptionExpressions = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.AwaitDurationKey{}).(time.Duration); ok {
		r.awaitDuration = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.MaxMessageNumKey{}).(int32); ok {
		r.maxMessageNum = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.InvisibleDurationKey{}).(time.Duration); ok {
		r.invisibleDuration = v
	}
	if v, ok := r.options.Context.Value(rocketmqOption.ReceiveIntervalKey{}).(time.Duration); ok {
		r.receiveInterval = v
	}

	if v, ok := r.options.Context.Value(rocketmqOption.LoggerLevelKey{}).(log.Level); ok {
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
		}
		_ = os.Setenv(rmqClient.CLIENT_LOG_LEVEL, strLevel)
	}

	if len(r.options.Tracings) > 0 {
		r.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "rocketmq-producer", r.options.Tracings...)
		r.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "rocketmq-consumer", r.options.Tracings...)
	}

	return nil
}

func (r *rocketmqBroker) Connect() error {
	r.RLock()
	if r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	p, err := r.createProducer()
	if err != nil {
		return err
	}

	if p != nil {
		_ = p.GracefulStop()
	}

	r.Lock()
	r.connected = true
	r.Unlock()

	return err
}

func (r *rocketmqBroker) Disconnect() error {
	r.RLock()
	if !r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	r.Lock()
	defer r.Unlock()
	for _, p := range r.producers {
		if err := p.GracefulStop(); err != nil {
			return err
		}
	}
	r.producers = make(map[string]rmqClient.Producer)

	r.connected = false
	return nil
}

func (r *rocketmqBroker) Publish(ctx context.Context, topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(r.options.Codec, msg)
	if err != nil {
		return err
	}

	return r.publish(ctx, topic, buf, opts...)
}

func (r *rocketmqBroker) publish(ctx context.Context, topic string, msg []byte, opts ...broker.PublishOption) error {
	rocketmqOptions := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&rocketmqOptions)
	}

	r.Lock()
	producer, ok := r.producers[topic]
	if !ok {
		var err error
		producer, err = r.createProducer()
		if err != nil {
			r.Unlock()
			return err
		}

		r.producers[topic] = producer
	}
	r.Unlock()

	rMsg := &rmqClient.Message{
		Topic: topic,
		Body:  msg,
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
		err = r.doSendTransaction(rocketmqOptions.Context, producer, rMsg)
	} else if sendAsync {
		err = r.doSendAsync(rocketmqOptions.Context, producer, rMsg)
	} else {
		err = r.doSend(rocketmqOptions.Context, producer, rMsg)
	}

	return err
}

func (r *rocketmqBroker) doSend(ctx context.Context, producer rmqClient.Producer, rMsg *rmqClient.Message) error {
	span := r.startProducerSpan(ctx, rMsg, false)

	var err error
	var receipts []*rmqClient.SendReceipt
	receipts, err = producer.Send(r.options.Context, rMsg)
	if err != nil {
		log.Errorf("[rocketmq]: send message error: %s\n", err)
		r.finishProducerSpan(ctx, span, nil, err)
	} else {
		r.finishProducerSpan(ctx, span, receipts[0], nil)
	}
	return err
}

func (r *rocketmqBroker) doSendAsync(ctx context.Context, producer rmqClient.Producer, rMsg *rmqClient.Message) error {
	span := r.startProducerSpan(ctx, rMsg, false)

	producer.SendAsync(ctx, rMsg, func(ctx context.Context, receipts []*rmqClient.SendReceipt, err error) {
		if err != nil {
			log.Errorf("[rocketmq]: send async message error: %s\n", err)
			r.finishProducerSpan(ctx, span, nil, err)
		} else {
			r.finishProducerSpan(ctx, span, receipts[0], nil)
		}
	})

	return nil
}

func (r *rocketmqBroker) doSendTransaction(ctx context.Context, producer rmqClient.Producer, rMsg *rmqClient.Message) error {
	span := r.startProducerSpan(ctx, rMsg, true)

	transaction := producer.BeginTransaction()

	var err error
	var receipts []*rmqClient.SendReceipt
	if receipts, err = producer.SendWithTransaction(ctx, rMsg, transaction); err != nil {
		return err
	}

	if err = transaction.Commit(); err != nil {
		return err
	}

	if err != nil {
		log.Errorf("[rocketmq]: send transaction message error: %s\n", err)
		r.finishProducerSpan(ctx, span, nil, err)
	} else {
		r.finishProducerSpan(ctx, span, receipts[0], nil)
	}

	return nil
}

func (r *rocketmqBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	rocketmqOptions := &broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
		Queue:   r.groupName,
	}
	for _, o := range opts {
		o(rocketmqOptions)
	}

	if r.consumer == nil {
		c, err := r.createConsumer(rocketmqOptions)
		if err != nil {
			return nil, err
		}
		r.consumer = c
	}

	sub := &subscriber{
		r:       r,
		options: *rocketmqOptions,
		topic:   topic,
		handler: handler,
		binder:  binder,
		reader:  r.consumer,
		done:    make(chan error),
	}

	var filterExpression *rmqClient.FilterExpression
	if v, ok := rocketmqOptions.Context.Value(rocketmqOption.SubscriptionFilterExpressionKey{}).(*rmqClient.FilterExpression); ok {
		filterExpression = v
	} else {
		filterExpression = rmqClient.SUB_ALL
	}

	if err := r.consumer.Subscribe(topic, filterExpression); err != nil {
		return nil, err
	}

	r.subscribers.Add(topic, sub)

	return sub, nil
}

func (r *rocketmqBroker) makeConfig() *rmqClient.Config {
	return &rmqClient.Config{
		Endpoint:      r.nameServers[0],
		NameSpace:     r.namespace,
		ConsumerGroup: r.groupName,
		Credentials: &credentials.SessionCredentials{
			AccessKey:     r.credentials.AccessKey,
			AccessSecret:  r.credentials.AccessSecret,
			SecurityToken: r.credentials.SecurityToken,
		},
	}
}

func (r *rocketmqBroker) createProducer(topic ...string) (producer rmqClient.Producer, err error) {
	cfg := r.makeConfig()

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

func (r *rocketmqBroker) createConsumer(options *broker.SubscribeOptions) (consumer rmqClient.SimpleConsumer, err error) {
	cfg := r.makeConfig()

	var opts []rmqClient.SimpleConsumerOption
	opts = append(opts, rmqClient.WithAwaitDuration(r.awaitDuration))
	if len(r.subscriptionExpressions) > 0 {
		opts = append(opts, rmqClient.WithSubscriptionExpressions(r.subscriptionExpressions))
	}

	// create consumer
	if consumer, err = rmqClient.NewSimpleConsumer(cfg, opts...); err != nil {
		return nil, err
	}

	// start consumer
	if err = consumer.Start(); err != nil {
		return nil, err
	}

	go r.run()

	return
}

func (r *rocketmqBroker) run() {
	ctx := r.options.Context
	for {
		//fmt.Println("start receive message")

		if r.consumer == nil {
			return
		}

		var err error

		// receive the message
		var messages []*rmqClient.MessageView
		if messages, err = r.consumer.Receive(ctx, r.maxMessageNum, r.invisibleDuration); err != nil {
			continue
		}

		// process message
		for _, mv := range messages {
			if mv == nil {
				log.Errorf("received message is nil")
				continue
			}

			newCtx, span := r.startConsumerSpan(ctx, mv)

			sub := r.subscribers.Get(mv.GetTopic())
			if sub == nil {
				err = errors.New(fmt.Sprintf("[%s] subscriber not found", mv.GetTopic()))
				log.Errorf(err.Error())
				r.finishConsumerSpan(span, err)
				continue
			}

			aSub := sub.(*subscriber)

			if err = aSub.onMessage(newCtx, mv); err != nil {
				log.Errorf("[%s] subscriber not found", mv.GetTopic())
				r.finishConsumerSpan(span, err)
				continue
			}

			r.finishConsumerSpan(span, nil)
		}

		time.Sleep(r.receiveInterval)
	}
}

func (r *rocketmqBroker) startProducerSpan(ctx context.Context, msg *rmqClient.Message, transaction bool) trace.Span {
	if r.producerTracer == nil {
		return nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingProtocolKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_MESSAGING_PROTOCOL),
		semConv.MessagingProtocolVersionKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_MESSAGING_PROTOCOL_VERSION),
		semConv.MessagingRocketmqNamespaceKey.String(r.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(r.groupName),
		semConv.MessagingRocketmqClientIDKey.String(r.instanceName),
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
	ctx, span = r.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (r *rocketmqBroker) finishProducerSpan(ctx context.Context, span trace.Span, receipt *rmqClient.SendReceipt, err error) {
	if r.producerTracer == nil {
		return
	}

	var messageId string
	if receipt != nil {
		messageId = receipt.MessageID
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingMessageIDKey.String(messageId),
		semConv.MessagingRocketmqNamespaceKey.String(r.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(r.groupName),
	}

	r.producerTracer.End(ctx, span, err, attrs...)
}

func (r *rocketmqBroker) startConsumerSpan(ctx context.Context, msg *rmqClient.MessageView) (context.Context, trace.Span) {
	if r.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingProtocolKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_MESSAGING_PROTOCOL),
		semConv.MessagingProtocolVersionKey.String(rmqClient.SPAN_ATTRIBUTE_VALUE_MESSAGING_PROTOCOL_VERSION),
		semConv.MessagingRocketmqNamespaceKey.String(r.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(r.groupName),
		semConv.MessagingRocketmqClientIDKey.String(r.instanceName),
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
	ctx, span = r.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (r *rocketmqBroker) finishConsumerSpan(span trace.Span, err error) {
	if r.consumerTracer == nil {
		return
	}

	r.consumerTracer.End(context.Background(), span, err)
}
