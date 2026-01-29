package broker

import (
	"context"
	"time"
)

// RequestOptions request options
type RequestOptions struct {
	// Context is request option context
	Context context.Context

	// Timeout is the request timeout
	Timeout time.Duration

	// ReplyTopic is the topic to which the reply should be sent
	ReplyTopic string

	// BodyCodec is the codec used for the message body (e.g., "json", "proto")
	BodyCodec string
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

// NewRequestOptions creates default RequestOptions.
func NewRequestOptions(opts ...RequestOption) RequestOptions {
	opt := RequestOptions{
		Context: context.Background(),
		Timeout: 10 * time.Second,
	}

	opt.Apply(opts...)

	return opt
}

// RequestContextWithValue sets a value in the request option context
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

// WithRequestContext sets the context for the request
func WithRequestContext(ctx context.Context) RequestOption {
	return func(o *RequestOptions) {
		if o == nil {
			return
		}
		o.Context = ctx
	}
}

// WithRequestTimeout sets the timeout for the request
func WithRequestTimeout(d time.Duration) RequestOption {
	return func(o *RequestOptions) {
		o.Timeout = d
	}
}

// WithReplyTopic sets the reply topic for the request
func WithReplyTopic(topic string) RequestOption {
	return func(o *RequestOptions) {
		o.ReplyTopic = topic
	}
}

// WithRequestBodyCodec sets the codec for the request body
func WithRequestBodyCodec(codec string) RequestOption {
	return func(o *RequestOptions) {
		o.BodyCodec = codec
	}
}
