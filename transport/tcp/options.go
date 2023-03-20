package tcp

import (
	"crypto/tls"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
)

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

func WithConnectHandler(h ConnectHandler) ServerOption {
	return func(s *Server) {
		s.connectHandler = h
	}
}

func WithRawDataHandler(h RawMessageHandler) ServerOption {
	return func(s *Server) {
		s.rawMessageHandler = h
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

func WithRecvBufferSize(size int) ServerOption {
	return func(_ *Server) {
		recvBufferSize = size
	}
}

////////////////////////////////////////////////////////////////////////////////

type ClientOption func(o *Client)

func WithClientCodec(codec string) ClientOption {
	return func(c *Client) {
		c.codec = encoding.GetCodec(codec)
	}
}

func WithEndpoint(uri string) ClientOption {
	return func(c *Client) {
		c.url = uri
	}
}

func WithClientRawDataHandler(h ClientRawMessageHandler) ClientOption {
	return func(c *Client) {
		c.rawMessageHandler = h
	}
}
