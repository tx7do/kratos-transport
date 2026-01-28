package aliyun

import (
	"context"
	"errors"

	aliyun "github.com/aliyunmq/mq-http-go-sdk"

	"github.com/tx7do/kratos-transport/broker"
)

type Publication struct {
	topic  string
	err    error
	m      *broker.Message
	ctx    context.Context
	reader aliyun.MQConsumer
	rm     []string
}

func (p *Publication) Topic() string {
	return p.topic
}

func (p *Publication) Message() *broker.Message {
	return p.m
}

func (p *Publication) RawMessage() any {
	return p.rm
}

func (p *Publication) Ack() error {
	if p.reader == nil {
		return errors.New("reader is nil")
	}
	p.err = p.reader.AckMessage(p.rm)
	return p.err
}

func (p *Publication) Error() error {
	return p.err
}
