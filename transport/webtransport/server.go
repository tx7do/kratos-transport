package webtransport

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/lucas-clemente/quic-go/quicvarint"
)

type Binder func() Any

type ConnectHandler func(SessionID, bool)

type MessageHandler func(SessionID, MessagePayload) error

type HandlerData struct {
	Handler MessageHandler
	Binder  Binder
}
type MessageHandlerMap map[MessageType]HandlerData

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http3.Server

	tlsConf  *tls.Config
	endpoint *url.URL
	timeout  time.Duration

	upgrader    *Upgrader
	mux         *http.ServeMux
	path        string
	strictSlash bool

	ctx       context.Context // is closed when Close is called
	ctxCancel context.CancelFunc
	refCount  sync.WaitGroup

	messageHandlers MessageHandlerMap
	connectHandler  ConnectHandler

	sessions *sessionManager
}

func NewServer(opts ...ServerOption) *Server {
	ctx, ctxCancel := context.WithCancel(context.Background())
	srv := &Server{
		ctx:       ctx,
		ctxCancel: ctxCancel,
		mux:       http.NewServeMux(),
		upgrader: &Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
	srv.init(opts...)
	return srv
}

func (s *Server) init(opts ...ServerOption) {
	s.Server = &http3.Server{
		Addr: ":443",
	}

	for _, o := range opts {
		o(s)
	}

	if s.tlsConf == nil {
		s.tlsConf = generateTLSConfig(alpnQuicTransport)
	}
	s.Server.TLSConfig = s.tlsConf

	timeout := s.timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	s.sessions = newSessionManager(timeout)

	if s.Server.AdditionalSettings == nil {
		s.Server.AdditionalSettings = make(map[uint64]uint64)
	}
	s.Server.AdditionalSettings[settingsEnableWebtransport] = 1
	s.Server.EnableDatagrams = true
	s.Server.StreamHijacker = func(ft http3.FrameType, qconn quic.Connection, str quic.Stream, err error) (bool /* hijacked */, error) {
		if isWebTransportError(err) {
			return true, nil
		}
		if ft != webTransportFrameType {
			return false, nil
		}
		id, err := quicvarint.Read(quicvarint.NewReader(str))
		if err != nil {
			if isWebTransportError(err) {
				return true, nil
			}
			return false, err
		}
		s.sessions.AddStream(qconn, str, SessionID(id))
		return true, nil
	}
	s.Server.UniStreamHijacker = func(st http3.StreamType, qconn quic.Connection, str quic.ReceiveStream, err error) (hijacked bool) {
		if st != webTransportUniStreamType && !isWebTransportError(err) {
			return false
		}
		s.sessions.AddUniStream(qconn, str)
		return true
	}

	s.mux.HandleFunc(s.path, s.addHandler)
	s.Server.Handler = s.mux
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.endpoint == nil {
		var err error
		s.endpoint, err = url.Parse(s.Addr)
		return s.endpoint, err
	}
	return s.endpoint, nil
}

func (s *Server) Start(ctx context.Context) error {
	log.Infof("[webtransport] server listening on: %s", s.Addr)

	if err := s.ListenAndServe(); err != nil {
		log.Errorf("[webtransport] start server failed: %s", err.Error())
		return err
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Info("[webtransport] server stopping")
	if s.ctxCancel != nil {
		s.ctxCancel()
	}
	if s.sessions != nil {
		s.sessions.Close()
	}
	err := s.Server.Close()
	s.refCount.Wait()
	return err
}

func (s *Server) addHandler(w http.ResponseWriter, r *http.Request) {
	httpStreamer, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("[webtransport] upgrade exception:", err)
		return
	}

	str := httpStreamer.HTTPStream()
	sID := SessionID(str.StreamID())

	hijacker, ok := w.(http3.Hijacker)
	if !ok { // should never happen, unless quic-go changed the API
		log.Error("[webtransport] failed to hijack")
		return
	}

	session := s.sessions.AddSession(
		hijacker.StreamCreator(),
		sID,
		r.Body.(http3.HTTPStreamer).HTTPStream(),
	)

	if s.connectHandler != nil {
		s.connectHandler(session.SessionID(), true)
	}
}
