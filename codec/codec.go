package codec

import (
	"errors"
	"io"
)

type MessageType int

const (
	Error MessageType = iota
	Request
	Response
	Event
)

var (
	ErrInvalidMessage = errors.New("invalid message")
)

type NewCodec func(io.ReadWriteCloser) Codec

type Codec interface {
	Reader
	Writer
	Close() error
	Name() string
}

type Reader interface {
	Read(interface{}) error
}

type Writer interface {
	Write(interface{}) error
}

type Marshaler interface {
	Marshal(interface{}) ([]byte, error)
	Unmarshal([]byte, interface{}) error
	Name() string
}
