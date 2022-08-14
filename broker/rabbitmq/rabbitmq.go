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
	mtx sync.Mutex
	wg  sync.WaitGroup

	conn *rabbitConnection
	opts broker.Options
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	return &rabbitBroker{
		opts: options,
	}
}

func (r *rabbitBroker) Name() string {
	return "rabbitmq"
}

func (r *rabbitBroker) Options() broker.Options {
	return r.opts
}

func (r *rabbitBroker) Address() string {
	if len(r.opts.Addrs) > 0 {
		return r.opts.Addrs[0]
	}
	return ""
}

func (r *rabbitBroker) Init(opts ...broker.Option) error {
	r.opts.Apply(opts...)

	var addrs []string
	for _, addr := range r.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		if !hasUrlPrefix(addr) {
			addr = "amqp://" + addr
		}
		addrs = append(addrs, addr)
	}
	if len(addrs) == 0 {
		addrs = []string{DefaultRabbitURL}
	}
	r.opts.Addrs = addrs

	return nil
}

func (r *rabbitBroker) Connect() error {
	if r.conn == nil {
		r.conn = newRabbitMQConnection(r.opts)
	}

	conf := DefaultAmqpConfig

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
	r.wg.Wait()
	return ret
}

func (r *rabbitBroker) Publish(routingKey string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(r.opts.Codec, msg)
	if err != nil {
		return err
	}

	return r.publish(routingKey, buf, opts...)
}

func (r *rabbitBroker) publish(routingKey string, buf []byte, opts ...broker.PublishOption) error {
	if r.conn == nil {
		return errors.New("connection is nil")
	}

	options := broker.PublishOptions{
		Context: context.Background(),
	}
	for _, o := range opts {
		o(&options)
	}

	msg := amqp.Publishing{
		Body:    buf,
		Headers: amqp.Table{},
	}

	if value, ok := options.Context.Value(deliveryModeKey{}).(uint8); ok {
		msg.DeliveryMode = value
	}

	if value, ok := options.Context.Value(priorityKey{}).(uint8); ok {
		msg.Priority = value
	}

	if value, ok := options.Context.Value(contentTypeKey{}).(string); ok {
		msg.ContentType = value
	}

	if value, ok := options.Context.Value(contentEncodingKey{}).(string); ok {
		msg.ContentEncoding = value
	}

	if value, ok := options.Context.Value(correlationIDKey{}).(string); ok {
		msg.CorrelationId = value
	}

	if value, ok := options.Context.Value(replyToKey{}).(string); ok {
		msg.ReplyTo = value
	}

	if value, ok := options.Context.Value(expirationKey{}).(string); ok {
		msg.Expiration = value
	}

	if value, ok := options.Context.Value(messageIDKey{}).(string); ok {
		msg.MessageId = value
	}

	if value, ok := options.Context.Value(timestampKey{}).(time.Time); ok {
		msg.Timestamp = value
	}

	if value, ok := options.Context.Value(messageTypeKey{}).(string); ok {
		msg.Type = value
	}

	if value, ok := options.Context.Value(userIDKey{}).(string); ok {
		msg.UserId = value
	}

	if value, ok := options.Context.Value(appIDKey{}).(string); ok {
		msg.AppId = value
	}

	if headers, ok := options.Context.Value(publishHeadersKey{}).(map[string]interface{}); ok {
		for k, v := range headers {
			msg.Headers[k] = v
		}
	}

	if val, ok := options.Context.Value(publishDeclareQueueKey{}).(*DeclarePublishQueueInfo); ok {
		if err := r.conn.DeclarePublishQueue(val.Queue, routingKey, val.BindArguments, val.QueueArguments, val.Durable); err != nil {
			return err
		}
	}

	return r.conn.Publish(r.conn.exchange.Name, routingKey, msg)
}

func (r *rabbitBroker) Subscribe(routingKey string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if r.conn == nil {
		return nil, errors.New("not connected")
	}

	options := broker.SubscribeOptions{
		Context: context.Background(),
		AutoAck: true,
	}
	for _, o := range opts {
		o(&options)
	}

	var requeueOnError = false
	if val, ok := options.Context.Value(requeueOnErrorKey{}).(bool); ok {
		requeueOnError = val
	}

	var ackSuccess = false
	if val, ok := options.Context.Value(ackSuccessKey{}).(bool); ok {
		options.AutoAck = val
		ackSuccess = !val
	}

	fn := func(msg amqp.Delivery) {
		m := &broker.Message{
			Headers: rabbitHeaderToMap(msg.Headers),
			Body:    nil,
		}

		p := &publication{d: msg, m: m, t: msg.RoutingKey}

		if binder != nil {
			m.Body = binder()
		}

		if err := broker.Unmarshal(r.opts.Codec, msg.Body, m.Body); err != nil {
			p.err = err
			r.opts.Logger.Error(err)
		}

		p.err = handler(r.opts.Context, p)
		if p.err == nil && ackSuccess && !options.AutoAck {
			_ = msg.Ack(false)
		} else if p.err != nil && !options.AutoAck {
			_ = msg.Nack(false, requeueOnError)
		}
	}

	sub := &subscriber{
		topic:        routingKey,
		opts:         options,
		mayRun:       true,
		r:            r,
		durableQueue: true,
		fn:           fn,
		headers:      nil,
		queueArgs:    nil,
	}

	if val, ok := options.Context.Value(durableQueueKey{}).(bool); ok {
		sub.durableQueue = val
	}

	if val, ok := options.Context.Value(subscribeBindArgsKey{}).(map[string]interface{}); ok {
		sub.headers = val
	}

	if val, ok := options.Context.Value(subscribeQueueArgsKey{}).(map[string]interface{}); ok {
		sub.queueArgs = val
	}

	go sub.resubscribe()

	return sub, nil
}
