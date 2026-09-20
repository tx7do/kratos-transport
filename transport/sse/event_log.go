package sse

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type EventLog []*Event

// maxEventLogSize 单流事件日志上限：防止 autoReplay 日志无界增长（内存泄漏）。
// 超限后丢弃最旧事件（Last-Event-ID 重放窗口随之缩小，属预期取舍）
const maxEventLogSize = 1024

func (e *EventLog) Add(ev *Event) {
	if !ev.hasContent() {
		return
	}

	ev.ID = []byte(newEventID())
	ev.timestamp = time.Now()
	*e = append(*e, ev)

	if n := len(*e); n > maxEventLogSize {
		*e = (*e)[n-maxEventLogSize:]
	}
}

func (e *EventLog) Clear() {
	*e = nil
}

func (e *EventLog) Replay(s *Subscriber) {
	for i := 0; i < len(*e); i++ {
		if string((*e)[i].ID) > s.eventId {
			s.connection <- (*e)[i]
		}
	}
}

// newEventID generates a new UUID v7 as the event identifier.
// UUID v7 is time-ordered, ensuring lexicographic sort matches chronological order.
func newEventID() string {
	return strings.ReplaceAll(uuid.Must(uuid.NewV7()).String(), "-", "")
}
