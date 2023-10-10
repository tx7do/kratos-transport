package kafka

import (
	"context"
	"errors"

	kafkaGo "github.com/segmentio/kafka-go"

	"github.com/tx7do/kratos-transport/broker"
)

type publication struct {
	topic  string
	err    error
	m      *broker.Message
	ctx    context.Context
	reader *kafkaGo.Reader
	km     kafkaGo.Message
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) RawMessage() interface{} {
	return p.km
}

func (p *publication) Ack() error {
	if p.reader == nil {
		return errors.New("read is nil")
	}
	return p.reader.CommitMessages(p.ctx, p.km)
}

func (p *publication) Error() error {
	return p.err
}
