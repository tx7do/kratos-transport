package websocket

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
)

type Any interface{}
type MessageType uint32
type MessagePayload Any

type BinaryMessage struct {
	Type MessageType
	Body []byte
}

func (m *BinaryMessage) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(m.Type)); err != nil {
		return nil, err
	}
	buf.Write(m.Body)
	return buf.Bytes(), nil
}

func (m *BinaryMessage) Unmarshal(buf []byte) error {
	network := new(bytes.Buffer)
	network.Write(buf)

	if err := binary.Read(network, binary.LittleEndian, &m.Type); err != nil {
		return err
	}

	m.Body = network.Bytes()

	return nil
}

type TextMessage struct {
	Type MessageType `json:"type" xml:"type"`
	Body string      `json:"body" xml:"body"`
}

func (m *TextMessage) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *TextMessage) Unmarshal(buf []byte) error {
	return json.Unmarshal(buf, m)
}
