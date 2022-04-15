// Package json provides a json codec
package json

import (
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/tx7do/kratos-transport/codec"
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
	return "json"
}

func (c *Codec) Read(b interface{}) error {
	buf, err := io.ReadAll(c.Conn)
	if err != nil {
		return err
	}

	if b == nil {
		return nil
	}
	if pb, ok := b.(proto.Message); ok {
		return protojson.Unmarshal(buf, pb)
	}
	return nil
}

func (c *Codec) Write(b interface{}) error {
	if b == nil {
		return nil
	}

	p, ok := b.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}

	buf, err := protojson.Marshal(p)
	if err != nil {
		return err
	}

	_, err = c.Conn.Write(buf)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}
