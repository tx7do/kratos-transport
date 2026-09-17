package stream

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
	redisOption "github.com/tx7do/kratos-transport/broker/redis/option"
)

const (
	defaultBroker = "redis://127.0.0.1:6379"
)

type streamBroker struct {
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

	return &streamBroker{
		options:     options,
		commonOpts:  commonOpts,
		subscribers: broker.NewSubscriberSyncMap(),
	}
}

func (b *streamBroker) Name() string {
	return "redis-stream"
}

func (b *streamBroker) Options() broker.Options {
	return b.options
}

func (b *streamBroker) Address() string {
	return b.addr
}

func (b *streamBroker) Init(opts ...broker.Option) error {
	if b.pool != nil {
		return errors.New("redis-stream: cannot init while connected")
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

	if v, ok := b.options.Context.Value(redisOption.OptionsKey).(*redisOption.CommonOptions); ok {
		b.commonOpts = v
	}

	return nil
}

func (b *streamBroker) Connect() error {
	if b.pool != nil {
		return nil
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

func (b *streamBroker) Disconnect() error {
	if b.pool == nil {
		return nil
	}
	err := b.pool.Close()
	b.pool = nil
	b.addr = ""

	b.subscribers.Clear()

	return err
}

func (b *streamBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return nil, errors.New("not implemented")
}

func (b *streamBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = b.internalPublish

	if len(b.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, b.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (b *streamBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(b.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return b.publish(ctx, topic, sendMsg, opts...)
}

// publish 使用 XADD 命令将消息写入 Redis Stream
func (b *streamBroker) publish(_ context.Context, stream string, msg *broker.Message, opts ...broker.PublishOption) error {
	publishOpts := broker.PublishOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&publishOpts)
	}

	conn := b.pool.Get()
	defer conn.Close()

	args := []any{stream}

	// MAXLEN 限制
	var maxLen int64
	if v, ok := publishOpts.Context.Value(redisOption.StreamMaxLenKey{}).(int64); ok {
		maxLen = v
	}
	if maxLen > 0 {
		args = append(args, "MAXLEN", "~", maxLen)
	}

	args = append(args, "*")
	args = append(args, "body", msg.BodyBytes())

	// 附加消息头（值转为 string）
	for k, v := range msg.Headers {
		args = append(args, k, fmt.Sprintf("%v", v))
	}

	_, err := conn.Do("XADD", args...)
	return err
}

func (b *streamBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	subOpts := broker.SubscribeOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&subOpts)
	}

	if len(b.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, b.options.SubscriberMiddlewares)
	}

	// 提取 Stream 专属配置
	group := redisOption.DefaultStreamGroup
	if v, ok := subOpts.Context.Value(redisOption.StreamGroupKey{}).(string); ok && v != "" {
		group = v
	}

	consumer := redisOption.DefaultStreamConsumer
	if v, ok := subOpts.Context.Value(redisOption.StreamConsumerKey{}).(string); ok && v != "" {
		consumer = v
	}

	blockTime := redisOption.DefaultStreamBlockTime
	if v, ok := subOpts.Context.Value(redisOption.StreamBlockTimeKey{}).(time.Duration); ok && v > 0 {
		blockTime = v
	}

	count := redisOption.DefaultStreamCount
	if v, ok := subOpts.Context.Value(redisOption.StreamCountKey{}).(int); ok && v > 0 {
		count = v
	}

	// 确保消费组存在
	if err := b.ensureGroup(topic, group); err != nil {
		return nil, err
	}

	sub := &subscriber{
		b:         b,
		topic:     topic,
		group:     group,
		consumer:  consumer,
		blockTime: blockTime,
		count:     count,
		handler:   handler,
		binder:    binder,
		options:   subOpts,
	}

	b.subscribers.Add(topic, sub)

	go sub.recv()

	return sub, nil
}

// ensureGroup 确保消费组存在，不存在则创建
func (b *streamBroker) ensureGroup(stream, group string) error {
	conn := b.pool.Get()
	defer conn.Close()

	// XGROUP CREATE stream group $ MKSTREAM
	_, err := conn.Do("XGROUP", "CREATE", stream, group, "$", "MKSTREAM")
	if err != nil {
		// BUSYGROUP: Consumer Group name already exists 是正常的
		if strings.Contains(err.Error(), "BUSYGROUP") {
			return nil
		}
		return err
	}
	return nil
}
