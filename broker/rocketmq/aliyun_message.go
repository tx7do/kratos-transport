package rocketmq

import (
	aliyun "github.com/aliyunmq/mq-http-go-sdk"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*AliyunProducerMessageCarrier)(nil)
var _ propagation.TextMapCarrier = (*AliyunConsumerMessageCarrier)(nil)

type AliyunProducerMessageCarrier struct {
	msg *aliyun.PublishMessageRequest
}

func NewAliyunProducerMessageCarrier(msg *aliyun.PublishMessageRequest) AliyunProducerMessageCarrier {
	return AliyunProducerMessageCarrier{msg: msg}
}

func (c AliyunProducerMessageCarrier) Get(key string) string {
	return c.msg.Properties[key]
}

func (c AliyunProducerMessageCarrier) Set(key, val string) {
	c.msg.Properties[key] = val
}

func (c AliyunProducerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Properties))
	var i = 0
	for k, _ := range c.msg.Properties {
		out[i] = k
		i++
	}
	return out
}

type AliyunConsumerMessageCarrier struct {
	msg *aliyun.ConsumeMessageEntry
}

func NewAliyunConsumerMessageCarrier(msg *aliyun.ConsumeMessageEntry) AliyunConsumerMessageCarrier {
	return AliyunConsumerMessageCarrier{msg: msg}
}

func (c AliyunConsumerMessageCarrier) Get(key string) string {
	return c.msg.Properties[key]
}

func (c AliyunConsumerMessageCarrier) Set(key, val string) {
	if c.msg.Properties == nil {
		c.msg.Properties = make(map[string]string)
	}
	c.msg.Properties[key] = val
}

func (c AliyunConsumerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Properties))
	var i = 0
	for k, _ := range c.msg.Properties {
		out[i] = k
		i++
	}
	return out
}
