package webtransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"

	"github.com/go-kratos/kratos/v2/encoding"
	kratosTransport "github.com/go-kratos/kratos/v2/transport"
	"github.com/tx7do/kratos-transport/broker"

	"github.com/quic-go/quic-go/http3"

	"github.com/tx7do/kratos-transport/transport"
)

const (
	KindWebtransport = "webtransport"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	*http3.Server

	tlsConf  *tls.Config
	endpoint *url.URL
	timeout  time.Duration

	mux         *http.ServeMux
	path        string
	strictSlash bool

	ctx       context.Context // is closed when Close is called
	ctxCancel context.CancelFunc
	refCount  sync.WaitGroup

	running atomic.Bool

	handlersMu      sync.RWMutex
	messageHandlers MessageHandlerMap
	connectHandler  ConnectHandler
	codec           encoding.Codec

	sessionCount atomic.Int64

	// sessions 在线的流会话表，用于服务端下行发送
	sessionsMu sync.Mutex
	sessions   map[SessionID]*session

	// sessionIDGen 会话 ID 生成器。
	// 不能用 h3 StreamID：QUIC StreamID 只在单条连接内唯一，
	// 多客户端的首条流恒为 0，会互相覆盖会话表
	sessionIDGen atomic.Uint64
}

// session 封装一条被劫持的 HTTP/3 双向流
type session struct {
	id SessionID
	// stream 被服务端劫持的 HTTP/3 流，读用于上行、写用于下行
	stream *http3.Stream
	// writeMu 单写者：串行化下行帧写入
	writeMu sync.Mutex
}

// writeWithTimeout 带写超时地向会话流写入一帧（QUIC 流在流控耗尽/对端不读时
// Write 可能阻塞到连接级 MaxIdleTimeout，需要 deadline 兜底）
func (s *Server) writeWithTimeout(sess *session, data []byte) error {
	sess.writeMu.Lock()
	defer sess.writeMu.Unlock()

	if s.timeout > 0 {
		_ = sess.stream.SetWriteDeadline(time.Now().Add(s.timeout))
	}

	return WriteFrame(sess.stream, data)
}

// SendRawData 向指定会话下行发送一帧原始数据
func (s *Server) SendRawData(sessionId SessionID, data []byte) error {
	s.sessionsMu.Lock()
	sess, ok := s.sessions[sessionId]
	s.sessionsMu.Unlock()

	if !ok {
		return errors.New("session not found")
	}

	return s.writeWithTimeout(sess, data)
}

// BroadcastRawData 向所有在线会话下行发送一帧原始数据
func (s *Server) BroadcastRawData(data []byte) error {
	s.sessionsMu.Lock()
	sessions := make([]*session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		sessions = append(sessions, sess)
	}
	s.sessionsMu.Unlock()

	var firstErr error
	for _, sess := range sessions {
		if err := s.writeWithTimeout(sess, data); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func NewServer(opts ...ServerOption) *Server {
	ctx, ctxCancel := context.WithCancel(context.Background())
	srv := &Server{
		ctx:       ctx,
		ctxCancel: ctxCancel,
		mux:       http.NewServeMux(),

		messageHandlers: make(MessageHandlerMap),
		sessions:        make(map[SessionID]*session),
		codec:           encoding.GetCodec("json"),
	}
	srv.init(opts...)
	return srv
}

func (s *Server) init(opts ...ServerOption) {
	const idleTimeout = 30 * time.Second

	s.Server = &http3.Server{
		Addr: ":443",
		QUICConfig: &quic.Config{
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

	if s.timeout == 0 {
		s.timeout = 5 * time.Second
	}

	// Advertise WebTransport support via HTTP/3 SETTINGS
	if s.Server.AdditionalSettings == nil {
		s.Server.AdditionalSettings = make(map[uint64]uint64)
	}
	s.Server.AdditionalSettings[settingsEnableWebtransport] = 1

	s.mux.HandleFunc(s.path, s.addHandler)
	s.Server.Handler = s.mux
}

func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, err
	}
	return s.endpoint, nil
}

func (s *Server) listenAndEndpoint() error {
	if s.endpoint == nil {
		host, port, err := net.SplitHostPort(s.Addr)
		if err != nil {
			return err
		}

		if host == "" {
			ip, _ := transport.GetLocalIP()
			host = ip
		}

		addr := host + ":" + fmt.Sprint(port)
		s.endpoint = transport.NewRegistryEndpoint("https", addr)
	}

	return nil
}

func (s *Server) Start(_ context.Context) error {
	if !s.running.CompareAndSwap(false, true) {
		return nil
	}

	if err := s.listenAndEndpoint(); err != nil {
		s.running.Store(false)
		return err
	}

	LogInfof("server listening on: %s", s.Addr)

	if err := s.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			LogErrorf("start server failed: %s", err.Error())
			s.running.Store(false)
			return err
		}
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	LogInfo("server stopping...")

	if s.ctxCancel != nil {
		s.ctxCancel()
	}

	// 先尝试优雅关闭（等待在途请求），失败或超时则硬关闭
	s.running.Store(false)

	err := s.Server.Shutdown(ctx)
	if err != nil {
		LogWarnf("graceful shutdown failed, closing: %s", err.Error())
		err = s.Server.Close()
	}

	// GOAWAY 对自定义帧协议的客户端无感知，必须主动关闭被劫持的流，
	// 否则 refCount.Wait 会一直等客户端主动断开
	s.sessionsMu.Lock()
	for _, sess := range s.sessions {
		sess.stream.CancelRead(0)
		sess.stream.Close()
	}
	s.sessionsMu.Unlock()

	s.refCount.Wait()

	LogInfo("server stopped.")

	return err
}

func (s *Server) RegisterMessageHandler(messageType MessageType, handler MessageHandler, binder Binder) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()

	if _, ok := s.messageHandlers[messageType]; ok {
		return
	}

	s.messageHandlers[messageType] = HandlerData{
		handler, binder,
	}
}

func (s *Server) DeregisterMessageHandler(messageType MessageType) {
	s.handlersMu.Lock()
	defer s.handlersMu.Unlock()

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
		LogErrorf("decode message exception: %s", err)
		return err
	}

	s.handlersMu.RLock()
	handlerData, ok := s.messageHandlers[msg.Type]
	s.handlersMu.RUnlock()
	if !ok {
		LogError("message type not found:", msg.Type)
		return errors.New("message handler not found")
	}

	var payload MessagePayload

	if handlerData.Binder != nil {
		payload = handlerData.Binder()

		if err := broker.Unmarshal(s.codec, msg.Body, &payload); err != nil {
			LogErrorf("unmarshal message exception: %s", err)
			return err
		}
	} else {
		payload = msg.Body
	}

	if err := handlerData.Handler(sessionId, payload); err != nil {
		LogErrorf("message handler exception: %s", err)
		return err
	}

	return nil
}

// addHandler handles WebTransport CONNECT requests.
// It validates the request, hijacks the HTTP/3 stream, and enters a read loop
// to process incoming messages.
func (s *Server) addHandler(w http.ResponseWriter, r *http.Request) {
	// Validate WebTransport CONNECT request
	if r.Method != http.MethodConnect {
		http.Error(w, "expected CONNECT request", http.StatusMethodNotAllowed)
		return
	}

	if r.Proto != protocolHeader {
		http.Error(w, "invalid protocol", http.StatusBadRequest)
		return
	}

	// Accept the session by sending 200 OK
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Hijack the HTTP/3 stream for bidirectional communication
	hijacker, ok := w.(http3.HTTPStreamer)
	if !ok {
		LogError("response writer does not support HTTPStreamer")
		http.Error(w, "stream hijacking not supported", http.StatusInternalServerError)
		return
	}

	stream := hijacker.HTTPStream()

	// Generate session ID：连接内唯一的自增 ID
	sessionId := SessionID(s.sessionIDGen.Add(1))

	sess := &session{id: sessionId, stream: stream}
	s.sessionsMu.Lock()
	s.sessions[sessionId] = sess
	s.sessionsMu.Unlock()

	s.sessionCount.Add(1)

	// Notify connect handler
	if s.connectHandler != nil {
		s.connectHandler(sessionId, true)
	}

	s.refCount.Add(1)
	go func() {
		defer s.refCount.Done()
		defer func() {
			s.sessionsMu.Lock()
			delete(s.sessions, sessionId)
			s.sessionsMu.Unlock()
			s.sessionCount.Add(-1)
			if s.connectHandler != nil {
				s.connectHandler(sessionId, false)
			}
		}()

		s.serveSession(sessionId, stream)
	}()
}

// serveSession reads messages from the hijacked stream and dispatches them
// to the registered message handlers.
func (s *Server) serveSession(sessionId SessionID, stream *http3.Stream) {
	for {
		// 按帧读取上行消息（4 字节小端长度前缀 + payload）
		frame, err := ReadFrame(stream)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				LogErrorf("session %d: read error: %s", sessionId, err)
			}
			return
		}

		// Dispatch to message handler
		if err := s.messageHandler(sessionId, frame); err != nil {
			LogErrorf("session %d: message handler error: %s", sessionId, err)
		}
	}
}
