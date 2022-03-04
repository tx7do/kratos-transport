package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/common"
)

// mqttPub is a common.Event
type mqttPub struct {
	topic string
	msg   *broker.Message
	err   error
}

// mqttPub is a common.Subscriber
type mqttSub struct {
	opts   common.SubscribeOptions
	topic  string
	client mqtt.Client
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

func (m *mqttSub) Options() common.SubscribeOptions {
	return m.opts
}

func (m *mqttSub) Topic() string {
	return m.topic
}

func (m *mqttSub) Unsubscribe() error {
	t := m.client.Unsubscribe(m.topic)
	return t.Error()
}
