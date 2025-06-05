package tcp

import (
	"bytes"
	"encoding/binary"
)

type NetMessageType uint32
type NetMessagePayload any

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
