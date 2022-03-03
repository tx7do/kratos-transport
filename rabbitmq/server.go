package rabbitmq

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/tx7do/kratos-transport/common"
	"net/url"
	"sync"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

// Server is a rabbitmq server wrapper.
type Server struct {
	common.Broker

	subscribers map[string]common.Subscriber
	handlers    map[string]common.Handler

	exit chan chan error
	sync.RWMutex
	started bool
	wg      *sync.WaitGroup

	log     *log.Helper
	baseCtx context.Context
	err     error
}

// NewServer creates a rabbitmq server by options.
func NewServer(opts ...common.Option) *Server {
	srv := &Server{
		baseCtx:     context.Background(),
		log:         log.NewHelper(log.GetLogger()),
		Broker:      NewBroker(opts...),
		subscribers: map[string]common.Subscriber{},
	}

	srv.err = srv.Connect()
	if srv.err != nil {
		return srv
	}

	return srv
}

// Start the rabbitmq server.
func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	s.baseCtx = ctx

	s.log.Infof("[rabbitmq] server listening on: %s", s.Address())

	return nil
}

// Stop the rabbitmq server.
func (s *Server) Stop(_ context.Context) error {
	s.log.Info("[rabbitmq] server stopping")
	return s.Disconnect()
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	return url.Parse(s.Address())
}

func (s *Server) Handle(_ common.Handler) error {
	s.Lock()
	defer s.Unlock()

	return nil
}

// RegisterSubscriber is syntactic sugar for registering a subscriber
func (s *Server) RegisterSubscriber(topic string, h common.Handler, opts ...common.SubscribeOption) error {
	sub, err := s.Subscribe(topic, h, opts...)
	if err != nil {
		return err
	}

	s.subscribers[topic] = sub

	return nil
}
