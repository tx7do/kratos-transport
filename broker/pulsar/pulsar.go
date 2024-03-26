package pulsar

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/tracing"

	"go.opentelemetry.io/otel/attribute"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultAddr = "pulsar://127.0.0.1:6650"
)

type pulsarBroker struct {
	sync.RWMutex

	connected bool
	options   broker.Options

	client    pulsar.Client
	producers map[string]pulsar.Producer

	subscribers *broker.SubscriberSyncMap

	producerTracer *tracing.Tracer
	consumerTracer *tracing.Tracer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &pulsarBroker{
		options:     options,
		producers:   make(map[string]pulsar.Producer),
		subscribers: broker.NewSubscriberSyncMap(),
	}

	return b
}

func (pb *pulsarBroker) Name() string {
	return "pulsar"
}

func (pb *pulsarBroker) Address() string {
	if len(pb.options.Addrs) > 0 {
		return pb.options.Addrs[0]
	}
	return defaultAddr
}

func (pb *pulsarBroker) Options() broker.Options {
	return pb.options
}

func (pb *pulsarBroker) Init(opts ...broker.Option) error {
	pb.options.Apply(opts...)

	pulsarOptions := pulsar.ClientOptions{
		URL:               defaultAddr,
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
	}

	if v, ok := pb.options.Context.Value(connectionTimeoutKey{}).(time.Duration); ok {
		pulsarOptions.OperationTimeout = v
	}
	if v, ok := pb.options.Context.Value(operationTimeoutKey{}).(time.Duration); ok {
		pulsarOptions.ConnectionTimeout = v
	}
	if v, ok := pb.options.Context.Value(listenerNameKey{}).(string); ok {
		pulsarOptions.ListenerName = v
	}
	if v, ok := pb.options.Context.Value(maxConnectionsPerBrokerKey{}).(int); ok {
		pulsarOptions.MaxConnectionsPerBroker = v
	}
	if v, ok := pb.options.Context.Value(customMetricsLabelsKey{}).(map[string]string); ok {
		pulsarOptions.CustomMetricsLabels = v
	}

	var enableTLS = false
	if v, ok := pb.options.Context.Value(tlsKey{}).(tlsConfig); ok {
		pulsarOptions.TLSTrustCertsFilePath = v.CaCertsPath
		if v.ClientCertPath != "" && v.ClientKeyPath != "" {
			pulsarOptions.Authentication = pulsar.NewAuthenticationTLS(v.ClientCertPath, v.ClientKeyPath)
		}
		pulsarOptions.TLSAllowInsecureConnection = v.AllowInsecureConnection
		pulsarOptions.TLSValidateHostname = v.ValidateHostname

		enableTLS = true
	}

	var cAddrs []string
	for _, addr := range pb.options.Addrs {
		if len(addr) == 0 {
			continue
		}
		addr = refitUrl(addr, enableTLS)
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{defaultAddr}
	}
	pb.options.Addrs = cAddrs
	pulsarOptions.URL = cAddrs[0]

	var err error
	pb.client, err = pulsar.NewClient(pulsarOptions)
	if err != nil {
		log.Errorf("Could not instantiate Pulsar client: %v", err)
		return err
	}

	if len(pb.options.Tracings) > 0 {
		pb.producerTracer = tracing.NewTracer(trace.SpanKindProducer, "pulsar-producer", pb.options.Tracings...)
		pb.consumerTracer = tracing.NewTracer(trace.SpanKindConsumer, "pulsar-consumer", pb.options.Tracings...)
	}

	return nil
}

func (pb *pulsarBroker) Connect() error {
	pb.RLock()
	if pb.connected {
		pb.RUnlock()
		return nil
	}
	pb.RUnlock()

	pb.Lock()
	pb.connected = true
	pb.Unlock()

	return nil
}

func (pb *pulsarBroker) Disconnect() error {
	pb.RLock()
	if !pb.connected {
		pb.RUnlock()
		return nil
	}
	pb.RUnlock()

	pb.Lock()
	defer pb.Unlock()

	for _, p := range pb.producers {
		p.Close()
	}

	pb.subscribers.Clear()

	pb.client.Close()

	pb.connected = false
	return nil
}

func (pb *pulsarBroker) Publish(ctx context.Context, topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(pb.options.Codec, msg)
	if err != nil {
		return err
	}

	return pb.publish(ctx, topic, buf, opts...)
}

func (pb *pulsarBroker) publish(ctx context.Context, topic string, msg []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	pulsarOptions := pulsar.ProducerOptions{
		Topic:           topic,
		DisableBatching: false,
	}

	if v, ok := options.Context.Value(producerNameKey{}).(string); ok {
		pulsarOptions.Name = v
	}
	if v, ok := options.Context.Value(producerPropertiesKey{}).(map[string]string); ok {
		pulsarOptions.Properties = v
	}
	if v, ok := options.Context.Value(sendTimeoutKey{}).(time.Duration); ok {
		pulsarOptions.SendTimeout = v
	}
	if v, ok := options.Context.Value(disableBatchingKey{}).(bool); ok {
		pulsarOptions.DisableBatching = v
	}
	if v, ok := options.Context.Value(batchingMaxPublishDelayKey{}).(time.Duration); ok {
		pulsarOptions.BatchingMaxPublishDelay = v
	}
	if v, ok := options.Context.Value(batchingMaxMessagesKey{}).(uint); ok {
		pulsarOptions.BatchingMaxMessages = v
	}
	if v, ok := options.Context.Value(batchingMaxSizeKey{}).(uint); ok {
		pulsarOptions.BatchingMaxSize = v
	}

	var cached bool
	pb.Lock()
	producer, ok := pb.producers[topic]
	if !ok {
		var err error
		producer, err = pb.client.CreateProducer(pulsarOptions)
		if err != nil {
			pb.Unlock()
			return err
		}

		pb.producers[topic] = producer
	} else {
		cached = true
	}
	pb.Unlock()

	pulsarMsg := pulsar.ProducerMessage{Payload: msg}

	if headers, ok := options.Context.Value(messageHeadersKey{}).(map[string]string); ok {
		pulsarMsg.Properties = headers
	}
	if v, ok := options.Context.Value(messageDeliverAfterKey{}).(time.Duration); ok {
		pulsarMsg.DeliverAfter = v
	}
	if v, ok := options.Context.Value(messageDeliverAtKey{}).(time.Time); ok {
		pulsarMsg.DeliverAt = v
	}
	if v, ok := options.Context.Value(messageSequenceIdKey{}).(*int64); ok {
		pulsarMsg.SequenceID = v
	}
	if v, ok := options.Context.Value(messageKeyKey{}).(string); ok {
		pulsarMsg.Key = v
	}
	if v, ok := options.Context.Value(messageValueKey{}).(interface{}); ok {
		pulsarMsg.Value = v
	}
	if v, ok := options.Context.Value(messageOrderingKeyKey{}).(string); ok {
		pulsarMsg.OrderingKey = v
	}
	if v, ok := options.Context.Value(messageEventTimeKey{}).(time.Time); ok {
		pulsarMsg.EventTime = v
	}
	if v, ok := options.Context.Value(messageDisableReplication{}).(bool); ok {
		pulsarMsg.DisableReplication = v
	}

	span := pb.startProducerSpan(options.Context, topic, &pulsarMsg)

	var err error
	var messageId pulsar.MessageID
	messageId, err = producer.Send(pb.options.Context, &pulsarMsg)
	if err != nil {
		log.Errorf("[pulsar]: send message error: %s\n", err)
		switch cached {
		case false:
		case true:
			pb.Lock()
			producer.Close()
			delete(pb.producers, topic)
			pb.Unlock()

			producer, err = pb.client.CreateProducer(pulsarOptions)
			if err != nil {
				pb.Unlock()
				break
			}
			if _, err = producer.Send(pb.options.Context, &pulsarMsg); err == nil {
				pb.Lock()
				pb.producers[topic] = producer
				pb.Unlock()
			}
		}
	}

	var msgId string
	if messageId != nil {
		msgId = strconv.FormatInt(messageId.EntryID(), 10)
	}

	pb.finishProducerSpan(span, msgId, err)

	return err
}

func (pb *pulsarBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
		Queue:   uuid.New().String(),
	}
	for _, o := range opts {
		o(&options)
	}

	pulsarOptions := pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: "my-subscription",
		Type:             pulsar.Shared,
	}

	channel := make(chan pulsar.ConsumerMessage, 100)
	pulsarOptions.MessageChannel = channel

	if v, ok := options.Context.Value(subscriptionNameKey{}).(string); ok {
		pulsarOptions.SubscriptionName = v
	}
	if v, ok := options.Context.Value(consumerPropertiesKey{}).(map[string]string); ok {
		pulsarOptions.Properties = v
	}
	if v, ok := options.Context.Value(subscriptionPropertiesKey{}).(map[string]string); ok {
		pulsarOptions.SubscriptionProperties = v
	}
	if v, ok := options.Context.Value(topicsPatternKey{}).(string); ok {
		pulsarOptions.TopicsPattern = v
	}
	if v, ok := options.Context.Value(autoDiscoveryPeriodKey{}).(time.Duration); ok {
		pulsarOptions.AutoDiscoveryPeriod = v
	}
	if v, ok := options.Context.Value(nackRedeliveryDelayKey{}).(time.Duration); ok {
		pulsarOptions.NackRedeliveryDelay = v
	}
	if v, ok := options.Context.Value(subscriptionRetryEnableKey{}).(bool); ok {
		pulsarOptions.RetryEnable = v
	}
	if v, ok := options.Context.Value(receiverQueueSizeKey{}).(int); ok {
		pulsarOptions.ReceiverQueueSize = v
	}

	c, _ := pb.client.Subscribe(pulsarOptions)
	if c == nil {
		return nil, errors.New("create consumer error")
	}

	sub := &subscriber{
		r:       pb,
		options: options,
		topic:   topic,
		handler: handler,
		reader:  c,
		channel: channel,
	}

	go func() {
		var err error
		var m broker.Message
		for cm := range channel {
			p := &publication{topic: cm.Topic(), reader: sub.reader, msg: &m, pulsarMsg: &cm.Message, ctx: options.Context}
			m.Headers = cm.Properties()

			ctx, span := pb.startConsumerSpan(sub.options.Context, &cm)

			if binder != nil {
				m.Body = binder()
			} else {
				m.Body = cm.Payload()
			}

			if err = broker.Unmarshal(pb.options.Codec, cm.Payload(), &m.Body); err != nil {
				p.err = err
				log.Errorf("[pulsar]: unmarshal message failed: %v", err)
				pb.finishConsumerSpan(span, err)
				continue
			}

			if err = sub.handler(ctx, p); err != nil {
				p.err = err
				log.Errorf("[pulsar]: handle message failed: %v", err)
				pb.finishConsumerSpan(span, err)
				continue
			}

			if sub.options.AutoAck {
				if err = p.Ack(); err != nil {
					p.err = err
					log.Errorf("[pulsar]: unable to commit msg: %v", err)
				}
			}

			pb.finishConsumerSpan(span, err)
		}
	}()

	pb.subscribers.Add(topic, sub)

	return sub, nil
}

func (pb *pulsarBroker) startProducerSpan(ctx context.Context, topic string, msg *pulsar.ProducerMessage) trace.Span {
	if pb.producerTracer == nil {
		return nil
	}

	carrier := NewProducerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("pulsar"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(topic),
	}

	var span trace.Span
	ctx, span = pb.producerTracer.Start(ctx, carrier, attrs...)

	return span
}

func (pb *pulsarBroker) finishProducerSpan(span trace.Span, messageId string, err error) {
	if pb.producerTracer == nil {
		return
	}

	attrs := []attribute.KeyValue{
		semConv.MessagingMessageIDKey.String(messageId),
	}

	pb.producerTracer.End(context.Background(), span, err, attrs...)
}

func (pb *pulsarBroker) startConsumerSpan(ctx context.Context, msg *pulsar.ConsumerMessage) (context.Context, trace.Span) {
	if pb.consumerTracer == nil {
		return ctx, nil
	}

	carrier := NewConsumerMessageCarrier(msg)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String(pb.Name()),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic()),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.Int64(msg.ID().EntryID()),
	}

	var span trace.Span
	ctx, span = pb.consumerTracer.Start(ctx, carrier, attrs...)

	return ctx, span
}

func (pb *pulsarBroker) finishConsumerSpan(span trace.Span, err error) {
	if pb.consumerTracer == nil {
		return
	}

	pb.consumerTracer.End(context.Background(), span, err)
}
