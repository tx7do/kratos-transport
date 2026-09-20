package webtransport

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	// frameLengthSize 帧长度前缀字节数
	frameLengthSize = 4
	// maxFrameSize 单帧最大字节数，超过视为协议错误
	maxFrameSize = 32 << 20
)

// WriteFrame 在流上写入一帧：4 字节小端长度前缀 + payload。
// QUIC 流是字节流，需要长度前缀来分包（与 server.serveSession 的分包逻辑对应）。
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > maxFrameSize {
		return errors.New("frame too large")
	}

	buf := make([]byte, frameLengthSize+len(payload))
	binary.LittleEndian.PutUint32(buf, uint32(len(payload)))
	copy(buf[frameLengthSize:], payload)

	_, err := w.Write(buf)
	return err
}

// ReadFrame 从流上读取一帧，返回去掉长度前缀的 payload。
func ReadFrame(r io.Reader) ([]byte, error) {
	hdr := make([]byte, frameLengthSize)
	if _, err := io.ReadFull(r, hdr); err != nil {
		return nil, err
	}

	size := binary.LittleEndian.Uint32(hdr)
	if size > maxFrameSize {
		return nil, errors.New("invalid frame size")
	}
	if size == 0 {
		// 与 WriteFrame 对称：空 payload 合法，不再视为致命协议错误
		return []byte{}, nil
	}

	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	return payload, nil
}
