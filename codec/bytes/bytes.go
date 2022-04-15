package bytes

import (
	"fmt"
	"github.com/tx7do/kratos-transport/codec"
	"io"
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
	return "bytes"
}

func (c *Codec) Read(b interface{}) error {
	// read bytes
	buf, err := io.ReadAll(c.Conn)
	if err != nil {
		return err
	}

	switch v := b.(type) {
	case *[]byte:
		*v = buf
	default:
		return fmt.Errorf("failed to read body: %v is not type of *[]byte", b)
	}

	return nil
}

func (c *Codec) Write(b interface{}) error {
	var v []byte
	switch vb := b.(type) {
	case *[]byte:
		v = *vb
	case []byte:
		v = vb
	default:
		return fmt.Errorf("failed to write: %v is not type of *[]byte or []byte", b)
	}
	_, err := c.Conn.Write(v)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}
