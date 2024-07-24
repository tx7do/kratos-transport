package nats

import (
	natsGo "github.com/nats-io/nats.go"
	"github.com/tx7do/kratos-transport/broker"
	"time"
)

type optionsKey struct{}
type drainConnectionKey struct{}

func Options(opts natsGo.Options) broker.Option {
	return broker.OptionContextWithValue(optionsKey{}, opts)
}

func DrainConnection() broker.Option {
	return broker.OptionContextWithValue(drainConnectionKey{}, struct{}{})
}

///////////////////////////////////////////////////////////////////////////////

type headersKey struct{}

func WithHeaders(h map[string][]string) broker.PublishOption {
	return broker.PublishContextWithValue(headersKey{}, h)
}

///////////////////////////////////////////////////////////////////////////////

type requestTimeoutKey struct{}

func WithRequestTimeout(timeout time.Duration) broker.RequestOption {
	return broker.RequestContextWithValue(requestTimeoutKey{}, timeout)
}

func WithRequestHeaders(h map[string][]string) broker.RequestOption {
	return broker.RequestContextWithValue(headersKey{}, h)
}
