package codec

import "errors"

var (
	ErrInvalidMessage = errors.New("invalid message")
)

type Marshaler interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	Name() string
}
