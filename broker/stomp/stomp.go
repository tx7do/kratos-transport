package stomp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/go-stomp/stomp/v3"
	"github.com/go-stomp/stomp/v3/frame"
	"github.com/tx7do/kratos-transport/broker"
)

type stompBroker struct {
	opts      broker.Options
	stompConn *stomp.Conn
}

func stompHeaderToMap(h *frame.Header) map[string]string {
	m := map[string]string{}
	for i := 0; i < h.Len(); i++ {
		k, v := h.GetAt(i)
		m[k] = v
	}
	return m
}

func (r *stompBroker) defaults() {
	WithConnectTimeout(30 * time.Second)(&r.opts)
	WithVirtualHost("/")(&r.opts)
}

func (r *stompBroker) Options() broker.Options {
	if r.opts.Context == nil {
		r.opts.Context = context.Background()
	}
	return r.opts
}

func (r *stompBroker) Address() string {
	if len(r.opts.Addrs) > 0 {
		return r.opts.Addrs[0]
	}
	return ""
}

func (r *stompBroker) Connect() error {
	connectTimeOut, _ := ConnectTimeoutFromContext(r.Options().Context)

	uri, err := url.Parse(r.Address())
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

	if auth, ok := AuthFromContext(r.Options().Context); ok && auth != nil {
		stompOpts = append(stompOpts, stomp.ConnOpt.Login(auth.username, auth.password))
	}
	if headers, ok := ConnectHeadersFromContext(r.Options().Context); ok && headers != nil {
		for k, v := range headers {
			stompOpts = append(stompOpts, stomp.ConnOpt.Header(k, v))
		}
	}
	if host, ok := VirtualHostFromContext(r.Options().Context); ok && host != "" {
		log.Printf("Adding host: %s", host)
		stompOpts = append(stompOpts, stomp.ConnOpt.Host(host))
	}

	r.stompConn, err = stomp.Connect(netConn, stompOpts...)
	if err != nil {
		_ = netConn.Close()
		return fmt.Errorf("failed to connect to %s: %v", uri.Host, err)
	}
	return nil
}

func (r *stompBroker) Disconnect() error {
	return r.stompConn.Disconnect()
}

func (r *stompBroker) Init(opts ...broker.Option) error {
	r.defaults()

	r.opts.Apply(opts...)

	return nil
}

func (r *stompBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	if r.stompConn == nil {
		return errors.New("not connected")
	}

	stompOpt := make([]func(*frame.Frame) error, 0, len(msg.Header))
	for k, v := range msg.Header {
		stompOpt = append(stompOpt, stomp.SendOpt.Header(k, v))
	}

	bOpt := broker.PublishOptions{}
	for _, o := range opts {
		o(&bOpt)
	}
	if withReceipt, ok := r.Options().Context.Value(receiptKey{}).(bool); ok && withReceipt {
		stompOpt = append(stompOpt, stomp.SendOpt.Receipt)
	}
	if withoutContentLength, ok := r.Options().Context.Value(suppressContentLengthKey{}).(bool); ok && withoutContentLength {
		stompOpt = append(stompOpt, stomp.SendOpt.NoContentLength)
	}

	if err := r.stompConn.Send(
		topic,
		"",
		msg.Body,
		stompOpt...); err != nil {
		return err
	}

	return nil
}

func (r *stompBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var ackSuccess bool

	if r.stompConn == nil {
		return nil, errors.New("not connected")
	}

	stompOpt := make([]func(*frame.Frame) error, 0, len(opts))
	bOpt := broker.SubscribeOptions{
		AutoAck: true,
	}
	for _, o := range opts {
		o(&bOpt)
	}
	if bOpt.Context == nil {
		bOpt.Context = context.Background()
	}

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

	sub, err := r.stompConn.Subscribe(topic, ackMode, stompOpt...)
	if err != nil {
		return nil, err
	}

	go func() {
		for msg := range sub.C {
			go func(msg *stomp.Message) {
				m := &broker.Message{
					Header: stompHeaderToMap(msg.Header),
					Body:   msg.Body,
				}
				p := &publication{msg: msg, m: m, topic: topic, broker: r}
				p.err = handler(r.opts.Context, p)
				if p.err == nil && !bOpt.AutoAck && ackSuccess {
					_ = msg.Conn.Ack(msg)
				}
			}(msg)
		}
	}()

	return &subscriber{sub: sub, topic: topic, opts: bOpt}, nil
}

func (r *stompBroker) Name() string {
	return "stomp"
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptions()

	r := &stompBroker{
		opts: options,
	}
	_ = r.Init(opts...)
	return r
}
