package rocketmq

import (
	"context"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	aliyun "github.com/aliyunmq/mq-http-go-sdk"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/gogap/errors"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"
)

type aliyunBroker struct {
	nameServers   []string
	nameServerUrl string

	accessKey     string
	secretKey     string
	securityToken string

	instanceName string
	groupName    string
	retryCount   int
	namespace    string

	connected bool
	sync.RWMutex
	opts broker.Options

	client    aliyun.MQClient
	producers map[string]aliyun.MQProducer

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func newAliyunHttpBroker(options broker.Options) broker.Broker {
	return &aliyunBroker{
		producers:  make(map[string]aliyun.MQProducer),
		opts:       options,
		retryCount: 2,
	}
}

func (r *aliyunBroker) Name() string {
	return "rocketmq_http"
}

func (r *aliyunBroker) Address() string {
	if len(r.nameServers) > 0 {
		return r.nameServers[0]
	} else if r.nameServerUrl != "" {
		return r.nameServerUrl
	}
	return defaultAddr
}

func (r *aliyunBroker) Options() broker.Options {
	return r.opts
}

func (r *aliyunBroker) Init(opts ...broker.Option) error {
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
	if v, ok := r.opts.Context.Value(securityTokenKey{}).(string); ok {
		r.securityToken = v
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

	if len(r.opts.Tracings) > 0 {
		r.producerTracer = tracing.NewTracer(trace.SpanKindProducer, tracing.WithSpanName("rocketmq-producer"))
		r.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, tracing.WithSpanName("rocketmq-consumer"))
	}

	return nil
}

func (r *aliyunBroker) Connect() error {
	r.RLock()
	if r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	endpoint := r.Address()
	client := aliyun.NewAliyunMQClient(endpoint, r.accessKey, r.secretKey, r.securityToken)
	r.client = client

	r.Lock()
	r.connected = true
	r.Unlock()

	return nil
}

func (r *aliyunBroker) Disconnect() error {
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

func (r *aliyunBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(r.opts.Codec, msg)
	if err != nil {
		return err
	}

	return r.publish(topic, buf, opts...)
}

func (r *aliyunBroker) publish(topic string, msg []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: context.Background(),
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

	if v, ok := options.Context.Value(propertiesKey{}).(map[string]string); ok {
		aMsg.Properties = v
	}
	if v, ok := options.Context.Value(delayTimeLevelKey{}).(int); ok {
		aMsg.StartDeliverTime = int64(v)
	}
	if v, ok := options.Context.Value(tagsKey{}).(string); ok {
		aMsg.MessageTag = v
	}
	if v, ok := options.Context.Value(keysKey{}).([]string); ok {
		var sb strings.Builder
		for _, k := range v {
			sb.WriteString(k)
			sb.WriteString(" ")
		}
		aMsg.MessageKey = sb.String()
	}
	if v, ok := options.Context.Value(shardingKeyKey{}).(string); ok {
		aMsg.ShardingKey = v
	}

	span := r.startProducerSpan(options.Context, topic, &aMsg)

	ret, err := p.PublishMessage(aMsg)
	if err != nil {
		log.Errorf("[rocketmq]: send message error: %s\n", err)
	}

	r.finishProducerSpan(span, ret.MessageId, err)

	return nil
}

func (r *aliyunBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
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

	sub := &aliyunSubscriber{
		opts:    options,
		topic:   topic,
		handler: handler,
		binder:  binder,
		reader:  mqConsumer,
		done:    make(chan struct{}),
	}

	go r.doConsume(sub)

	return sub, nil
}

func (r *aliyunBroker) doConsume(sub *aliyunSubscriber) {
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

						ctx, span := r.startConsumerSpan(sub.opts.Context, &msg)

						p := &aliyunPublication{
							topic:  msg.Message,
							reader: sub.reader,
							m:      &m,
							rm:     []string{msg.ReceiptHandle},
							ctx:    r.opts.Context,
						}

						m.Headers = msg.Properties

						if sub.binder != nil {
							m.Body = sub.binder()
						}

						if err := broker.Unmarshal(r.opts.Codec, []byte(msg.MessageBody), m.Body); err != nil {
							p.err = err
							log.Error("[rocketmq]: ", err)
						}

						err = sub.handler(ctx, p)
						if err != nil {
							log.Errorf("[rocketmq]: process message failed: %v", err)
						}

						if sub.opts.AutoAck {
							if err = p.Ack(); err != nil {
								// 某些消息的句柄可能超时，会导致消息消费状态确认不成功。
								if errAckItems, ok := err.(errors.ErrCode).Context()["Detail"].([]aliyun.ErrAckItem); ok {
									for _, errAckItem := range errAckItems {
										log.Errorf("[rocketmq]: ErrorHandle:%s, ErrorCode:%s, ErrorMsg:%s\n",
											errAckItem.ErrorHandle, errAckItem.ErrorCode, errAckItem.ErrorMsg)
									}
								} else {
									log.Error("[rocketmq]: ack err =", err)
								}
								time.Sleep(time.Duration(3) * time.Second)
							}
						}

						r.finishConsumerSpan(span)
					}

					endChan <- 1
				}
			case err := <-errChan:
				{
					// Topic中没有消息可消费。
					if strings.Contains(err.(errors.ErrCode).Error(), "MessageNotExist") {
						//log.Debug("[rocketmq] No new message, continue!")
					} else {
						log.Error("[rocketmq]: ", err)
						time.Sleep(time.Duration(3) * time.Second)
					}
					endChan <- 1
				}
			case <-time.After(35 * time.Second):
				{
					//log.Debug("[rocketmq] Timeout of consumer message ??")
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

func (r *aliyunBroker) startProducerSpan(ctx context.Context, topicName string, msg *aliyun.PublishMessageRequest) trace.Span {
	if r.producerTracer == nil {
		return nil
	}

	carrier := NewAliyunProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("rocketmq"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(topicName),
	}

	var span trace.Span
	ctx, span = r.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (r *aliyunBroker) finishProducerSpan(span trace.Span, messageId string, err error) {
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

func (r *aliyunBroker) startConsumerSpan(ctx context.Context, msg *aliyun.ConsumeMessageEntry) (context.Context, trace.Span) {
	if r.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewAliyunConsumerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("rocketmq"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Message),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(msg.MessageId),
	}

	var span trace.Span
	ctx, span = r.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (r *aliyunBroker) finishConsumerSpan(span trace.Span) {
	if r.consumerTracer == nil {
		return
	}

	r.consumerTracer.End(context.Background(), span, nil)
}
