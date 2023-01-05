package stomp

import (
	stompV3 "github.com/go-stomp/stomp/v3"

	"github.com/tx7do/kratos-transport/broker"
)

type publication struct {
	msg    *stompV3.Message
	m      *broker.Message
	broker *stompBroker
	topic  string
	err    error
}

func (p *publication) Ack() error {
	return p.broker.stompConn.Ack(p.msg)
}

func (p *publication) Error() error {
	return p.err
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.m
}
