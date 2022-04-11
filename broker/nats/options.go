package nats

import (
	"github.com/nats-io/nats.go"
	"github.com/tx7do/kratos-transport/broker"
)

type optionsKey struct{}
type drainConnectionKey struct{}

func Options(opts nats.Options) broker.Option {
	return broker.OptionContextWithValue(optionsKey{}, opts)
}

func DrainConnection() broker.Option {
	return broker.OptionContextWithValue(drainConnectionKey{}, struct{}{})
}
