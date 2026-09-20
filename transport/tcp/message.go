package tcp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type NetMessageType uint32
type NetMessagePayload any

const (
	// frameLengthSize 帧长度前缀字节数
	frameLengthSize = 4
	// maxFrameSize 单帧最大字节数，超过视为协议错误
	maxFrameSize = 32 << 20
)

type NetPacket struct {
	Type    NetMessageType
	Payload []byte
}

func (m *NetPacket) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, byteOrder, uint32(m.Type)); err != nil {
		return nil, err
	}
	buf.Write(m.Payload)
	return buf.Bytes(), nil
}

func (m *NetPacket) Unmarshal(buf []byte) error {
	network := new(bytes.Buffer)
	network.Write(buf)

	if err := binary.Read(network, byteOrder, &m.Type); err != nil {
		return err
	}

	m.Payload = network.Bytes()

	return nil
}

// WriteFrame 写入一帧：4 字节长度前缀 + payload。
// TCP 是字节流，不加分帧前缀的话，对端无法区分包边界（粘包/拆包）。
func WriteFrame(conn net.Conn, payload []byte) error {
	if len(payload) > maxFrameSize {
		return errors.New("frame too large")
	}

	buf := make([]byte, frameLengthSize+len(payload))
	byteOrder.PutUint32(buf, uint32(len(payload)))
	copy(buf[frameLengthSize:], payload)

	_, err := conn.Write(buf)
	return err
}

// ReadFrame 读取一帧，返回去掉长度前缀的 payload。
func ReadFrame(conn net.Conn) ([]byte, error) {
	hdr := make([]byte, frameLengthSize)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return nil, err
	}

	size := byteOrder.Uint32(hdr)
	if size > maxFrameSize {
		return nil, errors.New("invalid frame size")
	}
	if size == 0 {
		// 与 WriteFrame 对称：空 payload 合法，不再视为致命协议错误
		return []byte{}, nil
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	return payload, nil
}
