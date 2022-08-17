package kafka

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/google/uuid"
	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultAddr = "127.0.0.1:9092"
)

type kafkaBroker struct {
	sync.RWMutex

	readerConfig kafkaGo.ReaderConfig
	writers      map[string]*kafkaGo.Writer

	connected    bool
	opts         broker.Options
	retriesCount int
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &kafkaBroker{
		readerConfig: kafkaGo.ReaderConfig{WatchPartitionChanges: true},
		writers:      make(map[string]*kafkaGo.Writer),
		opts:         options,
		retriesCount: 1,
	}

	return b
}

func (b *kafkaBroker) Name() string {
	return "kafka"
}

func (b *kafkaBroker) Address() string {
	if len(b.opts.Addrs) > 0 {
		return b.opts.Addrs[0]
	}
	return defaultAddr
}

func (b *kafkaBroker) Options() broker.Options {
	return b.opts
}

func (b *kafkaBroker) Init(opts ...broker.Option) error {
	b.opts.Apply(opts...)

	var addrs []string
	for _, addr := range b.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		addrs = []string{defaultAddr}
	}
	b.opts.Addrs = addrs
	b.readerConfig.Brokers = addrs

	if value, ok := b.opts.Context.Value(queueCapacityKey{}).(int); ok {
		b.readerConfig.QueueCapacity = value
	}
	if value, ok := b.opts.Context.Value(minBytesKey{}).(int); ok {
		b.readerConfig.MinBytes = value
	}
	if value, ok := b.opts.Context.Value(maxBytesKey{}).(int); ok {
		b.readerConfig.MaxBytes = value
	}
	if value, ok := b.opts.Context.Value(maxWaitKey{}).(time.Duration); ok {
		b.readerConfig.MaxWait = value
	}
	if value, ok := b.opts.Context.Value(readLagIntervalKey{}).(time.Duration); ok {
		b.readerConfig.ReadLagInterval = value
	}
	if value, ok := b.opts.Context.Value(heartbeatIntervalKey{}).(time.Duration); ok {
		b.readerConfig.HeartbeatInterval = value
	}
	if value, ok := b.opts.Context.Value(commitIntervalKey{}).(time.Duration); ok {
		b.readerConfig.CommitInterval = value
	}
	if value, ok := b.opts.Context.Value(partitionWatchIntervalKey{}).(time.Duration); ok {
		b.readerConfig.PartitionWatchInterval = value
	}
	if value, ok := b.opts.Context.Value(watchPartitionChangesKey{}).(bool); ok {
		b.readerConfig.WatchPartitionChanges = value
	}
	if value, ok := b.opts.Context.Value(sessionTimeoutKey{}).(time.Duration); ok {
		b.readerConfig.SessionTimeout = value
	}
	if value, ok := b.opts.Context.Value(rebalanceTimeoutKey{}).(time.Duration); ok {
		b.readerConfig.RebalanceTimeout = value
	}
	if value, ok := b.opts.Context.Value(retentionTimeKey{}).(time.Duration); ok {
		b.readerConfig.RetentionTime = value
	}
	if value, ok := b.opts.Context.Value(startOffsetKey{}).(int64); ok {
		b.readerConfig.StartOffset = value
	}
	if value, ok := b.opts.Context.Value(maxAttemptsKey{}).(int); ok {
		b.readerConfig.MaxAttempts = value
	}

	if cnt, ok := b.opts.Context.Value(retriesCountKey{}).(int); ok {
		b.retriesCount = cnt
	}

	return nil
}

func (b *kafkaBroker) Connect() error {
	b.RLock()
	if b.connected {
		b.RUnlock()
		return nil
	}
	b.RUnlock()

	kAddrs := make([]string, 0, len(b.opts.Addrs))
	for _, addr := range b.opts.Addrs {
		conn, err := kafkaGo.DialContext(b.opts.Context, "tcp", addr)
		if err != nil {
			continue
		}
		if _, err = conn.Brokers(); err != nil {
			_ = conn.Close()
			continue
		}
		kAddrs = append(kAddrs, addr)
		_ = conn.Close()
	}

	if len(kAddrs) == 0 {
		return errors.New("no available commons")
	}

	b.Lock()
	b.opts.Addrs = kAddrs
	b.readerConfig.Brokers = kAddrs
	b.connected = true
	b.Unlock()

	return nil
}

func (b *kafkaBroker) Disconnect() error {
	b.RLock()
	if !b.connected {
		b.RUnlock()
		return nil
	}
	b.RUnlock()

	b.Lock()
	defer b.Unlock()
	for _, writer := range b.writers {
		if err := writer.Close(); err != nil {
			return err
		}
	}

	b.connected = false
	return nil
}

func (b *kafkaBroker) createProducer(opts ...broker.PublishOption) *kafkaGo.Writer {
	options := broker.PublishOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	writer := &kafkaGo.Writer{
		Addr:     kafkaGo.TCP(b.opts.Addrs...),
		Balancer: &kafkaGo.LeastBytes{},
	}

	if value, ok := options.Context.Value(balancerKey{}).(string); ok {
		switch value {
		default:
		case BalancerLeastBytes:
			writer.Balancer = &kafkaGo.LeastBytes{}
			break
		case BalancerRoundRobin:
			writer.Balancer = &kafkaGo.RoundRobin{}
			break
		case BalancerHash:
			writer.Balancer = &kafkaGo.Hash{}
			break
		case BalancerReferenceHash:
			writer.Balancer = &kafkaGo.ReferenceHash{}
			break
		case BalancerCRC32Balancer:
			writer.Balancer = &kafkaGo.CRC32Balancer{}
			break
		case BalancerMurmur2Balancer:
			writer.Balancer = &kafkaGo.Murmur2Balancer{}
			break
		}
	}

	if value, ok := options.Context.Value(batchSizeKey{}).(int); ok {
		writer.BatchSize = value
	}

	if value, ok := options.Context.Value(batchTimeoutKey{}).(time.Duration); ok {
		writer.BatchTimeout = value
	}

	if value, ok := options.Context.Value(batchBytesKey{}).(int64); ok {
		writer.BatchBytes = value
	}

	if value, ok := options.Context.Value(asyncKey{}).(bool); ok {
		writer.Async = value
	}

	if value, ok := options.Context.Value(maxAttemptsKey{}).(int); ok {
		writer.MaxAttempts = value
	}

	if value, ok := options.Context.Value(readTimeoutKey{}).(time.Duration); ok {
		writer.ReadTimeout = value
	}

	if value, ok := options.Context.Value(writeTimeoutKey{}).(time.Duration); ok {
		writer.WriteTimeout = value
	}

	if value, ok := options.Context.Value(allowAutoTopicCreationKey{}).(bool); ok {
		writer.AllowAutoTopicCreation = value
	}

	return writer
}

func (b *kafkaBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.opts.Codec, msg)
	if err != nil {
		return err
	}

	return b.publish(topic, buf, opts...)
}

func (b *kafkaBroker) publish(topic string, buf []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	kMsg := kafkaGo.Message{Topic: topic, Value: buf}

	if headers, ok := options.Context.Value(messageHeadersKey{}).(map[string]interface{}); ok {
		for k, v := range headers {
			header := kafkaGo.Header{Key: k}
			switch t := v.(type) {
			case string:
				header.Value = []byte(t)
			case []byte:
				header.Value = t
			default:
				var buf bytes.Buffer
				enc := gob.NewEncoder(&buf)
				if err := enc.Encode(v); err != nil {
					continue
				}
				header.Value = buf.Bytes()
			}
			kMsg.Headers = append(kMsg.Headers, header)
		}
	}

	if value, ok := options.Context.Value(messageKeyKey{}).([]byte); ok {
		kMsg.Key = value
	}

	if value, ok := options.Context.Value(messageOffsetKey{}).(int64); ok {
		kMsg.Offset = value
	}

	var cached bool
	b.Lock()
	writer, ok := b.writers[topic]
	if !ok {
		writer = b.createProducer(opts...)
		b.writers[topic] = writer
	} else {
		cached = true
	}
	b.Unlock()

	span := b.startProducerSpan(&kMsg)

	err := writer.WriteMessages(b.opts.Context, kMsg)
	if err != nil {
		switch cached {
		case false:
			if kerr, ok := err.(kafkaGo.Error); ok {
				if kerr.Temporary() && !kerr.Timeout() {
					time.Sleep(200 * time.Millisecond)
					err = writer.WriteMessages(b.opts.Context, kMsg)
				}
			}
		case true:
			b.Lock()
			if err = writer.Close(); err != nil {
				b.Unlock()
				break
			}
			delete(b.writers, topic)
			b.Unlock()

			writer := b.createProducer(opts...)
			for i := 0; i < b.retriesCount; i++ {
				if err = writer.WriteMessages(b.opts.Context, kMsg); err == nil {
					b.Lock()
					b.writers[topic] = writer
					b.Unlock()
					break
				}
			}
		}
	}

	b.finishProducerSpan(span, int32(kMsg.Partition), kMsg.Offset, err)

	return err
}

func (b *kafkaBroker) startProducerSpan(msg *kafkaGo.Message) trace.Span {
	if b.opts.Tracer.Tracer == nil {
		return nil
	}

	carrier := NewMessageCarrier(msg)
	ctx := b.opts.Tracer.Propagators.Extract(b.opts.Context, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("kafka"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic),
	}
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindProducer),
	}
	ctx, span := b.opts.Tracer.Tracer.Start(ctx, "kafka.produce", opts...)

	b.opts.Tracer.Propagators.Inject(ctx, carrier)

	return span
}

func (b *kafkaBroker) finishProducerSpan(span trace.Span, partition int32, offset int64, err error) {
	if span == nil {
		return
	}

	span.SetAttributes(
		semConv.MessagingMessageIDKey.String(strconv.FormatInt(offset, 10)),
		semConv.MessagingKafkaPartitionKey.Int64(int64(partition)),
	)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	span.End()
}

func (b *kafkaBroker) startConsumerSpan(msg *kafkaGo.Message) trace.Span {
	if b.opts.Tracer.Tracer == nil {
		return nil
	}

	carrier := NewMessageCarrier(msg)
	ctx := b.opts.Tracer.Propagators.Extract(b.opts.Context, carrier)

	attrs := []attribute.KeyValue{
		semConv.MessagingSystemKey.String("kafka"),
		semConv.MessagingDestinationKindTopic,
		semConv.MessagingDestinationKey.String(msg.Topic),
		semConv.MessagingOperationReceive,
		semConv.MessagingMessageIDKey.String(strconv.FormatInt(msg.Offset, 10)),
		semConv.MessagingKafkaPartitionKey.Int64(int64(msg.Partition)),
	}
	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
	}
	newCtx, span := b.opts.Tracer.Tracer.Start(ctx, "kafka.consume", opts...)

	b.opts.Tracer.Propagators.Inject(newCtx, carrier)

	return span
}

func (b *kafkaBroker) finishConsumerSpan(span trace.Span) {
	if span == nil {
		return
	}

	span.End()
}

func (b *kafkaBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
		Queue:   uuid.New().String(),
	}
	for _, o := range opts {
		o(&options)
	}

	readerConfig := b.readerConfig
	readerConfig.Topic = topic
	readerConfig.GroupID = options.Queue

	sub := &subscriber{
		opts:    options,
		topic:   topic,
		handler: handler,
		reader:  kafkaGo.NewReader(readerConfig),
	}

	go func() {

		for {
			select {
			case <-options.Context.Done():
				return
			default:
				msg, err := sub.reader.FetchMessage(options.Context)
				if err != nil {
					return
				}

				span := b.startConsumerSpan(&msg)

				m := &broker.Message{
					Headers: kafkaHeaderToMap(msg.Headers),
					Body:    nil,
				}

				p := &publication{topic: msg.Topic, reader: sub.reader, m: m, km: msg, ctx: options.Context}

				if binder != nil {
					m.Body = binder()
				}

				if err := broker.Unmarshal(b.opts.Codec, msg.Value, m.Body); err != nil {
					p.err = err
				}

				err = sub.handler(sub.opts.Context, p)
				if err != nil {
					b.opts.Logger.Errorf("[kafka]: process message failed: %v", err)
				}
				if sub.opts.AutoAck {
					if err = p.Ack(); err != nil {
						b.opts.Logger.Errorf("[kafka]: unable to commit msg: %v", err)
					}
				}

				b.finishConsumerSpan(span)
			}
		}
	}()

	return sub, nil
}
