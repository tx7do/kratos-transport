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

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/tracing"

	"github.com/tx7do/kratos-transport/broker"
	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"
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

func (r *rocketmqBroker) Name() string {
	return "rocketmqV2"
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

	rlog.SetLogger(r.logger)

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

	if v, ok := r.options.Context.Value(rocketmqOption.LoggerLevelKey{}).(log.Level); ok {
		r.logger.level = v
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

	_ = p.Shutdown()

	r.Lock()
	r.connected = true
	r.Unlock()

	return nil
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
		if err := p.Shutdown(); err != nil {
			return err
		}
	}
	r.producers = make(map[string]rocketmq.Producer)

	r.connected = false
	return nil
}

func (r *rocketmqBroker) createNsResolver() primitive.NsResolver {
	if len(r.nameServers) > 0 {
		return primitive.NewPassthroughResolver(r.nameServers)
	} else if r.nameServerUrl != "" {
		return primitive.NewHttpResolver("DEFAULT", r.nameServerUrl)
	} else {
		return primitive.NewHttpResolver("DEFAULT", rocketmqOption.DefaultAddr)
	}
}

func (r *rocketmqBroker) makeCredentials() *primitive.Credentials {
	return &primitive.Credentials{
		AccessKey:     r.credentials.AccessKey,
		SecretKey:     r.credentials.AccessSecret,
		SecurityToken: r.credentials.SecurityToken,
	}
}

func (r *rocketmqBroker) createProducer() (rocketmq.Producer, error) {
	credentials := r.makeCredentials()

	resolver := r.createNsResolver()

	var opts []producer.Option

	var traceCfg *primitive.TraceConfig = nil
	if r.enableTrace {
		traceCfg = &primitive.TraceConfig{
			Access:   primitive.Cloud,
			Resolver: resolver,
		}

		if r.groupName == "" {
			traceCfg.GroupName = "DEFAULT"
		} else {
			traceCfg.GroupName = r.groupName
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
	opts = append(opts, producer.WithRetry(r.retryCount))
	if r.instanceName != "" {
		opts = append(opts, producer.WithInstanceName(r.instanceName))
	}
	if r.groupName != "" {
		opts = append(opts, producer.WithGroupName(r.groupName))
	}

	p, err := rocketmq.NewProducer(opts...)
	if err != nil {
		r.logger.Errorf("new producer error: %s", err.Error())
		return nil, err
	}

	err = p.Start()
	if err != nil {
		r.logger.Errorf("[rocketmq]: start producer error: %s", err.Error())
		return nil, err
	}

	return p, nil
}

func (r *rocketmqBroker) createConsumer(options *broker.SubscribeOptions) (rocketmq.PushConsumer, error) {

	consumerOptions := []consumer.Option{
		consumer.WithGroupName(options.Queue),
		consumer.WithAutoCommit(options.AutoAck),
		consumer.WithRetry(r.retryCount),
		consumer.WithNamespace(r.namespace),
		consumer.WithInstance(r.instanceName),
	}

	credentials := r.makeCredentials()
	consumerOptions = append(consumerOptions, consumer.WithCredentials(*credentials))

	resolver := r.createNsResolver()
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
	if r.enableTrace {
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

func (r *rocketmqBroker) Request(ctx context.Context, topic string, msg broker.Any, opts ...broker.RequestOption) (broker.Any, error) {
	return nil, errors.New("not implemented")
}

func (r *rocketmqBroker) Publish(ctx context.Context, topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(r.options.Codec, msg)
	if err != nil {
		return err
	}

	return r.publish(ctx, topic, buf, opts...)
}

func (r *rocketmqBroker) publish(ctx context.Context, topic string, msg []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	var cached bool

	r.Lock()
	p, ok := r.producers[topic]
	if !ok {
		var err error
		p, err = r.createProducer()
		if err != nil {
			r.Unlock()
			return err
		}

		r.producers[topic] = p
	} else {
		cached = true
	}
	r.Unlock()

	rMsg := primitive.NewMessage(topic, msg)

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

	span := r.startProducerSpan(options.Context, rMsg)

	var err error
	var ret *primitive.SendResult
	ret, err = p.SendSync(r.options.Context, rMsg)
	if err != nil {
		r.logger.Errorf("[rocketmq]: send message error: %s\n", err)
		switch cached {
		case false:
		case true:
			r.Lock()
			if err = p.Shutdown(); err != nil {
				r.Unlock()
				break
			}
			delete(r.producers, topic)
			r.Unlock()

			p, err = r.createProducer()
			if err != nil {
				r.Unlock()
				break
			}
			if ret, err = p.SendSync(r.options.Context, rMsg); err == nil {
				r.Lock()
				r.producers[topic] = p
				r.Unlock()
				break
			}
		}
	}

	var messageId string
	if ret != nil {
		messageId = ret.MsgID
	}

	r.finishProducerSpan(span, messageId, err)

	return err
}

func (r *rocketmqBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
		Queue:   r.groupName,
	}
	for _, o := range opts {
		o(&options)
	}

	c, err := r.createConsumer(&options)
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
			//r.logger.Infof("[rocketmq] subscribe callback: %v \n", msgs)

			var errSub error
			var m broker.Message
			for _, msg := range msgs {
				p := &publication{topic: msg.Topic, reader: sub.reader, m: &m, rm: &msg.Message, ctx: options.Context}

				newCtx, span := r.startConsumerSpan(ctx, msg)

				m.Headers = msg.GetProperties()

				if binder != nil {
					m.Body = binder()

					if errSub = broker.Unmarshal(r.options.Codec, msg.Body, &m.Body); errSub != nil {
						p.err = errSub
						r.logger.Errorf("%s", errSub.Error())
						r.finishConsumerSpan(span, errSub)
						continue
					}
				} else {
					m.Body = msg.Body
				}

				if errSub = sub.handler(newCtx, p); errSub != nil {
					r.logger.Errorf("process message failed: %v", errSub)
					r.finishConsumerSpan(span, errSub)
					continue
				}

				if sub.options.AutoAck {
					if errSub = p.Ack(); errSub != nil {
						r.logger.Errorf("unable to commit msg: %v", errSub)
					}
				}

				r.finishConsumerSpan(span, errSub)
			}

			return consumer.ConsumeSuccess, nil
		}); err != nil {
		r.logger.Errorf("%s", err.Error())
		return nil, err
	}

	if err = c.Start(); err != nil {
		r.logger.Errorf("%s", err.Error())
		return nil, err
	}

	return sub, nil
}

func (r *rocketmqBroker) startProducerSpan(ctx context.Context, msg *primitive.Message) trace.Span {
	if r.producerTracer == nil {
		return nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rocketmqOption.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingRocketmqNamespaceKey.String(r.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(r.groupName),
		semConv.MessagingRocketmqClientIDKey.String(r.instanceName),
		semConv.MessagingDestinationKindTopic,

		semConv.MessagingDestinationKey.String(msg.Topic),
		semConv.MessagingRocketmqMessageTagKey.String(msg.GetTags()),
		semConv.MessagingRocketmqMessageKeysKey.String(msg.GetKeys()),
	}

	var span trace.Span
	ctx, span = r.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (r *rocketmqBroker) finishProducerSpan(span trace.Span, messageId string, err error) {
	if r.producerTracer == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingMessageIDKey.String(messageId),
		semConv.MessagingRocketmqNamespaceKey.String(r.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(r.groupName),
	}

	r.producerTracer.End(context.Background(), span, err, attrs...)
}

func (r *rocketmqBroker) startConsumerSpan(ctx context.Context, msg *primitive.MessageExt) (context.Context, trace.Span) {
	if r.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rocketmqOption.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(msg.MsgId),
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
