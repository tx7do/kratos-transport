package rabbitmq

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
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

	options broker.Options

	url                 string
	exchanges           map[string]*Exchange
	defaultExchangeName string
	qos                 Qos

	confirmMode bool
	onReturn    ReturnHandler
	onConfirm   ConfirmHandler

	connected      bool
	close          chan bool
	waitConnection chan struct{}
}

func newRabbitMQConnection(opts broker.Options) *rabbitConnection {
	conn := &rabbitConnection{
		options:        opts,
		url:            DefaultRabbitURL,
		qos:            DefaultQos,
		close:          make(chan bool),
		waitConnection: make(chan struct{}),
	}

	conn.init()

	return conn
}

func (r *rabbitConnection) init() {
	if len(r.options.Addrs) > 0 && hasUrlPrefix(r.options.Addrs[0]) {
		r.url = r.options.Addrs[0]
	}

	r.exchanges = make(map[string]*Exchange)

	// start with default exchange
	defaultEx := DefaultExchange
	r.exchanges[defaultEx.Name] = &defaultEx
	r.defaultExchangeName = defaultEx.Name

	// apply legacy single-exchange options (override default exchange)
	legacyEx := defaultEx
	if val, ok := r.options.Context.Value(exchangeNameKey{}).(string); ok {
		legacyEx.Name = val
	}
	if val, ok := r.options.Context.Value(exchangeKindKey{}).(string); ok {
		legacyEx.Type = val
	}
	if val, ok := r.options.Context.Value(exchangeDurableKey{}).(bool); ok {
		legacyEx.Durable = val
	}
	if legacyEx.Name != defaultEx.Name {
		delete(r.exchanges, defaultEx.Name)
	}
	r.exchanges[legacyEx.Name] = &legacyEx
	r.defaultExchangeName = legacyEx.Name

	// apply multi-exchange registration
	if vals, ok := r.options.Context.Value(exchangesKey{}).([]Exchange); ok && len(vals) > 0 {
		for i := range vals {
			ex := vals[i]
			r.exchanges[ex.Name] = &ex
		}
		// if no legacy exchange name was set, use first as default
		if _, hasLegacy := r.options.Context.Value(exchangeNameKey{}).(string); !hasLegacy {
			r.defaultExchangeName = vals[0].Name
		}
	}

	// apply explicit default exchange name override
	if val, ok := r.options.Context.Value(defaultExchangeNameKey{}).(string); ok {
		if _, exists := r.exchanges[val]; exists {
			r.defaultExchangeName = val
		}
	}

	if val, ok := r.options.Context.Value(prefetchCountKey{}).(int); ok {
		r.qos.PrefetchCount = val
	}
	if val, ok := r.options.Context.Value(prefetchSizeKey{}).(int); ok {
		r.qos.PrefetchSize = val
	}
	if val, ok := r.options.Context.Value(prefetchGlobalKey{}).(bool); ok {
		r.qos.PrefetchGlobal = val
	}

	if val, ok := r.options.Context.Value(confirmModeKey{}).(bool); ok {
		r.confirmMode = val
	}
	if val, ok := r.options.Context.Value(onReturnKey{}).(ReturnHandler); ok {
		r.onReturn = val
	}
	if val, ok := r.options.Context.Value(onConfirmKey{}).(ConfirmHandler); ok {
		r.onConfirm = val
	}
}

func (r *rabbitConnection) connect(secure bool, config *amqp.Config) error {
	if err := r.tryConnect(secure, config); err != nil {
		return err
	}

	r.Lock()
	r.connected = true
	r.ExchangeChannel = nil
	r.Unlock()
	close(r.waitConnection)

	go r.reconnect(secure, config)
	return nil
}

func (r *rabbitConnection) reconnect(secure bool, config *amqp.Config) {
	var connect bool
	reconnectDelay := 1 * time.Second
	maxReconnectDelay := 30 * time.Second

	for {
		if connect {
			if err := r.tryConnect(secure, config); err != nil {
				time.Sleep(reconnectDelay)
				if reconnectDelay < maxReconnectDelay {
					reconnectDelay *= 2
				}
				continue
			}

			r.Lock()
			r.connected = true
			r.ExchangeChannel = nil
			r.Unlock()
			close(r.waitConnection)
		}

		connect = true
		notifyClose := make(chan *amqp.Error)
		r.Connection.NotifyClose(notifyClose)

		select {
		case err := <-notifyClose:
			LogError(err)
			r.Lock()
			r.connected = false
			r.waitConnection = make(chan struct{})
			r.ExchangeChannel = nil
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
		// 重建 waitConnection：connect() 成功后会对它 close，
		// 不重建的话 Disconnect→Connect 会对已关闭 channel 再次 close 触发 panic
		r.waitConnection = make(chan struct{})
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

	// declare all registered exchanges
	for _, ex := range r.exchanges {
		if err := r.Channel.DeclareExchange(ex.Name, ex.Type, ex.Durable, false); err != nil {
			LogErrorf("declare exchange '%s' failed: %v", ex.Name, err)
		}
	}

	if !EnableLazyInitPublishChannel {
		r.ExchangeChannel, err = newRabbitChannel(r.Connection, r.qos)
	}

	return err
}

func (r *rabbitConnection) GetExchange(name string) (*Exchange, error) {
	if name == "" {
		name = r.defaultExchangeName
	}
	ex, ok := r.exchanges[name]
	if !ok {
		return nil, fmt.Errorf("exchange '%s' not registered", name)
	}
	return ex, nil
}

func (r *rabbitConnection) Consume(exchangeName, queueName, routingKey string, bindArgs amqp.Table, qArgs amqp.Table, autoAck, durableQueue, autoDel bool) (*rabbitChannel, <-chan amqp.Delivery, error) {
	ex, err := r.GetExchange(exchangeName)
	if err != nil {
		return nil, nil, err
	}

	consumerChannel, err := newRabbitChannel(r.Connection, r.qos)
	if err != nil {
		return nil, nil, err
	}

	if err = consumerChannel.DeclareExchange(ex.Name, ex.Type, ex.Durable, false); err != nil {
		return nil, nil, err
	}

	if err = consumerChannel.DeclareQueue(queueName, qArgs, durableQueue, autoDel); err != nil {
		return nil, nil, err
	}

	deliveries, err := consumerChannel.ConsumeQueue(queueName, autoAck)
	if err != nil {
		return nil, nil, err
	}

	if err = consumerChannel.BindQueue(queueName, routingKey, ex.Name, bindArgs); err != nil {
		return nil, nil, err
	}

	return consumerChannel, deliveries, nil
}

func (r *rabbitConnection) DeclarePublishQueue(exchangeName, queueName, routingKey string, bindArgs amqp.Table, queueArgs amqp.Table, durableQueue, autoDel bool) error {
	if err := r.lazyInitPublishChannel(); err != nil {
		return err
	}

	ex, err := r.GetExchange(exchangeName)
	if err != nil {
		return err
	}

	if err := r.ExchangeChannel.DeclareExchange(ex.Name, ex.Type, ex.Durable, false); err != nil {
		return err
	}

	if err := r.ExchangeChannel.DeclareQueue(queueName, queueArgs, durableQueue, autoDel); err != nil {
		return err
	}

	if err := r.ExchangeChannel.BindQueue(queueName, routingKey, ex.Name, bindArgs); err != nil {
		return err
	}

	return nil
}

func (r *rabbitConnection) Publish(ctx context.Context, exchangeName, routingKey string, mandatory bool, msg amqp.Publishing) error {

	if err := r.lazyInitPublishChannel(); err != nil {
		return err
	}

	return r.ExchangeChannel.Publish(ctx, exchangeName, routingKey, mandatory, msg)
}

func (r *rabbitConnection) lazyInitPublishChannel() error {
	r.Lock()
	defer r.Unlock()
	if r.ExchangeChannel == nil {
		var err error
		// lazy init publish channel
		r.ExchangeChannel, err = newRabbitChannel(r.Connection, r.qos)
		if err != nil {
			return err
		}

		// setup publisher confirms if enabled
		if r.confirmMode {
			if err := r.ExchangeChannel.Confirm(); err != nil {
				return err
			}
			confirmChan := make(chan amqp.Confirmation, 64)
			r.ExchangeChannel.NotifyPublish(confirmChan)
			go r.handleConfirms(confirmChan)
		}

		// setup return handler
		returnChan := make(chan amqp.Return, 64)
		r.ExchangeChannel.NotifyReturn(returnChan)
		go r.handleReturns(returnChan)
	}
	return nil
}

// handleConfirms processes publisher confirmations in a background goroutine.
func (r *rabbitConnection) handleConfirms(confirmChan chan amqp.Confirmation) {
	for conf := range confirmChan {
		if r.onConfirm != nil {
			r.onConfirm(conf)
		} else {
			if conf.Ack {
				LogDebugf("message confirmed, delivery tag: %d", conf.DeliveryTag)
			} else {
				LogWarnf("message nacked, delivery tag: %d", conf.DeliveryTag)
			}
		}
	}
}

// handleReturns processes returned messages in a background goroutine.
func (r *rabbitConnection) handleReturns(returnChan chan amqp.Return) {
	for ret := range returnChan {
		if r.onReturn != nil {
			r.onReturn(ret)
		} else {
			LogErrorf("message returned, reason: %s, exchange: %s, routing key: %s",
				ret.ReplyText, ret.Exchange, ret.RoutingKey)
		}
	}
}
