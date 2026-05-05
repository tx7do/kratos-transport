package webrtc

import (
	"crypto/tls"
	"net"
	"net/http"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/pion/webrtc/v4"
)

type PayloadType uint8

const (
	PayloadTypeBinary = 0
	PayloadTypeText   = 1
)

type ServerOption func(o *Server)

func WithNetwork(network string) ServerOption {
	return func(s *Server) {
		if network != "" {
			s.network = network
		}
	}
}

func WithAddress(addr string) ServerOption {
	return func(s *Server) {
		if addr != "" {
			s.address = addr
		}
	}
}

func WithStrictSlash(strictSlash bool) ServerOption {
	return func(s *Server) {
		s.strictSlash = strictSlash
	}
}

func WithPath(path string) ServerOption {
	return func(s *Server) {
		if path != "" {
			s.path = path
		}
	}
}

func WithTLSConfig(c *tls.Config) ServerOption {
	return func(o *Server) {
		o.tlsConf = c
	}
}

func WithListener(lis net.Listener) ServerOption {
	return func(s *Server) {
		s.lis = lis
	}
}

func WithCodec(c string) ServerOption {
	return func(s *Server) {
		if c != "" {
			s.codec = encoding.GetCodec(c)
		}
	}
}

// WithReadBufferSize is kept for backward compatibility and is ignored by WebRTC transport.
func WithReadBufferSize(_ int) ServerOption {
	return func(_ *Server) {}
}

// WithWriteBufferSize is kept for backward compatibility and is ignored by WebRTC transport.
func WithWriteBufferSize(_ int) ServerOption {
	return func(_ *Server) {}
}

func WithCheckOrigin(domain string) ServerOption {
	return func(s *Server) {
		s.checkOrigin = func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == domain
		}
	}
}

func WithCheckOriginFunc(fn func(*http.Request) bool) ServerOption {
	return func(s *Server) {
		if fn != nil {
			s.checkOrigin = fn
		}
	}
}

func WithEnableCORS(enable bool) ServerOption {
	return func(s *Server) {
		s.enableCORS = enable
	}
}

func WithCORS(allowOrigin, allowMethods, allowHeaders string) ServerOption {
	return func(s *Server) {
		if allowOrigin != "" {
			s.corsAllowOrigin = allowOrigin
		}
		if allowMethods != "" {
			s.corsAllowMethods = allowMethods
		}
		if allowHeaders != "" {
			s.corsAllowHeaders = allowHeaders
		}
	}
}

func WithChannelBufferSize(size int) ServerOption {
	return func(_ *Server) {
		if size > 0 {
			channelBufSize = size
		}
	}
}

func WithPayloadType(payloadType PayloadType) ServerOption {
	return func(s *Server) {
		s.payloadType = payloadType
	}
}

func WithInjectTokenToQuery(enable bool, tokenKey string) ServerOption {
	return func(s *Server) {
		s.injectToken = enable
		s.tokenKey = tokenKey
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

func WithWebRTCConfiguration(cfg webrtc.Configuration) ServerOption {
	return func(s *Server) {
		s.webrtcConfig = cfg
	}
}

func WithWebRTCAPI(api *webrtc.API) ServerOption {
	return func(s *Server) {
		s.webrtcAPI = api
	}
}

func WithDataChannelLabel(label string) ServerOption {
	return func(s *Server) {
		s.dataChannelLabel = label
	}
}

func WithAllowAnyDataChannelLabel(allow bool) ServerOption {
	return func(s *Server) {
		s.allowAnyDataLabel = allow
	}
}
