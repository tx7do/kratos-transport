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
	"github.com/tx7do/kratos-transport/transport"
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

	corsAllowOrigin string

	subscribeFunc   SubscriberFunction
	unsubscribeFunc SubscriberFunction
	authorizeFunc   AuthorizeFunc
	tokenExtractor  TokenExtractor

	streamMgr *StreamManager
}

// NewServer creates and initializes an SSE server with the provided options.
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

		corsAllowOrigin: "*",

		autoStream: false,
		autoReplay: true,
		headers:    map[string]string{},

		tokenExtractor: DefaultTokenExtractor,

		streamMgr: NewStreamManager(),
	}

	srv.init(opts...)

	srv.err = srv.listen()

	return srv
}

// Name returns the transport kind name.
func (s *Server) Name() string {
	return KindSSE
}

// Start starts serving SSE requests until the server is shut down.
func (s *Server) Start(ctx context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return err
	}

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

// Stop gracefully stops the server and cleans all streams.
func (s *Server) Stop(ctx context.Context) error {
	LogInfo("server stopping...")

	s.streamMgr.Clean()

	err := s.Shutdown(ctx)

	// Shutdown 后 http.Server 不可复用：重建实例并释放 listener，支持重启
	s.rebuildHTTPServer()
	if s.lis != nil {
		_ = s.lis.Close()
		s.lis = nil
	}
	s.endpoint = nil
	s.err = nil

	LogInfo("server stopped.")

	return err
}

// Endpoint returns the registry endpoint of this server.
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
		// If a full address is provided, keep it as-is.
		// If only a port is provided, derive a full address from the listener.
		// Note: the inferred IP may not always be the expected external address.
		addr, err := transport.AdjustAddress(s.address, s.lis)
		if err != nil {
			s.err = err
			return err
		}

		s.endpoint = transport.NewRegistryEndpoint(KindSSE, addr)
	}

	return nil
}

// Handle registers an HTTP handler for an exact path.
func (s *Server) Handle(path string, h http.Handler) {
	s.router.Handle(path, h)
}

// HandlePrefix registers an HTTP handler for a path prefix.
func (s *Server) HandlePrefix(prefix string, h http.Handler) {
	s.router.PathPrefix(prefix).Handler(h)
}

// HandleFunc registers an HTTP handler function for an exact path.
func (s *Server) HandleFunc(path string, h http.HandlerFunc) {
	s.router.HandleFunc(path, h)
}

// HandleHeader registers an HTTP handler function matching a specific header key/value.
func (s *Server) HandleHeader(key, val string, h http.HandlerFunc) {
	s.router.Headers(key, val).Handler(h)
}

// HandleServeHTTP binds the server's SSE handler to the specified path.
func (s *Server) HandleServeHTTP(path string) {
	s.router.HandleFunc(path, s.ServeHTTP)
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.router.StrictSlash(s.strictSlash)
	s.router.NotFoundHandler = http.NotFoundHandler()
	s.router.MethodNotAllowedHandler = http.NotFoundHandler()

	s.rebuildHTTPServer()
}

// rebuildHTTPServer 重建被 Shutdown 毒化的 http.Server（重启支持）
func (s *Server) rebuildHTTPServer() {
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

// Publish pushes an event to a specific stream if it exists.
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

// TryPublish attempts to push an event to a stream without blocking.
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

// PublishData encodes arbitrary data into an Event and publishes it to the target stream.
func (s *Server) PublishData(ctx context.Context, streamId StreamID, data MessagePayload) error {
	event, err := s.marshalEvent(data)
	if err != nil {
		return err
	}

	s.Publish(ctx, streamId, event)

	return nil
}

// PublishDataWithEventName encodes arbitrary data into an Event, sets its SSE event name,
// and publishes it to the target stream.
func (s *Server) PublishDataWithEventName(ctx context.Context, streamId StreamID, eventName string, data MessagePayload) error {
	return s.PublishDataWithMeta(ctx, streamId, data, WithEventName(eventName))
}

// PublishDataWithMeta encodes arbitrary data into an Event, applies the given metadata options,
// and publishes it to the target stream.
func (s *Server) PublishDataWithMeta(ctx context.Context, streamId StreamID, data MessagePayload, opts ...EventMetaOption) error {
	event, err := s.marshalEvent(data)
	if err != nil {
		return err
	}

	for _, o := range opts {
		o(event)
	}

	s.Publish(ctx, streamId, event)

	return nil
}

// Notify broadcasts an event to all registered streams.
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

// NotifyData encodes arbitrary data into an Event and broadcasts it to all registered streams.
func (s *Server) NotifyData(ctx context.Context, data MessagePayload) error {
	event, err := s.marshalEvent(data)
	if err != nil {
		return err
	}

	s.Notify(ctx, event)

	return nil
}

// NotifyDataWithEventName encodes arbitrary data into an Event, sets its SSE event name,
// and broadcasts it to all registered streams.
func (s *Server) NotifyDataWithEventName(ctx context.Context, eventName string, data MessagePayload) error {
	return s.NotifyDataWithMeta(ctx, data, WithEventName(eventName))
}

// NotifyDataWithMeta encodes arbitrary data into an Event, applies the given metadata options,
// and broadcasts it to all registered streams.
func (s *Server) NotifyDataWithMeta(ctx context.Context, data MessagePayload, opts ...EventMetaOption) error {
	event, err := s.marshalEvent(data)
	if err != nil {
		return err
	}

	for _, o := range opts {
		o(event)
	}

	s.Notify(ctx, event)

	return nil
}

func (s *Server) run() {
}

func (s *Server) createStream(streamId StreamID) *Stream {
	stream := newStream(streamId, s.bufferSize, s.autoReplay, s.autoStream, s.subscribeFunc, s.unsubscribeFunc)
	stream.run()
	return stream
}

// CreateStream creates a new stream, or returns the existing one if it already exists.
func (s *Server) CreateStream(streamId StreamID) *Stream {
	stream := s.streamMgr.Get(streamId)
	if stream != nil {
		return stream
	}

	stream = s.createStream(streamId)

	s.streamMgr.Add(stream)

	return stream
}

// process applies server-side event transformations before delivery.
func (s *Server) process(event *Event) *Event {
	if s.encodeBase64 {
		event.encodeBase64()
	}
	return event
}

// marshalEvent converts an arbitrary payload into an Event.
func (s *Server) marshalEvent(data MessagePayload) (*Event, error) {
	event := &Event{}
	if data != nil {
		var err error
		event.Data, err = broker.Marshal(s.codec, data)
		if err != nil {
			return nil, err
		}
	}
	return event, nil
}
