package nats

import natsGo "github.com/nats-io/nats.go"

func natsHeaderToMap(h natsGo.Header) map[string]string {
	m := map[string]string{}

	for k, v := range h {
		m[k] = v[0]
	}

	return m
}
