package nats

import NATS "github.com/nats-io/nats.go"

func natsHeaderToMap(h NATS.Header) map[string]string {
	m := map[string]string{}

	for k, v := range h {
		m[k] = v[0]
	}

	return m
}
