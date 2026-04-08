package tcp

import "encoding/binary"

// default byte order is little endian.
var byteOrder binary.ByteOrder = binary.LittleEndian

// WithLittleEndian set little endian
func WithLittleEndian() {
	byteOrder = binary.LittleEndian
}

// WithBigEndian set big endian
func WithBigEndian() {
	byteOrder = binary.BigEndian
}
