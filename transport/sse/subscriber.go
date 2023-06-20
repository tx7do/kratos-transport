package sse

import "net/url"

type Subscriber struct {
	quit       chan *Subscriber
	connection chan *Event
	removed    chan struct{}
	eventId    int
	URL        *url.URL
}

func (s *Subscriber) close() {
	s.quit <- s
	if s.removed != nil {
		<-s.removed
	}
}
