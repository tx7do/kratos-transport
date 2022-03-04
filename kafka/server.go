package kafka

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
	"github.com/tx7do/kratos-transport/common"
	"net/url"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

// Server is a kafka server wrapper.
type Server struct {
	broker.Broker

	subscribers map[string]broker.Subscriber
	baseCtx     context.Context
	err         error
	log         *log.Helper
	started     bool
}

// NewServer creates a kafka server by options.
func NewServer(opts ...common.Option) *Server {
	srv := &Server{
		baseCtx:     context.Background(),
		log:         log.NewHelper(log.GetLogger()),
		Broker:      kafka.NewBroker(opts...),
		subscribers: map[string]broker.Subscriber{},
		started:     false,
	}

	return srv
}

// Endpoint return a real address to registry endpoint.
func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}
	return url.Parse(s.Address())
}

// Start the kafka server.
func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	if s.started {
		return nil
	}

	s.log.Infof("[kafka] server listening on: %s", s.Address())

	s.baseCtx = ctx
	s.started = true

	return nil
}

// Stop the kafka server.
func (s *Server) Stop(_ context.Context) error {
	s.log.Info("[kafka] server stopping")
	s.started = false
	return s.Disconnect()
}

// RegisterSubscriber is syntactic sugar for registering a subscriber
func (s *Server) RegisterSubscriber(topic string, h broker.Handler, opts ...common.SubscribeOption) error {
	sub, err := s.Subscribe(topic, h, opts...)
	if err != nil {
		return err
	}

	s.subscribers[topic] = sub

	return nil
}
