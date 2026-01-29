package broker

import (
	"context"
	"time"
)

// SubscribeOptions subscribe options
type SubscribeOptions struct {
	// Context is subscribed option context
	Context context.Context

	// AutoAck indicates whether to automatically acknowledge messages
	AutoAck bool

	// Queue is the name of the queue to subscribe to
	Queue string

	// Concurrency is the number of concurrent handlers
	Concurrency int

	// MaxRetries is the maximum number of retries for message processing
	MaxRetries int

	// RetryDelay is the delay between retries
	RetryDelay time.Duration

	// Middlewares are subscriber middlewares
	Middlewares []SubscriberMiddleware
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

// NewSubscribeOptions creates default SubscribeOptions.
func NewSubscribeOptions(opts ...SubscribeOption) SubscribeOptions {
	opt := SubscribeOptions{
		Context:     context.Background(),
		AutoAck:     true,
		Queue:       "",
		Concurrency: 1,
		MaxRetries:  0,
	}

	opt.Apply(opts...)

	return opt
}

// SubscribeContextWithValue sets a value in the subscribe option context
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

// WithSubscribeAutoAck sets AutoAck to the given value
func WithSubscribeAutoAck(b bool) SubscribeOption {
	return func(o *SubscribeOptions) {
		if o == nil {
			return
		}
		o.AutoAck = b
	}
}

// WithSubscribeQueueName sets the queue name for the subscription
func WithSubscribeQueueName(name string) SubscribeOption {
	return func(o *SubscribeOptions) {
		if o == nil {
			return
		}
		o.Queue = name
	}
}

// WithSubscribeGroupID sets the group ID for the subscription (alias for WithQueueName) for kafka compatibility
func WithSubscribeGroupID(id string) SubscribeOption {
	return WithSubscribeQueueName(id)
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

// WithSubscribeConcurrency sets the number of concurrent handlers
func WithSubscribeConcurrency(n int) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Concurrency = n
	}
}

// WithSubscribeRetry sets the maximum number of retries and delay between retries
func WithSubscribeRetry(maxRetries int, delay time.Duration) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.MaxRetries = maxRetries
		o.RetryDelay = delay
	}
}

// WithSubscribeMiddlewares sets subscriber middlewares
func WithSubscribeMiddlewares(mws ...SubscriberMiddleware) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Middlewares = append(o.Middlewares, mws...)
	}
}
