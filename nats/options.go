package nats

import (
	"github.com/nats-io/nats.go"
	"github.com/tx7do/kratos-transport/common"
)

type optionsKey struct{}
type drainConnectionKey struct{}

func Options(opts nats.Options) common.Option {
	return setBrokerOption(optionsKey{}, opts)
}

func DrainConnection() common.Option {
	return setBrokerOption(drainConnectionKey{}, struct{}{})
}
