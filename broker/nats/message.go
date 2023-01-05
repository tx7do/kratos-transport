package nats

import (
	natsGo "github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*MessageCarrier)(nil)

type MessageCarrier struct {
	msg *natsGo.Msg
}

func NewMessageCarrier(msg *natsGo.Msg) MessageCarrier {
	return MessageCarrier{msg: msg}
}

func (c MessageCarrier) Get(key string) string {
	if c.msg.Header == nil {
		return ""
	}
	return c.msg.Header.Get(key)
}

func (c MessageCarrier) Set(key, val string) {
	if c.msg.Header == nil {
		c.msg.Header = make(natsGo.Header)
	}
	c.msg.Header.Set(key, val)
}

func (c MessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Header))
	var i int
	for k, _ := range c.msg.Header {
		out[i] = k
		i++
	}
	return out
}
