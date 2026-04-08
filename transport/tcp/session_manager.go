package tcp

import (
	"sync"
)

// SessionObserver defines the interface for observing session events such as addition and removal of sessions.
type SessionObserver interface {
	// OnSessionAdded is called when a new session is added to the SessionManager. It receives the newly added session as an argument.
	OnSessionAdded(session *Session)

	// OnSessionRemoved is called when a session is removed from the SessionManager. It receives the removed session as an argument.
	OnSessionRemoved(session *Session)
}

type SessionManager struct {
	sessions sync.Map
	observer SessionObserver
}

func NewSessionManager(observer SessionObserver) *SessionManager {
	return &SessionManager{
		sessions: sync.Map{},
		observer: observer,
	}
}

func (sm *SessionManager) RegisterObserver(observer SessionObserver) {
	sm.observer = observer
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

// addSession adds a new session to the manager and invokes the callback with the session ID and a boolean indicating whether the session was added successfully.
func (sm *SessionManager) addSession(session *Session) {
	if session == nil {
		return
	}

	sm.sessions.Store(session.SessionID(), session)

	if sm.observer != nil {
		sm.observer.OnSessionAdded(session)
	}
}

// removeSession removes a session from the manager and invokes the callback with the session ID and a boolean indicating whether the session was removed successfully.
func (sm *SessionManager) removeSession(session *Session) {
	if session == nil {
		return
	}

	sm.sessions.Delete(session.SessionID())

	if sm.observer != nil {
		sm.observer.OnSessionRemoved(session)
	}
}

// getSession retrieves a session by its session ID.
func (sm *SessionManager) getSession(sessionId SessionID) *Session {
	if val, ok := sm.sessions.Load(sessionId); ok {
		return val.(*Session)
	}
	return nil
}

// rangeSessions iterates over all sessions and applies the provided function to each session. If the function returns true, the iteration is stopped.
func (sm *SessionManager) rangeSessions(fn func(SessionID, *Session) bool) {
	var sessions []*Session
	sm.sessions.Range(func(key, val any) bool {
		sessions = append(sessions, val.(*Session))
		return true
	})

	for _, session := range sessions {
		fn(session.SessionID(), session)
	}
}
