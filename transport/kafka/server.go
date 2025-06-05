package kafka

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
	"github.com/tx7do/kratos-transport/transport"
)

var (
	_ kratosTransport.Server = (*Server)(nil)
)

type Server struct {
	broker.Broker
	brokerOpts []broker.Option

	subscribers    broker.SubscriberMap
	subscriberOpts transport.SubscribeOptionMap

	sync.RWMutex
	started atomic.Bool

	baseCtx context.Context
	err     error

	mws []broker.MiddlewareFunc
}

func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx:        context.Background(),
		subscribers:    make(broker.SubscriberMap),
		subscriberOpts: make(transport.SubscribeOptionMap),
		brokerOpts:     []broker.Option{},
		started:        atomic.Bool{},
	}

	srv.doInjectOptions(opts...)

	srv.Broker = kafka.NewBroker(srv.brokerOpts...)

	return srv
}

func (s *Server) doInjectOptions(opts ...ServerOption) {
	for _, o := range opts {
		o(s)
	}
}

func (s *Server) Name() string {
	return string(KindKafka)
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	if s.started.Load() {
		return nil
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

func (s *Server) Stop(_ context.Context) error {
	if s.started.Load() == false {
		return nil
	}

	LogInfo("server stopping...")

	s.err = nil

	for _, v := range s.subscribers {
		_ = v.Unsubscribe(false)
	}
	s.subscribers = make(broker.SubscriberMap)
	s.subscriberOpts = make(transport.SubscribeOptionMap)

	s.started.Store(false)
	err := s.Disconnect()

	LogInfo("server stopped.")

	return err
}

// RegisterSubscriber 注册一个订阅者
// @param ctx 上下文
// @param topic 订阅的主题
// @param queue 订阅的分组
// @param handler 订阅者的处理函数
func (s *Server) RegisterSubscriber(ctx context.Context, topic, queue string, disableAutoAck bool, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) error {
	s.Lock()
	defer s.Unlock()

	//var subscribeOptions []broker.SubscribeOption
	opts = append(opts, broker.WithQueueName(queue))
	if disableAutoAck {
		opts = append(opts, broker.DisableAutoAck())
	}

	// context必须要插入到头部，否则后续传入的配置会被覆盖掉。
	opts = append([]broker.SubscribeOption{broker.WithSubscribeContext(ctx)}, opts...)

	// handle middleware
	for i := len(s.mws) - 1; i >= 0; i-- {
		handler = s.mws[i](handler)
	}

	if s.started.Load() {
		return s.doRegisterSubscriber(topic, handler, binder, opts...)
	} else {
		s.subscriberOpts[topic] = &transport.SubscribeOption{Handler: handler, Binder: binder, SubscribeOptions: opts}
	}
	return nil
}

func RegisterSubscriber[T any](srv *Server, ctx context.Context, topic, queue string, disableAutoAck bool, handler func(context.Context, string, broker.Headers, *T) error, opts ...broker.SubscribeOption) error {
	return srv.RegisterSubscriber(ctx,
		topic,
		queue,
		disableAutoAck,
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
		func() broker.Any {
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
