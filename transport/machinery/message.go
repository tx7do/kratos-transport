package machinery

import (
	"github.com/RichardKnop/machinery/v2/tasks"
	"go.opentelemetry.io/otel/propagation"
	"strconv"
)

var _ propagation.TextMapCarrier = (*MessageCarrier)(nil)

type MessageCarrier struct {
	msg *tasks.Headers
}

func NewMessageCarrier(msg *tasks.Headers) MessageCarrier {
	return MessageCarrier{msg: msg}
}

func (c MessageCarrier) Get(key string) string {
	if c.msg == nil {
		*c.msg = make(tasks.Headers)
	}

	value := (*c.msg)[key]

	if value == nil {
		return ""
	}

	switch value.(type) {
	case float64:
		ft := value.(float64)
		return strconv.FormatFloat(ft, 'f', -1, 64)
	case float32:
		ft := value.(float32)
		return strconv.FormatFloat(float64(ft), 'f', -1, 64)
	case int:
		it := value.(int)
		return strconv.Itoa(it)
	case uint:
		it := value.(uint)
		return strconv.Itoa(int(it))
	case int8:
		it := value.(int8)
		return strconv.Itoa(int(it))
	case uint8:
		it := value.(uint8)
		return strconv.Itoa(int(it))
	case int16:
		it := value.(int16)
		return strconv.Itoa(int(it))
	case uint16:
		it := value.(uint16)
		return strconv.Itoa(int(it))
	case int32:
		it := value.(int32)
		return strconv.Itoa(int(it))
	case uint32:
		it := value.(uint32)
		return strconv.Itoa(int(it))
	case int64:
		it := value.(int64)
		return strconv.FormatInt(it, 10)
	case uint64:
		it := value.(uint64)
		return strconv.FormatUint(it, 10)
	case string:
		return value.(string)
	case []byte:
		return string(value.([]byte))
	default:
		return ""
	}
}

func (c MessageCarrier) Set(key, val string) {
	if c.msg == nil {
		*c.msg = make(tasks.Headers)
	}
	(*c.msg)[key] = val
}

func (c MessageCarrier) Keys() []string {
	if c.msg == nil {
		*c.msg = make(tasks.Headers)
	}
	var keys []string
	_ = c.msg.ForeachKey(func(key, _ string) error {
		keys = append(keys, key)
		return nil
	})
	return keys
}
