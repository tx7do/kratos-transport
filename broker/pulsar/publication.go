package pulsar

import (
	"context"
	"errors"
	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/tx7do/kratos-transport/broker"
)

type publication struct {
	topic     string
	err       error
	ctx       context.Context
	reader    pulsar.Consumer
	msg       *broker.Message
	pulsarMsg *pulsar.Message
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.msg
}

func (p *publication) RawMessage() any {
	return p.pulsarMsg
}

func (p *publication) Ack() error {
	if p.reader == nil {
		return errors.New("reader is nil")
	}
	return p.reader.Ack(*p.pulsarMsg)
}

func (p *publication) Error() error {
	return p.err
}
