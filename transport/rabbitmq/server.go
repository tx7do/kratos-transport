package rabbitmq

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/rabbitmq"
	"net/url"
	"strings"
	"sync"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

type SubscriberMap map[string]broker.Subscriber

type SubscribeOption struct {
	handler broker.Handler
	binder  broker.Binder
	opts    []broker.SubscribeOption
}
type SubscribeOptionMap map[string]*SubscribeOption

type Server struct {
	broker.Broker
	bOpts []broker.Option

	subscribers    SubscriberMap
	subscriberOpts SubscribeOptionMap

	sync.RWMutex
	started bool

	log     *log.Helper
	baseCtx context.Context
	err     error
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx:        context.Background(),
		log:            log.NewHelper(log.GetLogger()),
		subscribers:    SubscriberMap{},
		subscriberOpts: SubscribeOptionMap{},
		bOpts:          []broker.Option{},
		started:        false,
	}

	srv.init(opts...)

	srv.Broker = rabbitmq.NewBroker(srv.bOpts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Name() string {
	return "rabbitmq"
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	if s.started {
		return nil
	}

	_ = s.Init()

	s.err = s.Connect()
	if s.err != nil {
		return s.err
	}

	s.log.Infof("[rabbitmq] server listening on: %s", s.Address())

	s.err = s.doRegisterSubscriberMap()
	if s.err != nil {
		return s.err
	}

	s.baseCtx = ctx
	s.started = true

	return nil
}

func (s *Server) Stop(_ context.Context) error {
	s.log.Info("[rabbitmq] server stopping")
	s.started = false
	return s.Disconnect()
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.err != nil {
		return nil, s.err
	}

	addr := s.Address()
	if !strings.HasPrefix(addr, "amqp://") {
		addr = "amqp://" + addr
	}

	return url.Parse(addr)
}

func (s *Server) RegisterSubscriber(ctx context.Context, routingKey string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	s.Lock()
	defer s.Unlock()

	if s.baseCtx == nil {
		s.baseCtx = context.Background()
	}
	if ctx == nil {
		ctx = s.baseCtx
	}

	// context必须要插入到头部，否则后续传入的配置会被覆盖掉。
	opts = append([]broker.SubscribeOption{broker.WithSubscribeContext(ctx)}, opts...)

	if s.started {
		return s.doRegisterSubscriber(routingKey, handler, binder, opts...)
	} else {
		s.subscriberOpts[routingKey] = &SubscribeOption{handler: handler, binder: binder, opts: opts}
	}
	return nil
}

func (s *Server) doRegisterSubscriber(routingKey string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	sub, err := s.Subscribe(routingKey, handler, binder, opts...)
	if err != nil {
		return err
	}

	s.subscribers[routingKey] = sub

	return nil
}

func (s *Server) doRegisterSubscriberMap() error {
	for topic, opt := range s.subscriberOpts {
		_ = s.doRegisterSubscriber(topic, opt.handler, opt.binder, opt.opts...)
	}
	s.subscriberOpts = SubscribeOptionMap{}
	return nil
}
