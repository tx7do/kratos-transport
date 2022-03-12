package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tx7do/kratos-transport/broker"
)

type mqttPub struct {
	topic string
	msg   *broker.Message
	err   error
}

func (m *mqttPub) Ack() error {
	return nil
}

func (m *mqttPub) Error() error {
	return m.err
}

func (m *mqttPub) Topic() string {
	return m.topic
}

func (m *mqttPub) Message() *broker.Message {
	return m.msg
}

type mqttSub struct {
	opts   broker.SubscribeOptions
	topic  string
	client mqtt.Client
}

func (m *mqttSub) Options() broker.SubscribeOptions {
	return m.opts
}

func (m *mqttSub) Topic() string {
	return m.topic
}

func (m *mqttSub) Unsubscribe() error {
	t := m.client.Unsubscribe(m.topic)
	return t.Error()
}
