package broker

import (
	"context"
	"crypto/tls"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos/kratos/v2/encoding"
)

var (
	DefaultCodec      encoding.Codec = nil
	DefaultTracerName                = "kratos-broker"
)

///////////////////////////////////////////////////////////////////////////////

type TracingOptions struct {
	TracerProvider trace.TracerProvider
	Propagators    propagation.TextMapPropagator
	Tracer         trace.Tracer
}

///////////////////////////////////////////////////////////////////////////////

type Options struct {
	Addrs []string

	Codec encoding.Codec

	ErrorHandler Handler

	Secure    bool
	TLSConfig *tls.Config

	Context context.Context

	Tracer TracingOptions
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

func WithOptionContext(ctx context.Context) Option {
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

func WithAddress(addressList ...string) Option {
	return func(o *Options) {
		o.Addrs = addressList
	}
}

func WithCodec(codec encoding.Codec) Option {
	return func(o *Options) {
		o.Codec = codec
	}
}

func WithErrorHandler(handler Handler) Option {
	return func(o *Options) {
		o.ErrorHandler = handler
	}
}

func WithEnableSecure(enable bool) Option {
	return func(o *Options) {
		o.Secure = enable
	}
}

func WithTLSConfig(config *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = config
	}
}

func WithTracerProvider(provider trace.TracerProvider, tracerName string) Option {
	return func(opt *Options) {
		if provider != nil {
			opt.Tracer.TracerProvider = provider
		} else {
			opt.Tracer.TracerProvider = otel.GetTracerProvider()
		}

		if opt.Tracer.Propagators == nil {
			opt.Tracer.Propagators = otel.GetTextMapPropagator()
		}

		if len(tracerName) == 0 {
			tracerName = DefaultTracerName
		}

		opt.Tracer.Tracer = opt.Tracer.TracerProvider.Tracer(tracerName)
	}
}

func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return func(opt *Options) {
		if propagators != nil {
			opt.Tracer.Propagators = propagators
		} else {
			opt.Tracer.Propagators = otel.GetTextMapPropagator()
		}
		if opt.Tracer.TracerProvider == nil {
			opt.Tracer.TracerProvider = otel.GetTracerProvider()
			opt.Tracer.Tracer = opt.Tracer.TracerProvider.Tracer(DefaultTracerName)
		}
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

func PublishContextWithValue(k, v interface{}) PublishOption {
	return func(o *PublishOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func WithPublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		o.Context = ctx
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

func SubscribeContextWithValue(k, v interface{}) SubscribeOption {
	return func(o *SubscribeOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func DisableAutoAck() SubscribeOption {
	return func(o *SubscribeOptions) {
		o.AutoAck = false
	}
}

func WithQueueName(name string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Queue = name
	}
}

func WithSubscribeContext(ctx context.Context) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Context = ctx
	}
}

///////////////////////////////////////////////////////////////////////////////

type TracingOption func(*TracingOptions)

func (o *TracingOptions) Apply(opts ...TracingOption) {
	for _, opt := range opts {
		opt(o)
	}
}

func NewTracingOptions(opts ...TracingOption) TracingOptions {
	opt := TracingOptions{
		Propagators:    otel.GetTextMapPropagator(),
		TracerProvider: otel.GetTracerProvider(),
	}

	opt.Apply(opts...)

	opt.Tracer = opt.TracerProvider.Tracer(DefaultTracerName)

	return opt
}
