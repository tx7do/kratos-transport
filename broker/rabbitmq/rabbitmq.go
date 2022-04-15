// Package rabbitmq provides a RabbitMQ common
package rabbitmq

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"github.com/tx7do/kratos-transport/broker"
)

type rabbitBroker struct {
	conn           *rabbitConn
	addrs          []string
	opts           broker.Options
	prefetchCount  int
	prefetchGlobal bool
	mtx            sync.Mutex
	wg             sync.WaitGroup
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	return &rabbitBroker{
		addrs: options.Addrs,
		opts:  options,
	}
}

func (r *rabbitBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	m := amqp.Publishing{
		Body:    msg.Body,
		Headers: amqp.Table{},
	}

	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	if options.Context != nil {
		if value, ok := options.Context.Value(deliveryModeKey{}).(uint8); ok {
			m.DeliveryMode = value
		}

		if value, ok := options.Context.Value(priorityKey{}).(uint8); ok {
			m.Priority = value
		}

		if value, ok := options.Context.Value(contentTypeKey{}).(string); ok {
			m.ContentType = value
		}

		if value, ok := options.Context.Value(contentEncodingKey{}).(string); ok {
			m.ContentEncoding = value
		}

		if value, ok := options.Context.Value(correlationIDKey{}).(string); ok {
			m.CorrelationId = value
		}

		if value, ok := options.Context.Value(replyToKey{}).(string); ok {
			m.ReplyTo = value
		}

		if value, ok := options.Context.Value(expirationKey{}).(string); ok {
			m.Expiration = value
		}

		if value, ok := options.Context.Value(messageIDKey{}).(string); ok {
			m.MessageId = value
		}

		if value, ok := options.Context.Value(timestampKey{}).(time.Time); ok {
			m.Timestamp = value
		}

		if value, ok := options.Context.Value(typeMsgKey{}).(string); ok {
			m.Type = value
		}

		if value, ok := options.Context.Value(userIDKey{}).(string); ok {
			m.UserId = value
		}

		if value, ok := options.Context.Value(appIDKey{}).(string); ok {
			m.AppId = value
		}

	}

	for k, v := range msg.Header {
		m.Headers[k] = v
	}

	if r.conn == nil {
		return errors.New("connection is nil")
	}

	return r.conn.Publish(r.conn.exchange.Name, topic, m)
}

func (r *rabbitBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var ackSuccess bool

	if r.conn == nil {
		return nil, errors.New("not connected")
	}

	opt := broker.SubscribeOptions{
		AutoAck: true,
	}

	for _, o := range opts {
		o(&opt)
	}

	if opt.Context == nil {
		opt.Context = context.Background()
	}

	ctx := opt.Context
	if subscribeContext, ok := SubscribeContextFromContext(ctx); ok && subscribeContext != nil {
		ctx = subscribeContext
	}

	var requeueOnError bool
	requeueOnError, _ = ctx.Value(requeueOnErrorKey{}).(bool)

	var durableQueue bool
	durableQueue, _ = ctx.Value(durableQueueKey{}).(bool)

	var qArgs map[string]interface{}
	if qa, ok := ctx.Value(queueArgumentsKey{}).(map[string]interface{}); ok {
		qArgs = qa
	}

	var headers map[string]interface{}
	if h, ok := ctx.Value(headersKey{}).(map[string]interface{}); ok {
		headers = h
	}

	if bVal, ok := AckOnSuccessFromContext(ctx); ok && bVal {
		opt.AutoAck = false
		ackSuccess = true
	}

	fn := func(msg amqp.Delivery) {
		header := make(map[string]string)
		for k, v := range msg.Headers {
			header[k], _ = v.(string)
		}
		m := &broker.Message{
			Header: header,
			Body:   msg.Body,
		}
		p := &publication{d: msg, m: m, t: msg.RoutingKey}
		p.err = handler(r.opts.Context, p)
		if p.err == nil && ackSuccess && !opt.AutoAck {
			_ = msg.Ack(false)
		} else if p.err != nil && !opt.AutoAck {
			_ = msg.Nack(false, requeueOnError)
		}
	}

	sub := &subscriber{topic: topic, opts: opt, mayRun: true, r: r,
		durableQueue: durableQueue, fn: fn, headers: headers, queueArgs: qArgs}

	go sub.resubscribe()

	return sub, nil
}

func (r *rabbitBroker) Options() broker.Options {
	return r.opts
}

func (r *rabbitBroker) Name() string {
	return "rabbitmq"
}

func (r *rabbitBroker) Address() string {
	if len(r.addrs) > 0 {
		return r.addrs[0]
	}
	return ""
}

func (r *rabbitBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&r.opts)
	}
	r.addrs = r.opts.Addrs
	return nil
}

func (r *rabbitBroker) Connect() error {
	if r.conn == nil {
		r.conn = newRabbitMQConn(r.getExchange(), r.opts.Addrs, r.getPrefetchCount(), r.getPrefetchGlobal())
	}

	conf := defaultAmqpConfig

	if auth, ok := r.opts.Context.Value(externalAuthKey{}).(ExternalAuthentication); ok {
		conf.SASL = []amqp.Authentication{&auth}
	}

	conf.TLSClientConfig = r.opts.TLSConfig

	return r.conn.Connect(r.opts.Secure, &conf)
}

func (r *rabbitBroker) Disconnect() error {
	if r.conn == nil {
		return errors.New("connection is nil")
	}
	ret := r.conn.Close()
	r.wg.Wait() // wait all goroutines
	return ret
}

func (r *rabbitBroker) getExchange() Exchange {

	ex := DefaultExchange

	if e, ok := r.opts.Context.Value(exchangeKey{}).(string); ok {
		ex.Name = e
	}

	if d, ok := r.opts.Context.Value(durableExchangeKey{}).(bool); ok {
		ex.Durable = d
	}

	return ex
}

func (r *rabbitBroker) getPrefetchCount() int {
	if e, ok := r.opts.Context.Value(prefetchCountKey{}).(int); ok {
		return e
	}
	return DefaultPrefetchCount
}

func (r *rabbitBroker) getPrefetchGlobal() bool {
	if e, ok := r.opts.Context.Value(prefetchGlobalKey{}).(bool); ok {
		return e
	}
	return DefaultPrefetchGlobal
}
