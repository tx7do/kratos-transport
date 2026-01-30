package rocketmqClientGo

import (
	"context"
	"errors"
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/apache/rocketmq-client-go/v2/rlog"

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

	nameServers   []string
	nameServerUrl string

	credentials rocketmqOption.Credentials

	retryCount int

	instanceName string
	groupName    string
	namespace    string

	enableTrace bool

	connected bool
	options   broker.Options

	producers   map[string]rocketmq.Producer
	subscribers *broker.SubscriberSyncMap

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer

	logger *logger
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	return &rocketmqBroker{
		options:     options,
		retryCount:  2,
		producers:   make(map[string]rocketmq.Producer),
		subscribers: broker.NewSubscriberSyncMap(),
		logger: &logger{
			level: log.LevelInfo,
		},
	}
}

func (b *rocketmqBroker) Name() string {
	return "rocketmqV2"
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

	rlog.SetLogger(b.logger)

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

	if v, ok := b.options.Context.Value(rocketmqOption.LoggerLevelKey{}).(log.Level); ok {
		b.logger.level = v
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

	_ = p.Shutdown()

	b.Lock()
	b.connected = true
	b.Unlock()

	return nil
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
		if err := p.Shutdown(); err != nil {
			return err
		}
	}
	b.producers = make(map[string]rocketmq.Producer)

	b.connected = false
	return nil
}

func (b *rocketmqBroker) createNsResolver() primitive.NsResolver {
	if len(b.nameServers) > 0 {
		return primitive.NewPassthroughResolver(b.nameServers)
	} else if b.nameServerUrl != "" {
		return primitive.NewHttpResolver("DEFAULT", b.nameServerUrl)
	} else {
		return primitive.NewHttpResolver("DEFAULT", rocketmqOption.DefaultAddr)
	}
}

func (b *rocketmqBroker) makeCredentials() *primitive.Credentials {
	return &primitive.Credentials{
		AccessKey:     b.credentials.AccessKey,
		SecretKey:     b.credentials.AccessSecret,
		SecurityToken: b.credentials.SecurityToken,
	}
}

func (b *rocketmqBroker) createProducer() (rocketmq.Producer, error) {
	credentials := b.makeCredentials()

	resolver := b.createNsResolver()

	var opts []producer.Option

	var traceCfg *primitive.TraceConfig = nil
	if b.enableTrace {
		traceCfg = &primitive.TraceConfig{
			Access:   primitive.Cloud,
			Resolver: resolver,
		}

		if b.groupName == "" {
			traceCfg.GroupName = "DEFAULT"
		} else {
			traceCfg.GroupName = b.groupName
		}

		if credentials.AccessKey != "" && credentials.SecretKey != "" {
			traceCfg.Credentials = *credentials
		}

		opts = append(opts, producer.WithTrace(traceCfg))
	}

	if credentials.AccessKey != "" && credentials.SecretKey != "" {
		opts = append(opts, producer.WithCredentials(*credentials))
	}

	opts = append(opts, producer.WithNsResolver(resolver))
	opts = append(opts, producer.WithRetry(b.retryCount))
	if b.instanceName != "" {
		opts = append(opts, producer.WithInstanceName(b.instanceName))
	}
	if b.groupName != "" {
		opts = append(opts, producer.WithGroupName(b.groupName))
	}

	p, err := rocketmq.NewProducer(opts...)
	if err != nil {
		b.logger.Errorf("new producer error: %s", err.Error())
		return nil, err
	}

	err = p.Start()
	if err != nil {
		b.logger.Errorf("[rocketmq]: start producer error: %s", err.Error())
		return nil, err
	}

	return p, nil
}

func (b *rocketmqBroker) createConsumer(options *broker.SubscribeOptions) (rocketmq.PushConsumer, error) {

	consumerOptions := []consumer.Option{
		consumer.WithGroupName(options.Queue),
		consumer.WithAutoCommit(options.AutoAck),
		consumer.WithRetry(b.retryCount),
		consumer.WithNamespace(b.namespace),
		consumer.WithInstance(b.instanceName),
	}

	credentials := b.makeCredentials()
	consumerOptions = append(consumerOptions, consumer.WithCredentials(*credentials))

	resolver := b.createNsResolver()
	consumerOptions = append(consumerOptions, consumer.WithNsResolver(resolver))

	if v, ok := options.Context.Value(rocketmqOption.ConsumerModelKey{}).(rocketmqOption.MessageModel); ok {
		var m consumer.MessageModel
		switch v {
		case rocketmqOption.MessageModelClustering:
			m = consumer.Clustering
		case rocketmqOption.MessageModelBroadCasting:
			m = consumer.BroadCasting
		}
		consumerOptions = append(consumerOptions, consumer.WithConsumerModel(m))
	}

	// 消息追踪
	// 注意：阿里云线上的RocketMQ一定要加此选项，不然会导致消费不成功
	if b.enableTrace {
		traceCfg := &primitive.TraceConfig{
			GroupName:   options.Queue,
			Credentials: *credentials,
			Access:      primitive.Cloud,
			Resolver:    resolver,
		}
		consumerOptions = append(consumerOptions, consumer.WithTrace(traceCfg))
	}

	c, _ := rocketmq.NewPushConsumer(consumerOptions...)

	if c == nil {
		return nil, errors.New("create consumer error")
	}

	return c, nil
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
	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	var cached bool

	b.Lock()
	p, ok := b.producers[topic]
	if !ok {
		var err error
		p, err = b.createProducer()
		if err != nil {
			b.Unlock()
			return err
		}

		b.producers[topic] = p
	} else {
		cached = true
	}
	b.Unlock()

	rMsg := primitive.NewMessage(topic, msg.BodyBytes())

	if len(msg.Headers) > 0 {
		rMsg.WithProperties(msg.Headers)
	}

	if v, ok := options.Context.Value(rocketmqOption.CompressKey{}).(bool); ok {
		rMsg.Compress = v
	}
	if v, ok := options.Context.Value(rocketmqOption.BatchKey{}).(bool); ok {
		rMsg.Batch = v
	}
	if v, ok := options.Context.Value(rocketmqOption.PropertiesKey{}).(map[string]string); ok {
		rMsg.WithProperties(v)
	}
	if v, ok := options.Context.Value(rocketmqOption.DelayTimeLevelKey{}).(int); ok {
		rMsg.WithDelayTimeLevel(v)
	}
	if v, ok := options.Context.Value(rocketmqOption.TagsKey{}).(string); ok {
		rMsg.WithTag(v)
	}
	if v, ok := options.Context.Value(rocketmqOption.KeysKey{}).([]string); ok {
		rMsg.WithKeys(v)
	}
	if v, ok := options.Context.Value(rocketmqOption.ShardingKeyKey{}).(string); ok {
		rMsg.WithShardingKey(v)
	}

	var span trace.Span
	ctx, span = b.startProducerSpan(options.Context, rMsg)

	var err error
	var ret *primitive.SendResult
	ret, err = p.SendSync(b.options.Context, rMsg)
	if err != nil {
		b.logger.Errorf("[rocketmq]: send message error: %s\n", err)
		switch cached {
		case false:
		case true:
			b.Lock()
			if err = p.Shutdown(); err != nil {
				b.Unlock()
				break
			}
			delete(b.producers, topic)
			b.Unlock()

			p, err = b.createProducer()
			if err != nil {
				b.Unlock()
				break
			}
			if ret, err = p.SendSync(b.options.Context, rMsg); err == nil {
				b.Lock()
				b.producers[topic] = p
				b.Unlock()
				break
			}
		}
	}

	var messageId string
	if ret != nil {
		messageId = ret.MsgID
	}

	b.finishProducerSpan(ctx, span, messageId, err)

	return err
}

func (b *rocketmqBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
		Queue:   b.groupName,
	}
	for _, o := range opts {
		o(&options)
	}

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
	}

	c, err := b.createConsumer(&options)
	if err != nil {
		return nil, err
	}

	sub := &subscriber{
		options: options,
		topic:   topic,
		handler: handler,
		reader:  c,
	}

	if err = c.Subscribe(topic, consumer.MessageSelector{},
		func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			//b.logger.Infof("[rocketmq] subscribe callback: %v \n", msgs)

			var errSub error
			var m broker.Message
			for _, msg := range msgs {
				p := &publication{topic: msg.Topic, reader: sub.reader, m: &m, rm: &msg.Message, ctx: options.Context}

				newCtx, span := b.startConsumerSpan(ctx, msg)

				m.Headers = msg.GetProperties()

				if binder != nil {
					m.Body = binder()

					if errSub = broker.Unmarshal(b.options.Codec, msg.Body, &m.Body); errSub != nil {
						p.err = errSub
						b.logger.Errorf("%s", errSub.Error())
						b.finishConsumerSpan(newCtx, span, errSub)
						continue
					}
				} else {
					m.Body = msg.Body
				}

				if errSub = sub.handler(newCtx, p); errSub != nil {
					b.logger.Errorf("process message failed: %v", errSub)
					b.finishConsumerSpan(newCtx, span, errSub)
					continue
				}

				if sub.options.AutoAck {
					if errSub = p.Ack(); errSub != nil {
						b.logger.Errorf("unable to commit msg: %v", errSub)
					}
				}

				b.finishConsumerSpan(newCtx, span, errSub)
			}

			return consumer.ConsumeSuccess, nil
		}); err != nil {
		b.logger.Errorf("%s", err.Error())
		return nil, err
	}

	if err = c.Start(); err != nil {
		b.logger.Errorf("%s", err.Error())
		return nil, err
	}

	return sub, nil
}

func (b *rocketmqBroker) startProducerSpan(ctx context.Context, msg *primitive.Message) (context.Context, trace.Span) {
	if b.producerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rocketmqOption.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingRocketmqNamespaceKey.String(b.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(b.groupName),
		semConv.MessagingRocketmqClientIDKey.String(b.instanceName),
		semConv.MessagingDestinationKindTopic,

		semConv.MessagingDestinationKey.String(msg.Topic),
		semConv.MessagingRocketmqMessageTagKey.String(msg.GetTags()),
		semConv.MessagingRocketmqMessageKeysKey.String(msg.GetKeys()),
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	if span != nil {
		otel.GetTextMapPropagator().Inject(ctx, carrier)
	}

	return ctx, span
}

func (b *rocketmqBroker) finishProducerSpan(ctx context.Context, span trace.Span, messageId string, err error) {
	if b.producerTracer == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingMessageIDKey.String(messageId),
		semConv.MessagingRocketmqNamespaceKey.String(b.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(b.groupName),
	}

	b.producerTracer.End(ctx, span, err, attrs...)
}

func (b *rocketmqBroker) startConsumerSpan(ctx context.Context, msg *primitive.MessageExt) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rocketmqOption.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(msg.MsgId),
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
