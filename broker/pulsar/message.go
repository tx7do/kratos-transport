package pulsar

import (
	"github.com/apache/pulsar-client-go/pulsar"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*ProducerMessageCarrier)(nil)
var _ propagation.TextMapCarrier = (*ConsumerMessageCarrier)(nil)

type ProducerMessageCarrier struct {
	msg *pulsar.ProducerMessage
}

func NewProducerMessageCarrier(msg *pulsar.ProducerMessage) ProducerMessageCarrier {
	return ProducerMessageCarrier{msg: msg}
}

func (c ProducerMessageCarrier) Get(key string) string {
	if c.msg.Properties == nil {
		return ""
	}
	return c.msg.Properties[key]
}

func (c ProducerMessageCarrier) Set(key, val string) {
	if c.msg.Properties == nil {
		c.msg.Properties = make(map[string]string)
	}
	c.msg.Properties[key] = val
}

func (c ProducerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Properties))
	var i = 0
	for k, _ := range c.msg.Properties {
		out[i] = k
		i++
	}
	return out
}

type ConsumerMessageCarrier struct {
	msg *pulsar.ConsumerMessage
}

func NewConsumerMessageCarrier(msg *pulsar.ConsumerMessage) ConsumerMessageCarrier {
	return ConsumerMessageCarrier{msg: msg}
}

func (c ConsumerMessageCarrier) Get(key string) string {
	if c.msg.Properties() == nil {
		return ""
	}
	return c.msg.Properties()[key]
}

func (c ConsumerMessageCarrier) Set(key, val string) {
	if c.msg.Properties() == nil {
		return
	}
	c.msg.Properties()[key] = val
}

func (c ConsumerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Properties()))
	var i = 0
	for k, _ := range c.msg.Properties() {
		out[i] = k
		i++
	}
	return out
}
