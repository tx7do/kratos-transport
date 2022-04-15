package nsq

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	NSQ "github.com/nsqio/go-nsq"
	"github.com/tx7do/kratos-transport/broker"
)

var (
	DefaultConcurrentHandlers = 1
)

type nsqBroker struct {
	lookupAddrs []string
	addrs       []string
	opts        broker.Options
	config      *NSQ.Config

	sync.Mutex
	running bool
	p       []*NSQ.Producer
	c       []*subscriber
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
		config: NSQ.NewConfig(),
	}
	n.configure(n.opts.Context)

	return n
}

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
		cfgFlag := &NSQ.ConfigFlag{Config: n.config}
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

	producers := make([]*NSQ.Producer, 0, len(n.addrs))

	for _, addr := range n.addrs {
		p, err := NSQ.NewProducer(addr, n.config)
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

		cm, err := NSQ.NewConsumer(c.topic, channel, n.config)
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
		doneChan chan *NSQ.ProducerTransaction
		delay    time.Duration
	)
	if options.Context != nil {
		if v, ok := options.Context.Value(asyncPublishKey{}).(chan *NSQ.ProducerTransaction); ok {
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

	c, err := NSQ.NewConsumer(topic, channel, &config)
	if err != nil {
		return nil, err
	}

	h := NSQ.HandlerFunc(func(nm *NSQ.Message) error {
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
