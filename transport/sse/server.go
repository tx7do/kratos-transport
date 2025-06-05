package sse

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/errors"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/gorilla/mux"

	"github.com/tx7do/kratos-transport/broker"
)

type MessagePayload any

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
	_ http.Handler               = (*Server)(nil)
)

type Server struct {
	*http.Server

	lis      net.Listener
	tlsConf  *tls.Config
	endpoint *url.URL

	network     string
	address     string
	path        string
	streamIdKey string

	timeout time.Duration

	err   error
	codec encoding.Codec

	router      *mux.Router
	strictSlash bool

	headers    map[string]string
	eventTTL   time.Duration
	bufferSize int

	encodeBase64 bool
	splitData    bool
	autoStream   bool
	autoReplay   bool

	subscribeFunc   SubscriberFunction
	unsubscribeFunc SubscriberFunction

	streamMgr *StreamManager
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		router:      mux.NewRouter(),
		strictSlash: true,
		path:        "/",
		streamIdKey: "stream",

		bufferSize:   DefaultBufferSize,
		encodeBase64: false,

		autoStream: false,
		autoReplay: true,
		headers:    map[string]string{},

		streamMgr: NewStreamManager(),
	}

	srv.init(opts...)

	srv.err = srv.listen()

	return srv
}

func (s *Server) Name() string {
	return string(KindSSE)
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	s.BaseContext = func(net.Listener) context.Context {
		return ctx
	}

	LogInfof("server listening on: %s", s.lis.Addr().String())

	s.HandleServeHTTP(s.path)

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

	LogInfo("server stopping")
	return s.Shutdown(ctx)
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			return err
		}
		s.lis = lis
	}

	if s.endpoint == nil {
		endpoint, err := url.Parse(s.network + "://" + s.address)
		if err != nil {
			return err
		}
		s.endpoint = endpoint
	}

	return nil
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

func (s *Server) Publish(_ context.Context, streamId StreamID, event *Event) {
	stream := s.streamMgr.Get(streamId)
	if stream == nil {
		return
	}

	select {
	case <-stream.quit:
	case stream.event <- s.process(event):
	}
}

func (s *Server) TryPublish(_ context.Context, streamId StreamID, event *Event) bool {
	stream := s.streamMgr.Get(streamId)
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

func (s *Server) PublishData(ctx context.Context, streamId StreamID, data MessagePayload) error {
	event := &Event{}

	if data != nil {
		var err error
		event.Data, err = broker.Marshal(s.codec, data)
		if err != nil {
			return err
		}
	}

	s.Publish(ctx, streamId, event)

	return nil
}

func (s *Server) Notify(_ context.Context, event *Event) {
	s.streamMgr.Range(func(stream *Stream) {
		if stream == nil {
			return
		}

		select {
		case <-stream.quit:
		case stream.event <- s.process(event):
		}
	})
}

func (s *Server) NotifyData(_ context.Context, data MessagePayload) error {
	event := &Event{}

	if data != nil {
		var err error
		event.Data, err = broker.Marshal(s.codec, data)
		if err != nil {
			return err
		}
	}

	s.streamMgr.Range(func(stream *Stream) {
		if stream == nil {
			return
		}

		select {
		case <-stream.quit:
		case stream.event <- s.process(event):
		}
	})

	return nil
}

func (s *Server) run() {
}

func (s *Server) createStream(streamId StreamID) *Stream {
	stream := newStream(streamId, s.bufferSize, s.autoReplay, s.autoStream, s.subscribeFunc, s.unsubscribeFunc)
	stream.run()
	return stream
}

func (s *Server) CreateStream(streamId StreamID) *Stream {
	stream := s.streamMgr.Get(streamId)
	if stream != nil {
		return stream
	}

	stream = s.createStream(streamId)

	s.streamMgr.Add(stream)

	return stream
}

func (s *Server) process(event *Event) *Event {
	if s.encodeBase64 {
		event.encodeBase64()
	}
	return event
}
