package rabbitmq

import (
	"crypto/tls"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/streadway/amqp"

	"github.com/tx7do/kratos-transport/broker"
)

var (
	DefaultRabbitURL             = "amqp://guest:guest@127.0.0.1:5672"
	DefaultRequeueOnError        = false
	EnableLazyInitPublishChannel = true

	DefaultAmqpConfig = amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	}
)

type Exchange struct {
	Name    string
	Type    string // "direct", "fanout", "topic", "headers"
	Durable bool
}

var (
	DefaultExchange = Exchange{
		Name:    "amq.topic",
		Type:    "topic",
		Durable: true,
	}
)

type Qos struct {
	PrefetchCount  int
	PrefetchSize   int
	PrefetchGlobal bool
}

var (
	DefaultQos = Qos{
		PrefetchCount:  0,
		PrefetchSize:   0,
		PrefetchGlobal: false,
	}
)

type rabbitConnection struct {
	sync.Mutex

	Connection      *amqp.Connection
	Channel         *rabbitChannel
	ExchangeChannel *rabbitChannel

	opts broker.Options

	url      string
	exchange Exchange
	qos      Qos

	connected      bool
	close          chan bool
	waitConnection chan struct{}
}

func newRabbitMQConnection(opts broker.Options) *rabbitConnection {
	conn := &rabbitConnection{
		opts:           opts,
		url:            DefaultRabbitURL,
		qos:            DefaultQos,
		exchange:       DefaultExchange,
		close:          make(chan bool),
		waitConnection: make(chan struct{}),
	}

	conn.init()

	close(conn.waitConnection)

	return conn
}

func (r *rabbitConnection) init() {
	if len(r.opts.Addrs) > 0 && hasUrlPrefix(r.opts.Addrs[0]) {
		r.url = r.opts.Addrs[0]
	}

	if val, ok := r.opts.Context.Value(exchangeNameKey{}).(string); ok {
		r.exchange.Name = val
	}
	if val, ok := r.opts.Context.Value(exchangeKindKey{}).(string); ok {
		r.exchange.Type = val
	}
	if val, ok := r.opts.Context.Value(exchangeDurableKey{}).(bool); ok {
		r.exchange.Durable = val
	}

	if val, ok := r.opts.Context.Value(prefetchCountKey{}).(int); ok {
		r.qos.PrefetchCount = val
	}
	if val, ok := r.opts.Context.Value(prefetchSizeKey{}).(int); ok {
		r.qos.PrefetchSize = val
	}
	if val, ok := r.opts.Context.Value(prefetchGlobalKey{}).(bool); ok {
		r.qos.PrefetchGlobal = val
	}
}

func (r *rabbitConnection) connect(secure bool, config *amqp.Config) error {
	if err := r.tryConnect(secure, config); err != nil {
		return err
	}

	r.Lock()
	r.connected = true
	r.Unlock()

	go r.reconnect(secure, config)
	return nil
}

func (r *rabbitConnection) reconnect(secure bool, config *amqp.Config) {
	var connect bool

	for {
		if connect {
			if err := r.tryConnect(secure, config); err != nil {
				time.Sleep(1 * time.Second)
				continue
			}

			r.Lock()
			r.connected = true
			r.Unlock()
			close(r.waitConnection)
		}

		connect = true
		notifyClose := make(chan *amqp.Error)
		r.Connection.NotifyClose(notifyClose)

		chanNotifyClose := make(chan *amqp.Error)
		channelNotifyReturn := make(chan amqp.Return)

		if r.ExchangeChannel != nil {
			channel := r.ExchangeChannel.channel
			channel.NotifyClose(chanNotifyClose)

			channel.NotifyReturn(channelNotifyReturn)
		}

		select {
		case result, ok := <-channelNotifyReturn:
			if !ok {
				// Channel closed, probably also the channel or connection.
				return
			}
			log.Errorf("notify error reason: %s, description: %s", result.ReplyText, result.Exchange)
		case err := <-chanNotifyClose:
			log.Error(err)
			r.Lock()
			r.connected = false
			r.waitConnection = make(chan struct{})
			r.Unlock()
		case err := <-notifyClose:
			log.Error(err)

			select {
			case errs := <-chanNotifyClose:
				log.Error(errs)
			case <-time.After(time.Second):
			}

			r.Lock()
			r.connected = false
			r.waitConnection = make(chan struct{})
			r.Unlock()
		case <-r.close:
			return
		}
	}
}

func (r *rabbitConnection) Connect(secure bool, config *amqp.Config) error {
	r.Lock()

	if r.connected {
		r.Unlock()
		return nil
	}

	select {
	case <-r.close:
		r.close = make(chan bool)
	default:
	}

	r.Unlock()

	return r.connect(secure, config)
}

func (r *rabbitConnection) Close() error {
	r.Lock()
	defer r.Unlock()

	select {
	case <-r.close:
		return nil
	default:
		close(r.close)
		r.connected = false
	}

	if r.Connection == nil {
		return errors.New("connection is nil")
	}

	return r.Connection.Close()
}

func (r *rabbitConnection) tryConnect(secure bool, config *amqp.Config) error {
	if config == nil {
		config = &DefaultAmqpConfig
	}

	url := r.url

	if secure || config.TLSClientConfig != nil || strings.HasPrefix(r.url, "amqps://") {
		if config.TLSClientConfig == nil {
			config.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}

		url = strings.Replace(r.url, "amqp://", "amqps://", 1)
	}

	var err error
	r.Connection, err = amqp.DialConfig(url, *config)
	if err != nil {
		return err
	}

	if r.Channel, err = newRabbitChannel(r.Connection, r.qos); err != nil {
		return err
	}

	_ = r.Channel.DeclareExchange(r.exchange.Name, r.exchange.Type, r.exchange.Durable, false)

	if !EnableLazyInitPublishChannel {
		r.ExchangeChannel, err = newRabbitChannel(r.Connection, r.qos)
	}

	return err
}

func (r *rabbitConnection) Consume(queueName, routingKey string, bindArgs amqp.Table, qArgs amqp.Table, autoAck, durableQueue, autoDel bool) (*rabbitChannel, <-chan amqp.Delivery, error) {
	consumerChannel, err := newRabbitChannel(r.Connection, r.qos)
	if err != nil {
		return nil, nil, err
	}

	if err = consumerChannel.DeclareQueue(queueName, qArgs, durableQueue, autoDel); err != nil {
		return nil, nil, err
	}

	deliveries, err := consumerChannel.ConsumeQueue(queueName, autoAck)
	if err != nil {
		return nil, nil, err
	}

	if err = consumerChannel.BindQueue(queueName, routingKey, r.exchange.Name, bindArgs); err != nil {
		return nil, nil, err
	}

	return consumerChannel, deliveries, nil
}

func (r *rabbitConnection) DeclarePublishQueue(queueName, routingKey string, bindArgs amqp.Table, queueArgs amqp.Table, durableQueue, autoDel bool) error {
	if r.ExchangeChannel == nil {
		var err error
		r.ExchangeChannel, err = newRabbitChannel(r.Connection, r.qos)
		if err != nil {
			return err
		}
	}

	if err := r.ExchangeChannel.DeclareQueue(queueName, queueArgs, durableQueue, autoDel); err != nil {
		return err
	}

	if err := r.ExchangeChannel.BindQueue(queueName, routingKey, r.exchange.Name, bindArgs); err != nil {
		return err
	}

	return nil
}

func (r *rabbitConnection) Publish(exchangeName, routingKey string, msg amqp.Publishing) error {
	if r.ExchangeChannel == nil {
		var err error
		// lazy init publish channel
		r.ExchangeChannel, err = newRabbitChannel(r.Connection, r.qos)
		if err != nil {
			return err
		}
	}

	return r.ExchangeChannel.Publish(exchangeName, routingKey, msg)
}
