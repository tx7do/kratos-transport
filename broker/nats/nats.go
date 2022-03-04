package nats

import (
	"context"
	"errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/common"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
)

type natsBroker struct {
	sync.Once
	sync.RWMutex

	connected bool

	addrs []string
	conn  *nats.Conn
	opts  common.Options
	nopts nats.Options

	log *log.Helper

	drain   bool
	closeCh chan error
}

type subscriber struct {
	s    *nats.Subscription
	opts common.SubscribeOptions
}

type publication struct {
	t   string
	err error
	m   *broker.Message
}

func (p *publication) Topic() string {
	return p.t
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}

func (s *subscriber) Options() common.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.s.Subject
}

func (s *subscriber) Unsubscribe() error {
	return s.s.Unsubscribe()
}

func (n *natsBroker) Address() string {
	if n.conn != nil && n.conn.IsConnected() {
		return n.conn.ConnectedUrl()
	}

	if len(n.addrs) > 0 {
		return n.addrs[0]
	}

	return ""
}

func (n *natsBroker) setAddrs(addrs []string) []string {
	//nolint:prealloc
	var cAddrs []string
	for _, addr := range addrs {
		if len(addr) == 0 {
			continue
		}
		if !strings.HasPrefix(addr, "nats://") {
			addr = "nats://" + addr
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{nats.DefaultURL}
	}
	return cAddrs
}

func (n *natsBroker) Connect() error {
	n.Lock()
	defer n.Unlock()

	if n.connected {
		return nil
	}

	status := nats.CLOSED
	if n.conn != nil {
		status = n.conn.Status()
	}

	switch status {
	case nats.CONNECTED, nats.RECONNECTING, nats.CONNECTING:
		n.connected = true
		return nil
	default: // DISCONNECTED or CLOSED or DRAINING
		opts := n.nopts
		opts.Servers = n.addrs
		opts.Secure = n.opts.Secure
		opts.TLSConfig = n.opts.TLSConfig

		// secure might not be set
		if n.opts.TLSConfig != nil {
			opts.Secure = true
		}

		c, err := opts.Connect()
		if err != nil {
			return err
		}
		n.conn = c
		n.connected = true
		return nil
	}
}

func (n *natsBroker) Disconnect() error {
	n.Lock()
	defer n.Unlock()

	if n.drain {
		_ = n.conn.Drain()
		n.closeCh <- nil
	}

	n.conn.Close()

	n.connected = false

	return nil
}

func (n *natsBroker) Init(opts ...common.Option) error {
	n.setOption(opts...)
	return nil
}

func (n *natsBroker) Options() common.Options {
	return n.opts
}

func (n *natsBroker) Publish(topic string, msg *broker.Message, _ ...common.PublishOption) error {
	n.RLock()
	defer n.RUnlock()

	if n.conn == nil {
		return errors.New("not connected")
	}

	var data []byte
	if n.opts.Codec != nil {
		var err error
		data, err = n.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
	} else {
		data = msg.Body
	}

	return n.conn.Publish(topic, data)
}

func (n *natsBroker) Subscribe(topic string, handler broker.Handler, opts ...common.SubscribeOption) (broker.Subscriber, error) {
	n.RLock()
	if n.conn == nil {
		n.RUnlock()
		return nil, errors.New("not connected")
	}
	n.RUnlock()

	opt := common.SubscribeOptions{
		AutoAck: true,
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	fn := func(msg *nats.Msg) {
		var m broker.Message
		pub := &publication{t: msg.Subject}
		eh := n.opts.ErrorHandler

		var err error
		if n.opts.Codec != nil {
			err = n.opts.Codec.Unmarshal(msg.Data, &m)
			pub.err = err
		} else {
			m.Body = msg.Data
		}

		pub.m = &m
		if err != nil {
			m.Body = msg.Data
			n.log.Error(err)
			if eh != nil {
				_ = eh(pub)
			}
			return
		}
		if err := handler(pub); err != nil {
			pub.err = err
			n.log.Error(err)
			if eh != nil {
				_ = eh(pub)
			}
		}
	}

	var sub *nats.Subscription
	var err error

	n.RLock()
	if len(opt.Queue) > 0 {
		sub, err = n.conn.QueueSubscribe(topic, opt.Queue, fn)
	} else {
		sub, err = n.conn.Subscribe(topic, fn)
	}
	n.RUnlock()
	if err != nil {
		return nil, err
	}
	return &subscriber{s: sub, opts: opt}, nil
}

func (n *natsBroker) String() string {
	return "nats"
}

func (n *natsBroker) setOption(opts ...common.Option) {
	for _, o := range opts {
		o(&n.opts)
	}

	n.Once.Do(func() {
		n.nopts = nats.GetDefaultOptions()
	})

	if nopts, ok := n.opts.Context.Value(optionsKey{}).(nats.Options); ok {
		n.nopts = nopts
	}

	// common.Options have higher priority than nats.Options
	// only if Addrs, Secure or TLSConfig were not set through a common.Option
	// we read them from nats.Option
	if len(n.opts.Addrs) == 0 {
		n.opts.Addrs = n.nopts.Servers
	}

	if !n.opts.Secure {
		n.opts.Secure = n.nopts.Secure
	}

	if n.opts.TLSConfig == nil {
		n.opts.TLSConfig = n.nopts.TLSConfig
	}
	n.addrs = n.setAddrs(n.opts.Addrs)

	if n.opts.Context.Value(drainConnectionKey{}) != nil {
		n.drain = true
		n.closeCh = make(chan error)
		n.nopts.ClosedCB = n.onClose
		n.nopts.AsyncErrorCB = n.onAsyncError
		n.nopts.DisconnectedErrCB = n.onDisconnectedError
	}
}

func (n *natsBroker) onClose(_ *nats.Conn) {
	n.closeCh <- nil
}

func (n *natsBroker) onAsyncError(_ *nats.Conn, _ *nats.Subscription, err error) {
	if err == nats.ErrDrainTimeout {
		n.closeCh <- err
	}
}

func (n *natsBroker) onDisconnectedError(_ *nats.Conn, err error) {
	n.closeCh <- err
}

func NewBroker(opts ...common.Option) broker.Broker {
	options := common.Options{
		// Default codec
		//Codec:    json.Marshaler{},
		Context: context.Background(),
	}

	n := &natsBroker{
		opts: options,
	}
	n.setOption(opts...)

	return n
}
