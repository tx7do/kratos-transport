package kafka

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type kBroker struct {
	addrs []string

	readerConfig kafka.ReaderConfig
	writers      map[string]*kafka.Writer

	log *log.Helper

	connected bool
	sync.RWMutex
	opts broker.Options
}

type subscriber struct {
	k       *kBroker
	topic   string
	opts    broker.SubscribeOptions
	handler broker.Handler
	reader  *kafka.Reader
	closed  bool
	done    chan struct{}
	sync.RWMutex
}

type publication struct {
	topic  string
	err    error
	m      *broker.Message
	ctx    context.Context
	reader *kafka.Reader
	km     kafka.Message
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	return p.reader.CommitMessages(p.ctx, p.km)
}

func (p *publication) Error() error {
	return p.err
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	var err error
	s.Lock()
	defer s.Unlock()
	s.closed = true
	return err
}

func (k *kBroker) Address() string {
	if len(k.addrs) > 0 {
		return k.addrs[0]
	}
	return "127.0.0.1:9092"
}

func (k *kBroker) Connect() error {
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

func (k *kBroker) Disconnect() error {
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

func (k *kBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&k.opts)
	}
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

func (k *kBroker) Options() broker.Options {
	return k.opts
}

func (k *kBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
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

	kMsg := kafka.Message{Value: buf}

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

func (k *kBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
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

				err = sub.handler(p)
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

func (k *kBroker) String() string {
	return "kafka"
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.Options{
		//Codec:   json.Marshaler{},
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&options)
	}

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

	return &kBroker{
		readerConfig: readerConfig,
		writers:      make(map[string]*kafka.Writer),
		addrs:        cAddrs,
		opts:         options,
		log:          log.NewHelper(log.GetLogger()),
	}
}
