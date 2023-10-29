package rocketmqClients

import (
	rmqClient "github.com/apache/rocketmq-clients/golang/v5"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*ProducerMessageCarrier)(nil)
var _ propagation.TextMapCarrier = (*ConsumerMessageCarrier)(nil)

type ProducerMessageCarrier struct {
	msg *rmqClient.Message
}

func NewProducerMessageCarrier(msg *rmqClient.Message) ProducerMessageCarrier {
	return ProducerMessageCarrier{msg: msg}
}

func (c ProducerMessageCarrier) Get(key string) string {
	return c.msg.GetProperties()[key]
}

func (c ProducerMessageCarrier) Set(key, val string) {
	c.msg.AddProperty(key, val)
}

func (c ProducerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.GetProperties()))
	var i = 0
	for _, k := range c.msg.GetProperties() {
		out[i] = k
		i++
	}
	return out
}

type ConsumerMessageCarrier struct {
	msg *rmqClient.MessageView
}

func NewConsumerMessageCarrier(msg *rmqClient.MessageView) ConsumerMessageCarrier {
	return ConsumerMessageCarrier{msg: msg}
}

func (c ConsumerMessageCarrier) Get(key string) string {
	return c.msg.GetProperties()[key]
}

func (c ConsumerMessageCarrier) Set(key, val string) {
	c.msg.GetProperties()[key] = val
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
