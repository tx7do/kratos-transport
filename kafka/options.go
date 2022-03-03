package kafka

import (
	"context"
	"github.com/tx7do/kratos-transport/common"

	"github.com/segmentio/kafka-go"
)

var (
	DefaultReaderConfig = kafka.WriterConfig{}
	DefaultWriterConfig = kafka.ReaderConfig{}
)

type readerConfigKey struct{}
type writerConfigKey struct{}

func ReaderConfig(c kafka.ReaderConfig) common.Option {
	return setBrokerOption(readerConfigKey{}, c)
}

func WriterConfig(c kafka.WriterConfig) common.Option {
	return setBrokerOption(writerConfigKey{}, c)
}

type subscribeContextKey struct{}

// SubscribeContext set the context for common.SubscribeOption
func SubscribeContext(ctx context.Context) common.SubscribeOption {
	return setSubscribeOption(subscribeContextKey{}, ctx)
}

type subscribeReaderConfigKey struct{}

func SubscribeReaderConfig(c kafka.ReaderConfig) common.SubscribeOption {
	return setSubscribeOption(subscribeReaderConfigKey{}, c)
}

type subscribeWriterConfigKey struct{}

func SubscribeWriterConfig(c kafka.WriterConfig) common.SubscribeOption {
	return setSubscribeOption(subscribeWriterConfigKey{}, c)
}
