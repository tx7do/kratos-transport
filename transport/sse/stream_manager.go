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

// Range 对全部流执行回调。先快照再在锁外遍历：
// 回调里可能向流的事件通道阻塞发送（慢订阅者），持锁会拖死所有
// Get/Add/CreateStream 调用
func (s *StreamManager) Range(fn func(*Stream)) {
	s.mtx.Lock()
	streams := make([]*Stream, 0, len(s.streams))
	for _, v := range s.streams {
		streams = append(streams, v)
	}
	s.mtx.Unlock()

	for _, v := range streams {
		fn(v)
	}
}

func (s *StreamManager) Add(stream *Stream) {
	if stream == nil {
		return
	}

	// Exist 检查与写入合并进同一临界区，消除 TOCTOU（双写覆盖 → 孤儿流泄漏）
	s.mtx.Lock()
	defer s.mtx.Unlock()
	if _, ok := s.streams[stream.StreamID()]; ok {
		return
	}
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
			//LogInfo("remove stream: ", stream.StreamID())
			s.streams[k].close()
			delete(s.streams, k)
			return
		}
	}
}
