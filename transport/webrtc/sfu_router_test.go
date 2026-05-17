package webrtc

import (
	"fmt"
	"sync"
	"testing"

	"github.com/pion/webrtc/v4"
)

// TestSFURouter_BasicOperations 测试 SFU 路由器的基本操作
func TestSFURouter_BasicOperations(t *testing.T) {
	router := NewSFURouter()
	if router == nil {
		t.Fatal("failed to create SFU router")
	}

	// 测试获取发布者列表（应该为空）
	publishers := router.GetAllPublishers()
	if len(publishers) != 0 {
		t.Fatalf("expected 0 publishers, got %d", len(publishers))
	}
}

// TestSFURouter_AddAndGetTracks 测试添加和获取轨道
func TestSFURouter_AddAndGetTracks(t *testing.T) {
	router := NewSFURouter()

	sessionID := SessionID("test-session-1")

	// 模拟创建 TrackRemote（实际场景中由 WebRTC 提供）
	// 这里我们只测试路由器的逻辑，不测试真实的 RTP 流

	// 由于无法直接创建 TrackRemote，我们测试路由器的内部状态管理
	tracks := router.GetPublisherTracks(sessionID)
	if len(tracks) != 0 {
		t.Fatalf("expected 0 tracks, got %d", len(tracks))
	}
}

// TestSFURouter_SubscribeUnsubscribe 测试订阅和取消订阅
func TestSFURouter_SubscribeUnsubscribe(t *testing.T) {
	router := NewSFURouter()

	subscriberID := SessionID("subscriber-1")
	publisherID := SessionID("publisher-1")

	// 测试订阅（没有轨道时应该返回空列表）
	tracks := router.Subscribe(subscriberID, publisherID)
	if len(tracks) != 0 {
		t.Fatalf("expected 0 tracks, got %d", len(tracks))
	}

	// 测试取消订阅
	router.Unsubscribe(subscriberID, publisherID)
	// 不应该 panic
}

// TestSFURouter_RemoveSessionTracks 测试移除会话的所有轨道
func TestSFURouter_RemoveSessionTracks(t *testing.T) {
	router := NewSFURouter()

	sessionID := SessionID("test-session")

	// 移除不存在的会话（不应该 panic）
	router.RemoveSessionTracks(sessionID)

	// 再次调用（幂等性）
	router.RemoveSessionTracks(sessionID)
}

// TestMediaTrack_BasicProperties 测试媒体轨道的基本属性
func TestMediaTrack_BasicProperties(t *testing.T) {
	// 由于需要真实的 TrackRemote，这里只测试类型定义
	// 实际功能测试需要集成测试环境

	track := &MediaTrack{
		id:          "test-track-id",
		sessionID:   SessionID("test-session"),
		subscribers: make(map[SessionID]*webrtc.TrackLocalStaticRTP),
	}

	if track.ID() != "test-track-id" {
		t.Fatalf("expected ID 'test-track-id', got '%s'", track.ID())
	}

	if track.SessionID() != SessionID("test-session") {
		t.Fatalf("expected session ID 'test-session', got '%s'", track.SessionID())
	}
}

// TestMediaTrack_SubscriberManagement 测试订阅者管理
func TestMediaTrack_SubscriberManagement(t *testing.T) {
	track := &MediaTrack{
		id:          "test-track",
		sessionID:   SessionID("owner-session"),
		subscribers: make(map[SessionID]*webrtc.TrackLocalStaticRTP),
	}

	subscriberID1 := SessionID("subscriber-1")
	subscriberID2 := SessionID("subscriber-2")

	// 添加订阅者
	track.AddSubscriber(subscriberID1, nil)
	track.AddSubscriber(subscriberID2, nil)

	subscribers := track.GetSubscribers()
	if len(subscribers) != 2 {
		t.Fatalf("expected 2 subscribers, got %d", len(subscribers))
	}

	// 移除订阅者
	track.RemoveSubscriber(subscriberID1)
	subscribers = track.GetSubscribers()
	if len(subscribers) != 1 {
		t.Fatalf("expected 1 subscriber, got %d", len(subscribers))
	}

	// 移除不存在的订阅者（不应该 panic）
	track.RemoveSubscriber(SessionID("non-existent"))
}

// TestMediaTrack_Close 测试关闭媒体轨道
func TestMediaTrack_Close(t *testing.T) {
	track := &MediaTrack{
		id:          "test-track",
		sessionID:   SessionID("owner-session"),
		subscribers: make(map[SessionID]*webrtc.TrackLocalStaticRTP),
	}

	// 添加一些订阅者
	track.AddSubscriber(SessionID("sub-1"), nil)
	track.AddSubscriber(SessionID("sub-2"), nil)

	// 关闭轨道
	track.Close()

	// 验证订阅者列表已清空
	subscribers := track.GetSubscribers()
	if len(subscribers) != 0 {
		t.Fatalf("expected 0 subscribers after close, got %d", len(subscribers))
	}

	// 再次关闭（幂等性）
	track.Close()
}

// TestMediaTrack_ConcurrentAccess 测试并发访问
func TestMediaTrack_ConcurrentAccess(t *testing.T) {
	track := &MediaTrack{
		id:          "test-track",
		sessionID:   SessionID("owner-session"),
		subscribers: make(map[SessionID]*webrtc.TrackLocalStaticRTP),
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	// 并发添加订阅者
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			subscriberID := SessionID(fmt.Sprintf("subscriber-%d", id))
			track.AddSubscriber(subscriberID, nil)
		}(i)
	}

	wg.Wait()

	subscribers := track.GetSubscribers()
	if len(subscribers) != numGoroutines {
		t.Fatalf("expected %d subscribers, got %d", numGoroutines, len(subscribers))
	}

	// 并发读取
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = track.GetSubscribers()
		}()
	}

	wg.Wait()
}

// TestSFURouter_ConcurrentAccess 测试 SFU 路由器的并发访问
func TestSFURouter_ConcurrentAccess(t *testing.T) {
	router := NewSFURouter()

	var wg sync.WaitGroup
	numGoroutines := 50

	// 并发订阅
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			subscriberID := SessionID(fmt.Sprintf("subscriber-%d", id))
			publisherID := SessionID(fmt.Sprintf("publisher-%d", id%5)) // 5 个发布者
			router.Subscribe(subscriberID, publisherID)
		}(i)
	}

	wg.Wait()

	// 并发取消订阅
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			subscriberID := SessionID(fmt.Sprintf("subscriber-%d", id))
			publisherID := SessionID(fmt.Sprintf("publisher-%d", id%5))
			router.Unsubscribe(subscriberID, publisherID)
		}(i)
	}

	wg.Wait()
}
