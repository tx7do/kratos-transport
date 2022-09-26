package rocketmq

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"
)

const (
	defaultAddr = "127.0.0.1:9876"
)

type rocketmqBroker struct {
	nameServers   []string
	nameServerUrl string

	accessKey string
	secretKey string

	retryCount int

	instanceName string
	groupName    string
	namespace    string

	enableTrace bool

	connected bool
	sync.RWMutex
	opts broker.Options

	producers map[string]rocketmq.Producer

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	if v, ok := options.Context.Value(enableAliyunHttpKey{}).(bool); ok && v {
		return newAliyunHttpBroker(options)
	} else {
		return newBroker(options)
	}
}

func newBroker(options broker.Options) broker.Broker {
	return &rocketmqBroker{
		producers:  make(map[string]rocketmq.Producer),
		opts:       options,
		retryCount: 2,
	}
}

func (r *rocketmqBroker) Name() string {
	return "rocketmq"
}

func (r *rocketmqBroker) Address() string {
	if len(r.nameServers) > 0 {
		return r.nameServers[0]
	} else if r.nameServerUrl != "" {
		return r.nameServerUrl
	}
	return defaultAddr
}

func (r *rocketmqBroker) Options() broker.Options {
	return r.opts
}

func (r *rocketmqBroker) Init(opts ...broker.Option) error {
	r.opts.Apply(opts...)

	if v, ok := r.opts.Context.Value(nameServersKey{}).([]string); ok {
		r.nameServers = v
	}
	if v, ok := r.opts.Context.Value(nameServerUrlKey{}).(string); ok {
		r.nameServerUrl = v
	}
	if v, ok := r.opts.Context.Value(accessKey{}).(string); ok {
		r.accessKey = v
	}
	if v, ok := r.opts.Context.Value(secretKey{}).(string); ok {
		r.secretKey = v
	}
	if v, ok := r.opts.Context.Value(retryCountKey{}).(int); ok {
		r.retryCount = v
	}
	if v, ok := r.opts.Context.Value(namespaceKey{}).(string); ok {
		r.namespace = v
	}
	if v, ok := r.opts.Context.Value(instanceNameKey{}).(string); ok {
		r.instanceName = v
	}
	if v, ok := r.opts.Context.Value(groupNameKey{}).(string); ok {
		r.groupName = v
	}
	if v, ok := r.opts.Context.Value(enableTraceKey{}).(bool); ok {
		r.enableTrace = v
	}

	if len(r.opts.Tracings) > 0 {
		r.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "rocketmq-producer", r.opts.Tracings...)
		r.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "rocketmq-consumer", r.opts.Tracings...)
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

	r.connected = false
	return nil
}

func (r *rocketmqBroker) createNsResolver() primitive.NsResolver {
	if len(r.nameServers) > 0 {
		return primitive.NewPassthroughResolver(r.nameServers)
	} else if r.nameServerUrl != "" {
		return primitive.NewHttpResolver("DEFAULT", r.nameServerUrl)
	} else {
		return primitive.NewHttpResolver("DEFAULT", defaultAddr)
	}
}

func (r *rocketmqBroker) createProducer() (rocketmq.Producer, error) {

	var credentials primitive.Credentials
	if r.accessKey == "" || r.secretKey == "" {
		credentials = primitive.Credentials{
			AccessKey: r.accessKey,
			SecretKey: r.secretKey,
		}
	}

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
			traceCfg.Credentials = credentials
		}

		opts = append(opts, producer.WithTrace(traceCfg))
	}

	if credentials.AccessKey != "" && credentials.SecretKey != "" {
		producer.WithCredentials(credentials)
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
		log.Errorf("[rocketmq]: new producer error: " + err.Error())
		return nil, err
	}

	err = p.Start()
	if err != nil {
		log.Errorf("[rocketmq]: start producer error: %s", err.Error())
		return nil, err
	}

	return p, nil
}

func (r *rocketmqBroker) createConsumer(options *broker.SubscribeOptions) (rocketmq.PushConsumer, error) {
	credentials := primitive.Credentials{
		AccessKey: r.accessKey,
		SecretKey: r.secretKey,
	}

	resolver := r.createNsResolver()

	var traceCfg *primitive.TraceConfig = nil
	if r.enableTrace {
		traceCfg = &primitive.TraceConfig{
			GroupName:   options.Queue,
			Credentials: credentials,
			Access:      primitive.Cloud,
			Resolver:    resolver,
		}
	}

	c, _ := rocketmq.NewPushConsumer(
		consumer.WithNsResolver(resolver),
		consumer.WithCredentials(credentials),
		consumer.WithTrace(traceCfg),
		consumer.WithGroupName(options.Queue),
		consumer.WithAutoCommit(options.AutoAck),
		consumer.WithRetry(r.retryCount),
		consumer.WithNamespace(r.namespace),
		consumer.WithInstance(r.instanceName),
	)

	if c == nil {
		return nil, errors.New("create consumer error")
	}

	return c, nil
}

func (r *rocketmqBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(r.opts.Codec, msg)
	if err != nil {
		return err
	}

	return r.publish(topic, buf, opts...)
}

func (r *rocketmqBroker) publish(topic string, msg []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: context.Background(),
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

	if v, ok := options.Context.Value(compressKey{}).(bool); ok {
		rMsg.Compress = v
	}
	if v, ok := options.Context.Value(batchKey{}).(bool); ok {
		rMsg.Batch = v
	}
	if v, ok := options.Context.Value(propertiesKey{}).(map[string]string); ok {
		rMsg.WithProperties(v)
	}
	if v, ok := options.Context.Value(delayTimeLevelKey{}).(int); ok {
		rMsg.WithDelayTimeLevel(v)
	}
	if v, ok := options.Context.Value(tagsKey{}).(string); ok {
		rMsg.WithTag(v)
	}
	if v, ok := options.Context.Value(keysKey{}).([]string); ok {
		rMsg.WithKeys(v)
	}
	if v, ok := options.Context.Value(shardingKeyKey{}).(string); ok {
		rMsg.WithShardingKey(v)
	}

	span := r.startProducerSpan(options.Context, rMsg)

	var err error
	var ret *primitive.SendResult
	ret, err = p.SendSync(r.opts.Context, rMsg)
	if err != nil {
		log.Errorf("[rocketmq]: send message error: %s\n", err)
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
			if ret, err = p.SendSync(r.opts.Context, rMsg); err == nil {
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
		opts:    options,
		topic:   topic,
		handler: handler,
		reader:  c,
	}

	if err := c.Subscribe(topic, consumer.MessageSelector{},
		func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			//log.Infof("[rocketmq] subscribe callback: %v \n", msgs)

			var err error
			var m broker.Message
			for _, msg := range msgs {
				p := &publication{topic: msg.Topic, reader: sub.reader, m: &m, rm: &msg.Message, ctx: options.Context}

				newCtx, span := r.startConsumerSpan(ctx, msg)

				m.Headers = msg.GetProperties()

				if binder != nil {
					m.Body = binder()
				} else {
					m.Body = msg.Body
				}

				if err := broker.Unmarshal(r.opts.Codec, msg.Body, &m.Body); err != nil {
					p.err = err
					log.Error("[rocketmq] ", err)
				}

				err = sub.handler(newCtx, p)
				if err != nil {
					log.Errorf("[rocketmq]: process message failed: %v", err)
				}
				if sub.opts.AutoAck {
					if err = p.Ack(); err != nil {
						log.Errorf("[rocketmq]: unable to commit msg: %v", err)
					}
				}

				r.finishConsumerSpan(span)
			}

			return consumer.ConsumeSuccess, nil
		}); err != nil {
		log.Error("[rocketmq] ", err.Error())
		return nil, err
	}

	if err := c.Start(); err != nil {
		log.Error("[rocketmq] ", err.Error())
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
		semConv.MessagingSystemKey.String("rocketmq"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic),
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
		semConv.MessagingSystemKey.String("rocketmq"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(msg.MsgId),
	}

	var span trace.Span
	ctx, span = r.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (r *rocketmqBroker) finishConsumerSpan(span trace.Span) {
	if r.consumerTracer == nil {
		return
	}

	r.consumerTracer.End(context.Background(), span, nil)
}
