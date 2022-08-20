package redis

import (
	"fmt"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/gomodule/redigo/redis"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	codec   encoding.Codec
	conn    *redis.PubSubConn
	topic   string
	handler broker.Handler
	binder  broker.Binder
	opts    broker.SubscribeOptions
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

			if s.binder != nil {
				m.Body = s.binder()
			}

			p := publication{
				topic:   x.Channel,
				message: &m,
			}

			if err := broker.Unmarshal(s.codec, x.Data, m.Body); err != nil {
				p.err = err
				//log.Error("[redis]", err)
				break
			}

			if p.err = s.handler(s.opts.Context, &p); p.err != nil {
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
