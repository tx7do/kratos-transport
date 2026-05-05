package webrtc

import "sync"

// SessionObserver defines the interface for observing session events such as addition and removal of sessions.
type SessionObserver interface {
	// OnSessionAdded is called when a new session is added to the SessionManager. It receives the newly added session as an argument.
	OnSessionAdded(session *Session)

	// OnSessionRemoved is called when a session is removed from the SessionManager. It receives the removed session as an argument.
	OnSessionRemoved(session *Session)
}

type SessionManager struct {
	sessions sync.Map

	observerMu sync.RWMutex
	observer   SessionObserver
}

func NewSessionManager(observer SessionObserver) *SessionManager {
	return &SessionManager{
		sessions: sync.Map{},
		observer: observer,
	}
}

func (sm *SessionManager) RegisterObserver(observer SessionObserver) {
	sm.observerMu.Lock()
	defer sm.observerMu.Unlock()

	sm.observer = observer
}

func (sm *SessionManager) getObserver() SessionObserver {
	sm.observerMu.RLock()
	defer sm.observerMu.RUnlock()

	return sm.observer
}

func (sm *SessionManager) Clean() {
	sm.sessions.Clear()
}

// count returns the number of active sessions.
func (sm *SessionManager) count() int {
	count := 0
	sm.sessions.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// getSession retrieves a session by its session ID.
func (sm *SessionManager) getSession(sessionId SessionID) *Session {
	if val, ok := sm.sessions.Load(sessionId); ok {
		return val.(*Session)
	}
	return nil
}

func (sm *SessionManager) snapshot() []*Session {
	sessions := make([]*Session, 0)
	sm.sessions.Range(func(_, val any) bool {
		sessions = append(sessions, val.(*Session))
		return true
	})
	return sessions
}

// rangeSessions iterates over all sessions and applies the provided function to each session.
func (sm *SessionManager) rangeSessions(fn func(SessionID, *Session) bool) {
	for _, session := range sm.snapshot() {
		if !fn(session.SessionID(), session) {
			return
		}
	}
}

func (sm *SessionManager) closeAllAndWait() {
	for _, session := range sm.snapshot() {
		session.Close()
	}
	for _, session := range sm.snapshot() {
		session.Wait()
	}
}

// addSession adds a new session to the manager.
func (sm *SessionManager) addSession(session *Session) {
	if session == nil {
		return
	}

	if _, loaded := sm.sessions.LoadOrStore(session.SessionID(), session); loaded {
		return
	}

	if observer := sm.getObserver(); observer != nil {
		observer.OnSessionAdded(session)
	}
}

// removeSession removes a session from the manager.
func (sm *SessionManager) removeSession(session *Session) {
	if session == nil {
		return
	}

	if _, loaded := sm.sessions.LoadAndDelete(session.SessionID()); !loaded {
		return
	}

	if observer := sm.getObserver(); observer != nil {
		observer.OnSessionRemoved(session)
	}
}
