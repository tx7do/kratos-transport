package sse

import (
	"errors"
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
	s := NewServer(
		WithAddress(":8800"),
	)
	defer s.Stop(nil)

	s.CreateStream("test")
	stream := s.streamMgr.Get("test")
	sub := stream.addSubscriber(0, nil)

	s.Publish("test", &Event{Data: []byte("test")})

	msg, err := wait(sub.connection, time.Second*1)
	require.Nil(t, err)
	assert.Equal(t, []byte(`test`), msg)
}

func TestServerNonExistentStreamPublish(t *testing.T) {
	s := NewServer(
		WithAddress(":8800"),
	)
	defer s.Stop(nil)

	s.streamMgr.RemoveWithID("test")

	assert.NotPanics(t, func() { s.Publish("test", &Event{Data: []byte("test")}) })
}
