package webtransport

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/tx7do/kratos-transport/broker"
	"io"
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
	codec           encoding.Codec

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
		messageHandlers: make(MessageHandlerMap),
		codec:           encoding.GetCodec("json"),
	}
	srv.init(opts...)
	return srv
}

func (s *Server) init(opts ...ServerOption) {
	const idleTimeout = 30 * time.Second

	s.Server = &http3.Server{
		Addr: ":443",
		QuicConfig: &quic.Config{
			MaxIdleTimeout:  idleTimeout,
			KeepAlivePeriod: idleTimeout / 2,
		},
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
	s.Server.StreamHijacker = func(ft http3.FrameType, qConn quic.Connection, qStream quic.Stream, err error) (bool /* hijacked */, error) {
		if isWebTransportError(err) {
			return true, nil
		}
		if ft != webTransportFrameType {
			return false, nil
		}
		id, err := quicvarint.Read(quicvarint.NewReader(qStream))
		if err != nil {
			if isWebTransportError(err) {
				return true, nil
			}
			return false, err
		}
		s.sessions.AddStream(qConn, qStream, SessionID(id))
		return true, nil
	}
	s.Server.UniStreamHijacker = func(st http3.StreamType, qConn quic.Connection, qStream quic.ReceiveStream, err error) (hijacked bool) {
		if st != webTransportUniStreamType && !isWebTransportError(err) {
			return false
		}
		s.sessions.AddUniStream(qConn, qStream)
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

func (s *Server) RegisterMessageHandler(messageType MessageType, handler MessageHandler, binder Binder) {
	if _, ok := s.messageHandlers[messageType]; ok {
		return
	}

	s.messageHandlers[messageType] = HandlerData{
		handler, binder,
	}
}

func (s *Server) DeregisterMessageHandler(messageType MessageType) {
	delete(s.messageHandlers, messageType)
}

func (s *Server) marshalMessage(messageType MessageType, message MessagePayload) ([]byte, error) {
	var err error
	var msg Message
	msg.Type = messageType
	msg.Body, err = broker.Marshal(s.codec, message)
	if err != nil {
		return nil, err
	}

	buff, err := msg.Marshal()
	if err != nil {
		return nil, err
	}

	return buff, nil
}

func (s *Server) messageHandler(sessionId SessionID, buf []byte) error {
	var msg Message
	if err := msg.Unmarshal(buf); err != nil {
		log.Errorf("[webtransport] decode message exception: %s", err)
		return err
	}

	handlerData, ok := s.messageHandlers[msg.Type]
	if !ok {
		log.Error("[webtransport] message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	if handlerData.Binder != nil {
		payload = handlerData.Binder()
	}

	if err := broker.Unmarshal(s.codec, msg.Body, payload); err != nil {
		log.Errorf("[webtransport] unmarshal message exception: %s", err)
		return err
	}

	if err := handlerData.Handler(sessionId, payload); err != nil {
		log.Errorf("[webtransport] message handler exception: %s", err)
		return err
	}

	return nil
}

func (s *Server) addHandler(w http.ResponseWriter, r *http.Request) {
	httpStreamer, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("[webtransport] upgrade exception:", err)
		w.WriteHeader(404)
		return
	}

	httpStream := httpStreamer.HTTPStream()
	sID := SessionID(httpStream.StreamID())

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

	go s.doAcceptStream(session)
}

func (s *Server) doAcceptStream(session *Session) {
	for {
		acceptStream, err := session.AcceptStream(s.ctx)
		if err != nil {
			log.Debug("[webtransport] accept stream failed: ", err.Error())
			if s.connectHandler != nil {
				s.connectHandler(session.SessionID(), false)
			}
			break
		}
		data, err := io.ReadAll(acceptStream)
		if err != nil {
			log.Error("[webtransport] read data failed: ", err.Error())
		}
		//log.Debug("receive data: ", string(data))
		_ = s.messageHandler(session.SessionID(), data)
	}
}

func (s *Server) doAcceptUniStream(session *Session) {
	for {
		acceptStream, err := session.AcceptUniStream(s.ctx)
		if err != nil {
			log.Debug("[webtransport] accept uni stream failed: ", err.Error())
			if s.connectHandler != nil {
				s.connectHandler(session.SessionID(), false)
			}
			break
		}
		data, err := io.ReadAll(acceptStream)
		if err != nil {
			log.Error("[webtransport] read uni data failed: ", err.Error())
		}
		//log.Debug("receive data: ", string(data))
		_ = s.messageHandler(session.SessionID(), data)
	}
}
