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
		// 优先用订阅自身的 config（保留 WithMaxInFlight 等每订阅配置），nil 回退全局
		subConfig := b.config
		if c.config != nil {
			subConfig = c.config
		}
		if cm, err = NSQ.NewConsumer(c.topic, channel, subConfig); err != nil {
			return
		}

		if c.handlerFunc != nil {
			cm.AddConcurrentHandlers(c.handlerFunc, c.concurrency)
		} else if c.needsHandler {
			// Subscribe 先于 Connect 登记的订阅：此处补建 handler
			c.buildHandler(cm)
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
	// Foreach 闭包内的 return 只结束当次迭代：
	// 这里显式检查 err，避免订阅重连失败仍把服务标记为已启动
	if err != nil {
		return err
	}

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
	return broker.GenericRequest(ctx, b, topic, msg, opts...)
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

	// 未启动时只登记订阅（handler 在 Connect 里统一建 consumer 并连接），
	// 避免先 Subscribe 后 Connect 时产生两个 consumer，旧的泄漏
	if !b.running {
		sub := &subscriber{
			n:            b,
			options:      options,
			topic:        topic,
			needsHandler: true,
			binder:       binder,
			handler:      handler,
			config:       &config,
			concurrency:  concurrency,
		}

		if old := b.subscribers.Get(topic); old != nil {
			// 同主题重复订阅：先退订旧订阅，避免旧订阅继续消费（泄漏 + 重复消费）
			if uerr := old.Unsubscribe(false); uerr != nil {
				LogWarnf("unsubscribe old subscriber for topic %q failed: %v", topic, uerr)
			}
		}
		b.subscribers.Add(topic, sub)

		return sub, nil
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
				// 毒消息：通知 ErrorHandler 并 Finish，避免无限 Requeue 后被丢弃且无感知
				LogErrorf("unmarshal message failed: %v", errSub)
				p := &publication{topic: topic, nsqMsg: nm, msg: &m, err: errSub}
				if eh := b.options.ErrorHandler; eh != nil {
					_ = eh(b.options.Context, p)
				}
				nm.Finish()
				return nil
			}
		} else {
			m.Body = nm.Body
		}

		p := &publication{topic: topic, nsqMsg: nm, msg: &m}

		if errSub = handler(b.options.Context, p); errSub != nil {
			p.err = errSub
			if eh := b.options.ErrorHandler; eh != nil {
				_ = eh(b.options.Context, p)
			}
			// 处理失败即终止重投：显式 FIN 后必须返回 nil——
			// 返回 err 会让 go-nsq 对已 FIN 的消息再发 REQ（协议错误）
			if errFinish := p.Ack(); errFinish != nil {
				LogErrorf("unable to commit msg: %v", errFinish)
			}
			return nil
		}

		// AutoAck=false 的成功路径：由 handler 自行通过 publication.Ack 决定；
		// 未 ack 的消息由 in-flight 超时重投（标准手动语义）
		return nil
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

	if old := b.subscribers.Get(topic); old != nil {
		// 同主题重复订阅：先退订旧订阅，避免旧订阅继续消费（泄漏 + 重复消费）
		if uerr := old.Unsubscribe(false); uerr != nil {
			LogWarnf("unsubscribe old subscriber for topic %q failed: %v", topic, uerr)
		}
	}
	b.subscribers.Add(topic, sub)

	return sub, nil
}
