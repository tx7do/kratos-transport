package redis

import (
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/redis/option"
)

// DriverType 驱动类型别名，保持向后兼容
type DriverType = option.DriverType

const (
	DriverTypePubSub = option.DriverTypePubSub
	DriverTypeStream = option.DriverTypeStream
)

// WithConnectTimeout 连接Redis超时时间
func WithConnectTimeout(d time.Duration) broker.Option {
	return option.WithConnectTimeout(d)
}

// WithReadTimeout 从Redis读取数据超时时间
func WithReadTimeout(d time.Duration) broker.Option {
	return option.WithReadTimeout(d)
}

// WithWriteTimeout 向Redis写入数据超时时间
func WithWriteTimeout(d time.Duration) broker.Option {
	return option.WithWriteTimeout(d)
}

// WithIdleTimeout 最大的空闲连接等待时间
func WithIdleTimeout(d time.Duration) broker.Option {
	return option.WithIdleTimeout(d)
}

// WithMaxIdle 最大的空闲连接数
func WithMaxIdle(n int) broker.Option {
	return option.WithMaxIdle(n)
}

// WithMaxActive 最大的连接数
func WithMaxActive(n int) broker.Option {
	return option.WithMaxActive(n)
}

// WithPassword 设置密码
func WithPassword(password string) broker.Option {
	return option.WithPassword(password)
}

// WithDefaultOptions 全部置为默认的配置
func WithDefaultOptions() broker.Option {
	return option.WithDefaultOptions()
}

///
/// logger 转发
///

// SetLogger 注入库内部日志使用的 logger
func SetLogger(l log.Logger) {
	option.SetLogger(l)
}

func LogDebug(args ...any) {
	option.LogDebug(args...)
}

func LogInfo(args ...any) {
	option.LogInfo(args...)
}

func LogWarn(args ...any) {
	option.LogWarn(args...)
}

func LogError(args ...any) {
	option.LogError(args...)
}

func LogFatal(args ...any) {
	option.LogFatal(args...)
}

func LogDebugf(format string, args ...any) {
	option.LogDebugf(format, args...)
}

func LogInfof(format string, args ...any) {
	option.LogInfof(format, args...)
}

func LogWarnf(format string, args ...any) {
	option.LogWarnf(format, args...)
}

func LogErrorf(format string, args ...any) {
	option.LogErrorf(format, args...)
}

func LogFatalf(format string, args ...any) {
	option.LogFatalf(format, args...)
}

///
/// Stream 专属配置 转发
///

// WithStreamGroup Redis Stream 消费组名称
func WithStreamGroup(group string) broker.SubscribeOption {
	return option.WithStreamGroup(group)
}

// WithStreamConsumer Redis Stream 消费者名称
func WithStreamConsumer(consumer string) broker.SubscribeOption {
	return option.WithStreamConsumer(consumer)
}

// WithStreamBlockTime Redis Stream XREADGROUP 阻塞等待时间
func WithStreamBlockTime(d time.Duration) broker.SubscribeOption {
	return option.WithStreamBlockTime(d)
}

// WithStreamCount Redis Stream 每次读取的最大消息数
func WithStreamCount(n int) broker.SubscribeOption {
	return option.WithStreamCount(n)
}

// WithStreamMaxLen Redis Stream XADD 时的 MAXLEN 限制
func WithStreamMaxLen(n int64) broker.PublishOption {
	return option.WithStreamMaxLen(n)
}
