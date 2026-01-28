package broker

import "context"

// PublishHandler defines the publishing handler
type PublishHandler func(ctx context.Context, topic string, msg any, opts ...PublishOption) error

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
