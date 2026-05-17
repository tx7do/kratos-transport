package webrtc

import (
	"testing"
)

// TestTextNetPacket_MarshalUnmarshal 测试文本包的编解码
func TestTextNetPacket_MarshalUnmarshal(t *testing.T) {
	msg := TextNetPacket{
		Type:    100,
		Payload: "Hello, WebRTC!",
	}

	// 序列化
	buf, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// 反序列化
	var got TextNetPacket
	if err = got.Unmarshal(buf); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	// 验证
	if got.Type != msg.Type {
		t.Fatalf("unexpected type: got %d want %d", got.Type, msg.Type)
	}
	if got.Payload != msg.Payload {
		t.Fatalf("unexpected payload: got %q want %q", got.Payload, msg.Payload)
	}
}

// TestBinaryNetPacket_EmptyPayload 测试空负载的二进制包
func TestBinaryNetPacket_EmptyPayload(t *testing.T) {
	msg := BinaryNetPacket{
		Type:    200,
		Payload: []byte{},
	}

	buf, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got BinaryNetPacket
	if err = got.Unmarshal(buf); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.Type != msg.Type {
		t.Fatalf("unexpected type: got %d want %d", got.Type, msg.Type)
	}
	if len(got.Payload) != 0 {
		t.Fatalf("expected empty payload, got %d bytes", len(got.Payload))
	}
}

// TestBinaryNetPacket_LargePayload 测试大负载的二进制包
func TestBinaryNetPacket_LargePayload(t *testing.T) {
	largeData := make([]byte, 10000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	msg := BinaryNetPacket{
		Type:    300,
		Payload: largeData,
	}

	buf, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got BinaryNetPacket
	if err = got.Unmarshal(buf); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if got.Type != msg.Type {
		t.Fatalf("unexpected type: got %d want %d", got.Type, msg.Type)
	}
	if len(got.Payload) != len(msg.Payload) {
		t.Fatalf("unexpected payload length: got %d want %d", len(got.Payload), len(msg.Payload))
	}
	for i := range got.Payload {
		if got.Payload[i] != msg.Payload[i] {
			t.Fatalf("payload mismatch at index %d", i)
		}
	}
}

// TestMessageHandlerData_Create 测试消息处理器数据创建
func TestMessageHandlerData_Create(t *testing.T) {
	// 带 Creator 的处理器
	handler := &MessageHandlerData{
		Creator: func() any {
			return &chatMessage{}
		},
	}

	created := handler.Create()
	if created == nil {
		t.Fatal("expected non-nil created object")
	}

	// 不带 Creator 的处理器
	handler2 := &MessageHandlerData{
		Creator: nil,
	}

	created2 := handler2.Create()
	if created2 != nil {
		t.Fatal("expected nil when Creator is nil")
	}
}

// TestNetMessageHandlerMap 测试消息处理器映射
func TestNetMessageHandlerMap(t *testing.T) {
	handlers := make(NetMessageHandlerMap)

	// 添加处理器
	handlers[1] = &MessageHandlerData{
		Handler: func(_ SessionID, _ MessagePayload) error {
			return nil
		},
		Creator: func() any {
			return &chatMessage{}
		},
	}

	// 检查是否存在
	if _, ok := handlers[1]; !ok {
		t.Fatal("expected handler to exist")
	}

	// 删除处理器
	delete(handlers, 1)

	if _, ok := handlers[1]; ok {
		t.Fatal("expected handler to be deleted")
	}
}
