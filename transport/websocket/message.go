package websocket

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
)

type NetMessageType uint32
type MessagePayload any

type BinaryNetPacket struct {
	Type    NetMessageType
	Payload []byte
}

func (m *BinaryNetPacket) Marshal() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, uint32(m.Type)); err != nil {
		return nil, err
	}
	buf.Write(m.Payload)
	return buf.Bytes(), nil
}

func (m *BinaryNetPacket) Unmarshal(buf []byte) error {
	network := new(bytes.Buffer)
	network.Write(buf)

	if err := binary.Read(network, binary.LittleEndian, &m.Type); err != nil {
		return err
	}

	m.Payload = network.Bytes()

	return nil
}

type TextNetPacket struct {
	Type    NetMessageType `json:"type" xml:"type"`
	Payload string         `json:"payload" xml:"payload"`
}

func (m *TextNetPacket) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *TextNetPacket) Unmarshal(buf []byte) error {
	return json.Unmarshal(buf, m)
}
