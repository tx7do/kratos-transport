package bytes

import (
	"github.com/tx7do/kratos-transport/codec"
)

type Marshaler struct{}

func (n Marshaler) Name() string {
	return "bytes"
}

func (n Marshaler) Marshal(v interface{}) ([]byte, error) {
	switch ve := v.(type) {
	case *[]byte:
		return *ve, nil
	case []byte:
		return ve, nil
	}
	return nil, codec.ErrInvalidMessage
}

func (n Marshaler) Unmarshal(d []byte, v interface{}) error {
	switch ve := v.(type) {
	case *[]byte:
		*ve = d
	}
	return codec.ErrInvalidMessage
}
