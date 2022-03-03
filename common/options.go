package common

import (
	"context"
	"crypto/tls"
	"github.com/tx7do/kratos-transport/codec"
)

type PublishOptions struct {
	Context context.Context
}

type PublishOption func(*PublishOptions)

func PublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		o.Context = ctx
	}
}

type Options struct {
	Addrs  []string
	Secure bool
	Codec  codec.Marshaler

	ErrorHandler Handler

	TLSConfig *tls.Config
	Context   context.Context
}

type Option func(*Options)

func OptionContext(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

func Codec(c codec.Marshaler) Option {
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

type SubscribeOptions struct {
	AutoAck bool
	Queue   string
	Context context.Context
}

type SubscribeOption func(*SubscribeOptions)

func NewSubscribeOptions(opts ...SubscribeOption) SubscribeOptions {
	opt := SubscribeOptions{
		AutoAck: true,
	}

	for _, o := range opts {
		o(&opt)
	}

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
