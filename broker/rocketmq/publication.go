package rocketmq

import (
	"context"
	aliyun "github.com/aliyunmq/mq-http-go-sdk"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/tx7do/kratos-transport/broker"
)

type publication struct {
	topic  string
	err    error
	m      *broker.Message
	ctx    context.Context
	reader rocketmq.PushConsumer
	rm     *primitive.Message
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}

///
/// Aliyun Publication
///

type aliyunPublication struct {
	topic  string
	err    error
	m      *broker.Message
	ctx    context.Context
	reader aliyun.MQConsumer
	rm     []string
}

func (p *aliyunPublication) Topic() string {
	return p.topic
}

func (p *aliyunPublication) Message() *broker.Message {
	return p.m
}

func (p *aliyunPublication) Ack() error {
	p.err = p.reader.AckMessage(p.rm)
	return p.err
}

func (p *aliyunPublication) Error() error {
	return p.err
}
