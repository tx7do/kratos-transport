package kafka

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/common"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type kBroker struct {
	addrs []string

	readerConfig kafka.ReaderConfig
	writerConfig kafka.WriterConfig

	writers map[string]*kafka.Writer

	log *log.Helper

	connected bool
	sync.RWMutex
	opts common.Options
}

type subscriber struct {
	k         *kBroker
	topic     string
	opts      common.SubscribeOptions
	offset    int64
	gen       *kafka.Generation
	partition int
	handler   broker.Handler
	reader    *kafka.Reader
	closed    bool
	done      chan struct{}
	group     *kafka.ConsumerGroup
	cgcfg     kafka.ConsumerGroupConfig
	sync.RWMutex
}

type publication struct {
	topic      string
	err        error
	m          *broker.Message
	ctx        context.Context
	generation *kafka.Generation
	reader     *kafka.Reader
	km         kafka.Message
	offsets    map[string]map[int]int64 // for commit offsets
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	return p.generation.CommitOffsets(p.offsets)
}

func (p *publication) Error() error {
	return p.err
}

func (s *subscriber) Options() common.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	var err error
	s.Lock()
	defer s.Unlock()
	if s.group != nil {
		err = s.group.Close()
	}
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
	k.writerConfig.Brokers = k.addrs
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

func (k *kBroker) Init(opts ...common.Option) error {
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

func (k *kBroker) Options() common.Options {
	return k.opts
}

func (k *kBroker) Publish(topic string, msg *broker.Message, opts ...common.PublishOption) error {
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
		cfg := k.writerConfig
		cfg.Topic = topic
		if err := cfg.Validate(); err != nil {
			k.Unlock()
			return err
		}
		writer = kafka.NewWriter(cfg)
		k.writers[topic] = writer
	} else {
		cached = true
	}
	k.Unlock()

	err := writer.WriteMessages(k.opts.Context, kMsg)
	if err != nil {
		switch cached {
		case false:
			// non cached case, we can try to wait on some errors, but not timeout
			if kerr, ok := err.(kafka.Error); ok {
				if kerr.Temporary() && !kerr.Timeout() {
					// additional chanse to publish message
					time.Sleep(200 * time.Millisecond)
					err = writer.WriteMessages(k.opts.Context, kMsg)
				}
			}
		case true:
			// cached case, try to recreate writer and try again after that
			k.Lock()
			// close older writer to free memory
			if err = writer.Close(); err != nil {
				k.Unlock()
				return err
			}
			delete(k.writers, topic)
			k.Unlock()

			cfg := k.writerConfig
			cfg.Topic = topic
			if err = cfg.Validate(); err != nil {
				return err
			}
			writer := kafka.NewWriter(cfg)
			if err = writer.WriteMessages(k.opts.Context, kMsg); err == nil {
				k.Lock()
				k.writers[topic] = writer
				k.Unlock()
			}
		}
	}

	return err
}

func (k *kBroker) Subscribe(topic string, handler broker.Handler, opts ...common.SubscribeOption) (broker.Subscriber, error) {
	opt := common.SubscribeOptions{
		AutoAck: true,
		Queue:   uuid.New().String(),
	}
	for _, o := range opts {
		o(&opt)
	}

	cgcfg := kafka.ConsumerGroupConfig{
		ID:                    opt.Queue,
		WatchPartitionChanges: true,
		Brokers:               k.readerConfig.Brokers,
		Topics:                []string{topic},
		GroupBalancers:        []kafka.GroupBalancer{kafka.RangeGroupBalancer{}},
	}
	if err := cgcfg.Validate(); err != nil {
		return nil, err
	}

	cgroup, err := kafka.NewConsumerGroup(cgcfg)
	if err != nil {
		return nil, err
	}

	sub := &subscriber{opts: opt, topic: topic, group: cgroup, cgcfg: cgcfg}

	go func() {
		for {
			select {
			case <-k.opts.Context.Done():
				sub.RLock()
				closed := sub.closed
				sub.RUnlock()
				if closed {
					return
				}
				if k.opts.Context.Err() != nil {
					k.log.Errorf("[segmentio] context closed unexpected %v", k.opts.Context.Err())
				}
				return
			default:
				sub.RLock()
				group := sub.group
				sub.RUnlock()
				generation, err := group.Next(k.opts.Context)
				switch err {
				case nil:
					// normal execution
				case kafka.ErrGroupClosed:
					sub.RLock()
					closed := sub.closed
					sub.RUnlock()
					if !closed {
						k.log.Errorf("[segmentio] recreate consumer group, as it closed %v", k.opts.Context.Err())
						if err = group.Close(); err != nil {
							k.log.Errorf("[segmentio] consumer group close error %v", err)
						}
						sub.createGroup(k.opts.Context)
					}
					continue
				default:
					sub.RLock()
					closed := sub.closed
					sub.RUnlock()
					if !closed {
						k.log.Errorf("[segmentio] recreate consumer group, as unexpected consumer error %v", err)
					}
					if err = group.Close(); err != nil {
						k.log.Errorf("[segmentio] consumer group close error %v", err)
					}
					sub.createGroup(k.opts.Context)
					continue
				}

				for _, t := range cgcfg.Topics {
					assignments := generation.Assignments[t]
					for _, assignment := range assignments {
						cfg := k.readerConfig
						cfg.Topic = t
						cfg.Partition = assignment.ID
						cfg.GroupID = ""
						reader := kafka.NewReader(cfg)
						_ = reader.SetOffset(assignment.Offset)
						cgh := &cgHandler{generation: generation, commonOpts: k.opts, subOpts: opt, reader: reader, handler: handler}
						generation.Start(cgh.run)
					}
				}

			}
		}
	}()

	return sub, nil
}

type cgHandler struct {
	topic      string
	generation *kafka.Generation
	commonOpts common.Options
	subOpts    common.SubscribeOptions
	reader     *kafka.Reader
	handler    broker.Handler
	log        *log.Helper
}

func (h *cgHandler) run(ctx context.Context) {
	offsets := make(map[string]map[int]int64)
	offsets[h.reader.Config().Topic] = make(map[int]int64)

	defer func(reader *kafka.Reader) {
		err := reader.Close()
		if err != nil {

		}
	}(h.reader)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := h.reader.ReadMessage(ctx)
			switch err {
			default:
				return
			case kafka.ErrGenerationEnded:
				return
			case nil:
				var m broker.Message
				eh := h.commonOpts.ErrorHandler
				offsets[msg.Topic][msg.Partition] = msg.Offset
				p := &publication{topic: msg.Topic, generation: h.generation, m: &m, offsets: offsets}

				if h.commonOpts.Codec != nil {
					if err := h.commonOpts.Codec.Unmarshal(msg.Value, &m); err != nil {
						p.err = err
						p.m.Body = msg.Value
						if eh != nil {
							_ = eh(p)
						} else {
							h.log.Errorf("[segmentio]: failed to unmarshal: %v", err)
						}
						continue
					}
				} else {
					m.Body = msg.Value
				}

				err = h.handler(p)
				if err == nil && h.subOpts.AutoAck {
					if err = p.Ack(); err != nil {
						h.log.Errorf("[segmentio]: unable to commit msg: %v", err)
					}
				} else if err != nil {
					p.err = err
					if eh != nil {
						_ = eh(p)
					} else {
						h.log.Errorf("[segmentio]: subscriber error: %v", err)
					}
				}
			}
		}
	}
}

func (s *subscriber) createGroup(ctx context.Context) {
	s.RLock()
	cgcfg := s.cgcfg
	s.RUnlock()

	for {
		select {
		case <-ctx.Done():
			// closed
			return
		default:
			cGroup, err := kafka.NewConsumerGroup(cgcfg)
			if err != nil {
				continue
			}
			s.Lock()
			s.group = cGroup
			s.Unlock()
			// return
			return
		}
	}
}

func (k *kBroker) String() string {
	return "kafka"
}

func NewBroker(opts ...common.Option) broker.Broker {
	options := common.Options{
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

	writerConfig := kafka.WriterConfig{CompressionCodec: nil}
	if cfg, ok := options.Context.Value(writerConfigKey{}).(kafka.WriterConfig); ok {
		writerConfig = cfg
	}
	if len(writerConfig.Brokers) == 0 {
		writerConfig.Brokers = cAddrs
	}
	writerConfig.BatchSize = 1

	return &kBroker{
		readerConfig: readerConfig,
		writerConfig: writerConfig,
		writers:      make(map[string]*kafka.Writer),
		addrs:        cAddrs,
		opts:         options,
	}
}
