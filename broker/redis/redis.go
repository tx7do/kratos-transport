package redis

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
)

const (
	defaultBroker = "redis://127.0.0.1:6379"
)

type redisBroker struct {
	addr  string
	pool  *redis.Pool
	opts  broker.Options
	bOpts *commonOptions
}

// NewBroker returns a new common implemented using the Redis pub/sub
// protocol. The connection address may be a fully qualified IANA address such
// as: redis://user:secret@localhost:6379/0?foo=bar&qux=baz
func NewBroker(opts ...broker.Option) broker.Broker {
	bOpts := &commonOptions{
		maxIdle:        DefaultMaxIdle,
		maxActive:      DefaultMaxActive,
		idleTimeout:    DefaultIdleTimeout,
		connectTimeout: DefaultConnectTimeout,
		readTimeout:    DefaultReadTimeout,
		writeTimeout:   DefaultWriteTimeout,
	}

	options := broker.NewOptions()

	opts = append(opts, WithCommonOptions())
	options.Apply(opts...)

	return &redisBroker{
		opts:  options,
		bOpts: bOpts,
	}
}

func (b *redisBroker) Name() string {
	return "redis"
}

func (b *redisBroker) Options() broker.Options {
	return b.opts
}

func (b *redisBroker) Address() string {
	return b.addr
}

func (b *redisBroker) Init(opts ...broker.Option) error {
	if b.pool != nil {
		return errors.New("redis: cannot init while connected")
	}

	for _, o := range opts {
		o(&b.opts)
	}

	return nil
}

func (b *redisBroker) Connect() error {
	if b.pool != nil {
		return nil
	}

	var addr string

	if len(b.opts.Addrs) == 0 || b.opts.Addrs[0] == "" {
		addr = defaultBroker
	} else {
		addr = b.opts.Addrs[0]

		if !strings.HasPrefix(addr, "redis://") {
			addr = "redis://" + addr
		}
	}

	b.addr = addr

	b.pool = &redis.Pool{
		MaxIdle:     b.bOpts.maxIdle,
		MaxActive:   b.bOpts.maxActive,
		IdleTimeout: b.bOpts.idleTimeout,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(
				b.addr,
				redis.DialConnectTimeout(b.bOpts.connectTimeout),
				redis.DialReadTimeout(b.bOpts.readTimeout),
				redis.DialWriteTimeout(b.bOpts.writeTimeout),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			fmt.Printf("TestOnBorrow error: %X\n", err)
			return err
		},
	}

	return nil
}

func (b *redisBroker) Disconnect() error {
	err := b.pool.Close()
	b.pool = nil
	b.addr = ""
	return err
}

func (b *redisBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	if b.opts.Codec != nil {
		var err error
		buf, err := b.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
		return b.publish(topic, buf, opts...)
	} else {
		switch t := msg.(type) {
		case []byte:
			return b.publish(topic, t, opts...)
		case string:
			return b.publish(topic, []byte(t), opts...)
		default:
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(msg); err != nil {
				return err
			}
			return b.publish(topic, buf.Bytes(), opts...)
		}
	}
}

func (b *redisBroker) publish(topic string, msg []byte, _ ...broker.PublishOption) error {
	conn := b.pool.Get()
	_, err := redis.Int(conn.Do("PUBLISH", topic, msg))
	_ = conn.Close()
	return err
}

func (b *redisBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	s := subscriber{
		codec:   b.opts.Codec,
		conn:    &redis.PubSubConn{Conn: b.pool.Get()},
		topic:   topic,
		handler: handler,
		binder:  binder,
		opts:    options,
	}

	go s.recv()

	if err := s.conn.Subscribe(s.topic); err != nil {
		return nil, err
	}

	return &s, nil
}
