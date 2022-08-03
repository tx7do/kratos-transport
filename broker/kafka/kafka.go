package kafka

import (
	"bytes"
	"encoding/gob"
	"errors"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	kafkaGo "github.com/segmentio/kafka-go"
	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultAddr = "127.0.0.1:9092"
)

type kafkaBroker struct {
	addrs []string

	readerConfig kafkaGo.ReaderConfig
	writers      map[string]*kafkaGo.Writer

	log *log.Helper

	connected bool
	sync.RWMutex
	opts broker.Options
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	var cAddrs []string
	for _, addr := range options.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{defaultAddr}
	}

	readerConfig := kafkaGo.ReaderConfig{}
	if cfg, ok := options.Context.Value(readerConfigKey{}).(kafkaGo.ReaderConfig); ok {
		readerConfig = cfg
	}
	if len(readerConfig.Brokers) == 0 {
		readerConfig.Brokers = cAddrs
	}
	readerConfig.WatchPartitionChanges = true

	return &kafkaBroker{
		readerConfig: readerConfig,
		writers:      make(map[string]*kafkaGo.Writer),
		addrs:        cAddrs,
		opts:         options,
		log:          log.NewHelper(log.GetLogger()),
	}
}

func (b *kafkaBroker) Name() string {
	return "kafka"
}

func (b *kafkaBroker) Address() string {
	if len(b.addrs) > 0 {
		return b.addrs[0]
	}
	return defaultAddr
}

func (b *kafkaBroker) Options() broker.Options {
	return b.opts
}

func (b *kafkaBroker) Init(opts ...broker.Option) error {
	b.opts.Apply(opts...)

	var cAddrs []string
	for _, addr := range b.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{defaultAddr}
	}
	b.addrs = cAddrs
	return nil
}

func (b *kafkaBroker) Connect() error {
	b.RLock()
	if b.connected {
		b.RUnlock()
		return nil
	}
	b.RUnlock()

	kAddrs := make([]string, 0, len(b.addrs))
	for _, addr := range b.addrs {
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
	b.addrs = kAddrs
	b.readerConfig.Brokers = b.addrs
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

func (b *kafkaBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	if msg == nil {
		return errors.New("message is nil")
	}

	if b.opts.Codec != nil {
		var err error
		buf, err := b.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
		return b.publish(topic, buf, opts...)
	} else {
		switch t := msg.(type) {
		case []byte:
			return b.publish(topic, t, opts...)
		case string:
			return b.publish(topic, []byte(t), opts...)
		default:
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(msg); err != nil {
				return err
			}
			return b.publish(topic, buf.Bytes(), opts...)
		}
	}
}

func (b *kafkaBroker) createProducer(async bool, batchSize int, batchBytes int64, batchTimeout time.Duration) *kafkaGo.Writer {
	return &kafkaGo.Writer{
		Addr:         kafkaGo.TCP(b.addrs...),
		Balancer:     &kafkaGo.LeastBytes{},
		Async:        async,
		BatchSize:    batchSize,
		BatchTimeout: batchTimeout,
		BatchBytes:   batchBytes,
	}
}

func (b *kafkaBroker) publish(topic string, buf []byte, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	var async = false
	var batchSize = 1
	var batchBytes int64 = 1048576
	var batchTimeout = 10 * time.Millisecond
	var retriesCount = 1

	var cached bool

	kMsg := kafkaGo.Message{Topic: topic, Value: buf}

	if value, ok := options.Context.Value(batchSizeKey{}).(int); ok {
		batchSize = value
	}

	if value, ok := options.Context.Value(batchTimeoutKey{}).(time.Duration); ok {
		batchTimeout = value
	}

	if value, ok := options.Context.Value(batchBytesKey{}).(int64); ok {
		batchBytes = value
	}

	if value, ok := options.Context.Value(retriesCountKey{}).(int); ok {
		retriesCount = value
	}

	if value, ok := options.Context.Value(asyncKey{}).(bool); ok {
		async = value
	}

	if headers, ok := options.Context.Value(headersKey{}).(map[string]interface{}); ok {
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

	b.Lock()
	writer, ok := b.writers[topic]
	if !ok {
		writer = b.createProducer(async, batchSize, batchBytes, batchTimeout)
		b.writers[topic] = writer
	} else {
		cached = true
	}
	b.Unlock()

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
				return err
			}
			delete(b.writers, topic)
			b.Unlock()

			writer := b.createProducer(async, batchSize, batchBytes, batchTimeout)
			for i := 0; i < retriesCount; i++ {
				if err = writer.WriteMessages(b.opts.Context, kMsg); err == nil {
					b.Lock()
					b.writers[topic] = writer
					b.Unlock()
					break
				}
			}
		}
	}

	return err
}

func (b *kafkaBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	opt := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   uuid.New().String(),
	}
	for _, o := range opts {
		o(&opt)
	}

	readerConfig := b.readerConfig
	readerConfig.Topic = topic
	readerConfig.GroupID = opt.Queue

	sub := &subscriber{
		opts:    opt,
		topic:   topic,
		handler: handler,
		reader:  kafkaGo.NewReader(readerConfig),
	}

	go func() {

		for {
			select {
			case <-opt.Context.Done():
				return
			default:
				msg, err := sub.reader.FetchMessage(opt.Context)
				if err != nil {
					return
				}

				m := &broker.Message{
					Headers: kafkaHeaderToMap(msg.Headers),
					Body:    nil,
				}

				p := &publication{topic: msg.Topic, reader: sub.reader, m: m, km: msg, ctx: opt.Context}

				if binder != nil {
					m.Body = binder()
				}

				if b.opts.Codec != nil {
					if err := b.opts.Codec.Unmarshal(msg.Value, m.Body); err != nil {
						p.err = err
					}
				} else {
					m.Body = msg.Value
				}

				err = sub.handler(sub.opts.Context, p)
				if err != nil {
					b.log.Errorf("[kafka]: process message failed: %v", err)
				}
				if sub.opts.AutoAck {
					if err = p.Ack(); err != nil {
						b.log.Errorf("[kafka]: unable to commit msg: %v", err)
					}
				}
			}
		}
	}()

	return sub, nil
}
