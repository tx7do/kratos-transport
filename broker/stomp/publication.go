package stomp

import (
	"errors"
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
	if p.broker == nil {
		return errors.New("broker is nil")
	}
	if p.broker.stompConn == nil {
		return errors.New("stomp connection is nil")
	}
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

func (p *publication) RawMessage() any {
	return p.broker
}
