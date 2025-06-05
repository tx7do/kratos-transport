package tcp

import (
	"crypto/tls"
	"encoding/binary"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
)

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

////////////////////////////////////////////////////////////////////////////////

type ServerOption func(o *Server)

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

func WithTimeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = c
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		s.codec = encoding.GetCodec(c)
	}
}

func WithChannelBufferSize(size int) ServerOption {
	return func(_ *Server) {
		channelBufSize = size
	}
}

func WithReceiveBufferSize(size int) ServerOption {
	return func(_ *Server) {
		recvBufferSize = size
	}
}

func WithMessageMarshaler(m NetPacketMarshaler) ServerOption {
	return func(s *Server) {
		s.netPacketMarshaler = m
	}
}

func WithMessageUnmarshaler(m NetPacketUnmarshaler) ServerOption {
	return func(s *Server) {
		s.netPacketUnmarshaler = m
	}
}

func WithSocketConnectHandler(h SocketConnectHandler) ServerOption {
	return func(s *Server) {
		s.socketConnectHandler = h
	}
}

func WithSocketRawDataHandler(h SocketRawDataHandler) ServerOption {
	return func(s *Server) {
		if h != nil {
			s.socketRawDataHandler = h
		}
	}
}
