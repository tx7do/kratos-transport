package mqtt

import (
	"context"
	"errors"
	"strings"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/tx7do/kratos-transport/broker"
)

type mqttBroker struct {
	addrs   []string
	options broker.Options
	client  paho.Client

	subscribers *broker.SubscriberSyncMap
}

func NewBroker(opts ...broker.Option) broker.Broker {
	return newBroker(opts...)
}

func newClient(addrs []string, opts broker.Options, b *mqttBroker) paho.Client {
	cOpts := paho.NewClientOptions()

	// 是否清除会话，如果true，mqtt服务端将会清除掉
	cOpts.SetCleanSession(true)
	// 设置自动重连接
	cOpts.SetAutoReconnect(false)
	// 设置连接之后恢复订阅
	cOpts.SetResumeSubs(true)

	// 默认设置Client ID
	cOpts.SetClientID(generateClientId())

	// 连接成功回调
	cOpts.OnConnect = b.onConnect
	// 连接丢失回调
	cOpts.OnConnectionLost = b.onConnectionLost

	// 加入服务器地址列表
	for _, addr := range addrs {
		cOpts.AddBroker(addr)
	}

	if opts.TLSConfig != nil {
		cOpts.SetTLSConfig(opts.TLSConfig)
	}
	if auth, ok := opts.Context.Value(authKey{}).(*AuthRecord); ok && auth != nil {
		cOpts.SetUsername(auth.Username)
		cOpts.SetPassword(auth.Password)
	}
	if clientId, ok := opts.Context.Value(clientIdKey{}).(string); ok && clientId != "" {
		cOpts.SetClientID(clientId)
	}

	if pv, ok := opts.Context.Value(protocolVersionKey{}).(uint); ok {
		cOpts.SetProtocolVersion(pv)
	}

	if k, ok := opts.Context.Value(keepAliveKey{}).(time.Duration); ok {
		cOpts.SetKeepAlive(k)
	}
	if k, ok := opts.Context.Value(maxReconnectIntervalKey{}).(time.Duration); ok {
		cOpts.SetMaxReconnectInterval(k)
	}
	if k, ok := opts.Context.Value(connectRetryIntervalKey{}).(time.Duration); ok {
		cOpts.SetConnectRetryInterval(k)
	}
	if k, ok := opts.Context.Value(writeTimeoutKey{}).(time.Duration); ok {
		cOpts.SetWriteTimeout(k)
	}
	if k, ok := opts.Context.Value(connectTimeoutKey{}).(time.Duration); ok {
		cOpts.SetConnectTimeout(k)
	}
	if k, ok := opts.Context.Value(pingTimeoutKey{}).(time.Duration); ok {
		cOpts.SetPingTimeout(k)
	}

	if enabled, ok := opts.Context.Value(cleanSessionKey{}).(bool); ok {
		cOpts.SetCleanSession(enabled)
	}
	if enabled, ok := opts.Context.Value(autoReconnectKey{}).(bool); ok {
		cOpts.SetAutoReconnect(enabled)
	}
	if enabled, ok := opts.Context.Value(resumeSubsKey{}).(bool); ok {
		cOpts.SetResumeSubs(enabled)
	}
	if enabled, ok := opts.Context.Value(orderMattersKey{}).(bool); ok {
		cOpts.SetOrderMatters(enabled)
	}

	if _, ok := opts.Context.Value(errorLoggerKey{}).(bool); ok {
		paho.ERROR = ErrorLogger{}
	}
	if _, ok := opts.Context.Value(criticalLoggerKey{}).(bool); ok {
		paho.CRITICAL = CriticalLogger{}
	}
	if _, ok := opts.Context.Value(warnLoggerKey{}).(bool); ok {
		paho.WARN = WarnLogger{}
	}
	if _, ok := opts.Context.Value(debugLoggerKey{}).(bool); ok {
		paho.DEBUG = DebugLogger{}
	}
	if opt, ok := opts.Context.Value(debugLoggerKey{}).(LoggerOptions); ok {
		if opt.Error {
			paho.ERROR = ErrorLogger{}
		}
		if opt.Critical {
			paho.CRITICAL = CriticalLogger{}
		}
		if opt.Warn {
			paho.WARN = WarnLogger{}
		}
		if opt.Debug {
			paho.DEBUG = DebugLogger{}
		}
	}

	return paho.NewClient(cOpts)
}

func newBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &mqttBroker{
		options:     options,
		addrs:       options.Addrs,
		subscribers: broker.NewSubscriberSyncMap(),
	}

	b.client = newClient(options.Addrs, options, b)

	return b
}

func (m *mqttBroker) Name() string {
	return "mqtt"
}

func (m *mqttBroker) Options() broker.Options {
	return m.options
}

func (m *mqttBroker) Address() string {
	return strings.Join(m.addrs, ",")
}

func (m *mqttBroker) Init(opts ...broker.Option) error {
	if m.client.IsConnected() {
		return errors.New("cannot init while connected")
	}

	for _, o := range opts {
		o(&m.options)
	}

	m.addrs = setAddrs(m.options.Addrs)
	m.client = newClient(m.addrs, m.options, m)
	return nil
}

func (m *mqttBroker) Connect() error {
	if m.client.IsConnected() {
		return nil
	}

	t := m.client.Connect()

	if rs, err := checkClientToken(t); !rs {
		return err
	}

	return nil
}

func (m *mqttBroker) Disconnect() error {
	if !m.client.IsConnected() {
		return nil
	}
	m.client.Disconnect(0)

	m.subscribers.Clear()

	return nil
}

func (m *mqttBroker) Request(ctx context.Context, topic string, msg *broker.Message, opts ...broker.RequestOption) (*broker.Message, error) {
	return nil, errors.New("not implemented")
}

func (m *mqttBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	var finalTask = m.internalPublish

	if len(m.options.PublishMiddlewares) > 0 {
		finalTask = broker.ChainPublishMiddleware(finalTask, m.options.PublishMiddlewares)
	}

	return finalTask(ctx, topic, msg, opts...)
}

func (m *mqttBroker) internalPublish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(m.options.Codec, msg.Body)
	if err != nil {
		return err
	}

	sendMsg := msg.Clone()
	sendMsg.Body = buf

	return m.publish(ctx, topic, sendMsg, opts...)
}

func (m *mqttBroker) publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	if !m.client.IsConnected() {
		return errors.New("not connected")
	}

	options := broker.PublishOptions{
		Context: ctx,
	}
	for _, o := range opts {
		o(&options)
	}

	var qos byte = 1
	var retained = false

	if value, ok := options.Context.Value(qosPublishKey{}).(byte); ok {
		qos = value
	}
	if value, ok := options.Context.Value(retainedPublishKey{}).(bool); ok {
		retained = value
	}

	ret := m.client.Publish(topic, qos, retained, msg.Body)
	return ret.Error()
}

func (m *mqttBroker) Subscribe(topic string, handler broker.Handler, binder broker.Binder, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	if !m.client.IsConnected() {
		return nil, errors.New("not connected")
	}

	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	if len(m.options.SubscriberMiddlewares) > 0 {
		handler = broker.ChainSubscriberMiddleware(handler, m.options.SubscriberMiddlewares)
	}

	var qos byte = 1
	if value, ok := options.Context.Value(qosSubscribeKey{}).(byte); ok {
		qos = value
	}

	callback := func(c paho.Client, mq paho.Message) {
		var msg broker.Message

		p := &publication{topic: mq.Topic(), msg: &msg}

		if binder != nil {
			msg.Body = binder()

			if err := broker.Unmarshal(m.options.Codec, mq.Payload(), &msg.Body); err != nil {
				p.err = err
				LogError("unmarshal message failed:", err)
				return
			}
		} else {
			msg.Body = mq.Payload()
		}

		if err := handler(m.options.Context, p); err != nil {
			p.err = err
			LogError("handle message failed:", err)
		}
	}

	if err := m.doSubscribe(topic, qos, callback); err != nil {
		return nil, err
	}

	sub := &subscriber{
		m:        m,
		options:  options,
		topic:    topic,
		qos:      qos,
		callback: callback,
	}

	m.subscribers.Add(topic, sub)

	return sub, nil
}

func (m *mqttBroker) doSubscribe(topic string, qos byte, callback paho.MessageHandler) error {
	t := m.client.Subscribe(topic, qos, callback)

	if rs, err := checkClientToken(t); !rs {
		return err
	}

	return nil
}

func (m *mqttBroker) onConnect(_ paho.Client) {
	LogDebug("on connect")

	m.subscribers.Foreach(func(topic string, sub broker.Subscriber) {
		aSub := sub.(*subscriber)
		if err := m.doSubscribe(aSub.topic, aSub.qos, aSub.callback); err != nil {
			LogError("mqtt broker subscribe message failed:", err)
		}
	})
}

func (m *mqttBroker) onConnectionLost(client paho.Client, _ error) {
	LogDebug("on connect lost, try to reconnect")
	m.loopConnect(client)
}

func (m *mqttBroker) loopConnect(client paho.Client) {
	for {
		token := client.Connect()
		if rs, err := checkClientToken(token); !rs {
			LogErrorf("connect error: %s", err.Error())
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}
}
