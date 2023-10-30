package sse

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gopkg.in/cenkalti/backoff.v1"
)

var urlPath string
var srv *Server
var server *httptest.Server

var mldata = `{
	"key": "value",
	"array": [
		1,
		2,
		3
	]
}`

func setup(empty bool) {
	srv = newServer()
	go publishMsgs(srv, empty, 100000000)
}

func setupMultiline() {
	srv = newServer()
	srv.splitData = true
	go publishMultilineMessages(srv, 100000000)
}

func setupCount(empty bool, count int) {
	srv = newServer()
	go publishMsgs(srv, empty, count)
}

func newServer() *Server {
	srv = NewServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", srv.ServeHTTP)
	server = httptest.NewServer(mux)
	urlPath = server.URL + "/events"

	srv.CreateStream("test")

	return srv
}

func newServer401() *Server {
	srv = NewServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	server = httptest.NewServer(mux)
	urlPath = server.URL + "/events"

	srv.CreateStream("test")

	return srv
}

func publishMsgs(s *Server, empty bool, count int) {
	ctx := context.Background()

	for a := 0; a < count; a++ {
		if empty {
			s.Publish(ctx, "test", &Event{Data: []byte("\n")})
		} else {
			s.Publish(ctx, "test", &Event{Data: []byte("ping")})
		}
		time.Sleep(time.Millisecond * 50)
	}
}

func publishMultilineMessages(s *Server, count int) {
	ctx := context.Background()
	for a := 0; a < count; a++ {
		s.Publish(ctx, "test", &Event{ID: []byte("123456"), Data: []byte(mldata)})
	}
}

func cleanup() {
	server.CloseClientConnections()
	server.Close()
	_ = srv.Stop(nil)
}

func TestClientSubscribe(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(urlPath)

	events := make(chan *Event)
	var cErr error
	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if msg.Data != nil {
				events <- msg
				return
			}
		})
	}()

	for i := 0; i < 5; i++ {
		msg, err := wait(events, time.Second*1)
		require.Nil(t, err)
		assert.Equal(t, []byte(`ping`), msg)
	}

	assert.Nil(t, cErr)
}

func TestClientSubscribeMultiline(t *testing.T) {
	setupMultiline()
	defer cleanup()

	c := NewClient(urlPath)

	events := make(chan *Event)
	var cErr error

	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if msg.Data != nil {
				events <- msg
				return
			}
		})
	}()

	for i := 0; i < 5; i++ {
		msg, err := wait(events, time.Second*1)
		require.Nil(t, err)
		assert.Equal(t, []byte(mldata), msg)
	}

	assert.Nil(t, cErr)
}

func TestClientChanSubscribeEmptyMessage(t *testing.T) {
	setup(true)
	defer cleanup()

	c := NewClient(urlPath)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	for i := 0; i < 5; i++ {
		_, err := waitEvent(events, time.Second)
		require.Nil(t, err)
	}
}

func TestClientChanSubscribe(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(urlPath)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	for i := 0; i < 5; i++ {
		msg, merr := wait(events, time.Second*1)
		if msg == nil {
			i--
			continue
		}
		assert.Nil(t, merr)
		assert.Equal(t, []byte(`ping`), msg)
	}
	c.Unsubscribe(events)
}

func TestClientOnDisconnect(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(urlPath)

	called := make(chan struct{})
	c.OnDisconnect(func(client *Client) {
		called <- struct{}{}
	})

	go c.Subscribe("test", func(msg *Event) {})

	time.Sleep(time.Second)
	server.CloseClientConnections()

	assert.Equal(t, struct{}{}, <-called)
}

func TestClientOnConnect(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(urlPath)

	called := make(chan struct{})
	c.OnConnect(func(client *Client) {
		called <- struct{}{}
	})

	go c.Subscribe("test", func(msg *Event) {})

	time.Sleep(time.Second)
	assert.Equal(t, struct{}{}, <-called)

	server.CloseClientConnections()
}

func TestClientChanReconnect(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(urlPath)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	for i := 0; i < 10; i++ {
		if i == 5 {
			server.CloseClientConnections()
		}
		msg, merr := wait(events, time.Second*1)
		if msg == nil {
			i--
			continue
		}
		assert.Nil(t, merr)
		assert.Equal(t, []byte(`ping`), msg)
	}
	c.Unsubscribe(events)
}

func TestClientUnsubscribe(t *testing.T) {
	setup(false)
	defer cleanup()

	c := NewClient(urlPath)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	time.Sleep(time.Millisecond * 500)

	go c.Unsubscribe(events)
	go c.Unsubscribe(events)
}

func TestClientUnsubscribeNonBlock(t *testing.T) {
	count := 2
	setupCount(false, count)
	defer cleanup()

	c := NewClient(urlPath)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	for i := 0; i < count; i++ {
		msg, merr := wait(events, time.Second*1)
		assert.Nil(t, merr)
		assert.Equal(t, []byte(`ping`), msg)
	}
	doneCh := make(chan *Event)
	go func() {
		var e Event
		c.Unsubscribe(events)
		doneCh <- &e
	}()
	_, merr := wait(doneCh, time.Millisecond*100)
	assert.Nil(t, merr)
}

func TestClientUnsubscribe401(t *testing.T) {
	srv = newServer401()
	defer cleanup()

	c := NewClient(urlPath)

	c.ReconnectStrategy = backoff.WithMaxTries(
		backoff.NewExponentialBackOff(),
		3,
	)

	err := c.SubscribeRaw(func(ev *Event) {
		assert.False(t, true)
	})

	require.NotNil(t, err)
}

func TestClientLargeData(t *testing.T) {
	srv = newServer()
	defer cleanup()

	ctx := context.Background()

	c := NewClient(urlPath, ClientMaxBufferSize(1<<19))

	c.ReconnectStrategy = backoff.WithMaxTries(
		backoff.NewExponentialBackOff(),
		3,
	)

	data := make([]byte, 1<<17)
	rand.Read(data)
	data = []byte(hex.EncodeToString(data))

	ec := make(chan *Event, 1)

	srv.Publish(ctx, "test", &Event{Data: data})

	go func() {
		_ = c.Subscribe("test", func(ev *Event) {
			ec <- ev
		})
	}()

	d, err := wait(ec, time.Second)
	require.Nil(t, err)
	require.Equal(t, data, d)
}

func TestClientComment(t *testing.T) {
	srv = newServer()
	defer cleanup()

	ctx := context.Background()

	c := NewClient(urlPath)

	events := make(chan *Event)
	err := c.SubscribeChan("test", events)
	require.Nil(t, err)

	srv.Publish(ctx, "test", &Event{Comment: []byte("comment")})
	srv.Publish(ctx, "test", &Event{Data: []byte("test")})

	ev, err := waitEvent(events, time.Second*1)
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), ev.Data)

	c.Unsubscribe(events)
}

func TestTrimHeader(t *testing.T) {
	tests := []struct {
		input []byte
		want  []byte
	}{
		{
			input: []byte("data: real data"),
			want:  []byte("real data"),
		},
		{
			input: []byte("data:real data"),
			want:  []byte("real data"),
		},
		{
			input: []byte("data:"),
			want:  []byte(""),
		},
	}

	for _, tc := range tests {
		got := trimHeader(len(headerData), tc.input)
		require.Equal(t, tc.want, got)
	}
}

func TestSubscribeWithContextDone(t *testing.T) {
	setup(false)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	var n1 = runtime.NumGoroutine()

	c := NewClient(urlPath)

	for i := 0; i < 10; i++ {
		go c.SubscribeWithContext(ctx, "test", func(msg *Event) {})
	}

	time.Sleep(1 * time.Second)
	cancel()

	time.Sleep(1 * time.Second)
	var n2 = runtime.NumGoroutine()

	assert.Equal(t, n1, n2)
}
