package sse

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventLog(t *testing.T) {
	ev := make(EventLog, 0)
	testEvent := &Event{Data: []byte("test")}

	ev.Add(testEvent)
	ev.Clear()

	assert.Equal(t, 0, len(ev))

	ev.Add(testEvent)
	ev.Add(testEvent)

	assert.Equal(t, 2, len(ev))
}
