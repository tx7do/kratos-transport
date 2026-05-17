package webrtc

import (
	"testing"
)

// TestClient_MediaTrackManagement 测试客户端的媒体轨道管理
func TestClient_MediaTrackManagement(t *testing.T) {
	client := NewClient(
		WithSignalURL("http://localhost:9999/signal"),
	)

	if client == nil {
		t.Fatal("failed to create client")
	}

	// 初始状态应该没有轨道
	remoteTracks := client.GetRemoteTracks()
	if len(remoteTracks) != 0 {
		t.Fatalf("expected 0 remote tracks, got %d", len(remoteTracks))
	}

	// 测试移除不存在的轨道（不应该 panic）
	client.RemoveLocalTrack("non-existent-track")
}

// TestClient_MessageHandlerRegistration 测试消息处理器注册
func TestClient_MessageHandlerRegistration(t *testing.T) {
	client := NewClient()

	var handlerCalled bool

	RegisterClientMessageHandler(client, 1, func(msg *chatMessage) error {
		handlerCalled = true
		return nil
	})

	// 重复注册同一个类型（应该被忽略）
	RegisterClientMessageHandler(client, 1, func(msg *chatMessage) error {
		t.Fatal("duplicate handler should be ignored")
		return nil
	})

	// 注销处理器
	client.DeregisterMessageHandler(1)

	// 验证不会 panic
	if handlerCalled {
		t.Fatal("handler should not be called yet")
	}
}

// TestClient_PayloadType 测试负载类型配置
func TestClient_PayloadType(t *testing.T) {
	client := NewClient(
		WithClientPayloadType(PayloadTypeBinary),
	)

	if client.payloadType != PayloadTypeBinary {
		t.Fatalf("expected PayloadTypeBinary, got %d", client.payloadType)
	}

	client2 := NewClient(
		WithClientPayloadType(PayloadTypeText),
	)

	if client2.payloadType != PayloadTypeText {
		t.Fatalf("expected PayloadTypeText, got %d", client2.payloadType)
	}
}

// TestClient_DataChannelLabel 测试数据通道标签配置
func TestClient_DataChannelLabel(t *testing.T) {
	client := NewClient(
		WithClientDataChannelLabel("custom-label"),
	)

	if client.dataChannelLabel != "custom-label" {
		t.Fatalf("expected 'custom-label', got '%s'", client.dataChannelLabel)
	}
}
