package redis

import (
	"github.com/tx7do/kratos-transport/broker"
	"time"
)

var (
	DefaultMaxActive      = 0
	DefaultMaxIdle        = 5
	DefaultIdleTimeout    = 2 * time.Minute
	DefaultConnectTimeout = 5 * time.Second
	DefaultReadTimeout    = 5 * time.Second
	DefaultWriteTimeout   = 5 * time.Second

	optionsKey = optionsKeyType{}
)

// options contain additional options for the common.
type commonOptions struct {
	maxIdle        int
	maxActive      int
	idleTimeout    time.Duration
	connectTimeout time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

type optionsKeyType struct{}

func ConnectTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.connectTimeout = d
	}
}

func ReadTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.readTimeout = d
	}
}

func WriteTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.writeTimeout = d
	}
}

func MaxIdle(n int) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.maxIdle = n
	}
}

func MaxActive(n int) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.maxActive = n
	}
}

func IdleTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.idleTimeout = d
	}
}
