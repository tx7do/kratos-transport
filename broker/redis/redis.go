package redis

import (
	"context"
	"errors"
	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/codec"
	"github.com/tx7do/kratos-transport/common"
	"strings"
	"time"
)

// publication is an internal publication for the Redis common.
type publication struct {
	topic   string
	message *broker.Message
	err     error
}

// Topic returns the topic this publication applies to.
func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.message
}

func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}

type subscriber struct {
	codec  codec.Marshaler
	conn   *redis.PubSubConn
	topic  string
	handle broker.Handler
	opts   common.SubscribeOptions
}

func (s *subscriber) recv() {
	defer func(conn *redis.PubSubConn) {
		err := conn.Close()
		if err != nil {

		}
	}(s.conn)

	for {
		switch x := s.conn.Receive().(type) {
		case redis.Message:
			var m broker.Message

			if s.codec != nil {
				if err := s.codec.Unmarshal(x.Data, &m); err != nil {
					break
				}
			} else {
				m.Body = x.Data
			}

			p := publication{
				topic:   x.Channel,
				message: &m,
			}

			// Handle error? Retry?
			if p.err = s.handle(&p); p.err != nil {
				break
			}

			// Added for posterity, however Ack is a no-op.
			if s.opts.AutoAck {
				if err := p.Ack(); err != nil {
					break
				}
			}

		case redis.Subscription:
			if x.Count == 0 {
				return
			}

		case error:
			return
		}
	}
}

// Options returns the subscriber options.
func (s *subscriber) Options() common.SubscribeOptions {
	return s.opts
}

// Topic returns the topic of the subscriber.
func (s *subscriber) Topic() string {
	return s.topic
}

// Unsubscribe unsubscribes the subscriber and frees the connection.
func (s *subscriber) Unsubscribe() error {
	return s.conn.Unsubscribe()
}

// common implementation for Redis.
type redisBroker struct {
	addr  string
	pool  *redis.Pool
	opts  common.Options
	bopts *commonOptions
}

func (b *redisBroker) String() string {
	return "redis"
}

func (b *redisBroker) Options() common.Options {
	return b.opts
}

func (b *redisBroker) Address() string {
	return b.addr
}

// Init sets or overrides common options.
func (b *redisBroker) Init(opts ...common.Option) error {
	if b.pool != nil {
		return errors.New("redis: cannot init while connected")
	}

	for _, o := range opts {
		o(&b.opts)
	}

	return nil
}

// Connect establishes a connection to Redis which provides the
// pub/sub implementation.
func (b *redisBroker) Connect() error {
	if b.pool != nil {
		return nil
	}

	var addr string

	if len(b.opts.Addrs) == 0 || b.opts.Addrs[0] == "" {
		addr = "redis://127.0.0.1:6379"
	} else {
		addr = b.opts.Addrs[0]

		if !strings.HasPrefix("redis://", addr) {
			addr = "redis://" + addr
		}
	}

	b.addr = addr

	b.pool = &redis.Pool{
		MaxIdle:     b.bopts.maxIdle,
		MaxActive:   b.bopts.maxActive,
		IdleTimeout: b.bopts.idleTimeout,
		Dial: func() (redis.Conn, error) {
			return redis.DialURL(
				b.addr,
				redis.DialConnectTimeout(b.bopts.connectTimeout),
				redis.DialReadTimeout(b.bopts.readTimeout),
				redis.DialWriteTimeout(b.bopts.writeTimeout),
			)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}

	return nil
}

// Disconnect closes the connection pool.
func (b *redisBroker) Disconnect() error {
	err := b.pool.Close()
	b.pool = nil
	b.addr = ""
	return err
}

// Publish publishes a message.
func (b *redisBroker) Publish(topic string, msg *broker.Message, _ ...common.PublishOption) error {
	var err error
	var data []byte
	if b.opts.Codec != nil {
		data, err = b.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
	} else {
		data = msg.Body
	}

	conn := b.pool.Get()
	_, err = redis.Int(conn.Do("PUBLISH", topic, data))
	_ = conn.Close()

	return err
}

// Subscribe returns a subscriber for the topic and handler.
func (b *redisBroker) Subscribe(topic string, handler broker.Handler, opts ...common.SubscribeOption) (broker.Subscriber, error) {
	var options common.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	s := subscriber{
		codec:  b.opts.Codec,
		conn:   &redis.PubSubConn{Conn: b.pool.Get()},
		topic:  topic,
		handle: handler,
		opts:   options,
	}

	// Run the receiver routine.
	go s.recv()

	if err := s.conn.Subscribe(s.topic); err != nil {
		return nil, err
	}

	return &s, nil
}

// NewBroker returns a new common implemented using the Redis pub/sub
// protocol. The connection address may be a fully qualified IANA address such
// as: redis://user:secret@localhost:6379/0?foo=bar&qux=baz
func NewBroker(opts ...common.Option) broker.Broker {
	// Default options.
	bopts := &commonOptions{
		maxIdle:        DefaultMaxIdle,
		maxActive:      DefaultMaxActive,
		idleTimeout:    DefaultIdleTimeout,
		connectTimeout: DefaultConnectTimeout,
		readTimeout:    DefaultReadTimeout,
		writeTimeout:   DefaultWriteTimeout,
	}

	// Initialize with empty common options.
	options := common.Options{
		//Codec:   json.Marshaler{},
		Context: context.WithValue(context.Background(), optionsKey, bopts),
	}

	for _, o := range opts {
		o(&options)
	}

	return &redisBroker{
		opts:  options,
		bopts: bopts,
	}
}
