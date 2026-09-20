package option

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
)

type OptionsKeyType struct{}

type DriverType string

const (
	DriverTypePubSub DriverType = "pubsub"
	DriverTypeStream DriverType = "stream"
)

const (
	DefaultMaxActive         = 0
	DefaultMaxIdle           = 256
	DefaultIdleTimeout       = time.Duration(0)
	DefaultConnectTimeout    = 30 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultWriteTimeout      = 30 * time.Second
	DefaultHealthCheckPeriod = time.Minute

	// Stream 默认配置
	DefaultStreamGroup     = "kratos-group"
	DefaultStreamConsumer  = "kratos-consumer"
	DefaultStreamBlockTime = 5 * time.Second
	DefaultStreamCount     = 10
	DefaultStreamMaxLen    = 0
)

var OptionsKey = OptionsKeyType{}

// CommonOptions Redis连接池共享配置
// 子包（pubsub/stream）通过此类型读取连接池配置
type CommonOptions struct {
	MaxIdle        int
	MaxActive      int
	IdleTimeout    time.Duration
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	Password       string
}

///
/// Option
///

// defaultCommonOptions 返回填好默认值的连接池配置。
// 各 With* 首次触发时以此为基础，避免只带单字段的零值结构覆盖掉其余默认值。
func defaultCommonOptions() *CommonOptions {
	return &CommonOptions{
		MaxIdle:        DefaultMaxIdle,
		MaxActive:      DefaultMaxActive,
		IdleTimeout:    DefaultIdleTimeout,
		ConnectTimeout: DefaultConnectTimeout,
		ReadTimeout:    DefaultReadTimeout,
		WriteTimeout:   DefaultWriteTimeout,
	}
}

// commonOptions 取出（或创建）挂在 broker Options.Context 上的连接池配置
func commonOptions(o *broker.Options) *CommonOptions {
	if o.Context == nil {
		o.Context = context.Background()
	}
	if x, ok := o.Context.Value(OptionsKey).(*CommonOptions); ok && x != nil {
		return x
	}
	opts := defaultCommonOptions()
	o.Context = context.WithValue(o.Context, OptionsKey, opts)
	return opts
}

// WithConnectTimeout 连接Redis超时时间
func WithConnectTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		commonOptions(o).ConnectTimeout = d
	}
}

// WithReadTimeout 从Redis读取数据超时时间
func WithReadTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		commonOptions(o).ReadTimeout = d
	}
}

// WithWriteTimeout 向Redis写入数据超时时间
func WithWriteTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		commonOptions(o).WriteTimeout = d
	}
}

// WithIdleTimeout 最大的空闲连接等待时间，超过此时间后，空闲连接将被关闭。如果设置成0，空闲连接将不会被关闭。应该设置一个比redis服务端超时时间更短的时间。
func WithIdleTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		commonOptions(o).IdleTimeout = d
	}
}

// WithMaxIdle 最大的空闲连接数，表示即使没有redis连接时依然可以保持N个空闲的连接，而不被清除，随时处于待命状态。
func WithMaxIdle(n int) broker.Option {
	return func(o *broker.Options) {
		commonOptions(o).MaxIdle = n
	}
}

// WithMaxActive 最大的连接数，表示同时最多有N个连接。0表示不限制。
func WithMaxActive(n int) broker.Option {
	return func(o *broker.Options) {
		commonOptions(o).MaxActive = n
	}
}

// WithPassword 密码
func WithPassword(password string) broker.Option {
	return func(o *broker.Options) {
		// 走统一入口：单独设置密码时也保留其余默认值（零值结构会导致连接池配置丢失）
		commonOptions(o).Password = password
	}
}

// WithDefaultOptions 全部置为默认的配置
func WithDefaultOptions() broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, OptionsKey, defaultCommonOptions())
	}
}

///
/// logger
///

const (
	logKey = "[redis]"
)

func LogDebug(args ...any) {
	log.Debugf("%s %s", logKey, fmt.Sprint(args...))
}

func LogInfo(args ...any) {
	log.Infof("%s %s", logKey, fmt.Sprint(args...))
}

func LogWarn(args ...any) {
	log.Warnf("%s %s", logKey, fmt.Sprint(args...))
}

func LogError(args ...any) {
	log.Errorf("%s %s", logKey, fmt.Sprint(args...))
}

func LogFatal(args ...any) {
	log.Fatalf("%s %s", logKey, fmt.Sprint(args...))
}

func LogDebugf(format string, args ...any) {
	log.Debugf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogInfof(format string, args ...any) {
	log.Infof("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogWarnf(format string, args ...any) {
	log.Warnf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogErrorf(format string, args ...any) {
	log.Errorf("%s %s", logKey, fmt.Sprintf(format, args...))
}

func LogFatalf(format string, args ...any) {
	log.Fatalf("%s %s", logKey, fmt.Sprintf(format, args...))
}

///
/// Stream SubscribeOption
///

type StreamGroupKey struct{}
type StreamConsumerKey struct{}
type StreamBlockTimeKey struct{}
type StreamCountKey struct{}
type StreamMaxLenKey struct{}

// WithStreamGroup Redis Stream 消费组名称
func WithStreamGroup(group string) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(StreamGroupKey{}, group)
}

// WithStreamConsumer Redis Stream 消费者名称
func WithStreamConsumer(consumer string) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(StreamConsumerKey{}, consumer)
}

// WithStreamBlockTime Redis Stream XREADGROUP 阻塞等待时间
func WithStreamBlockTime(d time.Duration) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(StreamBlockTimeKey{}, d)
}

// WithStreamCount Redis Stream 每次读取的最大消息数
func WithStreamCount(n int) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(StreamCountKey{}, n)
}

// WithStreamMaxLen Redis Stream XADD 时的 MAXLEN 限制，0 表示不限制
func WithStreamMaxLen(n int64) broker.PublishOption {
	return broker.PublishContextWithValue(StreamMaxLenKey{}, n)
}
