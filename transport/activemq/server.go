package activemq

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/stomp"

	"github.com/tx7do/kratos-transport/transport"
	"github.com/tx7do/kratos-transport/transport/keepalive"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
)

type Server struct {
	sync.RWMutex
	broker.Broker
	brokerOpts []broker.Option

	subscribers    broker.SubscriberMap
	subscriberOpts transport.SubscribeOptionMap

	started atomic.Bool

	baseCtx context.Context
	err     error

	keepaliveServer *keepalive.Server
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx:        context.Background(),
		subscribers:    make(broker.SubscriberMap),
		subscriberOpts: make(transport.SubscribeOptionMap),
		brokerOpts:     []broker.Option{},
		started:        atomic.Bool{},
	}

	srv.init(opts...)

	return srv
}

func (s *Server) init(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}

	s.keepaliveServer = keepalive.NewServer(
		keepalive.WithServiceKind(KindActiveMQ),
	)

	s.Broker = stomp.NewBroker(s.brokerOpts...)

}

func (s *Server) Name() string {
	return KindActiveMQ
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	if s.started.Load() {
		return nil
	}

	if s.keepaliveServer != nil {
		go func() {
			if s.err = s.keepaliveServer.Start(ctx); s.err != nil {
				LogErrorf("keepalive server start failed: %s", s.err.Error())
			}
		}()
	}

	s.err = s.Init()
	if s.err != nil {
		LogErrorf("init broker failed: [%s]", s.err.Error())
		return s.err
	}

	s.err = s.Connect()
	if s.err != nil {
		return s.err
	}

	LogInfof("server listening on: %s", s.Address())

	s.err = s.doRegisterSubscriberMap()
	if s.err != nil {
		return s.err
	}

	s.baseCtx = ctx
	s.started.Store(true)

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	LogInfo("server stopping...")

	s.err = nil
	s.started.Store(false)
	err := s.Disconnect()

	if s.keepaliveServer != nil {
		if err := s.keepaliveServer.Stop(ctx); err != nil {
			LogError("keepalive server stop failed", s.err)
		}
		s.keepaliveServer = nil
	}

	LogInfo("server stopped.")

	return err
}

func (s *Server) RegisterSubscriber(ctx context.Context, topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
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

	if s.started.Load() {
		return s.doRegisterSubscriber(topic, handler, binder, opts...)
	} else {
		s.subscriberOpts[topic] = &transport.SubscribeOption{Handler: handler, Binder: binder, SubscribeOptions: opts}
	}
	return nil
}

func RegisterSubscriber[T any](srv *Server, ctx context.Context, topic string, handler func(context.Context, string, broker.Headers, *T) error, opts ...broker.SubscribeOption) error {
	return srv.RegisterSubscriber(ctx,
		topic,
		func(ctx context.Context, event broker.Event) error {
			switch t := event.Message().Body.(type) {
			case *T:
				if err := handler(ctx, event.Topic(), event.Message().Headers, t); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported type: %T", t)
			}
			return nil
		},
		func() any {
			var t T
			return &t
		},
		opts...,
	)
}

func (s *Server) doRegisterSubscriber(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	sub, err := s.Subscribe(topic, handler, binder, opts...)
	if err != nil {
		return err
	}

	s.subscribers[topic] = sub

	return nil
}

func (s *Server) doRegisterSubscriberMap() error {
	for topic, opt := range s.subscriberOpts {
		_ = s.doRegisterSubscriber(topic, opt.Handler, opt.Binder, opt.SubscribeOptions...)
	}
	s.subscriberOpts = make(transport.SubscribeOptionMap)
	return nil
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.keepaliveServer == nil {
		return nil, errors.New("keepalive server is nil")
	}

	return s.keepaliveServer.Endpoint()
}
