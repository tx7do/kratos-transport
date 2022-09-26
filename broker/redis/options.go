package redis

import (
	"context"
	"time"

	"github.com/tx7do/kratos-transport/broker"
)

type optionsKeyType struct{}

var (
	DefaultMaxActive         = 0
	DefaultMaxIdle           = 256
	DefaultIdleTimeout       = time.Duration(0)
	DefaultConnectTimeout    = 30 * time.Second
	DefaultReadTimeout       = 30 * time.Second
	DefaultWriteTimeout      = 30 * time.Second
	DefaultHealthCheckPeriod = time.Minute

	optionsKey = optionsKeyType{}
)

type commonOptions struct {
	maxIdle        int
	maxActive      int
	idleTimeout    time.Duration
	connectTimeout time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

// WithConnectTimeout 连接Redis超时时间
func WithConnectTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).connectTimeout = d
		} else {
			o.Context = context.WithValue(o.Context, optionsKey, &commonOptions{connectTimeout: d})
		}
	}
}

// WithReadTimeout 从Redis读取数据超时时间
func WithReadTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).readTimeout = d
		} else {
			o.Context = context.WithValue(o.Context, optionsKey, &commonOptions{readTimeout: d})
		}
	}
}

// WithWriteTimeout 向Redis写入数据超时时间
func WithWriteTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).writeTimeout = d
		} else {
			o.Context = context.WithValue(o.Context, optionsKey, &commonOptions{writeTimeout: d})
		}
	}
}

// WithIdleTimeout 最大的空闲连接等待时间，超过此时间后，空闲连接将被关闭。如果设置成0，空闲连接将不会被关闭。应该设置一个比redis服务端超时时间更短的时间。
func WithIdleTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).idleTimeout = d
		} else {
			o.Context = context.WithValue(o.Context, optionsKey, &commonOptions{idleTimeout: d})
		}
	}
}

// WithMaxIdle 最大的空闲连接数，表示即使没有redis连接时依然可以保持N个空闲的连接，而不被清除，随时处于待命状态。
func WithMaxIdle(n int) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).maxIdle = n
		} else {
			o.Context = context.WithValue(o.Context, optionsKey, &commonOptions{maxIdle: n})
		}
	}
}

// WithMaxActive 最大的连接数，表示同时最多有N个连接。0表示不限制。
func WithMaxActive(n int) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).maxActive = n
		} else {
			o.Context = context.WithValue(o.Context, optionsKey, &commonOptions{maxActive: n})
		}
	}
}

// WithDefaultOptions 全部置为默认的配置
func WithDefaultOptions() broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)

		opts := &commonOptions{
			maxIdle:        DefaultMaxIdle,
			maxActive:      DefaultMaxActive,
			idleTimeout:    DefaultIdleTimeout,
			connectTimeout: DefaultConnectTimeout,
			readTimeout:    DefaultReadTimeout,
			writeTimeout:   DefaultWriteTimeout,
		}

		if x != nil {
			x = opts
		} else {
			o.Context = context.WithValue(o.Context, optionsKey, opts)
		}
	}
}
