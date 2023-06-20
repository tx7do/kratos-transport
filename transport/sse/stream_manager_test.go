package sse

import (
	"github.com/stretchr/testify/assert"
	"runtime"
	"testing"
)

func createStream(id StreamID) *Stream {
	stream := newStream(id, 1024, true, true, nil, nil)
	stream.run()
	return stream
}

func TestCreateStream(t *testing.T) {
	streamMgr := NewStreamManager()

	stream := createStream("test")
	streamMgr.Add(stream)

	assert.NotNil(t, streamMgr.Get("test"))
}

func TestCreateExistingStream(t *testing.T) {
	streamMgr := NewStreamManager()

	stream := createStream("test")
	streamMgr.Add(stream)

	numGoRoutines := runtime.NumGoroutine()

	streamMgr.Add(stream)

	assert.NotNil(t, streamMgr.Get("test"))
	assert.Equal(t, numGoRoutines, runtime.NumGoroutine())
}

func TestRemoveStream(t *testing.T) {
	streamMgr := NewStreamManager()

	stream := createStream("test")
	streamMgr.Add(stream)

	streamMgr.RemoveWithID("test")

	assert.Nil(t, streamMgr.Get("test"))
}

func TestRemoveNonExistentStream(t *testing.T) {
	streamMgr := NewStreamManager()

	streamMgr.RemoveWithID("test")

	assert.NotPanics(t, func() { streamMgr.RemoveWithID("test") })
}
