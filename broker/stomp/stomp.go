package stomp

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/tx7do/kratos-transport/broker"
)

type stompBroker struct {
	opts      broker.Options
	stompConn *stomp.Conn
	log       *log.Helper
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptions()

	b := &stompBroker{
		opts: options,
		log:  log.NewHelper(log.GetLogger()),
	}
	_ = b.Init(opts...)
	return b
}

func (b *stompBroker) Name() string {
	return "stomp"
}

func (b *stompBroker) defaults() {
	WithConnectTimeout(30 * time.Second)(&b.opts)
	WithVirtualHost("/")(&b.opts)
}

func (b *stompBroker) Options() broker.Options {
	if b.opts.Context == nil {
		b.opts.Context = context.Background()
	}
	return b.opts
}

func (b *stompBroker) Address() string {
	if len(b.opts.Addrs) > 0 {
		return b.opts.Addrs[0]
	}
	return ""
}

func (b *stompBroker) Init(opts ...broker.Option) error {
	b.defaults()

	b.opts.Apply(opts...)

	return nil
}

func (b *stompBroker) Connect() error {
	connectTimeOut, _ := ConnectTimeoutFromContext(b.Options().Context)

	uri, err := url.Parse(b.Address())
	if err != nil {
		return err
	}

	if uri.Scheme != "stomp" {
		return fmt.Errorf("expected stomp:// protocol but was %s", uri.Scheme)
	}

	var stompOpts []func(*stomp.Conn) error
	if uri.User != nil && uri.User.Username() != "" {
		password, _ := uri.User.Password()
		stompOpts = append(stompOpts, stomp.ConnOpt.Login(uri.User.Username(), password))
	}

	netConn, err := net.DialTimeout("tcp", uri.Host, connectTimeOut)
	if err != nil {
		return fmt.Errorf("failed to dial %s: %v", uri.Host, err)
	}

	if auth, ok := AuthFromContext(b.Options().Context); ok && auth != nil {
		stompOpts = append(stompOpts, stomp.ConnOpt.Login(auth.username, auth.password))
	}
	if headers, ok := ConnectHeadersFromContext(b.Options().Context); ok && headers != nil {
		for k, v := range headers {
			stompOpts = append(stompOpts, stomp.ConnOpt.Header(k, v))
		}
	}
	if host, ok := VirtualHostFromContext(b.Options().Context); ok && host != "" {
		b.log.Infof("Adding host: %s", host)
		stompOpts = append(stompOpts, stomp.ConnOpt.Host(host))
	}

	b.stompConn, err = stomp.Connect(netConn, stompOpts...)
	if err != nil {
		_ = netConn.Close()
		return fmt.Errorf("failed to connect to %s: %v", uri.Host, err)
	}
	return nil
}

func (b *stompBroker) Disconnect() error {
	return b.stompConn.Disconnect()
}

func (b *stompBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	if b.opts.Codec != nil {
		var err error
		buf, err := b.opts.Codec.Marshal(msg)
		if err != nil {
			return err
		}
		return b.publish(topic, buf, opts...)
	} else {
		switch t := msg.(type) {
		case []byte:
			return b.publish(topic, t, opts...)
		case string:
			return b.publish(topic, []byte(t), opts...)
		default:
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(msg); err != nil {
				return err
			}
			return b.publish(topic, buf.Bytes(), opts...)
		}
	}
}

func (b *stompBroker) publish(topic string, msg []byte, opts ...broker.PublishOption) error {
	if b.stompConn == nil {
		return errors.New("not connected")
	}

	bOpt := broker.PublishOptions{}
	for _, o := range opts {
		o(&bOpt)
	}

	stompOpt := make([]func(*frame.Frame) error, 0, 0)

	if headers, ok := b.Options().Context.Value(headerKey{}).(map[string]interface{}); ok {
		for k, v := range headers {
			switch t := v.(type) {
			case string:
				stompOpt = append(stompOpt, stomp.SendOpt.Header(k, t))
			case []byte:
				stompOpt = append(stompOpt, stomp.SendOpt.Header(k, string(t)))
			default:
				var buf bytes.Buffer
				enc := gob.NewEncoder(&buf)
				if err := enc.Encode(v); err != nil {
					continue
				}
				stompOpt = append(stompOpt, stomp.SendOpt.Header(k, string(buf.Bytes())))
			}
		}
	}
	if withReceipt, ok := b.Options().Context.Value(receiptKey{}).(bool); ok && withReceipt {
		stompOpt = append(stompOpt, stomp.SendOpt.Receipt)
	}
	if withoutContentLength, ok := b.Options().Context.Value(suppressContentLengthKey{}).(bool); ok && withoutContentLength {
		stompOpt = append(stompOpt, stomp.SendOpt.NoContentLength)
	}

	if err := b.stompConn.Send(
		topic,
		"",
		msg,
		stompOpt...); err != nil {
		return err
	}

	return nil
}

func (b *stompBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if b.stompConn == nil {
		return nil, errors.New("not connected")
	}

	bOpt := broker.SubscribeOptions{
		AutoAck: true,
	}
	for _, o := range opts {
		o(&bOpt)
	}
	if bOpt.Context == nil {
		bOpt.Context = context.Background()
	}

	var ackSuccess bool
	stompOpt := make([]func(*frame.Frame) error, 0, len(opts))

	ctx := bOpt.Context
	if subscribeContext, ok := SubscribeContextFromContext(ctx); ok && subscribeContext != nil {
		ctx = subscribeContext
	}

	if durableQueue, ok := ctx.Value(durableQueueKey{}).(bool); ok && durableQueue {
		stompOpt = append(stompOpt, stomp.SubscribeOpt.Header("persistent", "true"))
	}

	if headers, ok := SubscribeHeadersFromContext(ctx); ok && len(headers) > 0 {
		for k, v := range headers {
			stompOpt = append(stompOpt, stomp.SubscribeOpt.Header(k, v))
		}
	}

	if bVal, ok := AckOnSuccessFromContext(ctx); ok && bVal {
		bOpt.AutoAck = false
		ackSuccess = true
	}

	var ackMode stomp.AckMode
	if bOpt.AutoAck {
		ackMode = stomp.AckAuto
	} else {
		ackMode = stomp.AckClientIndividual
	}

	sub, err := b.stompConn.Subscribe(topic, ackMode, stompOpt...)
	if err != nil {
		return nil, err
	}

	go func() {
		for msg := range sub.C {
			go func(msg *stomp.Message) {
				m := &broker.Message{
					Headers: stompHeaderToMap(msg.Header),
				}

				if binder != nil {
					m.Body = binder()
				}

				if b.opts.Codec != nil {
					if err := b.opts.Codec.Unmarshal(msg.Body, m.Body); err != nil {
						return
					}
				} else {
					m.Body = msg.Body
				}

				p := &publication{msg: msg, m: m, topic: topic, broker: b}
				p.err = handler(b.opts.Context, p)
				if p.err == nil && !bOpt.AutoAck && ackSuccess {
					_ = msg.Conn.Ack(msg)
				}
			}(msg)
		}
	}()

	return &subscriber{sub: sub, topic: topic, opts: bOpt}, nil
}
