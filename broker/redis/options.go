package redis

import (
	"github.com/tx7do/kratos-transport/common"
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

func ConnectTimeout(d time.Duration) common.Option {
	return func(o *common.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.connectTimeout = d
	}
}

func ReadTimeout(d time.Duration) common.Option {
	return func(o *common.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.readTimeout = d
	}
}

func WriteTimeout(d time.Duration) common.Option {
	return func(o *common.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.writeTimeout = d
	}
}

func MaxIdle(n int) common.Option {
	return func(o *common.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.maxIdle = n
	}
}

func MaxActive(n int) common.Option {
	return func(o *common.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.maxActive = n
	}
}

func IdleTimeout(d time.Duration) common.Option {
	return func(o *common.Options) {
		bo := o.Context.Value(optionsKey).(*commonOptions)
		bo.idleTimeout = d
	}
}
