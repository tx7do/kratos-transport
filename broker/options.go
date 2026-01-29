package broker

import (
	"context"
	"crypto/tls"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"
	_ "github.com/go-kratos/kratos/v2/encoding/proto"

	"github.com/tx7do/kratos-transport/tracing"
)

var (
	// DefaultCodec is the default codec for broker
	DefaultCodec = encoding.GetCodec("json")
)

///////////////////////////////////////////////////////////////////////////////

// Options broker options
type Options struct {
	// Addrs is a Broker addresses
	Addrs []string

	// Codec is a Broker codec
	Codec encoding.Codec

	// ErrorHandler is a Broker error handler
	ErrorHandler Handler

	// Secure enable secure connection
	Secure bool
	// TLSConfig is tls config for secure connection
	TLSConfig *tls.Config

	// Context is broker option context
	Context context.Context

	// Tracings are tracing options
	Tracings []tracing.Option

	// SubscriberMiddlewares applies to subscribe handlers
	SubscriberMiddlewares []SubscriberMiddleware

	// PublishMiddlewares applies to publish handlers
	PublishMiddlewares []PublishMiddleware
}

// Option defines a function which sets some option.
type Option func(*Options)

// Apply applies all options to the Options.
func (o *Options) Apply(opts ...Option) {
	if o == nil {
		return
	}
	for _, opt := range opts {
		opt(o)
	}
}

// NewOptions creates default Options.
func NewOptions() Options {
	opt := Options{
		Addrs: []string{},
		Codec: DefaultCodec,

		ErrorHandler: nil,

		Secure:    false,
		TLSConfig: nil,

		Context: context.Background(),

		Tracings: []tracing.Option{},
	}

	return opt
}

// NewOptionsAndApply creates Options and applies given Option functions.
func NewOptionsAndApply(opts ...Option) Options {
	opt := NewOptions()
	opt.Apply(opts...)
	return opt
}

// WithOptionContext sets the broker option context
func WithOptionContext(ctx context.Context) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Context = ctx
	}
}

// OptionContextWithValue sets a value in the broker option context
func OptionContextWithValue(k, v any) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// WithAddress set broker address
func WithAddress(addressList ...string) Option {
	addrsCopy := append([]string(nil), addressList...)
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Addrs = addrsCopy
	}
}

// WithCodec set codec, support: json, proto.
func WithCodec(name string) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Codec = encoding.GetCodec(name)
	}
}

// WithErrorHandler sets error handler
func WithErrorHandler(handler Handler) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.ErrorHandler = handler
	}
}

// WithEnableSecure sets enable secure connection
func WithEnableSecure(enable bool) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Secure = enable
	}
}

// WithTLSConfig sets tls config for secure connection
func WithTLSConfig(config *tls.Config) Option {
	return func(o *Options) {
		if o == nil {
			return
		}

		o.TLSConfig = config
		if o.TLSConfig != nil {
			o.Secure = true
		}
	}
}

// WithTracerProvider sets tracer provider
func WithTracerProvider(provider trace.TracerProvider) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Tracings = append(o.Tracings, tracing.WithTracerProvider(provider))
	}
}

// WithPropagator sets propagator
func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Tracings = append(o.Tracings, tracing.WithPropagator(propagator))
	}
}

// WithGlobalTracerProvider sets global tracer provider
func WithGlobalTracerProvider() Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Tracings = append(o.Tracings, tracing.WithGlobalTracerProvider())
	}
}

// WithGlobalPropagator sets global propagator
func WithGlobalPropagator() Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Tracings = append(o.Tracings, tracing.WithGlobalPropagator())
	}
}

// WithSubscriberMiddlewares sets subscriber middlewares
func WithSubscriberMiddlewares(mws ...SubscriberMiddleware) Option {
	m := append([]SubscriberMiddleware(nil), mws...)
	return func(o *Options) {
		if o == nil {
			return
		}
		o.SubscriberMiddlewares = m
	}
}

// WithPublishMiddlewares sets publish middlewares
func WithPublishMiddlewares(mws ...PublishMiddleware) Option {
	m := append([]PublishMiddleware(nil), mws...)
	return func(o *Options) {
		if o == nil {
			return
		}
		o.PublishMiddlewares = m
	}
}
