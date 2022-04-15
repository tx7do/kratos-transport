package mqtt

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tx7do/kratos-transport/broker"
)

type subscriber struct {
	opts   broker.SubscribeOptions
	topic  string
	client mqtt.Client
}

func (m *subscriber) Options() broker.SubscribeOptions {
	return m.opts
}

func (m *subscriber) Topic() string {
	return m.topic
}

func (m *subscriber) Unsubscribe() error {
	t := m.client.Unsubscribe(m.topic)
	return t.Error()
}
