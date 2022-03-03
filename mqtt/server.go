package mqtt

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type Handler func(context.Context, string, []byte) error
type TopicHandler func(context.Context, string, []byte) error

type subscribeMap map[string]byte

// ServerOption is mqtt server option.
type ServerOption func(o *Server)

// Network with server network.
func Network(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

// Address with server address.
func Address(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

// Timeout with server timeout.
func Timeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// Logger with server logger.
func Logger(logger log.Logger) ServerOption {
	return func(s *Server) {
		s.log = log.NewHelper(logger)
	}
}

// TLSConfig with TLS config.
func TLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = c
	}
}

// Options with mqtt options.
func Options(opt *mqtt.ClientOptions) ServerOption {
	return func(s *Server) {
		s.mqttOpts = opt
	}
}

func Topic(topic string, qos byte) ServerOption {
	return func(s *Server) {
		s.topics[topic] = qos
	}
}

func Handle(h Handler) ServerOption {
	return func(s *Server) {
		s.handler = h
	}
}

// Server is a mqtt server wrapper.
type Server struct {
	mqtt.Client
	baseCtx  context.Context
	tlsConf  *tls.Config
	err      error
	network  string
	address  string
	endpoint *url.URL
	timeout  time.Duration
	log      *log.Helper
	mqttOpts *mqtt.ClientOptions
	topics   subscribeMap
	handler  Handler
	running  bool
}

// NewServer creates a mqtt server by options.
func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx:  context.Background(),
		network:  "tcp",
		address:  ":0",
		timeout:  1 * time.Second,
		log:      log.NewHelper(log.GetLogger()),
		topics:   subscribeMap{},
		mqttOpts: nil,
		running:  false,
	}

	for _, o := range opts {
		o(srv)
	}

	srv.mqttOpts = mqtt.NewClientOptions()

	srv.mqttOpts.AddBroker(srv.address)

	// setup tls
	if srv.tlsConf != nil {
		srv.mqttOpts.SetTLSConfig(srv.tlsConf)
	}

	srv.mqttOpts.SetClientID(fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Intn(10)))
	srv.mqttOpts.SetCleanSession(false)

	srv.mqttOpts.SetKeepAlive(60 * time.Second)
	srv.mqttOpts.SetPingTimeout(1 * time.Second)

	srv.mqttOpts.SetDefaultPublishHandler(srv.receive)

	var err error
	srv.endpoint, err = url.Parse(srv.address)
	if err != nil {
		srv.log.Errorf("error Endpoint: %X", err)
	}

	srv.Client = mqtt.NewClient(srv.mqttOpts)

	return srv
}

// Endpoint return a real address to registry endpoint.
// examples:
//   tcp://emqx:public@broker.emqx.io:1883
func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.endpoint, nil
}

// Start the mqtt server.
func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	s.baseCtx = ctx
	s.running = true

	s.log.Infof("[mqtt] server listening on: %s", s.address)

	if token := s.Client.Connect(); token.Wait() && token.Error() != nil {
		s.err = token.Error()
		return token.Error()
	}

	if token := s.Client.SubscribeMultiple(s.topics, nil); token.Wait() && token.Error() != nil {
		s.err = token.Error()
		return s.err
	}

	return nil
}

// Stop the mqtt server.
func (s *Server) Stop(_ context.Context) error {
	s.log.Info("[mqtt] server stopping")
	s.running = false
	s.Client.Disconnect(250)
	return nil
}

func (s *Server) receive(_ mqtt.Client, msg mqtt.Message) {
	//fmt.Printf("TOPIC: %s\n", msg.Topic())
	//fmt.Printf("MSG: %s\n", msg.Payload())

	err := s.handler(s.baseCtx, msg.Topic(), msg.Payload())
	if err != nil {
		log.Fatal("message handling exception:", err)
	}
}

func (s *Server) AddTopic(topic string, qos byte) error {
	s.topics[topic] = qos
	return nil
}
