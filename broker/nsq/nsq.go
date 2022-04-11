package nsq

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nsqio/go-nsq"
	"github.com/tx7do/kratos-transport/broker"
)

type nsqBroker struct {
	lookupAddrs []string
	addrs       []string
	opts        broker.Options
	config      *nsq.Config

	sync.Mutex
	running bool
	p       []*nsq.Producer
	c       []*subscriber
}

type publication struct {
	topic string
	m     *broker.Message
	nm    *nsq.Message
	opts  broker.PublishOptions
	err   error
}

type subscriber struct {
	topic string
	opts  broker.SubscribeOptions
	c     *nsq.Consumer
	h     nsq.HandlerFunc
	n     int
}

var (
	DefaultConcurrentHandlers = 1
)

func (n *nsqBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&n.opts)
	}

	var addrs []string

	for _, addr := range n.opts.Addrs {
		if len(addr) > 0 {
			addrs = append(addrs, addr)
		}
	}

	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:4150"}
	}

	n.addrs = addrs
	n.configure(n.opts.Context)
	return nil
}

func (n *nsqBroker) configure(ctx context.Context) {
	if v, ok := ctx.Value(lookupdAddrsKey{}).([]string); ok {
		n.lookupAddrs = v
	}

	if v, ok := ctx.Value(consumerOptsKey{}).([]string); ok {
		cfgFlag := &nsq.ConfigFlag{Config: n.config}
		for _, opt := range v {
			_ = cfgFlag.Set(opt)
		}
	}
}

func (n *nsqBroker) Options() broker.Options {
	return n.opts
}

func (n *nsqBroker) Address() string {
	return n.addrs[rand.Intn(len(n.addrs))]
}

func (n *nsqBroker) Connect() error {
	n.Lock()
	defer n.Unlock()

	if n.running {
		return nil
	}

	producers := make([]*nsq.Producer, 0, len(n.addrs))

	for _, addr := range n.addrs {
		p, err := nsq.NewProducer(addr, n.config)
		if err != nil {
			return err
		}
		if err = p.Ping(); err != nil {
			return err
		}
		producers = append(producers, p)
	}

	for _, c := range n.c {
		channel := c.opts.Queue
		if len(channel) == 0 {
			channel = uuid.New().String() + "#ephemeral"
		}

		cm, err := nsq.NewConsumer(c.topic, channel, n.config)
		if err != nil {
			return err
		}

		cm.AddConcurrentHandlers(c.h, c.n)

		c.c = cm

		if len(n.lookupAddrs) > 0 {
			_ = c.c.ConnectToNSQLookupds(n.lookupAddrs)
		} else {
			err = c.c.ConnectToNSQDs(n.addrs)
			if err != nil {
				return err
			}
		}
	}

	n.p = producers
	n.running = true
	return nil
}

func (n *nsqBroker) Disconnect() error {
	n.Lock()
	defer n.Unlock()

	if !n.running {
		return nil
	}

	for _, p := range n.p {
		p.Stop()
	}

	for _, c := range n.c {
		c.c.Stop()

		if len(n.lookupAddrs) > 0 {
			for _, addr := range n.lookupAddrs {
				_ = c.c.DisconnectFromNSQLookupd(addr)
			}
		} else {
			for _, addr := range n.addrs {
				_ = c.c.DisconnectFromNSQD(addr)
			}
		}
	}

	n.p = nil
	n.running = false
	return nil
}

func (n *nsqBroker) Publish(topic string, message *broker.Message, opts ...broker.PublishOption) error {
	p := n.p[rand.Intn(len(n.p))]

	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	var (
		doneChan chan *nsq.ProducerTransaction
		delay    time.Duration
	)
	if options.Context != nil {
		if v, ok := options.Context.Value(asyncPublishKey{}).(chan *nsq.ProducerTransaction); ok {
			doneChan = v
		}
		if v, ok := options.Context.Value(deferredPublishKey{}).(time.Duration); ok {
			delay = v
		}
	}

	var sendBuffer []byte
	if n.opts.Codec != nil {
		var err error
		sendBuffer, err = n.opts.Codec.Marshal(message)
		if err != nil {
			return err
		}
	} else {
		sendBuffer = message.Body
	}

	if doneChan != nil {
		if delay > 0 {
			return p.DeferredPublishAsync(topic, delay, sendBuffer, doneChan)
		}
		return p.PublishAsync(topic, sendBuffer, doneChan)
	} else {
		if delay > 0 {
			return p.DeferredPublish(topic, delay, sendBuffer)
		}
		return p.Publish(topic, sendBuffer)
	}
}

func (n *nsqBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		AutoAck: true,
	}

	for _, o := range opts {
		o(&options)
	}

	concurrency, maxInFlight := DefaultConcurrentHandlers, DefaultConcurrentHandlers
	if options.Context != nil {
		if v, ok := options.Context.Value(concurrentHandlerKey{}).(int); ok {
			maxInFlight, concurrency = v, v
		}
		if v, ok := options.Context.Value(maxInFlightKey{}).(int); ok {
			maxInFlight = v
		}
	}
	channel := options.Queue
	if len(channel) == 0 {
		channel = uuid.New().String() + "#ephemeral"
	}
	config := *n.config
	config.MaxInFlight = maxInFlight

	c, err := nsq.NewConsumer(topic, channel, &config)
	if err != nil {
		return nil, err
	}

	h := nsq.HandlerFunc(func(nm *nsq.Message) error {
		if !options.AutoAck {
			nm.DisableAutoResponse()
		}

		var m broker.Message

		if n.opts.Codec != nil {
			if err := n.opts.Codec.Unmarshal(nm.Body, &m); err != nil {
				return err
			}
		} else {
			m.Body = nm.Body
		}

		p := &publication{topic: topic, nm: nm, m: &m}
		p.err = handler(n.opts.Context, p)
		return p.err
	})

	c.AddConcurrentHandlers(h, concurrency)

	if len(n.lookupAddrs) > 0 {
		err = c.ConnectToNSQLookupds(n.lookupAddrs)
	} else {
		err = c.ConnectToNSQDs(n.addrs)
	}
	if err != nil {
		return nil, err
	}

	sub := &subscriber{
		c:     c,
		opts:  options,
		topic: topic,
		h:     h,
		n:     concurrency,
	}

	n.c = append(n.c, sub)

	return sub, nil
}

func (n *nsqBroker) Name() string {
	return "nsq"
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	p.nm.Finish()
	return nil
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
	s.c.Stop()
	return nil
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	var addrs []string

	for _, addr := range options.Addrs {
		if len(addr) > 0 {
			addrs = append(addrs, addr)
		}
	}

	if len(addrs) == 0 {
		addrs = []string{"127.0.0.1:4150"}
	}

	n := &nsqBroker{
		addrs:  addrs,
		opts:   options,
		config: nsq.NewConfig(),
	}
	n.configure(n.opts.Context)

	return n
}
