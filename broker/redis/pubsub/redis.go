package pubsub

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
	redisOption "github.com/tx7do/kratos-transport/broker/redis/option"
)

const (
	defaultBroker = "redis://127.0.0.1:6379"
)

type pubsubBroker struct {
	addr       string
	pool       *redis.Pool
	options    broker.Options
	commonOpts *redisOption.CommonOptions

	subscribers *broker.SubscriberSyncMap
}

func NewBroker(opts ...broker.Option) broker.Broker {
	commonOpts := &redisOption.CommonOptions{
		MaxIdle:        redisOption.DefaultMaxIdle,
		MaxActive:      redisOption.DefaultMaxActive,
		IdleTimeout:    redisOption.DefaultIdleTimeout,
		ConnectTimeout: redisOption.DefaultConnectTimeout,
		ReadTimeout:    redisOption.DefaultReadTimeout,
		WriteTimeout:   redisOption.DefaultWriteTimeout,
	}

	options := broker.NewOptionsAndApply(opts...)

	if l := broker.LoggerFromOptions(&options); l != nil {
		redisOption.SetLogger(l)
	}

	return &pubsubBroker{
		options:     options,
		commonOpts:  commonOpts,
		subscribers: broker.NewSubscriberSyncMap(),
	}
}

func (b *pubsubBroker) Name() string {
	return "redis"
}

func (b *pubsubBroker) Options() broker.Options {
	return b.options
}

func (b *pubsubBroker) Address() string {
	return b.addr
}

func (b *pubsubBroker) Init(opts ...broker.Option) error {
	if b.pool != nil {
		return errors.New("redis: cannot init while connected")
	}

	b.options.Apply(opts...)

	if v, ok := b.options.Context.Value(redisOption.OptionsKey).(*redisOption.CommonOptions); ok && v != nil {
		b.commonOpts = v
	}

	b.addr = normalizeAddr(b.options.Addrs)

	return nil
}

func normalizeAddr(addressList []string) string {
	if len(addressList) == 0 || addressList[0] == "" {
		return defaultBroker
	}

	addr := addressList[0]
	if !strings.HasPrefix(addr, "redis://") {
		addr = "redis://" + addr
	}

	return addr
}

func (b *pubsubBroker) Connect() error {
	if b.pool != nil {
		return nil
	}

	if b.addr == "" {
		b.addr = normalizeAddr(b.options.Addrs)
	}

	b.pool = &redis.Pool{
		MaxIdle:     b.commonOpts.MaxIdle,
		MaxActive:   b.commonOpts.MaxActive,
		IdleTimeout: b.commonOpts.IdleTimeout,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(
				b.addr,
				redis.DialConnectTimeout(b.commonOpts.ConnectTimeout),
				redis.DialReadTimeout(redisOption.DefaultHealthCheckPeriod+b.commonOpts.ReadTimeout),
				redis.DialWriteTimeout(b.commonOpts.WriteTimeout),
				redis.DialPassword(b.commonOpts.Password),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if nil != err {
				redisOption.LogError("ping error:" + err.Error())
			}
			return err
		},
	}

	return nil
}

func (b *pubsubBroker) Disconnect() error {
	if b.pool == nil {
		return nil
	}

	err := b.pool.Close()
	b.pool = nil
	b.addr = ""

	b.subscribers.Clear()

	return err
}

func (b *pubsubBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return broker.GenericRequest(ctx, b, topic, msg, opts...)
}

func (b *pubsubBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *pubsubBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

func (b *pubsubBroker) publish(_ context.Context, topic string, msg *broker.Message, _ ...broker.PublishOption) error {
	if b.pool == nil {
		return errors.New("redis: not connected")
	}
	conn := b.pool.Get()
	_, err := redis.Int(conn.Do("PUBLISH", topic, msg.BodyBytes()))
	_ = conn.Close()
	return err
}

func (b *pubsubBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if b.pool == nil {
		return nil, errors.New("redis: not connected")
	}

	options := broker.SubscribeOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
	}

	sub := &subscriber{
		b:       b,
		conn:    &redis.PubSubConn{Conn: b.pool.Get()},
		topic:   topic,
		handler: handler,
		binder:  binder,
		options: options,
	}

	if err := sub.conn.Subscribe(sub.topic); err != nil {
		_ = sub.conn.Close()
		return nil, err
	}

	if old := b.subscribers.Get(topic); old != nil {
		// 同主题重复订阅：先退订旧订阅，避免旧订阅继续消费（泄漏 + 重复消费）
		if uerr := old.Unsubscribe(false); uerr != nil {
			redisOption.LogWarnf("unsubscribe old subscriber for topic %q failed: %v", topic, uerr)
		}
	}
	b.subscribers.Add(topic, sub)

	go sub.recv()

	return sub, nil
}
