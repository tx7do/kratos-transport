package sse

import (
	"net/url"
	"sync"
	"sync/atomic"
)

type StreamID string

type SubscriberFunction func(streamID StreamID, sub *Subscriber)

type Stream struct {
	ID           StreamID
	event        chan *Event
	quit         chan struct{}
	quitOnce     sync.Once
	EventLog     EventLog
	AutoReplay   bool
	isAutoStream bool

	register   chan *Subscriber
	deregister chan *Subscriber

	subscribers     []*Subscriber
	subscriberCount int32

	OnSubscribe   SubscriberFunction
	OnUnsubscribe SubscriberFunction
}

func newStream(id StreamID, buffSize int, replay, isAutoStream bool, onSubscribe, onUnsubscribe SubscriberFunction) *Stream {
	return &Stream{
		ID:            id,
		AutoReplay:    replay,
		subscribers:   make([]*Subscriber, 0),
		isAutoStream:  isAutoStream,
		register:      make(chan *Subscriber),
		deregister:    make(chan *Subscriber),
		event:         make(chan *Event, buffSize),
		quit:          make(chan struct{}),
		EventLog:      make(EventLog, 0),
		OnSubscribe:   onSubscribe,
		OnUnsubscribe: onUnsubscribe,
	}
}

func (s *Stream) StreamID() StreamID {
	return s.ID
}

func (s *Stream) run() {
	go func(stream *Stream) {
		for {
			select {
			case subscriber := <-stream.register:
				stream.subscribers = append(stream.subscribers, subscriber)
				if stream.AutoReplay {
					stream.EventLog.Replay(subscriber)
				}

			case subscriber := <-stream.deregister:
				i := stream.getSubIndex(subscriber)
				if i != -1 {
					stream.removeSubscriber(i)
				}

				if stream.OnUnsubscribe != nil {
					go stream.OnUnsubscribe(stream.ID, subscriber)
				}

			case event := <-stream.event:
				if stream.AutoReplay {
					stream.EventLog.Add(event)
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

	if s.isAutoStream {
		sub.removed = make(chan struct{}, 1)
	}

	s.register <- sub

	if s.OnSubscribe != nil {
		go s.OnSubscribe(s.ID, sub)
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
