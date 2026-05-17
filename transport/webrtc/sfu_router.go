package webrtc

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

// SFURouter SFU 路由器，负责媒体流的转发
type SFURouter struct {
	mu            sync.RWMutex
	tracks        map[SessionID]map[string]*MediaTrack // sessionID -> trackID -> MediaTrack
	subscriptions map[SessionID][]string               // subscriberID -> []trackID
}

// NewSFURouter 创建新的 SFU 路由器
func NewSFURouter() *SFURouter {
	return &SFURouter{
		tracks:        make(map[SessionID]map[string]*MediaTrack),
		subscriptions: make(map[SessionID][]string),
	}
}

// AddTrack 添加媒体轨道
func (r *SFURouter) AddTrack(sessionID SessionID, track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) *MediaTrack {
	r.mu.Lock()
	defer r.mu.Unlock()

	mediaTrack := NewMediaTrack(sessionID, track, receiver)

	if _, ok := r.tracks[sessionID]; !ok {
		r.tracks[sessionID] = make(map[string]*MediaTrack)
	}

	r.tracks[sessionID][track.ID()] = mediaTrack

	// 自动启动转发
	mediaTrack.Start()

	LogInfof("added track: session=%s, trackID=%s, kind=%s", sessionID, track.ID(), track.Kind())

	return mediaTrack
}

// RemoveTrack 移除媒体轨道
func (r *SFURouter) RemoveTrack(sessionID SessionID, trackID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tracks, ok := r.tracks[sessionID]; ok {
		if track, exists := tracks[trackID]; exists {
			track.Close()
			delete(tracks, trackID)

			// 清理所有订阅关系
			for subscriberID, subscribedTracks := range r.subscriptions {
				var newSubscribedTracks []string
				for _, tid := range subscribedTracks {
					if tid != trackID {
						newSubscribedTracks = append(newSubscribedTracks, tid)
					}
				}
				if len(newSubscribedTracks) == 0 {
					delete(r.subscriptions, subscriberID)
				} else {
					r.subscriptions[subscriberID] = newSubscribedTracks
				}
			}

			LogInfof("removed track: session=%s, trackID=%s", sessionID, trackID)
		}

		if len(tracks) == 0 {
			delete(r.tracks, sessionID)
		}
	}
}

// RemoveSessionTracks 移除会话的所有轨道
func (r *SFURouter) RemoveSessionTracks(sessionID SessionID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tracks, ok := r.tracks[sessionID]; ok {
		for trackID, track := range tracks {
			track.Close()
			LogInfof("closed track: session=%s, trackID=%s", sessionID, trackID)
		}
		delete(r.tracks, sessionID)
	}

	// 清理该会话的订阅
	delete(r.subscriptions, sessionID)

	LogInfof("removed all tracks for session: %s", sessionID)
}

// Subscribe 订阅指定会话的媒体轨道
func (r *SFURouter) Subscribe(subscriberID SessionID, publisherID SessionID) []*MediaTrack {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var subscribedTracks []*MediaTrack

	if tracks, ok := r.tracks[publisherID]; ok {
		for trackID, track := range tracks {
			subscribedTracks = append(subscribedTracks, track)

			// 记录订阅关系
			if _, ok := r.subscriptions[subscriberID]; !ok {
				r.subscriptions[subscriberID] = make([]string, 0)
			}
			r.subscriptions[subscriberID] = append(r.subscriptions[subscriberID], trackID)
		}
	}

	LogInfof("session %s subscribed to %d tracks from %s", subscriberID, len(subscribedTracks), publisherID)

	return subscribedTracks
}

// Unsubscribe 取消订阅
func (r *SFURouter) Unsubscribe(subscriberID SessionID, publisherID SessionID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if trackIDs, ok := r.subscriptions[subscriberID]; ok {
		var remaining []string
		for _, trackID := range trackIDs {
			// 检查这个轨道是否属于指定的发布者
			if tracks, exists := r.tracks[publisherID]; exists {
				if _, hasTrack := tracks[trackID]; hasTrack {
					// 从轨道中移除订阅者
					if track, ok := tracks[trackID]; ok {
						track.RemoveSubscriber(subscriberID)
					}
					continue // 跳过，不加入 remaining
				}
			}
			remaining = append(remaining, trackID)
		}

		if len(remaining) == 0 {
			delete(r.subscriptions, subscriberID)
		} else {
			r.subscriptions[subscriberID] = remaining
		}
	}

	LogInfof("session %s unsubscribed from %s", subscriberID, publisherID)
}

// GetPublisherTracks 获取发布者的所有轨道
func (r *SFURouter) GetPublisherTracks(publisherID SessionID) []*MediaTrack {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tracks []*MediaTrack

	if publisherTracks, ok := r.tracks[publisherID]; ok {
		for _, track := range publisherTracks {
			tracks = append(tracks, track)
		}
	}

	return tracks
}

// GetAllPublishers 获取所有发布者会话 ID
func (r *SFURouter) GetAllPublishers() []SessionID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var publishers []SessionID
	for sessionID := range r.tracks {
		publishers = append(publishers, sessionID)
	}

	return publishers
}

// CreateLocalTrackForSubscriber 为订阅者创建本地轨道
func (r *SFURouter) CreateLocalTrackForSubscriber(subscriberID SessionID, mediaTrack *MediaTrack, pc *webrtc.PeerConnection) (*webrtc.TrackLocalStaticRTP, error) {
	localTrack, err := webrtc.NewTrackLocalStaticRTP(
		mediaTrack.Codec().RTPCodecCapability,
		mediaTrack.ID(),
		string(mediaTrack.SessionID()),
	)
	if err != nil {
		return nil, err
	}

	// 添加到 PeerConnection
	if _, err = pc.AddTrack(localTrack); err != nil {
		return nil, err
	}

	// 注册到媒体轨道
	mediaTrack.AddSubscriber(subscriberID, localTrack)

	LogInfof("created local track for subscriber %s: trackID=%s", subscriberID, mediaTrack.ID())

	return localTrack, nil
}
