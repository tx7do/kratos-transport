package mqtt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/tx7do/kratos-transport/broker"
)

type mqttBroker struct {
	addrs  []string
	opts   broker.Options
	client MQTT.Client
}

func NewBroker(opts ...broker.Option) broker.Broker {
	return newBroker(opts...)
}

func newClient(addrs []string, opts broker.Options, b *mqttBroker) MQTT.Client {
	cOpts := MQTT.NewClientOptions()

	// 是否清除会话，如果true，mqtt服务端将会清除掉
	cOpts.SetCleanSession(true)
	// 设置自动重连接
	cOpts.SetAutoReconnect(false)
	// 设置连接之后恢复订阅
	cOpts.SetResumeSubs(true)
	// 设置保活时间
	//cOpts.SetKeepAlive(60)
	// 设置最大重连时间间隔
	//cOpts.SetMaxReconnectInterval(30)
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

	return MQTT.NewClient(cOpts)
}

func newBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptionsAndApply(opts...)

	b := &mqttBroker{
		opts:  options,
		addrs: options.Addrs,
	}

	b.client = newClient(options.Addrs, options, b)

	return b
}

func (m *mqttBroker) Name() string {
	return "MQTT"
}

func (m *mqttBroker) Options() broker.Options {
	return m.opts
}

func (m *mqttBroker) Address() string {
	return strings.Join(m.addrs, ",")
}

func (m *mqttBroker) Init(opts ...broker.Option) error {
	if m.client.IsConnected() {
		return errors.New("cannot init while connected")
	}

	for _, o := range opts {
		o(&m.opts)
	}

	m.addrs = setAddrs(m.opts.Addrs)
	m.client = newClient(m.addrs, m.opts, m)
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
	return nil
}

func (m *mqttBroker) Publish(topic string, msg broker.Any, opts ...broker.PublishOption) error {
	buf, err := broker.Marshal(m.opts.Codec, msg)
	if err != nil {
		return err
	}

	return m.publish(topic, buf, opts...)
}

func (m *mqttBroker) publish(topic string, buf []byte, opts ...broker.PublishOption) error {
	if !m.client.IsConnected() {
		return errors.New("not connected")
	}

	options := broker.PublishOptions{
		Context: context.Background(),
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

	ret := m.client.Publish(topic, qos, retained, buf)
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

	var qos byte = 1

	if value, ok := options.Context.Value(qosSubscribeKey{}).(byte); ok {
		qos = value
	}

	t := m.client.Subscribe(topic, qos, func(c MQTT.Client, mq MQTT.Message) {
		var msg broker.Message

		p := &publication{topic: mq.Topic(), msg: &msg}

		if binder != nil {
			msg.Body = binder()
		} else {
			msg.Body = mq.Payload()
		}

		if err := broker.Unmarshal(m.opts.Codec, mq.Payload(), &msg.Body); err != nil {
			p.err = err
			log.Error(err)
			return
		}

		if err := handler(m.opts.Context, p); err != nil {
			p.err = err
			log.Error(err)
		}
	})

	if rs, err := checkClientToken(t); !rs {
		return nil, err
	}

	return &subscriber{
		opts:   options,
		client: m.client,
		topic:  topic,
	}, nil
}

func (m *mqttBroker) onConnect(_ MQTT.Client) {
	fmt.Println("on connect")
}

func (m *mqttBroker) onConnectionLost(client MQTT.Client, _ error) {
	fmt.Println("on connect lost, try to reconnect")
	m.loopConnect(client)
}

func (m *mqttBroker) loopConnect(client MQTT.Client) {
	for {
		token := client.Connect()
		if rs, err := checkClientToken(token); !rs {
			fmt.Printf("connect error: %s\n", err.Error())
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}
}
