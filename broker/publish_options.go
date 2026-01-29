package broker

import (
	"context"
	"time"
)

// PublishOptions publish options
type PublishOptions struct {
	// Context is published option context
	Context context.Context

	// Timeout is the publishing timeout
	Timeout time.Duration

	// Retries is the number of retries for publishing
	Retries int

	// Async indicates whether to publish asynchronously
	Async bool

	// Callback is the callback function for asynchronous publishing
	Callback func(err error)

	// RequiredAcks indicates the number of required acknowledgments
	RequiredAcks int

	// BodyCodec is the codec used for the message body (e.g., "json", "proto")
	BodyCodec string
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

// NewPublishOptions creates default PublishOptions.
func NewPublishOptions(opts ...PublishOption) PublishOptions {
	opt := PublishOptions{
		Context: context.Background(),
	}

	opt.Apply(opts...)

	return opt
}

// PublishContextWithValue sets a value in the publish option context
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

// WithPublishContext sets the context for publishing
func WithPublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) {
		if o == nil {
			return
		}
		o.Context = ctx
	}
}

// WithPublishTimeout sets the publishing timeout
func WithPublishTimeout(d time.Duration) PublishOption {
	return func(o *PublishOptions) {
		o.Timeout = d
	}
}

// WithPublishAsync sets the publishing to be asynchronous with a callback
func WithPublishAsync(callback func(error)) PublishOption {
	return func(o *PublishOptions) {
		o.Async = true
		o.Callback = callback
	}
}

// WithPublishRetries sets the number of retries for publishing
func WithPublishRetries(n int) PublishOption {
	return func(o *PublishOptions) {
		o.Retries = n
	}
}

// WithPublishRequiredAcks sets the number of required acknowledgments for publishing
func WithPublishRequiredAcks(acks int) PublishOption {
	return func(o *PublishOptions) {
		o.RequiredAcks = acks
	}
}

// WithPublishBodyCodec sets the body codec for publishing
func WithPublishBodyCodec(codec string) PublishOption {
	return func(o *PublishOptions) {
		o.BodyCodec = codec
	}
}

func WithPublishCallback(callback func(error)) PublishOption {
	return func(o *PublishOptions) {
		o.Callback = callback
	}
}
