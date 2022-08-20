package pulsar

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
	"strconv"
	"sync"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultAddr = "pulsar://127.0.0.1:6650"
)

type pulsarBroker struct {
	sync.RWMutex

	connected bool
	opts      broker.Options

	client    pulsar.Client
	producers map[string]pulsar.Producer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	r := &pulsarBroker{
		producers: make(map[string]pulsar.Producer),
		opts:      options,
	}

	return r
}

func (pb *pulsarBroker) Name() string {
	return "pulsar"
}

func (pb *pulsarBroker) Address() string {
	if len(pb.opts.Addrs) > 0 {
		return pb.opts.Addrs[0]
	}
	return defaultAddr
}

func (pb *pulsarBroker) Options() broker.Options {
	return pb.opts
}

func (pb *pulsarBroker) Init(opts ...broker.Option) error {
	pb.opts.Apply(opts...)

	pulsarOptions := pulsar.ClientOptions{
		URL:               defaultAddr,
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
	}

	if v, ok := pb.opts.Context.Value(connectionTimeoutKey{}).(time.Duration); ok {
		pulsarOptions.OperationTimeout = v
	}
	if v, ok := pb.opts.Context.Value(operationTimeoutKey{}).(time.Duration); ok {
		pulsarOptions.ConnectionTimeout = v
	}
	if v, ok := pb.opts.Context.Value(listenerNameKey{}).(string); ok {
		pulsarOptions.ListenerName = v
	}
	if v, ok := pb.opts.Context.Value(maxConnectionsPerBrokerKey{}).(int); ok {
		pulsarOptions.MaxConnectionsPerBroker = v
	}
	if v, ok := pb.opts.Context.Value(customMetricsLabelsKey{}).(map[string]string); ok {
		pulsarOptions.CustomMetricsLabels = v
	}

	var enableTLS = false
	if v, ok := pb.opts.Context.Value(tlsKey{}).(tlsConfig); ok {
		pulsarOptions.TLSTrustCertsFilePath = v.CaCertsPath
		if v.ClientCertPath != "" && v.ClientKeyPath != "" {
			pulsarOptions.Authentication = pulsar.NewAuthenticationTLS(v.ClientCertPath, v.ClientKeyPath)
		}
		pulsarOptions.TLSAllowInsecureConnection = v.AllowInsecureConnection
		pulsarOptions.TLSValidateHostname = v.ValidateHostname

		enableTLS = true
	}

	var cAddrs []string
	for _, addr := range pb.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		addr = refitUrl(addr, enableTLS)
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{defaultAddr}
	}
	pb.opts.Addrs = cAddrs
	pulsarOptions.URL = cAddrs[0]

	var err error
	pb.client, err = pulsar.NewClient(pulsarOptions)
	if err != nil {
		log.Errorf("Could not instantiate Pulsar client: %v", err)
		return err
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

	pb.client.Close()

	pb.connected = false
	return nil
}

func (pb *pulsarBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(pb.opts.Codec, msg)
	if err != nil {
		return err
	}

	return pb.publish(topic, buf, opts...)
}

func (pb *pulsarBroker) publish(topic string, msg []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: context.Background(),
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

	span := pb.startProducerSpan(topic, &pulsarMsg)

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

	var err error
	var messageId pulsar.MessageID
	messageId, err = producer.Send(pb.opts.Context, &pulsarMsg)
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
			if _, err = producer.Send(pb.opts.Context, &pulsarMsg); err == nil {
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
		opts:    options,
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

			span := pb.startConsumerSpan(&cm)

			if binder != nil {
				m.Body = binder()
			}

			if err := broker.Unmarshal(pb.opts.Codec, cm.Payload(), m.Body); err != nil {
				p.err = err
				log.Error(err)
				continue
			}

			err = sub.handler(sub.opts.Context, p)
			if err != nil {
				log.Errorf("[pulsar]: process message failed: %v", err)
			}
			if sub.opts.AutoAck {
				if err = p.Ack(); err != nil {
					log.Errorf("[pulsar]: unable to commit msg: %v", err)
				}
			}

			pb.finishConsumerSpan(span)
		}
	}()

	return sub, nil
}

func (pb *pulsarBroker) startProducerSpan(topic string, msg *pulsar.ProducerMessage) trace.Span {
	if pb.opts.Tracer.Tracer == nil {
		return nil
	}

	carrier := NewProducerMessageCarrier(msg)
	ctx := pb.opts.Tracer.Propagators.Extract(pb.opts.Context, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("pulsar"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(topic),
	}
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindProducer),
	}
	ctx, span := pb.opts.Tracer.Tracer.Start(ctx, "pulsar.produce", opts...)

	pb.opts.Tracer.Propagators.Inject(ctx, carrier)

	return span
}

func (pb *pulsarBroker) finishProducerSpan(span trace.Span, messageId string, err error) {
	if span == nil {
		return
	}

	span.SetAttributes(
		semConv.MessagingMessageIDKey.String(messageId),
	)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}

func (pb *pulsarBroker) startConsumerSpan(msg *pulsar.ConsumerMessage) trace.Span {
	if pb.opts.Tracer.Tracer == nil {
		return nil
	}

	carrier := NewConsumerMessageCarrier(msg)
	ctx := pb.opts.Tracer.Propagators.Extract(pb.opts.Context, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("pulsar"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic()),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.Int64(msg.ID().EntryID()),
	}
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
	}
	newCtx, span := pb.opts.Tracer.Tracer.Start(ctx, "pulsar.consume", opts...)

	pb.opts.Tracer.Propagators.Inject(newCtx, carrier)

	return span
}

func (pb *pulsarBroker) finishConsumerSpan(span trace.Span) {
	if span == nil {
		return
	}

	span.End()
}
