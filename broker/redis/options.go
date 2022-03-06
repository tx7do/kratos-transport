package redis

import (
	"context"
	"fmt"
	"github.com/tx7do/kratos-transport/broker"
	"time"
)

var (
	DefaultMaxActive      = 0
	DefaultMaxIdle        = 5
	DefaultIdleTimeout    = 0 * time.Minute
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
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).connectTimeout = d
		} else {
			fmt.Println("ConnectTimeout set error")
		}
	}
}

func ReadTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).readTimeout = d
		} else {
			fmt.Println("ReadTimeout set error")
		}
	}
}

func WriteTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).writeTimeout = d
		} else {
			fmt.Println("WriteTimeout set error")
		}
	}
}

func MaxIdle(n int) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).maxIdle = n
		} else {
			fmt.Println("MaxIdle set error")
		}
	}
}

func MaxActive(n int) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).maxActive = n
		} else {
			fmt.Println("MaxActive set error")
		}
	}
}

func IdleTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		x := o.Context.Value(optionsKey)
		if x != nil {
			x.(*commonOptions).idleTimeout = d
		} else {
			fmt.Println("IdleTimeout set error")
		}
	}
}
