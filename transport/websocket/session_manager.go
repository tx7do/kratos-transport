package websocket

import "sync"

type SessionMap map[SessionID]*Session

type SessionManager struct {
	sessions       SessionMap
	mtx            sync.RWMutex
	connectHandler SocketConnectHandler
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(SessionMap),
	}
}

func (s *SessionManager) RegisterConnectHandler(handler SocketConnectHandler) {
	s.connectHandler = handler
}

func (s *SessionManager) Clean() {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.sessions = SessionMap{}
}

func (s *SessionManager) Count() int {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	return len(s.sessions)
}

func (s *SessionManager) Get(sessionId SessionID) (*Session, bool) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	c, ok := s.sessions[sessionId]
	return c, ok
}

func (s *SessionManager) Range(fn func(*Session)) {
	s.mtx.RLock()
	defer s.mtx.RUnlock()

	for _, v := range s.sessions {
		fn(v)
	}
}

func (s *SessionManager) Add(session *Session) {
	s.mtx.Lock()
	s.sessions[session.SessionID()] = session
	s.mtx.Unlock()

	//log.Info("[websocket] add session: ", session.SessionID())

	if s.connectHandler != nil {
		s.connectHandler(session.SessionID(), session.queries, true)
	}
}

func (s *SessionManager) Remove(session *Session) {
	s.mtx.Lock()

	for k, v := range s.sessions {
		if session == v {
			//log.Info("[websocket] remove session: ", session.SessionID())
			delete(s.sessions, k)
			s.mtx.Unlock()

			if s.connectHandler != nil {
				s.connectHandler(session.SessionID(), session.queries, false)
			}
			return
		}
	}

	s.mtx.Unlock()
}
