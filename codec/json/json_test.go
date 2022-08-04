package json

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"sync"
	"testing"
)

type Hygrothermograph struct {
	Humidity    float64 `json:"humidity"`
	Temperature float64 `json:"temperature"`
}

type readWriteCloser struct {
	sync.RWMutex
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

func (rwc *readWriteCloser) Read(p []byte) (n int, err error) {
	rwc.RLock()
	defer rwc.RUnlock()
	return rwc.rbuf.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (n int, err error) {
	rwc.Lock()
	defer rwc.Unlock()
	return rwc.wbuf.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	return nil
}

type safeBuffer struct {
	sync.RWMutex
	buf []byte
	off int
}

func (b *safeBuffer) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	p2 := make([]byte, len(p))
	copy(p2, p)
	b.Lock()
	b.buf = append(b.buf, p2...)
	b.Unlock()
	return len(p2), nil
}

func (b *safeBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	b.RLock()
	n = copy(p, b.buf[b.off:])
	b.RUnlock()
	if n == 0 {
		return 0, io.EOF
	}
	b.off += n
	return n, nil
}

func (b *safeBuffer) Close() error {
	return nil
}

func TestMarshal(t *testing.T) {
	codec := Marshaler{}
	assert.Equal(t, "json", codec.Name())

	var msg Hygrothermograph
	msg.Humidity = 100
	msg.Temperature = 200

	buf, err := codec.Marshal(msg)
	assert.Nil(t, err)

	var msg1 Hygrothermograph
	var body interface{}
	body = &msg1
	err = codec.Unmarshal(buf, body)
	assert.Nil(t, err)
	msg2, _ := body.(*Hygrothermograph)
	assert.Equal(t, msg, *msg2)
}

func TestCodec(t *testing.T) {
	rwc := readWriteCloser{
		rbuf: bytes.NewBuffer(nil),
		wbuf: bytes.NewBuffer(nil),
	}
	codec := NewCodec(&rwc)
	assert.Equal(t, "json", codec.Name())

	var msg Hygrothermograph
	msg.Humidity = 100
	msg.Temperature = 200

	err := codec.Write(&msg)
	assert.Nil(t, err)

	fmt.Printf("buf: %s\n", rwc.wbuf.String())

	rwc.rbuf.Reset()
	rwc.rbuf.Write(rwc.wbuf.Bytes())

	var msg1 Hygrothermograph
	err = codec.Read(&msg1)
	assert.Nil(t, err)
	assert.Equal(t, msg, msg1)
}

func TestCodec1(t *testing.T) {
	buffer := new(safeBuffer)
	codec := NewCodec(buffer)
	assert.Equal(t, "json", codec.Name())

	var msg Hygrothermograph
	msg.Humidity = 100
	msg.Temperature = 200

	err := codec.Write(&msg)
	assert.Nil(t, err)

	var msg1 Hygrothermograph
	err = codec.Read(&msg1)
	assert.Nil(t, err)
}

func TestMemory(t *testing.T) {
	var i int = 20
	fmt.Printf("i: %p\n", &i)
	i1 := &i
	fmt.Printf("i1: %p\n", i1)
	i2 := i
	fmt.Printf("i2: %p\n", &i2)

	var a1 []byte
	fmt.Printf("a1: %p\n", a1)
	a2 := a1
	fmt.Printf("a2: %p\n", a2)

	type Bytes []byte

	s1 := "hello"
	fmt.Printf("s1: %p\n", &s1)
	b1 := []byte(s1) // 产生内存分配
	fmt.Printf("b1: %p\n", b1)
	b2 := b1
	fmt.Printf("b2: %p\n", b2)
	b3 := Bytes(b1)
	fmt.Printf("b3: %p\n", b3)
	b2[0] = 'c'
	fmt.Printf("s1: %s\n", s1)
	fmt.Printf("b1: %s\n", string(b1))
	fmt.Printf("b2: %s\n", string(b2))

	s2 := string(b2) // 产生内存分配
	fmt.Printf("s2: %p\n", &s2)
	s2 = "world"
	fmt.Printf("s2: %p\n", &s2)
	fmt.Printf("b1: %s\n", string(b1))
	fmt.Printf("s2: %s\n", s2)
}
