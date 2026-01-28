package broker

import (
	"fmt"
	"sync"
	"time"
)

// Subscriber defines the subscriber interface
type Subscriber interface {
	// Options returns the subscription options
	Options() SubscribeOptions

	// Topic returns the subscribed topic
	Topic() string

	// Unsubscribe unsubscribes from the topic
	Unsubscribe(removeFromManager bool) error
}

// SubscriberMap is a map of topic strings to Subscribers
type SubscriberMap map[string]Subscriber

// SubscriberSyncMap is a thread-safe map for managing subscribers
type SubscriberSyncMap struct {
	sync.RWMutex
	m SubscriberMap
}

func NewSubscriberSyncMap() *SubscriberSyncMap {
	return &SubscriberSyncMap{
		m: make(SubscriberMap),
	}
}

// Topics returns a slice of all subscribed topics
func (sm *SubscriberSyncMap) Topics() []string {
	sm.RLock()
	topics := make([]string, 0, len(sm.m))
	for k := range sm.m {
		topics = append(topics, k)
	}
	sm.RUnlock()
	return topics
}

// Len returns the number of subscribers in the map
func (sm *SubscriberSyncMap) Len() int {
	sm.RLock()
	n := len(sm.m)
	sm.RUnlock()
	return n
}

// Get retrieves the Subscriber for a given topic
func (sm *SubscriberSyncMap) Get(topic string) Subscriber {
	sm.RLock()
	defer sm.RUnlock()

	return sm.m[topic]
}

// GetAll returns a copy of the entire SubscriberMap
func (sm *SubscriberSyncMap) GetAll() SubscriberMap {
	sm.RLock()
	cp := make(SubscriberMap, len(sm.m))
	for k, v := range sm.m {
		cp[k] = v
	}
	sm.RUnlock()
	return cp
}

// Has checks if a topic exists in the map
func (sm *SubscriberSyncMap) Has(topic string) bool {
	sm.RLock()
	_, ok := sm.m[topic]
	sm.RUnlock()
	return ok
}

// Add adds a Subscriber to the map
func (sm *SubscriberSyncMap) Add(topic string, sub Subscriber) {
	sm.Lock()
	defer sm.Unlock()

	sm.m[topic] = sub
}

// Remove removes a Subscriber from the map and unsubscribes it
func (sm *SubscriberSyncMap) Remove(topic string) error {
	sm.Lock()
	sub, ok := sm.m[topic]
	if ok {
		delete(sm.m, topic)
	}
	sm.Unlock()

	if !ok {
		return fmt.Errorf("topic[%s] not found", topic)
	}

	return sub.Unsubscribe(true)
}

// RemoveOnly removes a Subscriber from the map without unsubscribing it
func (sm *SubscriberSyncMap) RemoveOnly(topic string) bool {
	sm.Lock()
	defer sm.Unlock()

	if _, ok := sm.m[topic]; ok {
		delete(sm.m, topic)
		return true
	} else {
		return false
	}
}

// Clear clears all Subscribers from the map and unsubscribes them
func (sm *SubscriberSyncMap) Clear() {
	sm.Lock()
	subs := make([]Subscriber, 0, len(sm.m))
	for _, sub := range sm.m {
		subs = append(subs, sub)
	}
	sm.m = make(SubscriberMap)
	sm.Unlock()

	for _, sub := range subs {
		_ = sub.Unsubscribe(false)
	}
}

// ForceClear forces clearing the map without unsubscribing
func (sm *SubscriberSyncMap) ForceClear() {
	sm.Lock()
	defer sm.Unlock()

	sm.m = make(SubscriberMap)
}

// RemoveWithTimeout removes a Subscriber with a timeout for the unsubscribe operation
func (sm *SubscriberSyncMap) RemoveWithTimeout(topic string, timeout time.Duration) error {
	sm.Lock()
	sub, ok := sm.m[topic]
	if ok {
		delete(sm.m, topic)
	}
	sm.Unlock()

	if !ok {
		return fmt.Errorf("topic[%s] not found", topic)
	}

	done := make(chan error, 1)
	go func() {
		done <- sub.Unsubscribe(true)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("unsubscribe topic[%s] timeout after %s", topic, timeout)
	}
}

// ClearWithTimeout clears all Subscribers with a timeout for each unsubscribe operation
func (sm *SubscriberSyncMap) ClearWithTimeout(timeout time.Duration) []string {
	sm.Lock()
	subs := make(map[string]Subscriber, len(sm.m))
	for k, sub := range sm.m {
		subs[k] = sub
	}
	sm.m = make(SubscriberMap)
	sm.Unlock()

	var failed []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	for topic, sub := range subs {
		wg.Add(1)
		go func(t string, s Subscriber) {
			defer wg.Done()
			done := make(chan error, 1)
			go func() { done <- s.Unsubscribe(false) }()
			select {
			case _ = <-done:
			case <-time.After(timeout):
				mu.Lock()
				failed = append(failed, t)
				mu.Unlock()
			}
		}(topic, sub)
	}
	wg.Wait()
	return failed
}

// Foreach for each Subscriber in the map, applies the given function
func (sm *SubscriberSyncMap) Foreach(fnc func(topic string, sub Subscriber)) {
	sm.RLock()
	entries := make([]struct {
		topic string
		sub   Subscriber
	}, 0, len(sm.m))
	for k, v := range sm.m {
		entries = append(entries, struct {
			topic string
			sub   Subscriber
		}{topic: k, sub: v})
	}
	sm.RUnlock()

	for _, e := range entries {
		fnc(e.topic, e.sub)
	}
}
