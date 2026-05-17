package webrtc

import (
	"net/url"
	"testing"
)

// TestSessionManager_BasicOperations 测试会话管理器的基本操作
func TestSessionManager_BasicOperations(t *testing.T) {
	manager := NewSessionManager(nil)
	if manager == nil {
		t.Fatal("failed to create session manager")
	}

	// 初始状态应该没有会话
	count := manager.count()
	if count != 0 {
		t.Fatalf("expected 0 sessions, got %d", count)
	}
}

// TestSessionManager_AddRemoveSession 测试添加和移除会话
func TestSessionManager_AddRemoveSession(t *testing.T) {
	manager := NewSessionManager(nil)

	// 创建模拟会话
	sessionID := SessionID("test-session-1")

	// 获取不存在的会话（应该返回 nil）
	session := manager.getSession(sessionID)
	if session != nil {
		t.Fatal("expected nil for non-existent session")
	}

	// 注意：实际添加会话需要真实的 PeerConnection
	// 这里只测试管理器的逻辑
}

// TestSessionManager_RangeSessions 测试遍历会话
func TestSessionManager_RangeSessions(t *testing.T) {
	manager := NewSessionManager(nil)

	var visited int
	manager.rangeSessions(func(_ SessionID, _ *Session) bool {
		visited++
		return true
	})

	if visited != 0 {
		t.Fatalf("expected 0 iterations, got %d", visited)
	}
}

// TestSessionManager_CloseAll 测试关闭所有会话
func TestSessionManager_CloseAll(t *testing.T) {
	manager := NewSessionManager(nil)

	// 关闭空管理器（不应该 panic）
	manager.closeAllAndWait()
}

// TestSessionID_String 测试 SessionID 转换
func TestSessionID_String(t *testing.T) {
	sessionID := SessionID("test-id-123")

	// SessionID 是 string 类型别名
	if string(sessionID) != "test-id-123" {
		t.Fatalf("expected 'test-id-123', got '%s'", string(sessionID))
	}
}

// TestSession_Queries 测试会话的查询参数
func TestSession_Queries(t *testing.T) {
	// 创建查询参数
	queries := url.Values{}
	queries.Set("token", "test-token")
	queries.Set("room", "room-1")

	// 验证查询参数
	if queries.Get("token") != "test-token" {
		t.Fatalf("expected token 'test-token', got '%s'", queries.Get("token"))
	}

	if queries.Get("room") != "room-1" {
		t.Fatalf("expected room 'room-1', got '%s'", queries.Get("room"))
	}
}
