// Package json provides a json codec
package json

import (
	"encoding/json"
	"io"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/tx7do/kratos-transport/codec"
)

type Codec struct {
	Conn    io.ReadWriteCloser
	Encoder *json.Encoder
	Decoder *json.Decoder
}

func NewCodec(c io.ReadWriteCloser) codec.Codec {
	return &Codec{
		Conn:    c,
		Decoder: json.NewDecoder(c),
		Encoder: json.NewEncoder(c),
	}
}

func (c *Codec) Name() string {
	return "json"
}

func (c *Codec) Read(b interface{}) error {
	if b == nil {
		return nil
	}

	if pb, ok := b.(proto.Message); ok {
		return jsonpb.UnmarshalNext(c.Decoder, pb)
	}

	return c.Decoder.Decode(b)
}

func (c *Codec) Write(b interface{}) error {
	if b == nil {
		return nil
	}

	return c.Encoder.Encode(b)
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}
