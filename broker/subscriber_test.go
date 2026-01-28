package broker

import (
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
)

// mockSubscriber 用于测试
type mockSubscriber struct {
	name  string
	delay time.Duration
	err   error

	mu     sync.Mutex
	called bool
}

func (m *mockSubscriber) Options() SubscribeOptions { return SubscribeOptions{} }
func (m *mockSubscriber) Topic() string             { return m.name }
func (m *mockSubscriber) Unsubscribe(removeFromManager bool) error {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	m.mu.Lock()
	m.called = true
	m.mu.Unlock()
	return m.err
}
func (m *mockSubscriber) Called() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.called
}

func sliceEqualIgnoreOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}

func TestSubscriberSyncMap_BasicOperations(t *testing.T) {
	sm := NewSubscriberSyncMap()

	// Add
	s1 := &mockSubscriber{name: "a"}
	s2 := &mockSubscriber{name: "b"}
	sm.Add("a", s1)
	sm.Add("b", s2)

	// Len / Has / Get
	if l := sm.Len(); l != 2 {
		t.Fatalf("Len() = %d, want 2", l)
	}
	if !sm.Has("a") || !sm.Has("b") {
		t.Fatalf("Has() failed")
	}
	if got := sm.Get("a"); got == nil {
		t.Fatalf("Get(a) returned nil")
	}

	// Topics (order not guaranteed)
	topics := sm.Topics()
	if !sliceEqualIgnoreOrder(topics, []string{"a", "b"}) {
		t.Fatalf("Topics() = %v, want [a b] (order ignored)", topics)
	}

	// GetAll returns a copy
	cp := sm.GetAll()
	if _, ok := cp["a"]; !ok {
		t.Fatalf("GetAll missing key a")
	}
	delete(cp, "a")
	// original must remain
	if !sm.Has("a") {
		t.Fatalf("modifying GetAll copy affected original map")
	}

	// RemoveOnly
	ok := sm.RemoveOnly("a")
	if !ok {
		t.Fatalf("RemoveOnly(a) should return true")
	}
	if sm.Has("a") {
		t.Fatalf("a should be removed")
	}
	if l := sm.Len(); l != 1 {
		t.Fatalf("Len() after RemoveOnly = %d, want 1", l)
	}

	// Remove not found
	if err := sm.Remove("nonexistent"); err == nil {
		t.Fatalf("Remove(nonexistent) should return error")
	}

	// Remove with unsubscribe called
	sm.Add("r1", &mockSubscriber{name: "r1"})
	if err := sm.Remove("r1"); err != nil {
		t.Fatalf("Remove(r1) unexpected error: %v", err)
	}
	if sm.Has("r1") {
		t.Fatalf("r1 should be removed")
	}
}

func TestSubscriberSyncMap_RemoveWithTimeout(t *testing.T) {
	sm := NewSubscriberSyncMap()

	// fast unsub
	fast := &mockSubscriber{name: "fast", delay: 5 * time.Millisecond}
	sm.Add("fast", fast)
	if err := sm.RemoveWithTimeout("fast", 50*time.Millisecond); err != nil {
		t.Fatalf("RemoveWithTimeout(fast) unexpected error: %v", err)
	}
	if !fast.Called() {
		t.Fatalf("fast unsub not called")
	}

	// slow unsub -> timeout
	slow := &mockSubscriber{name: "slow", delay: 100 * time.Millisecond}
	sm.Add("slow", slow)
	err := sm.RemoveWithTimeout("slow", 20*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error for slow unsub")
	}
	contains := err != nil && (stringContains(err.Error(), "timeout") || stringContains(err.Error(), "timeout after"))
	if !contains {
		t.Fatalf("unexpected error message for timeout: %v", err)
	}
}

func TestSubscriberSyncMap_ClearAndClearWithTimeout_ForceClear(t *testing.T) {
	sm := NewSubscriberSyncMap()

	a := &mockSubscriber{name: "a"}
	b := &mockSubscriber{name: "b", delay: 5 * time.Millisecond}
	c := &mockSubscriber{name: "c", delay: 100 * time.Millisecond}
	sm.Add("a", a)
	sm.Add("b", b)
	sm.Add("c", c)

	// Clear should call Unsubscribe for all
	sm.Clear()
	if sm.Len() != 0 {
		t.Fatalf("Clear did not empty map")
	}
	if !a.Called() || !b.Called() || !c.Called() {
		t.Fatalf("Clear did not call Unsubscribe for all subscribers")
	}

	// ClearWithTimeout: some slow, should be reported as failed
	sm.Add("x", &mockSubscriber{name: "x", delay: 5 * time.Millisecond})
	sm.Add("y", &mockSubscriber{name: "y", delay: 100 * time.Millisecond})
	sm.Add("z", &mockSubscriber{name: "z", delay: 200 * time.Millisecond})

	failed := sm.ClearWithTimeout(50 * time.Millisecond)
	// y and z expected to timeout (x should succeed)
	if !sliceEqualIgnoreOrder(failed, []string{"y", "z"}) {
		t.Fatalf("ClearWithTimeout failed = %v, want [y z] (order ignored)", failed)
	}

	// ForceClear should clear without calling Unsubscribe
	sa := &mockSubscriber{name: "sa"}
	sb := &mockSubscriber{name: "sb"}
	sm.Add("sa", sa)
	sm.Add("sb", sb)
	sm.ForceClear()
	if sm.Len() != 0 {
		t.Fatalf("ForceClear did not empty map")
	}
	if sa.Called() || sb.Called() {
		t.Fatalf("ForceClear should not call Unsubscribe")
	}
}

func TestSubscriberSyncMap_Foreach(t *testing.T) {
	sm := NewSubscriberSyncMap()
	sm.Add("t1", &mockSubscriber{name: "t1"})
	sm.Add("t2", &mockSubscriber{name: "t2"})

	var got []string
	sm.Foreach(func(topic string, sub Subscriber) {
		got = append(got, topic)
	})
	if !sliceEqualIgnoreOrder(got, []string{"t1", "t2"}) {
		t.Fatalf("Foreach topics = %v, want [t1 t2] (order ignored)", got)
	}
}

// small helper: check substring (avoid importing strings repeatedly)
func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (reflect.ValueOf(s).String() == reflect.ValueOf(sub).String() || (len(s) > 0 && len(sub) > 0 && (indexOf(s, sub) >= 0)))
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
