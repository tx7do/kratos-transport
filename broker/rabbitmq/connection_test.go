package rabbitmq

import (
	"testing"
)

func TestNewRabbitMQConnURL(t *testing.T) {
	testcases := []struct {
		title string
		urls  []string
		want  string
	}{
		{"Multiple URLs", []string{"amqp://example.com/one", "amqp://example.com/two"}, "amqp://example.com/one"},
		{"Insecure URL", []string{"amqp://example.com"}, "amqp://example.com"},
		{"Secure URL", []string{"amqps://example.com"}, "amqps://example.com"},
		{"Invalid URL", []string{"http://example.com"}, DefaultRabbitURL},
		{"No URLs", []string{}, DefaultRabbitURL},
	}

	for _, test := range testcases {
		conn := newRabbitMQConn(Exchange{Name: "exchange"}, test.urls, 0, false)

		if have, want := conn.url, test.want; have != want {
			t.Errorf("%s: invalid url, want %q, have %q", test.title, want, have)
		}
	}
}

func TestNewRabbitMQPrefetch(t *testing.T) {
	testcases := []struct {
		title          string
		urls           []string
		prefetchCount  int
		prefetchGlobal bool
	}{
		{"Multiple URLs", []string{"amqp://example.com/one", "amqp://example.com/two"}, 1, true},
		{"Insecure URL", []string{"amqp://example.com"}, 1, true},
		{"Secure URL", []string{"amqps://example.com"}, 1, true},
		{"Invalid URL", []string{"http://example.com"}, 1, true},
		{"No URLs", []string{}, 1, true},
	}

	for _, test := range testcases {
		conn := newRabbitMQConn(Exchange{Name: "exchange"}, test.urls, test.prefetchCount, test.prefetchGlobal)

		if have, want := conn.prefetchCount, test.prefetchCount; have != want {
			t.Errorf("%s: invalid prefetch count, want %d, have %d", test.title, want, have)
		}

		if have, want := conn.prefetchGlobal, test.prefetchGlobal; have != want {
			t.Errorf("%s: invalid prefetch global setting, want %t, have %t", test.title, want, have)
		}
	}
}
