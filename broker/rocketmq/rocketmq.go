package rocketmq

import (
	"context"
	"errors"
	"sync"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultAddr = "127.0.0.1:9876"
)

type rocketmqBroker struct {
	nameServers   []string
	nameServerUrl string

	accessKey    string
	secretKey    string
	instanceName string
	groupName    string
	retryCount   int
	namespace    string

	enableTrace bool

	log *log.Helper

	connected bool
	sync.RWMutex
	opts broker.Options

	producers map[string]rocketmq.Producer
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	if v, ok := options.Context.Value(enableAliyunHttpKey{}).(bool); ok && v {
		return newAliyunHttpBroker(options)
	} else {
		return newBroker(options)
	}
}

func newBroker(options broker.Options) broker.Broker {
	return &rocketmqBroker{
		producers:  make(map[string]rocketmq.Producer),
		opts:       options,
		log:        log.NewHelper(log.GetLogger()),
		retryCount: 2,
	}
}

func (r *rocketmqBroker) Name() string {
	return "rocketmq"
}

func (r *rocketmqBroker) Address() string {
	if len(r.nameServers) > 0 {
		return r.nameServers[0]
	} else if r.nameServerUrl != "" {
		return r.nameServerUrl
	}
	return defaultAddr
}

func (r *rocketmqBroker) Options() broker.Options {
	return r.opts
}

func (r *rocketmqBroker) Init(opts ...broker.Option) error {
	r.opts.Apply(opts...)

	if v, ok := r.opts.Context.Value(nameServersKey{}).([]string); ok {
		r.nameServers = v
	}
	if v, ok := r.opts.Context.Value(nameServerUrlKey{}).(string); ok {
		r.nameServerUrl = v
	}
	if v, ok := r.opts.Context.Value(accessKey{}).(string); ok {
		r.accessKey = v
	}
	if v, ok := r.opts.Context.Value(secretKey{}).(string); ok {
		r.secretKey = v
	}
	if v, ok := r.opts.Context.Value(retryCountKey{}).(int); ok {
		r.retryCount = v
	}
	if v, ok := r.opts.Context.Value(namespaceKey{}).(string); ok {
		r.namespace = v
	}
	if v, ok := r.opts.Context.Value(instanceNameKey{}).(string); ok {
		r.instanceName = v
	}
	if v, ok := r.opts.Context.Value(groupNameKey{}).(string); ok {
		r.groupName = v
	}
	if v, ok := r.opts.Context.Value(enableTraceKey{}).(bool); ok {
		r.enableTrace = v
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

func (r *rocketmqBroker) createNsResolver() primitive.NsResolver {
	if len(r.nameServers) > 0 {
		return primitive.NewPassthroughResolver(r.nameServers)
	} else if r.nameServerUrl != "" {
		return primitive.NewHttpResolver("DEFAULT", r.nameServerUrl)
	} else {
		return primitive.NewHttpResolver("DEFAULT", defaultAddr)
	}
}

func (r *rocketmqBroker) createProducer() (rocketmq.Producer, error) {
	credentials := primitive.Credentials{
		AccessKey: r.accessKey,
		SecretKey: r.secretKey,
	}

	resolver := r.createNsResolver()

	var traceCfg *primitive.TraceConfig = nil
	if r.enableTrace {
		traceCfg = &primitive.TraceConfig{
			GroupName:   r.groupName,
			Credentials: credentials,
			Access:      primitive.Cloud,
			Resolver:    resolver,
		}
	}

	p, err := rocketmq.NewProducer(
		producer.WithNsResolver(resolver),
		producer.WithCredentials(credentials),
		producer.WithTrace(traceCfg),
		producer.WithRetry(r.retryCount),
		producer.WithInstanceName(r.instanceName),
		producer.WithNamespace(r.namespace),
		producer.WithGroupName(r.groupName),
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

func (r *rocketmqBroker) createConsumer(options *broker.SubscribeOptions) (rocketmq.PushConsumer, error) {
	credentials := primitive.Credentials{
		AccessKey: r.accessKey,
		SecretKey: r.secretKey,
	}

	resolver := r.createNsResolver()

	var traceCfg *primitive.TraceConfig = nil
	if r.enableTrace {
		traceCfg = &primitive.TraceConfig{
			GroupName:   options.Queue,
			Credentials: credentials,
			Access:      primitive.Cloud,
			Resolver:    resolver,
		}
	}

	c, _ := rocketmq.NewPushConsumer(
		consumer.WithNsResolver(resolver),
		consumer.WithCredentials(credentials),
		consumer.WithTrace(traceCfg),
		consumer.WithGroupName(options.Queue),
		consumer.WithAutoCommit(options.AutoAck),
		consumer.WithRetry(r.retryCount),
		consumer.WithNamespace(r.namespace),
		consumer.WithInstance(r.instanceName),
	)

	if c == nil {
		return nil, errors.New("create consumer error")
	}

	return c, nil
}

func (r *rocketmqBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

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
	rMsg.WithProperties(msg.Header)
	if v, ok := options.Context.Value(compressKey{}).(bool); ok {
		rMsg.Compress = v
	}
	if v, ok := options.Context.Value(batchKey{}).(bool); ok {
		rMsg.Batch = v
	}

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
	options := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   r.groupName,
	}
	for _, o := range opts {
		o(&options)
	}

	c, err := r.createConsumer(&options)
	if err != nil {
		return nil, err
	}

	sub := &subscriber{
		opts:    options,
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
				p := &publication{topic: msg.Topic, reader: sub.reader, m: &m, rm: &msg.Message, ctx: options.Context}

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
