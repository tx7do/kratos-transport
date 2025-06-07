package nsq

import (
	"errors"

	NSQ "github.com/nsqio/go-nsq"
	"github.com/tx7do/kratos-transport/broker"
)

type publication struct {
	topic   string
	msg     *broker.Message
	nsqMsg  *NSQ.Message
	options broker.PublishOptions
	err     error
}

func (p *publication) Topic() string {
	return p.topic
}

func (p *publication) Message() *broker.Message {
	return p.msg
}

func (p *publication) RawMessage() interface{} {
	return p.nsqMsg
}

func (p *publication) Ack() error {
	if p.nsqMsg == nil {
		p.err = errors.New("nsq message is nil")
		return p.err
	}

	p.nsqMsg.Finish()
	return nil
}

func (p *publication) Error() error {
	return p.err
}
