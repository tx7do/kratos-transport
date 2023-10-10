package rocketmq

import (
	"context"
	"errors"
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

func (p *publication) RawMessage() interface{} {
	return p.rm
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

func (p *aliyunPublication) RawMessage() interface{} {
	return p.rm
}

func (p *aliyunPublication) Ack() error {
	if p.reader == nil {
		return errors.New("reader is nil")
	}
	p.err = p.reader.AckMessage(p.rm)
	return p.err
}

func (p *aliyunPublication) Error() error {
	return p.err
}
