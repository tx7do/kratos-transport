package aliyun

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	aliyun "github.com/aliyunmq/mq-http-go-sdk"
	"github.com/gogap/errors"

	"github.com/tx7do/kratos-transport/broker"
	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"
	"github.com/tx7do/kratos-transport/tracing"
)

const (
	TracerMessageSystemKey = "rocketmq"
	SpanNameProducer       = "rocketmq-producer"
	SpanNameConsumer       = "rocketmq-consumer"
)

type aliyunmqBroker struct {
	sync.RWMutex

	nameServers   []string
	nameServerUrl string

	credentials rocketmqOption.Credentials

	instanceName string
	groupName    string
	retryCount   int
	namespace    string

	connected bool
	options   broker.Options

	client    aliyun.MQClient
	producers map[string]aliyun.MQProducer

	subscribers *broker.SubscriberSyncMap

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)
	return &aliyunmqBroker{
		producers:   make(map[string]aliyun.MQProducer),
		options:     options,
		retryCount:  2,
		subscribers: broker.NewSubscriberSyncMap(),
	}
}

func (b *aliyunmqBroker) Name() string {
	return "rocketmq-http"
}

func (b *aliyunmqBroker) Address() string {
	if len(b.nameServers) > 0 {
		return b.nameServers[0]
	} else if b.nameServerUrl != "" {
		return b.nameServerUrl
	}
	return rocketmqOption.DefaultAddr
}

func (b *aliyunmqBroker) Options() broker.Options {
	return b.options
}

func (b *aliyunmqBroker) Init(opts ...broker.Option) error {
	b.options.Apply(opts...)

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

	if len(b.options.Tracings) > 0 {
		b.producerTracer = tracing.NewTracer(trace.SpanKindProducer, SpanNameProducer, b.options.Tracings...)
		b.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, SpanNameConsumer, b.options.Tracings...)
	}

	return nil
}

func (b *aliyunmqBroker) Connect() error {
	b.RLock()
	if b.connected {
		b.RUnlock()
		return nil
	}
	b.RUnlock()

	endpoint := b.Address()
	client := aliyun.NewAliyunMQClient(endpoint, b.credentials.AccessKey, b.credentials.AccessSecret, b.credentials.SecurityToken)
	b.client = client

	b.Lock()
	b.connected = true
	b.Unlock()

	return nil
}

func (b *aliyunmqBroker) Disconnect() error {
	b.RLock()
	if !b.connected {
		b.RUnlock()
		return nil
	}
	b.RUnlock()

	b.Lock()
	defer b.Unlock()

	b.client = nil

	b.connected = false
	return nil
}

func (b *aliyunmqBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return nil, errors.New("not implemented")
}

func (b *aliyunmqBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *aliyunmqBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *aliyunmqBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	if b.client == nil {
		return errors.New("client is nil")
	}

	b.Lock()
	p, ok := b.producers[topic]
	if !ok {
		p = b.client.GetProducer(b.instanceName, topic)
		if p == nil {
			b.Unlock()
			return errors.New("create producer failed")
		}

		b.producers[topic] = p
	} else {
	}
	b.Unlock()

	aMsg := aliyun.PublishMessageRequest{
		MessageBody: string(msg.BodyBytes()),
		Properties:  msg.Headers,
	}

	if v, ok := options.Context.Value(rocketmqOption.PropertiesKey{}).(map[string]string); ok {
		aMsg.Properties = v
	}
	if v, ok := options.Context.Value(rocketmqOption.DelayTimeLevelKey{}).(int); ok {
		aMsg.StartDeliverTime = int64(v)
	}
	if v, ok := options.Context.Value(rocketmqOption.TagsKey{}).(string); ok {
		aMsg.MessageTag = v
	}
	if v, ok := options.Context.Value(rocketmqOption.KeysKey{}).([]string); ok {
		var sb strings.Builder
		for _, k := range v {
			sb.WriteString(k)
			sb.WriteString(" ")
		}
		aMsg.MessageKey = sb.String()
	}
	if v, ok := options.Context.Value(rocketmqOption.ShardingKeyKey{}).(string); ok {
		aMsg.ShardingKey = v
	}

	var span trace.Span
	ctx, span = b.startProducerSpan(options.Context, topic, &aMsg)

	ret, err := p.PublishMessage(aMsg)
	if err != nil {
		LogErrorf("send message error: %s\n", err)
	}

	b.finishProducerSpan(ctx, span, ret.MessageId, err)

	return nil
}

func (b *aliyunmqBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if b.client == nil {
		return nil, errors.New("client is nil")
	}

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

	mqConsumer := b.client.GetConsumer(b.instanceName, topic, options.Queue, "")

	sub := &Subscriber{
		options: options,
		topic:   topic,
		handler: handler,
		binder:  binder,
		reader:  mqConsumer,
		done:    make(chan struct{}),
	}

	go b.doConsume(sub)

	return sub, nil
}

func (b *aliyunmqBroker) doConsume(sub *Subscriber) {
	for {
		endChan := make(chan int)
		respChan := make(chan aliyun.ConsumeMessageResponse)
		errChan := make(chan error)
		go func() {
			select {
			case resp := <-respChan:
				{
					var err error
					var m broker.Message
					for _, msg := range resp.Messages {

						ctx, span := b.startConsumerSpan(sub.options.Context, &msg)

						p := &Publication{
							topic:  msg.Message,
							reader: sub.reader,
							m:      &m,
							rm:     []string{msg.ReceiptHandle},
							ctx:    b.options.Context,
						}

						m.Headers = msg.Properties

						if sub.binder != nil {
							m.Body = sub.binder()

							if err = broker.Unmarshal(b.options.Codec, []byte(msg.MessageBody), &m.Body); err != nil {
								LogError(err)
								b.finishConsumerSpan(ctx, span, err)
								continue
							}
						} else {
							m.Body = msg.MessageBody
						}

						if err = sub.handler(ctx, p); err != nil {
							LogErrorf("process message failed: %v", err)
							b.finishConsumerSpan(ctx, span, err)
							continue
						}

						if sub.options.AutoAck {
							if err = p.Ack(); err != nil {
								// 某些消息的句柄可能超时，会导致消息消费状态确认不成功。
								if errAckItems, ok := err.(errors.ErrCode).Context()["Detail"].([]aliyun.ErrAckItem); ok {
									for _, errAckItem := range errAckItems {
										LogErrorf("ErrorHandle:%s, ErrorCode:%s, ErrorMsg:%s\n",
											errAckItem.ErrorHandle, errAckItem.ErrorCode, errAckItem.ErrorMsg)
									}
								} else {
									LogError("ack err =", err)
								}
								time.Sleep(time.Duration(3) * time.Second)
							}
						}

						b.finishConsumerSpan(ctx, span, err)
					}

					endChan <- 1
				}
			case err := <-errChan:
				{
					// Topic中没有消息可消费。
					if strings.Contains(err.(errors.ErrCode).Error(), "MessageNotExist") {
						//LogDebug("No new message, continue!")
					} else {
						LogError(err)
						time.Sleep(time.Duration(3) * time.Second)
					}
					endChan <- 1
				}
			case <-time.After(35 * time.Second):
				{
					//LogDebug("Timeout of consumer message ??")
					endChan <- 1
				}

			case sub.done <- struct{}{}:
				return
			}
		}()

		// 长轮询消费消息，网络超时时间默认为35s。
		// 长轮询表示如果Topic没有消息，则客户端请求会在服务端挂起3s，3s内如果有消息可以消费则立即返回响应。
		sub.reader.ConsumeMessage(respChan, errChan,
			3, // 一次最多消费3条（最多可设置为16条）。
			3, // 长轮询时间3s（最多可设置为30s）。
		)
		<-endChan
	}
}

func (b *aliyunmqBroker) startProducerSpan(
	ctx context.Context,
	topicName string,
	msg *aliyun.PublishMessageRequest,
) (context.Context, trace.Span) {
	if b.producerTracer == nil {
		return ctx, nil
	}

	if msg == nil {
		return ctx, nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rocketmqOption.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingRocketmqNamespaceKey.String(b.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(b.groupName),
		semConv.MessagingRocketmqClientIDKey.String(b.instanceName),

		semConv.MessagingDestinationKey.String(topicName),
	}

	if len(msg.MessageTag) > 0 {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTagKey.String(msg.MessageTag))
	}
	if len(msg.MessageKey) > 0 {
		attrs = append(attrs, semConv.MessagingRocketmqMessageKeysKey.String(msg.MessageKey))
	}

	var span trace.Span
	ctx, span = b.producerTracer.Start(ctx, carrier, attrs...)

	if span != nil {
		otel.GetTextMapPropagator().Inject(ctx, carrier)
	}

	return ctx, span
}

func (b *aliyunmqBroker) finishProducerSpan(ctx context.Context, span trace.Span, messageId string, err error) {
	if b.producerTracer == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingMessageIDKey.String(messageId),
	}

	b.producerTracer.End(ctx, span, err, attrs...)
}

func (b *aliyunmqBroker) startConsumerSpan(ctx context.Context, msg *aliyun.ConsumeMessageEntry) (context.Context, trace.Span) {
	if b.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rocketmqOption.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Message),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(msg.MessageId),
	}

	if len(msg.MessageTag) > 0 {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTagKey.String(msg.MessageTag))
	}
	if len(msg.MessageKey) > 0 {
		attrs = append(attrs, semConv.MessagingRocketmqMessageKeysKey.String(msg.MessageKey))
	}

	var span trace.Span
	ctx, span = b.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (b *aliyunmqBroker) finishConsumerSpan(ctx context.Context, span trace.Span, err error) {
	if b.consumerTracer == nil {
		return
	}

	b.consumerTracer.End(ctx, span, err)
}
