package text

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
	return "text"
}

func (c *Codec) Read(b interface{}) error {
	buf, err := io.ReadAll(c.Conn)
	if err != nil {
		return err
	}

	switch v := b.(type) {
	case *string:
		*v = string(buf)
	case *[]byte:
		*v = buf
	default:
		return fmt.Errorf("failed to read body: %v is not type of *[]byte", b)
	}

	return nil
}

func (c *Codec) Write(b interface{}) error {
	var v []byte
	switch ve := b.(type) {
	case *[]byte:
		v = *ve
	case *string:
		v = []byte(*ve)
	case string:
		v = []byte(ve)
	case []byte:
		v = ve
	default:
		return fmt.Errorf("failed to write: %v is not type of *[]byte or []byte", b)
	}
	_, err := c.Conn.Write(v)
	return err
}

func (c *Codec) Close() error {
	return c.Conn.Close()
}
