package nats

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	NATS "github.com/nats-io/nats.go"
	"github.com/tx7do/kratos-transport/broker"
)

type natsBroker struct {
	sync.Once
	sync.RWMutex

	connected bool

	addrs []string
	opts  broker.Options

	conn     *NATS.Conn
	natsOpts NATS.Options

	log *log.Helper

	drain   bool
	closeCh chan error
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &natsBroker{
		opts: options,
		log:  log.NewHelper(log.GetLogger()),
	}
	b.setOption(opts...)

	return b
}

func (b *natsBroker) Address() string {
	if b.conn != nil && b.conn.IsConnected() {
		return b.conn.ConnectedUrl()
	}

	if len(b.addrs) > 0 {
		return b.addrs[0]
	}

	return ""
}

func (b *natsBroker) Name() string {
	return "nats"
}

func (b *natsBroker) Options() broker.Options {
	return b.opts
}

func (b *natsBroker) Init(opts ...broker.Option) error {
	b.setOption(opts...)
	return nil
}

func (b *natsBroker) setAddrs(addrs []string) []string {
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
		cAddrs = []string{NATS.DefaultURL}
	}
	return cAddrs
}

func (b *natsBroker) setOption(opts ...broker.Option) {
	for _, o := range opts {
		o(&b.opts)
	}

	b.Once.Do(func() {
		b.natsOpts = NATS.GetDefaultOptions()
	})

	if value, ok := b.opts.Context.Value(optionsKey{}).(NATS.Options); ok {
		b.natsOpts = value
	}

	if len(b.opts.Addrs) == 0 {
		b.opts.Addrs = b.natsOpts.Servers
	}

	if !b.opts.Secure {
		b.opts.Secure = b.natsOpts.Secure
	}

	if b.opts.TLSConfig == nil {
		b.opts.TLSConfig = b.natsOpts.TLSConfig
	}
	b.addrs = b.setAddrs(b.opts.Addrs)

	if b.opts.Context.Value(drainConnectionKey{}) != nil {
		b.drain = true
		b.closeCh = make(chan error)
		b.natsOpts.ClosedCB = b.onClose
		b.natsOpts.AsyncErrorCB = b.onAsyncError
		b.natsOpts.DisconnectedErrCB = b.onDisconnectedError
	}
}

func (b *natsBroker) Connect() error {
	b.Lock()
	defer b.Unlock()

	if b.connected {
		return nil
	}

	status := NATS.CLOSED
	if b.conn != nil {
		status = b.conn.Status()
	}

	switch status {
	case NATS.CONNECTED, NATS.RECONNECTING, NATS.CONNECTING:
		b.connected = true
		return nil
	default: // DISCONNECTED or CLOSED or DRAINING
		opts := b.natsOpts
		opts.Servers = b.addrs
		opts.Secure = b.opts.Secure
		opts.TLSConfig = b.opts.TLSConfig

		if b.opts.TLSConfig != nil {
			opts.Secure = true
		}

		c, err := opts.Connect()
		if err != nil {
			return err
		}
		b.conn = c
		b.connected = true
		return nil
	}
}

func (b *natsBroker) Disconnect() error {
	b.Lock()
	defer b.Unlock()

	if b.drain {
		_ = b.conn.Drain()
		b.closeCh <- nil
	}

	b.conn.Close()

	b.connected = false

	return nil
}

func (b *natsBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	b.RLock()
	defer b.RUnlock()

	if b.conn == nil {
		return errors.New("not connected")
	}

	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	if b.opts.Codec != nil {
		var err error
		buf, err := b.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
		return b.conn.Publish(topic, buf)
	} else {
		switch t := msg.(type) {
		case []byte:
			return b.conn.Publish(topic, t)
		case string:
			return b.conn.Publish(topic, []byte(t))
		default:
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(msg); err != nil {
				return err
			}
			return b.conn.Publish(topic, buf.Bytes())
		}
	}
}

func (b *natsBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	b.RLock()
	if b.conn == nil {
		b.RUnlock()
		return nil, errors.New("not connected")
	}
	b.RUnlock()

	opt := broker.SubscribeOptions{
		AutoAck: true,
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	subs := &subscriber{s: nil, opts: opt}

	fn := func(msg *NATS.Msg) {
		m := &broker.Message{
			Headers: natsHeaderToMap(msg.Header),
			Body:    nil,
		}

		pub := &publication{t: msg.Subject, m: m}

		eh := b.opts.ErrorHandler

		if binder != nil {
			m.Body = binder()
		}

		if b.opts.Codec != nil {
			if err := b.opts.Codec.Unmarshal(msg.Data, m.Body); err != nil {
				pub.err = err
				if err != nil {
					b.log.Error(err)
					if eh != nil {
						_ = eh(b.opts.Context, pub)
					}
					return
				}
			}
		} else {
			m.Body = msg.Data
		}

		if err := handler(b.opts.Context, pub); err != nil {
			pub.err = err
			b.log.Error(err)
			if eh != nil {
				_ = eh(b.opts.Context, pub)
			}
		}
		if opt.AutoAck {
			if err := pub.Ack(); err != nil {
				b.log.Errorf("[nats]: unable to commit msg: %v", err)
			}
		}
	}

	var sub *NATS.Subscription
	var err error

	b.RLock()
	if len(opt.Queue) > 0 {
		sub, err = b.conn.QueueSubscribe(topic, opt.Queue, fn)
	} else {
		sub, err = b.conn.Subscribe(topic, fn)
	}
	b.RUnlock()
	if err != nil {
		return nil, err
	}

	subs.s = sub

	return subs, nil
}

func (b *natsBroker) onClose(_ *NATS.Conn) {
	b.closeCh <- nil
}

func (b *natsBroker) onAsyncError(_ *NATS.Conn, _ *NATS.Subscription, err error) {
	if err == NATS.ErrDrainTimeout {
		b.closeCh <- err
	}
}

func (b *natsBroker) onDisconnectedError(_ *NATS.Conn, err error) {
	b.closeCh <- err
}
