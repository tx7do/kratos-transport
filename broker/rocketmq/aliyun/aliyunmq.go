package aliyun

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	aliyun "github.com/aliyunmq/mq-http-go-sdk"
	"github.com/gogap/errors"

	"github.com/tx7do/kratos-transport/broker"
	rocketmqOption "github.com/tx7do/kratos-transport/broker/rocketmq/option"
	"github.com/tx7do/kratos-transport/tracing"
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

func (r *aliyunmqBroker) Name() string {
	return "rocketmq-http"
}

func (r *aliyunmqBroker) Address() string {
	if len(r.nameServers) > 0 {
		return r.nameServers[0]
	} else if r.nameServerUrl != "" {
		return r.nameServerUrl
	}
	return rocketmqOption.DefaultAddr
}

func (r *aliyunmqBroker) Options() broker.Options {
	return r.options
}

func (r *aliyunmqBroker) Init(opts ...broker.Option) error {
	r.options.Apply(opts...)

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

	if len(r.options.Tracings) > 0 {
		r.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "rocketmq-producer", r.options.Tracings...)
		r.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "rocketmq-consumer", r.options.Tracings...)
	}

	return nil
}

func (r *aliyunmqBroker) Connect() error {
	r.RLock()
	if r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	endpoint := r.Address()
	client := aliyun.NewAliyunMQClient(endpoint, r.credentials.AccessKey, r.credentials.AccessSecret, r.credentials.SecurityToken)
	r.client = client

	r.Lock()
	r.connected = true
	r.Unlock()

	return nil
}

func (r *aliyunmqBroker) Disconnect() error {
	r.RLock()
	if !r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	r.Lock()
	defer r.Unlock()

	r.client = nil

	r.connected = false
	return nil
}

func (r *aliyunmqBroker) Request(ctx context.Context, topic string, msg any, opts ...broker.RequestOption) (any, error) {
	return nil, errors.New("not implemented")
}

func (r *aliyunmqBroker) Publish(ctx context.Context, topic string, msg any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(r.options.Codec, msg)
	if err != nil {
		return err
	}

	return r.publish(ctx, topic, buf, opts...)
}

func (r *aliyunmqBroker) publish(ctx context.Context, topic string, msg []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	if r.client == nil {
		return errors.New("client is nil")
	}

	r.Lock()
	p, ok := r.producers[topic]
	if !ok {
		p = r.client.GetProducer(r.instanceName, topic)
		if p == nil {
			r.Unlock()
			return errors.New("create producer failed")
		}

		r.producers[topic] = p
	} else {
	}
	r.Unlock()

	aMsg := aliyun.PublishMessageRequest{
		MessageBody: string(msg),
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

	span := r.startProducerSpan(options.Context, topic, &aMsg)

	ret, err := p.PublishMessage(aMsg)
	if err != nil {
		LogErrorf("send message error: %s\n", err)
	}

	r.finishProducerSpan(span, ret.MessageId, err)

	return nil
}

func (r *aliyunmqBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if r.client == nil {
		return nil, errors.New("client is nil")
	}

	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
		Queue:   r.groupName,
	}
	for _, o := range opts {
		o(&options)
	}

	mqConsumer := r.client.GetConsumer(r.instanceName, topic, options.Queue, "")

	sub := &Subscriber{
		options: options,
		topic:   topic,
		handler: handler,
		binder:  binder,
		reader:  mqConsumer,
		done:    make(chan struct{}),
	}

	go r.doConsume(sub)

	return sub, nil
}

func (r *aliyunmqBroker) doConsume(sub *Subscriber) {
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

						ctx, span := r.startConsumerSpan(sub.options.Context, &msg)

						p := &Publication{
							topic:  msg.Message,
							reader: sub.reader,
							m:      &m,
							rm:     []string{msg.ReceiptHandle},
							ctx:    r.options.Context,
						}

						m.Headers = msg.Properties

						if sub.binder != nil {
							m.Body = sub.binder()

							if err = broker.Unmarshal(r.options.Codec, []byte(msg.MessageBody), &m.Body); err != nil {
								LogError(err)
								r.finishConsumerSpan(span, err)
								continue
							}
						} else {
							m.Body = msg.MessageBody
						}

						if err = sub.handler(ctx, p); err != nil {
							LogErrorf("process message failed: %v", err)
							r.finishConsumerSpan(span, err)
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

						r.finishConsumerSpan(span, err)
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

func (r *aliyunmqBroker) startProducerSpan(ctx context.Context, topicName string, msg *aliyun.PublishMessageRequest) trace.Span {
	if r.producerTracer == nil {
		return nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(rocketmqOption.SPAN_ATTRIBUTE_VALUE_ROCKETMQ_MESSAGING_SYSTEM),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingRocketmqNamespaceKey.String(r.namespace),
		semConv.MessagingRocketmqClientGroupKey.String(r.groupName),
		semConv.MessagingRocketmqClientIDKey.String(r.instanceName),

		semConv.MessagingDestinationKey.String(topicName),
	}

	if len(msg.MessageTag) > 0 {
		attrs = append(attrs, semConv.MessagingRocketmqMessageTagKey.String(msg.MessageTag))
	}
	if len(msg.MessageKey) > 0 {
		attrs = append(attrs, semConv.MessagingRocketmqMessageKeysKey.String(msg.MessageKey))
	}

	var span trace.Span
	ctx, span = r.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (r *aliyunmqBroker) finishProducerSpan(span trace.Span, messageId string, err error) {
	if r.producerTracer == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingMessageIDKey.String(messageId),
	}

	r.producerTracer.End(context.Background(), span, err, attrs...)
}

func (r *aliyunmqBroker) startConsumerSpan(ctx context.Context, msg *aliyun.ConsumeMessageEntry) (context.Context, trace.Span) {
	if r.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

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
	ctx, span = r.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (r *aliyunmqBroker) finishConsumerSpan(span trace.Span, err error) {
	if r.consumerTracer == nil {
		return
	}

	r.consumerTracer.End(context.Background(), span, err)
}
