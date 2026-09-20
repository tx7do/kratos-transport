package tcp

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	payloads := [][]byte{
		[]byte("hello"),
		make([]byte, 70000), // 大帧，验证长度前缀跨段正确
		[]byte("tiny"),
	}
	binary.LittleEndian.PutUint32(payloads[1][:4], 0xdeadbeef)

	go func() {
		for _, p := range payloads {
			if err := WriteFrame(c1, p); err != nil {
				t.Errorf("write frame: %v", err)
			}
		}
	}()

	for i, want := range payloads {
		got, err := ReadFrame(c2)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if string(got) != string(want) {
			t.Fatalf("frame %d mismatch: got %d bytes want %d", i, len(got), len(want))
		}
	}
}

func TestFrameRoundTripFragmented(t *testing.T) {
	// 模拟粘包：三帧连写，逐帧读出
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	go func() {
		buf := []byte{}
		for _, p := range [][]byte{[]byte("a"), []byte("bb"), []byte("ccc")} {
			f := make([]byte, 4+len(p))
			binary.LittleEndian.PutUint32(f, uint32(len(p)))
			copy(f[4:], p)
			buf = append(buf, f...)
		}
		_, _ = c1.Write(buf) // 一次性写入，模拟粘包
	}()

	for i, want := range []string{"a", "bb", "ccc"} {
		got, err := ReadFrame(c2)
		if err != nil {
			t.Fatalf("frame %d: %v", i, err)
		}
		if string(got) != want {
			t.Fatalf("frame %d: got %q want %q", i, got, want)
		}
	}
}
