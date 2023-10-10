package nats

import "github.com/tx7do/kratos-transport/broker"

type publication struct {
	t   string
	err error
	m   *broker.Message
}

func (p *publication) Topic() string {
	return p.t
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) RawMessage() interface{} {
	return p.m
}

func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}
