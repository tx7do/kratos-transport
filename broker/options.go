package broker

import (
	"context"
	"crypto/tls"

	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-kratos/kratos/v2/encoding"

	"github.com/tx7do/kratos-transport/tracing"
)

var (
	DefaultCodec encoding.Codec
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

	// Tracings is tracing options
	Tracings []tracing.Option
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

func NewOptionsAndApply(opts ...Option) Options {
	opt := NewOptions()
	opt.Apply(opts...)
	return opt
}

func WithOptionContext(ctx context.Context) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Context = ctx
	}
}

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

func WithErrorHandler(handler Handler) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.ErrorHandler = handler
	}
}

func WithEnableSecure(enable bool) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Secure = enable
	}
}

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

func WithTracerProvider(provider trace.TracerProvider) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Tracings = append(o.Tracings, tracing.WithTracerProvider(provider))
	}
}

func WithPropagator(propagator propagation.TextMapPropagator) Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Tracings = append(o.Tracings, tracing.WithPropagator(propagator))
	}
}

func WithGlobalTracerProvider() Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Tracings = append(o.Tracings, tracing.WithGlobalTracerProvider())
	}
}

func WithGlobalPropagator() Option {
	return func(o *Options) {
		if o == nil {
			return
		}
		o.Tracings = append(o.Tracings, tracing.WithGlobalPropagator())
	}
}

///////////////////////////////////////////////////////////////////////////////

// PublishOptions publish options
type PublishOptions struct {
	// Context is publish option context
	Context context.Context
}

// PublishOption defines a function which sets some publish option.
type PublishOption func(*PublishOptions)

// Apply applies all options to the PublishOptions.
func (o *PublishOptions) Apply(opts ...PublishOption) {
	if o == nil {
		return
	}
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

func PublishContextWithValue(k, v any) PublishOption {
	return func(o *PublishOptions) {
		if o == nil {
			return
		}
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func WithPublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		if o == nil {
			return
		}
		o.Context = ctx
	}
}

///////////////////////////////////////////////////////////////////////////////

// SubscribeOptions subscribe options
type SubscribeOptions struct {
	// AutoAck indicates whether to automatically acknowledge messages
	AutoAck bool

	// Queue is the name of the queue to subscribe to
	Queue string

	// Context is subscribe option context
	Context context.Context
}

// SubscribeOption defines a function which sets some subscribe option.
type SubscribeOption func(*SubscribeOptions)

// Apply applies all options to the SubscribeOptions.
func (o *SubscribeOptions) Apply(opts ...SubscribeOption) {
	if o == nil {
		return
	}
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

func SubscribeContextWithValue(k, v any) SubscribeOption {
	return func(o *SubscribeOptions) {
		if o == nil {
			return
		}
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// DisableAutoAck sets AutoAck to false
func DisableAutoAck() SubscribeOption {
	return func(o *SubscribeOptions) {
		if o == nil {
			return
		}
		o.AutoAck = false
	}
}

// WithQueueName sets the queue name for the subscription
func WithQueueName(name string) SubscribeOption {
	return func(o *SubscribeOptions) {
		if o == nil {
			return
		}
		o.Queue = name
	}
}

// WithSubscribeContext sets the context for the subscription
func WithSubscribeContext(ctx context.Context) SubscribeOption {
	return func(o *SubscribeOptions) {
		if o == nil {
			return
		}
		o.Context = ctx
	}
}

///////////////////////////////////////////////////////////////////////////////

// RequestOptions request options
type RequestOptions struct {
	// Context is request option context
	Context context.Context
}

// RequestOption defines a function which sets some request option.
type RequestOption func(*RequestOptions)

// Apply applies all options to the RequestOptions.
func (o *RequestOptions) Apply(opts ...RequestOption) {
	if o == nil {
		return
	}
	for _, opt := range opts {
		opt(o)
	}
}

func NewRequestOptions(opts ...RequestOption) RequestOptions {
	opt := RequestOptions{
		Context: context.Background(),
	}

	opt.Apply(opts...)

	return opt
}

func RequestContextWithValue(k, v any) RequestOption {
	return func(o *RequestOptions) {
		if o == nil {
			return
		}
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func WithRequestContext(ctx context.Context) RequestOption {
	return func(o *RequestOptions) {
		if o == nil {
			return
		}
		o.Context = ctx
	}
}
