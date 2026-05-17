package webrtc

import (
	"errors"
	"io"
	"sync"

	"github.com/pion/webrtc/v4"
)

// MediaTrack 表示一个媒体轨道（音频或视频）
type MediaTrack struct {
	id        string
	sessionID SessionID
	track     *webrtc.TrackRemote
	receiver  *webrtc.RTPReceiver
	kind      webrtc.RTPCodecType
	codec     webrtc.RTPCodecParameters

	mu          sync.RWMutex
	subscribers map[SessionID]*webrtc.TrackLocalStaticRTP
	closed      bool
}

// NewMediaTrack 创建新的媒体轨道
func NewMediaTrack(sessionID SessionID, track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) *MediaTrack {
	return &MediaTrack{
		id:          track.ID(),
		sessionID:   sessionID,
		track:       track,
		receiver:    receiver,
		kind:        track.Kind(),
		codec:       track.Codec(),
		subscribers: make(map[SessionID]*webrtc.TrackLocalStaticRTP),
	}
}

// ID 返回轨道 ID
func (m *MediaTrack) ID() string {
	return m.id
}

// SessionID 返回所有者会话 ID
func (m *MediaTrack) SessionID() SessionID {
	return m.sessionID
}

// Kind 返回媒体类型（音频/视频）
func (m *MediaTrack) Kind() webrtc.RTPCodecType {
	return m.kind
}

// Codec 返回编解码器信息
func (m *MediaTrack) Codec() webrtc.RTPCodecParameters {
	return m.codec
}

// AddSubscriber 添加订阅者
func (m *MediaTrack) AddSubscriber(subscriberID SessionID, localTrack *webrtc.TrackLocalStaticRTP) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	m.subscribers[subscriberID] = localTrack
}

// RemoveSubscriber 移除订阅者
func (m *MediaTrack) RemoveSubscriber(subscriberID SessionID) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.subscribers, subscriberID)
}

// GetSubscribers 获取所有订阅者
func (m *MediaTrack) GetSubscribers() map[SessionID]*webrtc.TrackLocalStaticRTP {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[SessionID]*webrtc.TrackLocalStaticRTP)
	for k, v := range m.subscribers {
		result[k] = v
	}
	return result
}

// Start 开始转发媒体流
func (m *MediaTrack) Start() {
	go m.forwardRTP()
}

// forwardRTP 转发 RTP 包到所有订阅者
func (m *MediaTrack) forwardRTP() {
	buf := make([]byte, 1500) // MTU size

	for {
		n, _, err := m.track.Read(buf)
		if err != nil {
			if err == io.EOF || errors.Is(err, io.ErrClosedPipe) {
				LogDebugf("track %s closed", m.id)
			} else {
				LogErrorf("read RTP error: %s", err)
			}
			m.Close()
			return
		}

		packet := buf[:n]

		m.mu.RLock()
		if m.closed {
			m.mu.RUnlock()
			return
		}

		// 发送给所有订阅者
		for _, localTrack := range m.subscribers {
			if _, writeErr := localTrack.Write(packet); writeErr != nil {
				LogErrorf("write to subscriber error: %s", writeErr)
			}
		}
		m.mu.RUnlock()
	}
}

// Close 关闭媒体轨道
func (m *MediaTrack) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	m.closed = true

	// TrackLocalStaticRTP 不需要手动关闭
	// 当 PeerConnection 关闭时会自动清理
	// 这里只需要清空订阅者列表
	m.subscribers = nil
}
