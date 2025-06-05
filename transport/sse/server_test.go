package sse

import (
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wait(ch chan *Event, duration time.Duration) ([]byte, error) {
	var err error
	var msg []byte

	select {
	case event := <-ch:
		msg = event.Data
	case <-time.After(duration):
		err = errors.New("timeout")
	}
	return msg, err
}

func waitEvent(ch chan *Event, duration time.Duration) (*Event, error) {
	select {
	case event := <-ch:
		return event, nil
	case <-time.After(duration):
		return nil, errors.New("timeout")
	}
}

func TestServerExistingStreamPublish(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	s := NewServer(
		WithAddress(":8800"),
		WithCodec("json"),
		WithPath("/events"),
		WithSubscriberFunction(func(streamID StreamID, sub *Subscriber) {
			var token string
			if sub.URL != nil {
				token = sub.URL.Query().Get("token")
			}
			LogInfof("subscriber [%s] [%+v] connected", streamID, token)
		}),
	)
	defer s.Stop(ctx)

	s.CreateStream("test")

	stream := s.streamMgr.Get("test")
	sub := stream.addSubscriber(0, nil)

	go func() {
		_ = s.Start(ctx)
	}()

	s.Publish(ctx, "test", &Event{Data: []byte("ping")})

	msg, err := wait(sub.connection, time.Second*1)
	require.Nil(t, err)
	assert.Equal(t, []byte(`ping`), msg)

	<-interrupt
}

func TestServerNonExistentStreamPublish(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	s := NewServer(
		WithAddress(":8800"),
		WithCodec("json"),
		WithPath("/events"),
	)
	defer s.Stop(ctx)

	s.CreateStream("test")

	go func() {
		s.streamMgr.RemoveWithID("test")
		_ = s.Start(ctx)
	}()

	assert.NotPanics(t, func() {
		_ = s.PublishData(ctx, "test", &Event{Data: []byte("test")})
	})

	<-interrupt
}

func TestServerPublishData(t *testing.T) {
	host, port, err := net.SplitHostPort("127.0.0.1:8800")
	require.NoError(t, err)
	t.Logf("host: %s, port: %s", host, port)
}
