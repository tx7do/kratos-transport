package kcp

import (
	"testing"

	"github.com/xtaci/kcp-go/v5"
)

func TestSessionManager(t *testing.T) {
	conn := &kcp.UDPSession{}
	session := NewSession(conn, nil)
	id := session.SessionID()

	sm := NewSessionManager()

	for i := 0; i < 100; i++ {
		go func() { sm.addSession(session, nil) }()
		go func() { sm.removeSession(session, nil) }()
		go func() { sm.getSession(id) }()
	}
}
