package nsq

import (
	"context"
	"errors"
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
	sync.Mutex

	lookupAddrs []string
	addrs       []string

	options broker.Options
	config  *NSQ.Config

	running bool

	producers []*NSQ.Producer

	subscribers *broker.SubscriberSyncMap
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &nsqBroker{
		options: options,
		config:  NSQ.NewConfig(),

		producers: make([]*NSQ.Producer, 0),

		subscribers: broker.NewSubscriberSyncMap(),
	}

	return b
}

func (b *nsqBroker) Name() string {
	return "NSQ"
}

func (b *nsqBroker) Options() broker.Options {
	return b.options
}

func (b *nsqBroker) Address() string {
	if len(b.options.Addrs) > 0 {
		return b.options.Addrs[0]
	}

	return defaultAddr
}

func (b *nsqBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&b.options)
	}

	var addrs []string

	for _, addr := range b.options.Addrs {
		if len(addr) > 0 {
			addrs = append(addrs, addr)
		}
	}

	if len(addrs) == 0 {
		addrs = []string{defaultAddr}
	}

	b.addrs = addrs
	b.configure(b.options.Context)

	return nil
}

func (b *nsqBroker) configure(ctx context.Context) {
	if v, ok := ctx.Value(lookupdAddrsKey{}).([]string); ok {
		b.lookupAddrs = v
	}

	if v, ok := ctx.Value(consumerOptsKey{}).([]string); ok {
		cfgFlag := &NSQ.ConfigFlag{Config: b.config}
		for _, opt := range v {
			_ = cfgFlag.Set(opt)
		}
	}
}

func (b *nsqBroker) Connect() error {
	b.Lock()
	defer b.Unlock()

	if b.running {
		return nil
	}

	producers := make([]*NSQ.Producer, 0, len(b.addrs))
	for _, addr := range b.addrs {
		p, err := NSQ.NewProducer(addr, b.config)
		if err != nil {
			return err
		}
		if err = p.Ping(); err != nil {
			return err
		}
		producers = append(producers, p)
	}
	b.producers = producers

	var err error
	b.subscribers.Foreach(func(topic string, sub broker.Subscriber) {
		c := sub.(*subscriber)

		channel := c.options.Queue
		if len(channel) == 0 {
			channel = uuid.New().String() + "#ephemeral"
		}

		var cm *NSQ.Consumer
		if cm, err = NSQ.NewConsumer(c.topic, channel, b.config); err != nil {
			return
		}

		if c.handlerFunc != nil {
			cm.AddConcurrentHandlers(c.handlerFunc, c.concurrency)
		}

		c.consumer = cm

		if len(b.lookupAddrs) > 0 {
			_ = c.consumer.ConnectToNSQLookupds(b.lookupAddrs)
		} else {
			if err = c.consumer.ConnectToNSQDs(b.addrs); err != nil {
				return
			}
		}
	})

	b.running = true

	return nil
}

func (b *nsqBroker) Disconnect() error {
	b.Lock()
	defer b.Unlock()

	if !b.running {
		return nil
	}

	for _, p := range b.producers {
		p.Stop()
	}

	b.subscribers.Foreach(func(topic string, sub broker.Subscriber) {
		c := sub.(*subscriber)

		c.consumer.Stop()

		if len(b.lookupAddrs) > 0 {
			for _, addr := range b.lookupAddrs {
				_ = c.consumer.DisconnectFromNSQLookupd(addr)
			}
		} else {
			for _, addr := range b.addrs {
				_ = c.consumer.DisconnectFromNSQD(addr)
			}
		}
	})
	b.subscribers.Clear()

	b.producers = nil
	b.running = false

	return nil
}

func (b *nsqBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return nil, errors.New("not implemented")
}

func (b *nsqBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *nsqBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *nsqBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{
		Context: ctx,
	}
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

	p := b.getProducer()
	if p == nil {
		return errors.New("producer is null")
	}

	if doneChan != nil {
		if delay > 0 {
			return p.DeferredPublishAsync(topic, delay, msg.BodyBytes(), doneChan)
		}
		return p.PublishAsync(topic, msg.BodyBytes(), doneChan)
	} else {
		if delay > 0 {
			return p.DeferredPublish(topic, delay, msg.BodyBytes())
		}
		return p.Publish(topic, msg.BodyBytes())
	}
}

func (b *nsqBroker) getProducer() *NSQ.Producer {
	producerLen := len(b.producers)
	if producerLen == 0 {
		return nil
	}
	return b.producers[rand.Intn(producerLen)]
}

func (b *nsqBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
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

	config := *b.config
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

		//fmt.Println("receive message:", nm.ID, nm.Payload)

		var m broker.Message
		var errSub error

		if binder != nil {
			m.Body = binder()

			if errSub = broker.Unmarshal(b.options.Codec, nm.Body, &m.Body); errSub != nil {
				return errSub
			}
		} else {
			m.Body = nm.Body
		}

		p := &publication{topic: topic, nsqMsg: nm, msg: &m}

		if errSub = handler(b.options.Context, p); errSub != nil {
			p.err = errSub
			return errSub
		}

		if options.AutoAck {
			if errSub = p.Ack(); err != nil {
				LogErrorf("unable to commit msg: %v", errSub)
			}
		}

		return p.err
	})

	c.AddConcurrentHandlers(h, concurrency)

	if len(b.lookupAddrs) > 0 {
		err = c.ConnectToNSQLookupds(b.lookupAddrs)
	} else {
		err = c.ConnectToNSQDs(b.addrs)
	}
	if err != nil {
		return nil, err
	}

	sub := &subscriber{
		n:           b,
		consumer:    c,
		options:     options,
		topic:       topic,
		handlerFunc: h,
		concurrency: concurrency,
	}

	b.subscribers.Add(topic, sub)

	return sub, nil
}
