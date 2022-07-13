package nsq

import (
	NSQ "github.com/nsqio/go-nsq"
	"github.com/tx7do/kratos-transport/broker"
)

type publication struct {
	topic  string
	msg    *broker.Message
	nsqMsg *NSQ.Message
	opts   broker.PublishOptions
	err    error
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.msg
}

func (p *publication) Ack() error {
	p.nsqMsg.Finish()
	return nil
}

func (p *publication) Error() error {
	return p.err
}
