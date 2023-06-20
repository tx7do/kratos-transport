package sse

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/gorilla/mux"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
	_ http.Handler         = (*Server)(nil)
)

type Server struct {
	*http.Server

	lis     net.Listener
	tlsConf *tls.Config

	network string
	address string

	timeout time.Duration

	err   error
	codec encoding.Codec

	router      *mux.Router
	strictSlash bool

	Headers      map[string]string
	EventTTL     time.Duration
	bufferSize   int
	encodeBase64 bool
	SplitData    bool
	AutoStream   bool
	AutoReplay   bool

	OnSubscribe   SubscriberFunction
	OnUnsubscribe SubscriberFunction

	subscriber   chan *Subscriber
	unsubscriber chan *Subscriber

	streamMgr *StreamManager
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		router:      mux.NewRouter(),
		strictSlash: true,

		bufferSize:   DefaultBufferSize,
		encodeBase64: false,

		AutoStream: false,
		AutoReplay: true,
		Headers:    map[string]string{},

		streamMgr: NewStreamManager(),
	}

	srv.init(opts...)

	srv.err = srv.listen()

	return srv
}

func (s *Server) Name() string {
	return "sse"
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}
	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}
	log.Infof("[sse] server listening on: %s", s.lis.Addr().String())

	var err error
	if s.tlsConf != nil {
		err = s.ServeTLS(s.lis, "", "")
	} else {
		err = s.Serve(s.lis)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.streamMgr.Clean()

	log.Info("[sse] server stopping")
	return s.Shutdown(ctx)
}

func (s *Server) Endpoint() (*url.URL, error) {
	addr := s.address

	prefix := "http://"
	if s.tlsConf == nil {
		if !strings.HasPrefix(addr, "http://") {
			prefix = "http://"
		}
	} else {
		if !strings.HasPrefix(addr, "https://") {
			prefix = "https://"
		}
	}
	addr = prefix + addr

	var endpoint *url.URL
	endpoint, s.err = url.Parse(addr)
	return endpoint, nil
}

func (s *Server) Handle(path string, h http.Handler) {
	s.router.Handle(path, h)
}

func (s *Server) HandlePrefix(prefix string, h http.Handler) {
	s.router.PathPrefix(prefix).Handler(h)
}

func (s *Server) HandleFunc(path string, h http.HandlerFunc) {
	s.router.HandleFunc(path, h)
}

func (s *Server) HandleHeader(key, val string, h http.HandlerFunc) {
	s.router.Headers(key, val).Handler(h)
}

func (s *Server) HandleServeHTTP(path string) {
	s.router.HandleFunc(path, s.ServeHTTP)
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.router.StrictSlash(s.strictSlash)
	s.router.NotFoundHandler = http.DefaultServeMux
	s.router.MethodNotAllowedHandler = http.DefaultServeMux

	s.Server = &http.Server{
		Handler:   s.router,
		TLSConfig: s.tlsConf,
	}
}

func (s *Server) listen() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	return nil
}

func (s *Server) Publish(id StreamID, event *Event) {
	stream := s.streamMgr.Get(id)
	if stream == nil {
		return
	}

	select {
	case <-stream.quit:
	case stream.event <- s.process(event):
	}
}

func (s *Server) TryPublish(id StreamID, event *Event) bool {
	stream := s.streamMgr.Get(id)
	if stream == nil {
		return false
	}

	select {
	case stream.event <- s.process(event):
		return true
	default:
		return false
	}
}

func (s *Server) run() {
	//for {
	//	select {
	//	case client := <-s.subscriber:
	//		s.CreateStream(client)
	//	case client := <-s.unsubscriber:
	//		s.removeStream(client)
	//	}
	//}
}

func (s *Server) createStream(id StreamID) *Stream {
	stream := newStream(id, s.bufferSize, s.AutoReplay, s.AutoStream, s.OnSubscribe, s.OnUnsubscribe)
	stream.run()
	return stream
}

func (s *Server) CreateStream(id StreamID) *Stream {
	stream := s.streamMgr.Get(id)
	if stream != nil {
		return stream
	}

	stream = s.createStream(id)

	s.streamMgr.Add(stream)

	return stream
}

func (s *Server) process(event *Event) *Event {
	if s.encodeBase64 {
		event.encodeBase64()
	}
	return event
}
