package broker

import "context"

// PublishHandler defines the publishing handler
type PublishHandler func(ctx context.Context, topic string, msg *Message, opts ...PublishOption) error

// PublishMiddleware defines the publishing middleware
type PublishMiddleware func(PublishHandler) PublishHandler

// ChainPublishMiddleware chains multiple PublishMiddleware into a single PublishHandler.
func ChainPublishMiddleware(h PublishHandler, mws []PublishMiddleware) PublishHandler {
	if len(mws) == 0 {
		return h
	}
	for i := len(mws) - 1; i >= 0; i-- {
		if mws[i] == nil {
			continue
		}
		h = mws[i](h)
	}
	return h
}

// WrapLegacyPublishHandler wraps a legacy PublishHandler to the new PublishHandler
func WrapLegacyPublishHandler(h func(ctx context.Context, topic string, msg any, opts ...PublishOption) error) PublishHandler {
	return func(ctx context.Context, topic string, msg *Message, opts ...PublishOption) error {
		var payload any
		if msg != nil {
			if msg.Msg != nil {
				payload = msg.Msg
			} else {
				payload = msg.Body
			}
		}
		return h(ctx, topic, payload, opts...)
	}
}
