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

const (
	defaultAddr = "127.0.0.1:4150"
)

type nsqBroker struct {
	lookupAddrs []string
	addrs       []string

	opts   broker.Options
	config *NSQ.Config

	sync.Mutex
	running bool

	producers   []*NSQ.Producer
	subscribers []*subscriber
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
		addrs = []string{defaultAddr}
	}

	n := &nsqBroker{
		addrs:  addrs,
		opts:   options,
		config: NSQ.NewConfig(),
	}
	n.configure(n.opts.Context)

	return n
}

func (n *nsqBroker) Name() string {
	return "nsq"
}

func (n *nsqBroker) Options() broker.Options {
	return n.opts
}

func (n *nsqBroker) Address() string {
	return n.addrs[rand.Intn(len(n.addrs))]
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
		addrs = []string{defaultAddr}
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

	for _, c := range n.subscribers {
		channel := c.opts.Queue
		if len(channel) == 0 {
			channel = uuid.New().String() + "#ephemeral"
		}

		cm, err := NSQ.NewConsumer(c.topic, channel, n.config)
		if err != nil {
			return err
		}

		if c.handlerFunc != nil {
			cm.AddConcurrentHandlers(c.handlerFunc, c.concurrency)
		}

		c.consumer = cm

		if len(n.lookupAddrs) > 0 {
			_ = c.consumer.ConnectToNSQLookupds(n.lookupAddrs)
		} else {
			err = c.consumer.ConnectToNSQDs(n.addrs)
			if err != nil {
				return err
			}
		}
	}

	n.producers = producers
	n.running = true
	return nil
}

func (n *nsqBroker) Disconnect() error {
	n.Lock()
	defer n.Unlock()

	if !n.running {
		return nil
	}

	for _, p := range n.producers {
		p.Stop()
	}

	for _, c := range n.subscribers {
		c.consumer.Stop()

		if len(n.lookupAddrs) > 0 {
			for _, addr := range n.lookupAddrs {
				_ = c.consumer.DisconnectFromNSQLookupd(addr)
			}
		} else {
			for _, addr := range n.addrs {
				_ = c.consumer.DisconnectFromNSQD(addr)
			}
		}
	}

	n.producers = nil
	n.running = false
	return nil
}

func (n *nsqBroker) Publish(topic string, message *broker.Message, opts ...broker.PublishOption) error {
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

	p := n.producers[rand.Intn(len(n.producers))]

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

	config := *n.config
	config.MaxInFlight = maxInFlight

	channel := options.Queue
	if len(channel) == 0 {
		channel = uuid.New().String() + "#ephemeral"
	}

	c, err := NSQ.NewConsumer(topic, channel, &config)
	if err != nil {
		return nil, err
	}

	h := NSQ.HandlerFunc(func(nm *NSQ.Message) error {
		if !options.AutoAck {
			nm.DisableAutoResponse()
		}

		//fmt.Println("receive message:", nm.ID, nm.Body)

		var m broker.Message
		if n.opts.Codec != nil {
			if err := n.opts.Codec.Unmarshal(nm.Body, &m); err != nil {
				return err
			}
		} else {
			m.Body = nm.Body
		}

		p := &publication{topic: topic, nsqMsg: nm, msg: &m}
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
		consumer:    c,
		opts:        options,
		topic:       topic,
		handlerFunc: h,
		concurrency: concurrency,
	}

	n.subscribers = append(n.subscribers, sub)

	return sub, nil
}
