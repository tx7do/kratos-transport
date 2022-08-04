package json

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Hygrothermograph struct {
	Humidity    float64 `json:"humidity"`
	Temperature float64 `json:"temperature"`
}

func TestCodec(t *testing.T) {
	codec := Marshaler{}

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
	assert.Equal(t, msg, msg2)
}

func TestPoint(t *testing.T) {
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
