package broker

import "context"

// PublishFunc defines the publish function
type PublishFunc func(ctx context.Context, topic string, msg *Message, opts ...PublishOption) error

// PublishMiddleware defines the publishing middleware
type PublishMiddleware func(PublishFunc) PublishFunc
