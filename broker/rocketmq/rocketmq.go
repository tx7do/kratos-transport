package rocketmq

import (
	"context"
	"errors"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/tx7do/kratos-transport/broker"
	"sync"
)

const (
	defaultAddr = "127.0.0.1:9876"
)

type rocketmqBroker struct {
	addrs      []string
	accessKey  string
	secretKey  string
	retryCount int
	namespace  string

	log *log.Helper

	connected bool
	sync.RWMutex
	opts broker.Options

	producers map[string]rocketmq.Producer
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

	r := &rocketmqBroker{
		producers:  make(map[string]rocketmq.Producer),
		addrs:      cAddrs,
		opts:       options,
		log:        log.NewHelper(log.GetLogger()),
		retryCount: 2,
	}

	return r
}

func (r *rocketmqBroker) Name() string {
	return "rocketmq"
}

func (r *rocketmqBroker) Address() string {
	if len(r.addrs) > 0 {
		return r.addrs[0]
	}
	return defaultAddr
}

func (r *rocketmqBroker) Options() broker.Options {
	return r.opts
}

func (r *rocketmqBroker) Init(opts ...broker.Option) error {
	r.opts.Apply(opts...)

	var cAddrs []string
	for _, addr := range r.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{defaultAddr}
	}
	r.addrs = cAddrs

	ok := false
	var accesskey string
	accesskey, ok = r.opts.Context.Value(accessKey{}).(string)
	if ok {
		r.accessKey = accesskey
	}

	var secretkey string
	secretkey, ok = r.opts.Context.Value(secretKey{}).(string)
	if ok {
		r.secretKey = secretkey
	}

	var retryCount int
	retryCount, ok = r.opts.Context.Value(retryCountKey{}).(int)
	if ok {
		r.retryCount = retryCount
	}

	var namespace string
	namespace, ok = r.opts.Context.Value(namespaceKey{}).(string)
	if ok {
		r.namespace = namespace
	}

	return nil
}

func (r *rocketmqBroker) Connect() error {
	r.RLock()
	if r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	p, err := r.createProducer()
	if err != nil {
		return err
	}

	_ = p.Shutdown()

	r.Lock()
	r.addrs = r.opts.Addrs
	r.connected = true
	r.Unlock()

	return nil
}

func (r *rocketmqBroker) Disconnect() error {
	r.RLock()
	if !r.connected {
		r.RUnlock()
		return nil
	}
	r.RUnlock()

	r.Lock()
	defer r.Unlock()
	for _, p := range r.producers {
		if err := p.Shutdown(); err != nil {
			return err
		}
	}

	r.connected = false
	return nil
}

func (r *rocketmqBroker) createProducer() (rocketmq.Producer, error) {
	p, err := rocketmq.NewProducer(
		producer.WithNsResolver(primitive.NewPassthroughResolver(r.opts.Addrs)),
		producer.WithRetry(r.retryCount),
		producer.WithCredentials(primitive.Credentials{
			AccessKey: r.accessKey,
			SecretKey: r.secretKey,
		}),
		producer.WithNamespace(r.namespace),
	)
	if err != nil {
		r.log.Errorf("[rocketmq]: new producer error: " + err.Error())
		return nil, err
	}

	err = p.Start()
	if err != nil {
		r.log.Errorf("[rocketmq]: start producer error: %s", err.Error())
		return nil, err
	}

	return p, nil
}

func (r *rocketmqBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var cached bool

	r.Lock()
	p, ok := r.producers[topic]
	if !ok {
		var err error
		p, err = r.createProducer()
		if err != nil {
			r.Unlock()
			return err
		}

		r.producers[topic] = p
	} else {
		cached = true
	}
	r.Unlock()

	var buf []byte
	if r.opts.Codec != nil {
		var err error
		buf, err = r.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
	} else {
		buf = msg.Body
	}

	rMsg := primitive.NewMessage(topic, buf)
	_, err := p.SendSync(r.opts.Context, rMsg)
	if err != nil {
		r.log.Errorf("[rocketmq]: send message error: %s\n", err)
		switch cached {
		case false:
		case true:
			r.Lock()
			if err = p.Shutdown(); err != nil {
				r.Unlock()
				return err
			}
			delete(r.producers, topic)
			r.Unlock()

			p, err = r.createProducer()
			if err != nil {
				r.Unlock()
				return err
			}
			if _, err = p.SendSync(r.opts.Context, rMsg); err == nil {
				r.Lock()
				r.producers[topic] = p
				r.Unlock()
			}
		}
	}

	return nil
}

func (r *rocketmqBroker) Subscribe(topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	opt := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   uuid.New().String(),
	}
	for _, o := range opts {
		o(&opt)
	}

	c, _ := rocketmq.NewPushConsumer(
		consumer.WithGroupName(opt.Queue),
		consumer.WithNsResolver(primitive.NewPassthroughResolver(r.opts.Addrs)),
	)
	if c == nil {
		return nil, errors.New("create consumer error")
	}

	sub := &subscriber{
		opts:    opt,
		topic:   topic,
		handler: h,
		reader:  c,
	}

	if err := c.Subscribe(topic, consumer.MessageSelector{},
		func(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
			//r.log.Infof("subscribe callback: %v \n", msgs)

			var err error
			var m broker.Message
			for _, msg := range msgs {
				p := &publication{topic: msg.Topic, reader: sub.reader, m: &m, rm: &msg.Message, ctx: opt.Context}

				m.Header = msg.GetProperties()
				if r.opts.Codec != nil {
					if err := r.opts.Codec.Unmarshal(msg.Body, &m); err != nil {
						p.err = err
					}
				} else {
					m.Body = msg.Body
				}

				err = sub.handler(sub.opts.Context, p)
				if err != nil {
					r.log.Errorf("[rocketmq]: process message failed: %v", err)
				}
				if sub.opts.AutoAck {
					if err = p.Ack(); err != nil {
						r.log.Errorf("[rocketmq]: unable to commit msg: %v", err)
					}
				}
			}

			return consumer.ConsumeSuccess, nil
		}); err != nil {
		r.log.Errorf(err.Error())
		return nil, err
	}

	if err := c.Start(); err != nil {
		r.log.Errorf(err.Error())
		return nil, err
	}

	return sub, nil
}
