package kafka

import (
	"context"
	"crypto/tls"
	"github.com/segmentio/kafka-go"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type Handler func(context.Context, string, string, []byte) error

// ServerOption is kafka server option.
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
		s.kafkaOpts.Brokers = append(s.kafkaOpts.Brokers, addr)
	}
}

func Handle(h Handler) ServerOption {
	return func(s *Server) {
		s.handler = h
	}
}

func GroupID(groupId string) ServerOption {
	return func(s *Server) {
		s.kafkaOpts.GroupID = groupId
	}
}

func Topics(topics []string) ServerOption {
	return func(s *Server) {
		s.kafkaOpts.GroupTopics = topics
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

// Server is a kafka server wrapper.
type Server struct {
	*kafka.Reader
	baseCtx   context.Context
	tlsConf   *tls.Config
	err       error
	network   string
	address   string
	endpoint  *url.URL
	timeout   time.Duration
	log       *log.Helper
	kafkaOpts kafka.ReaderConfig
	handler   Handler
	running   bool
}

// NewServer creates a kafka server by options.
func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx:   context.Background(),
		network:   "tcp",
		address:   ":0",
		timeout:   1 * time.Second,
		log:       log.NewHelper(log.GetLogger()),
		kafkaOpts: kafka.ReaderConfig{},
		running:   false,
	}

	for _, o := range opts {
		o(srv)
	}

	var err error
	srv.endpoint, err = url.Parse(srv.address)
	if err != nil {
		srv.log.Errorf("error Endpoint: %X", err)
	}

	srv.kafkaOpts.MinBytes = 10e3 // 10KB
	srv.kafkaOpts.MaxBytes = 10e6 // 10MB

	srv.Reader = kafka.NewReader(srv.kafkaOpts)

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

// Start the kafka server.
func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}
	s.baseCtx = ctx
	s.log.Infof("[kafka] server listening on: %s", s.address)

	s.running = true

	for {
		if s.running == false {
			break
		}

		m, err := s.Reader.FetchMessage(ctx)
		if err != nil {
			s.err = err
			break
		}

		//s.log.Infof("[kafka] receive msg: k:[%s] v:[%s]", string(m.Key), string(m.Value))
		err = s.handler(s.baseCtx, m.Topic, string(m.Key), m.Value)
		if err != nil {
			s.err = err
			log.Fatal("message handling exception:", err)
		}

		if err := s.Reader.CommitMessages(ctx, m); err != nil {
			s.err = err
			log.Fatal("failed to commit messages:", err)
		}
	}
	return nil
}

// Stop the kafka server.
func (s *Server) Stop(_ context.Context) error {
	s.log.Info("[kafka] server stopping")
	s.running = false
	return s.Reader.Close()
}
