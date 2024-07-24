package redis

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultBroker = "redis://127.0.0.1:6379"
)

type redisBroker struct {
	addr       string
	pool       *redis.Pool
	options    broker.Options
	commonOpts *commonOptions

	subscribers *broker.SubscriberSyncMap
}

// NewBroker returns a new common implemented using the Redis pub/sub
// protocol. The connection address may be a fully qualified IANA address such
// as: redis://user:secret@localhost:6379/0?foo=bar&qux=baz
func NewBroker(opts ...broker.Option) broker.Broker {
	commonOpts := &commonOptions{
		maxIdle:        DefaultMaxIdle,
		maxActive:      DefaultMaxActive,
		idleTimeout:    DefaultIdleTimeout,
		connectTimeout: DefaultConnectTimeout,
		readTimeout:    DefaultReadTimeout,
		writeTimeout:   DefaultWriteTimeout,
	}

	options := broker.NewOptionsAndApply(opts...)

	return &redisBroker{
		options:     options,
		commonOpts:  commonOpts,
		subscribers: broker.NewSubscriberSyncMap(),
	}
}

func (b *redisBroker) Name() string {
	return "redis"
}

func (b *redisBroker) Options() broker.Options {
	return b.options
}

func (b *redisBroker) Address() string {
	return b.addr
}

func (b *redisBroker) Init(opts ...broker.Option) error {
	if b.pool != nil {
		return errors.New("redis: cannot init while connected")
	}

	var addr string

	if len(b.options.Addrs) == 0 || b.options.Addrs[0] == "" {
		addr = defaultBroker
	} else {
		addr = b.options.Addrs[0]

		if !strings.HasPrefix(addr, "redis://") {
			addr = "redis://" + addr
		}
	}

	b.addr = addr

	b.options.Apply(opts...)

	if v, ok := b.options.Context.Value(optionsKey).(*commonOptions); ok {
		b.commonOpts = v
	}

	return nil
}

func (b *redisBroker) Connect() error {
	if b.pool != nil {
		return nil
	}

	b.pool = &redis.Pool{
		MaxIdle:     b.commonOpts.maxIdle,
		MaxActive:   b.commonOpts.maxActive,
		IdleTimeout: b.commonOpts.idleTimeout,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(
				b.addr,
				redis.DialConnectTimeout(b.commonOpts.connectTimeout),
				redis.DialReadTimeout(DefaultHealthCheckPeriod+b.commonOpts.readTimeout),
				redis.DialWriteTimeout(b.commonOpts.writeTimeout),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			if nil != err {
				log.Error("[redis] ping error:" + err.Error())
			}
			return err
		},
	}

	return nil
}

func (b *redisBroker) Disconnect() error {
	err := b.pool.Close()
	b.pool = nil
	b.addr = ""

	b.subscribers.Clear()

	return err
}

func (b *redisBroker) Request(ctx context.Context, topic string, msg broker.Any, opts ...broker.RequestOption) (broker.Any, error) {
	return nil, errors.New("not implemented")
}

func (b *redisBroker) Publish(ctx context.Context, topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg)
	if err != nil {
		return err
	}

	return b.publish(ctx, topic, buf, opts...)
}

func (b *redisBroker) publish(_ context.Context, topic string, msg []byte, _ ...broker.PublishOption) error {
	conn := b.pool.Get()
	_, err := redis.Int(conn.Do("PUBLISH", topic, msg))
	_ = conn.Close()
	return err
}

func (b *redisBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.SubscribeOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
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
		return nil, err
	}

	b.subscribers.Add(topic, sub)

	go sub.recv()

	return sub, nil
}
