package rocketmqClientGo

import (
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*ProducerMessageCarrier)(nil)
var _ propagation.TextMapCarrier = (*ConsumerMessageCarrier)(nil)

type ProducerMessageCarrier struct {
	msg *primitive.Message
}

func NewProducerMessageCarrier(msg *primitive.Message) ProducerMessageCarrier {
	return ProducerMessageCarrier{msg: msg}
}

func (c ProducerMessageCarrier) Get(key string) string {
	return c.msg.GetProperty(key)
}

func (c ProducerMessageCarrier) Set(key, val string) {
	c.msg.WithProperty(key, val)
}

func (c ProducerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.GetProperties()))
	var i = 0
	for k := range c.msg.GetProperties() {
		out[i] = k
		i++
	}
	return out
}

type ConsumerMessageCarrier struct {
	msg *primitive.MessageExt
}

func NewConsumerMessageCarrier(msg *primitive.MessageExt) ConsumerMessageCarrier {
	return ConsumerMessageCarrier{msg: msg}
}

func (c ConsumerMessageCarrier) Get(key string) string {
	return c.msg.GetProperty(key)
}

func (c ConsumerMessageCarrier) Set(key, val string) {
	c.msg.WithProperty(key, val)
}

func (c ConsumerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.GetProperties()))
	var i = 0
	for k := range c.msg.GetProperties() {
		out[i] = k
		i++
	}
	return out
}
