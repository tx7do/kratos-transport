package broker

import "github.com/nats-io/nats.go"

type Any interface{}

type Binder func() Any

type Headers map[string]string

type Message struct {
	Headers   Headers
	Body      Any
	Partition int
	Offset    int64
	Reply     Any
	NatsMsg   *nats.Msg
}

func (m Message) GetHeaders() Headers {
	return m.Headers
}

func (m Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

