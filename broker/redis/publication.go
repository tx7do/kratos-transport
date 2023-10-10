package redis

import "github.com/tx7do/kratos-transport/broker"

type publication struct {
	topic   string
	message *broker.Message
	err     error
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.message
}

func (p *publication) RawMessage() interface{} {
	return p.message
}

func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}
