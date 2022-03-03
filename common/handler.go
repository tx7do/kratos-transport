package common

import "context"

type HandlerOption func(*HandlerOptions)

type HandlerOptions struct {
	Internal bool
	Metadata map[string]map[string]string
}

type SubscriberOption func(*SubscriberOptions)

type SubscriberOptions struct {
	AutoAck  bool
	Queue    string
	Internal bool
	Context  context.Context
}

func EndpointMetadata(name string, md map[string]string) HandlerOption {
	return func(o *HandlerOptions) {
		o.Metadata[name] = md
	}
}

func InternalHandler(b bool) HandlerOption {
	return func(o *HandlerOptions) {
		o.Internal = b
	}
}

func InternalSubscriber(b bool) SubscriberOption {
	return func(o *SubscriberOptions) {
		o.Internal = b
	}
}
func NewSubscriberOptions(opts ...SubscriberOption) SubscriberOptions {
	opt := SubscriberOptions{
		AutoAck: true,
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

func SubscriberQueue(n string) SubscriberOption {
	return func(o *SubscriberOptions) {
		o.Queue = n
	}
}

func SubscriberContext(ctx context.Context) SubscriberOption {
	return func(o *SubscriberOptions) {
		o.Context = ctx
	}
}
