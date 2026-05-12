package sse

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPStreamHandler(t *testing.T) {
	s := NewServer(
		WithAddress(":8800"),
	)
	defer s.Stop(nil)

	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	c := NewClient(server.URL + "/events")

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

	time.Sleep(time.Millisecond * 200)
	require.Nil(t, cErr)
	s.Publish(ctx, "test", &Event{Data: []byte("test")})

	msg, err := wait(events, time.Millisecond*500)
	require.Nil(t, err)
	assert.Equal(t, []byte(`test`), msg)
}

func TestHTTPStreamHandlerExistingEvents(t *testing.T) {
	s := NewServer(
		WithAddress(":8800"),
	)
	defer s.Stop(nil)

	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	s.Publish(ctx, "test", &Event{Data: []byte("test 1")})
	s.Publish(ctx, "test", &Event{Data: []byte("test 2")})
	s.Publish(ctx, "test", &Event{Data: []byte("test 3")})

	time.Sleep(time.Millisecond * 100)

	c := NewClient(server.URL + "/events")

	events := make(chan *Event)
	var cErr error
	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if len(msg.Data) > 0 {
				events <- msg
			}
		})
	}()

	require.Nil(t, cErr)

	for i := 1; i <= 3; i++ {
		msg, err := wait(events, time.Millisecond*500)
		require.Nil(t, err)
		assert.Equal(t, []byte("test "+strconv.Itoa(i)), msg)
	}
}

func TestHTTPStreamHandlerEventID(t *testing.T) {
	s := NewServer(
		WithAddress(":8800"),
	)
	defer s.Stop(nil)

	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	s.Publish(ctx, "test", &Event{Data: []byte("test 1")})
	s.Publish(ctx, "test", &Event{Data: []byte("test 2")})
	s.Publish(ctx, "test", &Event{Data: []byte("test 3")})

	time.Sleep(time.Millisecond * 100)

	c := NewClient(server.URL + "/events")
	c.LastEventID.Store([]byte("2"))

	events := make(chan *Event)
	var cErr error
	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if len(msg.Data) > 0 {
				events <- msg
			}
		})
	}()

	require.Nil(t, cErr)

	msg, err := wait(events, time.Millisecond*500)
	require.Nil(t, err)
	assert.Equal(t, []byte("test 3"), msg)
}

func TestHTTPStreamHandlerEventTTL(t *testing.T) {
	s := NewServer(
		WithAddress(":8800"),
	)
	defer s.Stop(nil)

	s.eventTTL = time.Second * 1

	ctx := context.Background()

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)

	s.CreateStream("test")

	s.Publish(ctx, "test", &Event{Data: []byte("test 1")})
	s.Publish(ctx, "test", &Event{Data: []byte("test 2")})
	time.Sleep(time.Second * 2)
	s.Publish(ctx, "test", &Event{Data: []byte("test 3")})

	time.Sleep(time.Millisecond * 100)

	c := NewClient(server.URL + "/events")

	events := make(chan *Event)
	var cErr error
	go func() {
		cErr = c.Subscribe("test", func(msg *Event) {
			if len(msg.Data) > 0 {
				events <- msg
			}
		})
	}()

	require.Nil(t, cErr)

	msg, err := wait(events, time.Millisecond*500)
	require.Nil(t, err)
	assert.Equal(t, []byte("test 3"), msg)
}

func TestHTTPStreamHandlerHeaderFlushIfNoEvents(t *testing.T) {
	ctx := context.Background()

	//go func() {
	s := NewServer(
		WithAddress(":8800"),
	)
	defer s.Stop(ctx)

	s.HandleServeHTTP("/events")
	s.CreateStream("test")

	s.Start(ctx)
	//}()

	c := NewClient("localhost:8800/events")

	subscribed := make(chan struct{})
	events := make(chan *Event)
	go func() {
		assert.NoError(t, c.SubscribeChan("test", events))
		subscribed <- struct{}{}
	}()

	select {
	case <-subscribed:
	case <-time.After(1000 * time.Millisecond):
		assert.Fail(t, "Subscribe should returned in 100 milliseconds")
	}
}

func TestHTTPStreamHandlerSubscriberCanReadAuthorizationToken(t *testing.T) {
	tokenCh := make(chan string, 1)

	s := NewServer(
		WithAddress(":8800"),
		WithSubscriberFunction(func(streamID StreamID, sub *Subscriber) {
			if streamID == "test" {
				tokenCh <- sub.Token("")
			}
		}),
	)
	defer s.Stop(nil)

	s.CreateStream("test")

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/events?stream=test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer header-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	select {
	case token := <-tokenCh:
		assert.Equal(t, "header-token", token)
	case <-time.After(time.Second):
		t.Fatal("subscriber callback did not receive token in time")
	}
}

func TestHTTPStreamHandlerAuthorizeUnauthorized(t *testing.T) {
	s := NewServer(
		WithAutoStream(true),
		WithAuthorizeFunc(func(_ *http.Request, token string) error {
			if token == "" {
				return ErrUnauthorized
			}
			return nil
		}),
	)
	defer s.Stop(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/events?stream=test", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestHTTPStreamHandlerAuthorizeForbidden(t *testing.T) {
	s := NewServer(
		WithAutoStream(true),
		WithAuthorizeFunc(func(_ *http.Request, token string) error {
			if token != "ok-token" {
				return errors.Join(ErrForbidden, errors.New("invalid token"))
			}
			return nil
		}),
	)
	defer s.Stop(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/events?stream=test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer bad-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestHTTPStreamHandlerAuthorizeSuccessWithAuthorizationHeader(t *testing.T) {
	s := NewServer(
		WithAutoStream(true),
		WithAuthorizeFunc(func(_ *http.Request, token string) error {
			if token != "ok-token" {
				return ErrUnauthorized
			}
			return nil
		}),
	)
	defer s.Stop(nil)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.ServeHTTP)
	server := httptest.NewServer(mux)
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/events?stream=test", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer ok-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHTTPStreamHandlerAutoStream(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sseServer := NewServer()
	defer sseServer.Stop(nil)

	sseServer.autoReplay = false

	sseServer.autoStream = true

	mux := http.NewServeMux()
	mux.HandleFunc("/events", sseServer.ServeHTTP)
	srv := httptest.NewServer(mux)

	c := NewClient(srv.URL + "/events")

	events := make(chan *Event)

	cErr := make(chan error)

	go func() {
		cErr <- c.SubscribeChan("test", events)
	}()

	require.Nil(t, <-cErr)

	sseServer.Publish(ctx, "test", &Event{Data: []byte("test")})

	msg, err := wait(events, 1*time.Second)

	require.Nil(t, err)

	assert.Equal(t, []byte(`test`), msg)

	c.Unsubscribe(events)

	_, _ = wait(events, 1*time.Second)

	assert.Equal(t, (*Stream)(nil), sseServer.streamMgr.Get("test"))
}
