package sse

import (
	"net/url"
	"sync"
	"sync/atomic"
)

type StreamID string

type SubscriberFunction func(streamID StreamID, sub *Subscriber)

type Stream struct {
	id StreamID

	event    chan *Event
	quit     chan struct{}
	quitOnce sync.Once
	eventLog EventLog

	autoReplay bool
	autoStream bool

	register   chan *Subscriber
	deregister chan *Subscriber

	subscribers     []*Subscriber
	subscriberCount int32

	onSubscribe   SubscriberFunction
	onUnsubscribe SubscriberFunction
}

func newStream(id StreamID, buffSize int, replay, autoStream bool, onSubscribe, onUnsubscribe SubscriberFunction) *Stream {
	return &Stream{
		id:            id,
		autoStream:    autoStream,
		autoReplay:    replay,
		subscribers:   make([]*Subscriber, 0),
		register:      make(chan *Subscriber),
		deregister:    make(chan *Subscriber),
		event:         make(chan *Event, buffSize),
		quit:          make(chan struct{}),
		eventLog:      make(EventLog, 0),
		onSubscribe:   onSubscribe,
		onUnsubscribe: onUnsubscribe,
	}
}

func (s *Stream) StreamID() StreamID {
	return s.id
}

func (s *Stream) run() {
	go func(stream *Stream) {
		for {
			select {
			case subscriber := <-stream.register:
				stream.subscribers = append(stream.subscribers, subscriber)
				if stream.autoReplay {
					stream.eventLog.Replay(subscriber)
				}

			case subscriber := <-stream.deregister:
				i := stream.getSubIndex(subscriber)
				if i != -1 {
					stream.removeSubscriber(i)
				}

				if stream.onUnsubscribe != nil {
					go stream.onUnsubscribe(stream.id, subscriber)
				}

			case event := <-stream.event:
				if stream.autoReplay {
					stream.eventLog.Add(event)
				}
				for i := range stream.subscribers {
					stream.subscribers[i].connection <- event
				}

			case <-stream.quit:
				stream.removeAllSubscribers()
				return
			}
		}
	}(s)
}

func (s *Stream) close() {
	s.quitOnce.Do(func() {
		close(s.quit)
	})
}

func (s *Stream) getSubIndex(sub *Subscriber) int {
	for i := range s.subscribers {
		if s.subscribers[i] == sub {
			return i
		}
	}
	return -1
}

func (s *Stream) addSubscriber(eventId int, url *url.URL) *Subscriber {
	atomic.AddInt32(&s.subscriberCount, 1)
	sub := &Subscriber{
		eventId:    eventId,
		quit:       s.deregister,
		connection: make(chan *Event, 64),
		URL:        url,
	}

	if s.autoStream {
		sub.removed = make(chan struct{}, 1)
	}

	s.register <- sub

	if s.onSubscribe != nil {
		go s.onSubscribe(s.id, sub)
	}

	return sub
}

func (s *Stream) removeSubscriber(i int) {
	atomic.AddInt32(&s.subscriberCount, -1)
	close(s.subscribers[i].connection)
	if s.subscribers[i].removed != nil {
		s.subscribers[i].removed <- struct{}{}
		close(s.subscribers[i].removed)
	}
	s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
}

func (s *Stream) removeAllSubscribers() {
	for i := 0; i < len(s.subscribers); i++ {
		close(s.subscribers[i].connection)
		if s.subscribers[i].removed != nil {
			s.subscribers[i].removed <- struct{}{}
			close(s.subscribers[i].removed)
		}
	}
	atomic.StoreInt32(&s.subscriberCount, 0)
	s.subscribers = s.subscribers[:0]
}

func (s *Stream) getSubscriberCount() int {
	return int(atomic.LoadInt32(&s.subscriberCount))
}
