package kafka

import (
	"context"
	"github.com/segmentio/kafka-go"
	"github.com/tx7do/kratos-transport/broker"
)

var (
	DefaultReaderConfig = kafka.WriterConfig{}
	DefaultWriterConfig = kafka.ReaderConfig{}
)

type readerConfigKey struct{}
type writerConfigKey struct{}
type subscribeContextKey struct{}
type subscribeReaderConfigKey struct{}
type subscribeWriterConfigKey struct{}

func WithReaderConfig(c kafka.ReaderConfig) broker.Option {
	return broker.OptionContextWithValue(readerConfigKey{}, c)
}

func WithWriterConfig(c kafka.WriterConfig) broker.Option {
	return broker.OptionContextWithValue(writerConfigKey{}, c)
}

func WithSubscribeContext(ctx context.Context) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeContextKey{}, ctx)
}

func WithSubscribeContextFromContext(ctx context.Context) (context.Context, bool) {
	c, ok := ctx.Value(subscribeContextKey{}).(context.Context)
	return c, ok
}

func WithSubscribeReaderConfig(c kafka.ReaderConfig) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeReaderConfigKey{}, c)
}

func WithSubscribeWriterConfig(c kafka.WriterConfig) broker.SubscribeOption {
	return broker.SubscribeContextWithValue(subscribeWriterConfigKey{}, c)
}
