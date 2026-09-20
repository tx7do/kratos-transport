package kcp

import (
	"sync"
	"testing"

	"github.com/xtaci/kcp-go/v5"
)

func TestSessionManager(t *testing.T) {
	conn := &kcp.UDPSession{}
	session := NewSession(conn, nil)
	id := session.SessionID()

	sm := NewSessionManager(nil)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); sm.addSession(session) }()
		go func() { defer wg.Done(); sm.removeSession(session) }()
		go func() { defer wg.Done(); sm.getSession(id) }()
	}
	wg.Wait()
}
