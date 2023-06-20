package rabbitmq

import (
	"strconv"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*ProducerMessageCarrier)(nil)
var _ propagation.TextMapCarrier = (*ConsumerMessageCarrier)(nil)

type ProducerMessageCarrier struct {
	msg *amqp.Publishing
}

func NewProducerMessageCarrier(msg *amqp.Publishing) ProducerMessageCarrier {
	return ProducerMessageCarrier{msg: msg}
}

func (c ProducerMessageCarrier) Get(key string) string {
	for k, v := range c.msg.Headers {
		if k == key {
			switch t := v.(type) {
			case []byte:
				return string(t)
			case string:
				return t
			case int, int8, int16, int32, int64:
				return strconv.FormatInt(t.(int64), 10)
			case uint, uint8, uint16, uint32, uint64:
				return strconv.FormatUint(t.(uint64), 10)
			case float32, float64:
				return strconv.FormatFloat(t.(float64), 'f', -1, 64)
			default:
				return ""
			}
		}
	}
	return ""
}

func (c ProducerMessageCarrier) Set(key, val string) {
	c.msg.Headers[key] = val
}

func (c ProducerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Headers))
	var i = 0
	for k := range c.msg.Headers {
		out[i] = k
		i++
	}
	return out
}

type ConsumerMessageCarrier struct {
	msg *amqp.Delivery
}

func NewConsumerMessageCarrier(msg *amqp.Delivery) ConsumerMessageCarrier {
	return ConsumerMessageCarrier{msg: msg}
}

func (c ConsumerMessageCarrier) Get(key string) string {
	for k, v := range c.msg.Headers {
		if k == key {
			switch t := v.(type) {
			case []byte:
				return string(t)
			case string:
				return t
			case int, int8, int16, int32, int64:
				return strconv.FormatInt(t.(int64), 10)
			case uint, uint8, uint16, uint32, uint64:
				return strconv.FormatUint(t.(uint64), 10)
			case float32, float64:
				return strconv.FormatFloat(t.(float64), 'f', -1, 64)
			default:
				return ""
			}
		}
	}
	return ""
}

func (c ConsumerMessageCarrier) Set(key, val string) {
	if c.msg.Headers == nil {
		c.msg.Headers = make(amqp.Table)
	}
	c.msg.Headers[key] = val
}

func (c ConsumerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Headers))
	var i = 0
	for k := range c.msg.Headers {
		out[i] = k
		i++
	}
	return out
}
