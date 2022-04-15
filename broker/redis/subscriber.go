package redis

import (
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/codec"
)

type subscriber struct {
	codec  codec.Marshaler
	conn   *redis.PubSubConn
	topic  string
	handle broker.Handler
	opts   broker.SubscribeOptions
}

func (s *subscriber) recv() {
	defer func(conn *redis.PubSubConn) {
		err := conn.Close()
		if err != nil {

		}
	}(s.conn)

	for {
		switch x := s.conn.Receive().(type) {
		case redis.Message:
			var m broker.Message

			if s.codec != nil {
				if err := s.codec.Unmarshal(x.Data, &m); err != nil {
					break
				}
			} else {
				m.Body = x.Data
			}

			p := publication{
				topic:   x.Channel,
				message: &m,
			}

			if p.err = s.handle(s.opts.Context, &p); p.err != nil {
				break
			}

			if s.opts.AutoAck {
				if err := p.Ack(); err != nil {
					break
				}
			}

		case redis.Subscription:
			if x.Count == 0 {
				return
			}

		case error:
			fmt.Printf("redis recv error: %s\n", x.Error())
			return
		}
	}
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.topic
}

func (s *subscriber) Unsubscribe() error {
	return s.conn.Unsubscribe()
}
