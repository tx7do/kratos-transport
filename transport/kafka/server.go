package kafka

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"

	kratosTransport "github.com/go-kratos/kratos/v2/transport"

	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"

	"github.com/tx7do/kratos-transport/transport"
	"github.com/tx7do/kratos-transport/transport/keepalive"
)

var (
	_ kratosTransport.Server     = (*Server)(nil)
	_ kratosTransport.Endpointer = (*Server)(nil)
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
		keepalive.WithServiceKind(KindKafka),
	)

	s.Broker = kafka.NewBroker(s.brokerOpts...)

}

func (s *Server) Name() string {
	return KindKafka
}

func (s *Server) Start(ctx context.Context) error {
	if s.err != nil {
		return s.err
	}

	if s.started.Load() {
		return nil
	}

	if s.keepaliveServer == nil {
		s.keepaliveServer = keepalive.NewServer(keepalive.WithServiceKind(KindKafka))
	}

	if s.keepaliveServer != nil {
		go func() {
			if err := s.keepaliveServer.Start(ctx); err != nil {
				LogErrorf("keepalive server start failed: %s", err.Error())
			}
		}()
	}

	if s.err = s.Init(); s.err != nil {
		LogErrorf("init broker failed: [%s]", s.err.Error())
		return s.err
	}

	if s.err = s.Connect(); s.err != nil {
		LogErrorf("connect broker failed: [%s]", s.err.Error())
		return s.err
	}

	LogInfof("server listening on: %s", s.Address())

	if s.err = s.doRegisterSubscriberMap(); s.err != nil {
		return s.err
	}

	s.baseCtx = ctx
	s.started.Store(true)

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if !s.started.Load() {
		// 允许 Start 失败后重试：清掉残留的错误状态
		s.err = nil
		return nil
	}

	LogInfo("server stopping...")

	s.started.Store(false)

	// 持锁快照订阅表，避免与并发的 RegisterSubscriber 竞争
	s.Lock()
	subs := s.subscribers
	s.subscribers = make(broker.SubscriberMap)
	s.Unlock()

	for _, v := range subs {
		_ = v.Unsubscribe(false)
	}

	// 保留 subscriberOpts，下一次 Start 会通过 doRegisterSubscriberMap 重新注册

	err := s.Disconnect()
	s.err = nil

	if s.keepaliveServer != nil {
		if keepaliveErr := s.keepaliveServer.Stop(ctx); keepaliveErr != nil {
			LogErrorf("keepalive server stop failed: %s", keepaliveErr.Error())
		}
		s.keepaliveServer = nil
	}

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
	//var subscribeOptions []broker.SubscribeOption
	opts = append(opts, broker.WithSubscribeQueueName(queue))
	if disableAutoAck {
		opts = append(opts, broker.WithSubscribeAutoAck(false))
	}

	// context必须要插入到头部，否则后续传入的配置会被覆盖掉。
	opts = append([]broker.SubscribeOption{broker.WithSubscribeContext(ctx)}, opts...)
	started := s.started.Load()
	if !started {
		s.subscriberOpts[topic] = &transport.SubscribeOption{Handler: handler, Binder: binder, SubscribeOptions: opts}
		s.Unlock()
		return nil
	}
	s.Unlock()

	// 订阅动作放在锁外执行，避免 broker 阻塞拖住整个注册面
	return s.doRegisterSubscriber(topic, handler, binder, opts...)
}

func RegisterSubscriber[T any](
	srv *Server,
	ctx context.Context,
	topic, queue string,
	disableAutoAck bool,
	handler broker.TypedHandler[T],
	opts ...broker.SubscribeOption,
) error {
	return srv.RegisterSubscriber(ctx,
		topic,
		queue,
		disableAutoAck,
		func(ctx context.Context, event broker.Event) error {
			if event == nil || event.Message() == nil || event.Message().Body == nil {
				return fmt.Errorf("event or message body is nil")
			}

			var zero T
			expectedType := fmt.Sprintf("%T", &zero)

			switch t := event.Message().Body.(type) {
			case *T:
				if err := handler(ctx, event.Topic(), event.Message().Headers, t); err != nil {
					return err
				}
			case T:
				if err := handler(ctx, event.Topic(), event.Message().Headers, &t); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported type: expected %s, got %T", expectedType, event.Message().Body)
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

	s.Lock()
	old, exists := s.subscribers[topic]
	s.subscribers[topic] = sub
	s.Unlock()

	if exists {
		// 旧订阅先退订，避免旧订阅继续消费造成泄漏
		LogWarnf("subscriber for topic '%s' already exists, unsubscribing the old one", topic)
		_ = old.Unsubscribe(false)
	}
	return nil
}

func (s *Server) doRegisterSubscriberMap() error {
	// 持锁取出并清空缓存表，避免与并发的 RegisterSubscriber 竞争
	s.Lock()
	optsMap := s.subscriberOpts
	s.subscriberOpts = make(transport.SubscribeOptionMap)
	s.Unlock()

	var errs []error
	for topic, opt := range optsMap {
		if err := s.doRegisterSubscriber(topic, opt.Handler, opt.Binder, opt.SubscribeOptions...); err != nil {
			LogErrorf("register subscriber failed, topic: %s, error: %s", topic, err.Error())
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Server) Endpoint() (*url.URL, error) {
	if s.keepaliveServer == nil {
		return nil, errors.New("keepalive server is nil")
	}

	return s.keepaliveServer.Endpoint()
}
