package stomp

import (
	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*ProducerMessageCarrier)(nil)
var _ propagation.TextMapCarrier = (*ConsumerMessageCarrier)(nil)

type ProducerMessageCarrier struct {
	msg *[]func(*frame.Frame) error
}

func NewProducerMessageCarrier(msg *[]func(*frame.Frame) error) ProducerMessageCarrier {
	return ProducerMessageCarrier{msg: msg}
}

func (c ProducerMessageCarrier) Get(key string) string {
	//fmt.Printf("ProducerMessageCarrier.Get %s\n", key)
	return ""
}

func (c ProducerMessageCarrier) Set(key, val string) {
	//fmt.Println("ProducerMessageCarrier.Set", key, val)
	*c.msg = append(*c.msg, stomp.SendOpt.Header(key, val))
}

func (c ProducerMessageCarrier) Keys() []string {
	//fmt.Printf("ProducerMessageCarrier.Keys\n")
	return nil
}

type ConsumerMessageCarrier struct {
	msg *stomp.Message
}

func NewConsumerMessageCarrier(msg *stomp.Message) ConsumerMessageCarrier {
	return ConsumerMessageCarrier{msg: msg}
}

func (c ConsumerMessageCarrier) Get(key string) string {
	if c.msg.Header == nil {
		return ""
	}
	return c.msg.Header.Get(key)
}

func (c ConsumerMessageCarrier) Set(key, val string) {
	if c.msg.Header == nil {
		c.msg.Header = frame.NewHeader()
	}
	c.msg.Header.Set(key, val)
}

func (c ConsumerMessageCarrier) Keys() []string {
	if c.msg.Header == nil {
		return nil
	}

	out := make([]string, c.msg.Header.Len())
	for i := 0; i < c.msg.Header.Len(); i++ {
		k, _ := c.msg.Header.GetAt(i)
		out[i] = k
	}

	return out
}
