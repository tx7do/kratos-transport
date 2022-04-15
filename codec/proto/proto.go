package proto

import (
	"io"

	"github.com/tx7do/kratos-transport/codec"
	"google.golang.org/protobuf/proto"
)

type Codec struct {
	Conn io.ReadWriteCloser
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn: c,
	}
}

func (c *Codec) Name() string {
	return "proto"
}

func (c *Codec) Read(b interface{}) error {
	if b == nil {
		return nil
	}
	buf, err := io.ReadAll(c.Conn)
	if err != nil {
		return err
	}
	m, ok := b.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}
	return proto.Unmarshal(buf, m)
}

func (c *Codec) Write(b interface{}) error {
	if b == nil {
		return nil
	}
	p, ok := b.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}
	buf, err := proto.Marshal(p)
	if err != nil {
		return err
	}
	_, err = c.Conn.Write(buf)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}
