package aliyun

import (
	aliyun "github.com/aliyunmq/mq-http-go-sdk"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*ProducerMessageCarrier)(nil)
var _ propagation.TextMapCarrier = (*ConsumerMessageCarrier)(nil)

type ProducerMessageCarrier struct {
	msg *aliyun.PublishMessageRequest
}

func NewProducerMessageCarrier(msg *aliyun.PublishMessageRequest) ProducerMessageCarrier {
	return ProducerMessageCarrier{msg: msg}
}

func (c ProducerMessageCarrier) Get(key string) string {
	return c.msg.Properties[key]
}

func (c ProducerMessageCarrier) Set(key, val string) {
	c.msg.Properties[key] = val
}

func (c ProducerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Properties))
	var i = 0
	for k := range c.msg.Properties {
		out[i] = k
		i++
	}
	return out
}

type ConsumerMessageCarrier struct {
	msg *aliyun.ConsumeMessageEntry
}

func NewConsumerMessageCarrier(msg *aliyun.ConsumeMessageEntry) ConsumerMessageCarrier {
	return ConsumerMessageCarrier{msg: msg}
}

func (c ConsumerMessageCarrier) Get(key string) string {
	return c.msg.Properties[key]
}

func (c ConsumerMessageCarrier) Set(key, val string) {
	if c.msg.Properties == nil {
		c.msg.Properties = make(map[string]string)
	}
	c.msg.Properties[key] = val
}

func (c ConsumerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Properties))
	var i = 0
	for k := range c.msg.Properties {
		out[i] = k
		i++
	}
	return out
}
