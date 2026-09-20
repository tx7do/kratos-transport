package nats

import (
	"strings"
	natsGo "github.com/nats-io/nats.go"

	"github.com/tx7do/kratos-transport/broker"
)

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

func (p *publication) RawMessage() any {
	return p.m
}

// Ack acknowledges the message.
// For JetStream messages (identified by having a Reply subject), this calls msg.Ack().
// For core NATS messages, this is a no-op since core NATS does not support acknowledgments.
func (p *publication) Ack() error {
	if p.m != nil {
		if msg, ok := p.m.Msg.(*natsGo.Msg); ok {
			// 仅 JetStream 消息可 Ack：其回执主题固定为 $JS.ACK.* 前缀。
			// core NATS 的 request-reply 消息也有 Reply 主题，
			// 误 Ack 会向响应方发送垃圾帧
			if strings.HasPrefix(msg.Reply, "$JS.ACK.") {
				return msg.Ack()
			}
		}
	}
	return nil
}

func (p *publication) Error() error {
	return p.err
}
