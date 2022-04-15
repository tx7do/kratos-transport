package kafka

import (
	"errors"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/tx7do/kratos-transport/broker"
)

type kafkaBroker struct {
	addrs []string

	readerConfig kafka.ReaderConfig
	writers      map[string]*kafka.Writer

	log *log.Helper

	connected bool
	sync.RWMutex
	opts broker.Options
}

func (k *kafkaBroker) Address() string {
	if len(k.addrs) > 0 {
		return k.addrs[0]
	}
	return "127.0.0.1:9092"
}

func (k *kafkaBroker) Connect() error {
	k.RLock()
	if k.connected {
		k.RUnlock()
		return nil
	}
	k.RUnlock()

	kAddrs := make([]string, 0, len(k.addrs))
	for _, addr := range k.addrs {
		conn, err := kafka.DialContext(k.opts.Context, "tcp", addr)
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

	k.Lock()
	k.addrs = kAddrs
	k.readerConfig.Brokers = k.addrs
	k.connected = true
	k.Unlock()

	return nil
}

func (k *kafkaBroker) Disconnect() error {
	k.RLock()
	if !k.connected {
		k.RUnlock()
		return nil
	}
	k.RUnlock()

	k.Lock()
	defer k.Unlock()
	for _, writer := range k.writers {
		if err := writer.Close(); err != nil {
			return err
		}
	}

	k.connected = false
	return nil
}

func (k *kafkaBroker) Init(opts ...broker.Option) error {
	k.opts.Apply(opts...)

	var cAddrs []string
	for _, addr := range k.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{"127.0.0.1:9092"}
	}
	k.addrs = cAddrs
	return nil
}

func (k *kafkaBroker) Options() broker.Options {
	return k.opts
}

func (k *kafkaBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var cached bool

	var buf []byte
	if k.opts.Codec != nil {
		var err error
		buf, err = k.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
	} else {
		buf = msg.Body
	}

	kMsg := kafka.Message{Topic: topic, Value: buf}

	k.Lock()
	writer, ok := k.writers[topic]
	if !ok {
		writer =
			&kafka.Writer{
				Addr:     kafka.TCP(k.addrs...),
				Balancer: &kafka.LeastBytes{},
			}
		k.writers[topic] = writer
	} else {
		cached = true
	}
	k.Unlock()

	err := writer.WriteMessages(k.opts.Context, kMsg)
	if err != nil {
		switch cached {
		case false:
			if kerr, ok := err.(kafka.Error); ok {
				if kerr.Temporary() && !kerr.Timeout() {
					time.Sleep(200 * time.Millisecond)
					err = writer.WriteMessages(k.opts.Context, kMsg)
				}
			}
		case true:
			k.Lock()
			if err = writer.Close(); err != nil {
				k.Unlock()
				return err
			}
			delete(k.writers, topic)
			k.Unlock()

			writer := &kafka.Writer{
				Addr:     kafka.TCP(k.addrs...),
				Balancer: &kafka.LeastBytes{},
			}
			if err = writer.WriteMessages(k.opts.Context, kMsg); err == nil {
				k.Lock()
				k.writers[topic] = writer
				k.Unlock()
			}
		}
	}

	return err
}

func (k *kafkaBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	opt := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   uuid.New().String(),
	}
	for _, o := range opts {
		o(&opt)
	}

	readerConfig := k.readerConfig
	readerConfig.Topic = topic
	readerConfig.GroupID = opt.Queue

	sub := &subscriber{
		opts:    opt,
		topic:   topic,
		handler: handler,
		reader:  kafka.NewReader(readerConfig),
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

				var m broker.Message
				p := &publication{topic: msg.Topic, reader: sub.reader, m: &m, km: msg, ctx: opt.Context}

				if k.opts.Codec != nil {
					if err := k.opts.Codec.Unmarshal(msg.Value, &m); err != nil {
						p.err = err
					}
				} else {
					m.Body = msg.Value
				}

				err = sub.handler(sub.opts.Context, p)
				if err != nil {
					k.log.Errorf("[segmentio]: process message failed: %v", err)
				}
				if sub.opts.AutoAck {
					if err = p.Ack(); err != nil {
						k.log.Errorf("[segmentio]: unable to commit msg: %v", err)
					}
				}
			}
		}
	}()

	return sub, nil
}

func (k *kafkaBroker) Name() string {
	return "kafka"
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
		cAddrs = []string{"127.0.0.1:9092"}
	}

	readerConfig := kafka.ReaderConfig{}
	if cfg, ok := options.Context.Value(readerConfigKey{}).(kafka.ReaderConfig); ok {
		readerConfig = cfg
	}
	if len(readerConfig.Brokers) == 0 {
		readerConfig.Brokers = cAddrs
	}
	readerConfig.WatchPartitionChanges = true

	return &kafkaBroker{
		readerConfig: readerConfig,
		writers:      make(map[string]*kafka.Writer),
		addrs:        cAddrs,
		opts:         options,
		log:          log.NewHelper(log.GetLogger()),
	}
}
