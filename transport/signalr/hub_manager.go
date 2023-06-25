package signalr

import "sync"

type HubMap map[HubID]*Hub

type HubManager struct {
	sessions       HubMap
	mtx            sync.RWMutex
	connectHandler ConnectHandler
}

func NewHubManager() *HubManager {
	return &HubManager{
		sessions: make(HubMap),
	}
}

func (s *HubManager) RegisterConnectHandler(handler ConnectHandler) {
	s.connectHandler = handler
}

func (s *HubManager) Clean() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.sessions = HubMap{}
}

func (s *HubManager) Count() int {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return len(s.sessions)
}

func (s *HubManager) Get(hubId HubID) (*Hub, bool) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	c, ok := s.sessions[hubId]
	return c, ok
}

func (s *HubManager) Range(fn func(*Hub)) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for _, v := range s.sessions {
		fn(v)
	}
}

func (s *HubManager) Add(c *Hub) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	//log.Info("[signalr] add session: ", c.HubID())
	s.sessions[c.HubID()] = c

	if s.connectHandler != nil {
		s.connectHandler(c.HubID(), true)
	}
}

func (s *HubManager) Remove(c *Hub) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	for k, v := range s.sessions {
		if c == v {
			//log.Info("[signalr] remove session: ", c.HubID())
			if s.connectHandler != nil {
				s.connectHandler(c.HubID(), false)
			}
			delete(s.sessions, k)
			return
		}
	}
}
