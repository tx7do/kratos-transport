package rabbitmq

import (
	"github.com/stretchr/testify/assert"
	"github.com/tx7do/kratos-transport/broker"
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
		var opts []broker.Option
		opts = append(opts, WithExchangeName("exchange"))
		opts = append(opts, broker.WithAddress(test.urls...))
		options := broker.NewOptionsAndApply(opts...)

		conn := newRabbitMQConnection(options)

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
		prefetchSize   int
		prefetchGlobal bool
	}{
		{"Multiple URLs", []string{"amqp://example.com/one", "amqp://example.com/two"}, 1, 0, true},
		{"Insecure URL", []string{"amqp://example.com"}, 1, 0, true},
		{"Secure URL", []string{"amqps://example.com"}, 1, 0, true},
		{"Invalid URL", []string{"http://example.com"}, 1, 0, true},
		{"No URLs", []string{}, 1, 0, true},
	}

	for _, test := range testcases {

		var opts []broker.Option
		opts = append(opts, WithExchangeName("exchange"))
		opts = append(opts, broker.WithAddress(test.urls...))
		opts = append(opts, WithPrefetchCount(test.prefetchCount))
		opts = append(opts, WithPrefetchSize(test.prefetchSize))
		if test.prefetchGlobal {
			opts = append(opts, WithPrefetchGlobal())
		}

		options := broker.NewOptionsAndApply(opts...)

		conn := newRabbitMQConnection(options)

		if have, want := conn.qos.PrefetchCount, test.prefetchCount; have != want {
			t.Errorf("%s: invalid prefetch count, want %d, have %d", test.title, want, have)
		}

		if have, want := conn.qos.PrefetchGlobal, test.prefetchGlobal; have != want {
			t.Errorf("%s: invalid prefetch global setting, want %t, have %t", test.title, want, have)
		}
	}
}

func TestHasRabbitUrlPrefix(t *testing.T) {
	assert.True(t, hasUrlPrefix("amqp://example.com"))
	assert.True(t, hasUrlPrefix("amqps://example.com"))
	assert.False(t, hasUrlPrefix("http://example.com"))
	assert.False(t, hasUrlPrefix("https://example.com"))
	assert.False(t, hasUrlPrefix("example.com"))
}
