package proto

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	"github.com/oxtoacart/bpool"
	"github.com/tx7do/kratos-transport/codec"
)

var bufferPool = bpool.NewSizedBufferPool(16, 256)

type Marshaler struct{}

func (Marshaler) Name() string {
	return "proto"
}

func (Marshaler) Marshal(v interface{}) ([]byte, error) {
	pb, ok := v.(proto.Message)
	if !ok {
		return nil, codec.ErrInvalidMessage
	}

	buf := bufferPool.Get()
	pBuf := proto.NewBuffer(buf.Bytes())
	defer func() {
		bufferPool.Put(bytes.NewBuffer(pBuf.Bytes()))
	}()

	if err := pBuf.Marshal(pb); err != nil {
		return nil, err
	}

	return pBuf.Bytes(), nil
}

func (Marshaler) Unmarshal(data []byte, v interface{}) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return codec.ErrInvalidMessage
	}

	return proto.Unmarshal(data, pb)
}
