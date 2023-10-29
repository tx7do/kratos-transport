package rocketmqClients

import (
	"context"
	"errors"

	rmqClient "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/tx7do/kratos-transport/broker"
)

type publication struct {
	topic string
	err   error
	ctx   context.Context

	message *broker.Message

	reader     rmqClient.SimpleConsumer
	rmqMessage *rmqClient.MessageView
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.message
}

func (p *publication) RawMessage() interface{} {
	return p.rmqMessage
}

func (p *publication) Ack() error {
	if p.reader == nil {
		p.err = errors.New("reader is nil")
		return p.err
	}
	p.err = p.reader.Ack(p.ctx, p.rmqMessage)
	return p.err
}

func (p *publication) Error() error {
	return p.err
}
