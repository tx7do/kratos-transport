package sse

import "sync"

type StreamMap map[StreamID]*Stream

type StreamManager struct {
	streams StreamMap
	mtx     sync.RWMutex
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(StreamMap),
	}
}

func (s *StreamManager) Clean() {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for _, v := range s.streams {
		v.close()
	}
	s.streams = make(StreamMap)
}

func (s *StreamManager) Count() int {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	return len(s.streams)
}

func (s *StreamManager) Get(streamId StreamID) *Stream {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	c, _ := s.streams[streamId]
	return c
}

func (s *StreamManager) Exist(streamId StreamID) bool {
	stream := s.Get(streamId)
	return stream != nil
}

func (s *StreamManager) Range(fn func(*Stream)) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for _, v := range s.streams {
		fn(v)
	}
}

func (s *StreamManager) Add(stream *Stream) {
	if stream == nil {
		return
	}

	if s.Exist(stream.StreamID()) {
		return
	}

	//log.Info("[sse] add stream: ", stream.StreamID())
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.streams[stream.StreamID()] = stream
}

func (s *StreamManager) RemoveWithID(streamId StreamID) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if s.streams[streamId] != nil {
		s.streams[streamId].close()
		delete(s.streams, streamId)
	}
}

func (s *StreamManager) Remove(stream *Stream) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for k, v := range s.streams {
		if stream == v {
			//log.Info("[sse] remove stream: ", stream.StreamID())
			s.streams[k].close()
			delete(s.streams, k)
			return
		}
	}
}
