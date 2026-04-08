package tcp

import (
	"sync"
)

type SessionManager struct {
	sessions sync.Map
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: sync.Map{},
	}
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
func (sm *SessionManager) addSession(c *Session, callback func(SessionID, bool)) {
	if c == nil {
		return
	}
	sm.sessions.Store(c.SessionID(), c)
	if callback != nil {
		callback(c.SessionID(), true)
	}
}

// removeSession removes a session from the manager and invokes the callback with the session ID and a boolean indicating whether the session was removed successfully.
func (sm *SessionManager) removeSession(c *Session, callback func(SessionID, bool)) {
	if c == nil {
		return
	}
	sm.sessions.Delete(c.SessionID())
	if callback != nil {
		callback(c.SessionID(), true)
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
