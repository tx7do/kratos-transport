package broker

import (
	"context"
	"crypto/tls"

	"github.com/go-kratos/kratos/v2/encoding"
)

//var DefaultCodec = encoding.GetCodec("json")
var DefaultCodec encoding.Codec = nil

type Options struct {
	Addrs []string
	Codec encoding.Codec

	ErrorHandler Handler

	Secure    bool
	TLSConfig *tls.Config

	Context context.Context
}

type Option func(*Options)

func (o *Options) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(o)
	}
}

func NewOptions() Options {
	opt := Options{
		Addrs: []string{},
		Codec: DefaultCodec,

		ErrorHandler: nil,

		Secure:    false,
		TLSConfig: nil,

		Context: context.Background(),
	}

	return opt
}

func NewOptionsAndApply(opts ...Option) Options {
	opt := NewOptions()
	opt.Apply(opts...)
	return opt
}

func OptionContext(ctx context.Context) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = ctx
		}
	}
}

func OptionContextWithValue(k, v interface{}) Option {
	return func(o *Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

func Codec(c encoding.Codec) Option {
	return func(o *Options) {
		o.Codec = c
	}
}

func ErrorHandler(h Handler) Option {
	return func(o *Options) {
		o.ErrorHandler = h
	}
}

func Secure(b bool) Option {
	return func(o *Options) {
		o.Secure = b
	}
}

func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

///////////////////////////////////////////////////////////////////////////////

type PublishOptions struct {
	Context context.Context
}

type PublishOption func(*PublishOptions)

func (o *PublishOptions) Apply(opts ...PublishOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func NewPublishOptions(opts ...PublishOption) PublishOptions {
	opt := PublishOptions{
		Context: context.Background(),
	}

	opt.Apply(opts...)

	return opt
}

func PublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		o.Context = ctx
	}
}

func PublishContextWithValue(k, v interface{}) PublishOption {
	return func(o *PublishOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

///////////////////////////////////////////////////////////////////////////////

type SubscribeOptions struct {
	AutoAck bool
	Queue   string
	Context context.Context
}

type SubscribeOption func(*SubscribeOptions)

func (o *SubscribeOptions) Apply(opts ...SubscribeOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func NewSubscribeOptions(opts ...SubscribeOption) SubscribeOptions {
	opt := SubscribeOptions{
		AutoAck: true,
		Queue:   "",
		Context: context.Background(),
	}

	opt.Apply(opts...)

	return opt
}

func DisableAutoAck() SubscribeOption {
	return func(o *SubscribeOptions) {
		o.AutoAck = false
	}
}

func Queue(name string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Queue = name
	}
}

func SubscribeContext(ctx context.Context) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Context = ctx
	}
}

func SubscribeContextWithValue(k, v interface{}) SubscribeOption {
	return func(o *SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}
