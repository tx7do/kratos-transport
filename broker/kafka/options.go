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

func ReaderConfig(c kafka.ReaderConfig) broker.Option {
	return setBrokerOption(readerConfigKey{}, c)
}

func WriterConfig(c kafka.WriterConfig) broker.Option {
	return setBrokerOption(writerConfigKey{}, c)
}

type subscribeContextKey struct{}

// SubscribeContext set the context for common.SubscribeOption
func SubscribeContext(ctx context.Context) broker.SubscribeOption {
	return setSubscribeOption(subscribeContextKey{}, ctx)
}

type subscribeReaderConfigKey struct{}

func SubscribeReaderConfig(c kafka.ReaderConfig) broker.SubscribeOption {
	return setSubscribeOption(subscribeReaderConfigKey{}, c)
}

type subscribeWriterConfigKey struct{}

func SubscribeWriterConfig(c kafka.WriterConfig) broker.SubscribeOption {
	return setSubscribeOption(subscribeWriterConfigKey{}, c)
}
