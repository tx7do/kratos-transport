package kafka

import (
	"context"
	"errors"

	kafkaGo "github.com/segmentio/kafka-go"

	"github.com/tx7do/kratos-transport/broker"
)

type publication struct {
	topic string

	bm *broker.Message
	km kafkaGo.Message

	reader *kafkaGo.Reader

	ctx context.Context
	err error
}

func newPublication(ctx context.Context, reader *kafkaGo.Reader, km kafkaGo.Message, bm *broker.Message) *publication {
	pub := &publication{
		topic:  km.Topic,
		reader: reader,
		bm:     bm,
		km:     km,
		ctx:    ctx,
	}

	return pub
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.bm
}

func (p *publication) RawMessage() any {
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
