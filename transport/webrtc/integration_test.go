package webrtc

import (
	"testing"
)

// TestSFUIntegration_BasicFlow 测试 SFU 基本工作流程
func TestSFUIntegration_BasicFlow(t *testing.T) {
	// 1. 创建服务器
	server := NewServer(
		WithAddress("127.0.0.1:0"),
		WithPath("/signal"),
	)

	if server == nil {
		t.Fatal("failed to create server")
	}

	// 2. 验证 SFU 路由器已初始化
	if server.sfuRouter == nil {
		t.Fatal("SFU router should be initialized")
	}

	// 3. 注册消息处理器
	RegisterServerMessageHandler(server, messageTypeChat, func(_ SessionID, msg *chatMessage) error {
		t.Logf("Received message: %+v", msg)
		return nil
	})

	// 4. 验证处理器已注册
	handler := server.GetMessageHandler(messageTypeChat)
	if handler == nil {
		t.Fatal("message handler should be registered")
	}

	// 5. 注销处理器
	server.DeregisterMessageHandler(messageTypeChat)
	handler = server.GetMessageHandler(messageTypeChat)
	if handler != nil {
		t.Fatal("message handler should be deregistered")
	}
}

// TestSFUIntegration_ServerOptions 测试服务器选项配置
func TestSFUIntegration_ServerOptions(t *testing.T) {
	// 测试各种选项
	server := NewServer(
		WithAddress("127.0.0.1:9999"),
		WithPath("/custom-signal"),
		WithCodec("json"),
		WithPayloadType(PayloadTypeBinary),
		WithDataChannelLabel("kratos-data"),
		WithAllowAnyDataChannelLabel(true),
		WithEnableCORS(true),
		WithCORS("*", "GET, POST, OPTIONS", "Content-Type, Authorization"),
		WithStrictSlash(true),
		WithInjectTokenToQuery(true, "token"),
	)

	if server == nil {
		t.Fatal("failed to create server with options")
	}

	// 验证配置
	if server.path != "/custom-signal" {
		t.Fatalf("expected path '/custom-signal', got '%s'", server.path)
	}

	if server.payloadType != PayloadTypeBinary {
		t.Fatalf("expected PayloadTypeBinary, got %d", server.payloadType)
	}

	if server.dataChannelLabel != "kratos-data" {
		t.Fatalf("expected label 'kratos-data', got '%s'", server.dataChannelLabel)
	}

	if !server.allowAnyDataLabel {
		t.Fatal("expected allowAnyDataLabel to be true")
	}
}

// TestSFUIntegration_ClientOptions 测试客户端选项配置
func TestSFUIntegration_ClientOptions(t *testing.T) {
	client := NewClient(
		WithSignalURL("http://localhost:9999/signal"),
		WithAuthorization("Bearer test-token"),
		WithClientCodec("json"),
		WithClientPayloadType(PayloadTypeText),
		WithClientDataChannelLabel("client-data"),
		WithClientConnectTimeout(5000000000), // 5 seconds
		WithClientSignalTimeout(3000000000),  // 3 seconds
	)

	if client == nil {
		t.Fatal("failed to create client with options")
	}

	// 验证配置
	if client.signalURL != "http://localhost:9999/signal" {
		t.Fatalf("unexpected signal URL: %s", client.signalURL)
	}

	if client.authorization != "Bearer test-token" {
		t.Fatalf("unexpected authorization: %s", client.authorization)
	}

	if client.payloadType != PayloadTypeText {
		t.Fatalf("expected PayloadTypeText, got %d", client.payloadType)
	}
}

// TestSFUIntegration_MessageMarshalUnmarshal 测试消息编解码流程
func TestSFUIntegration_MessageMarshalUnmarshal(t *testing.T) {
	server := NewServer(
		WithAddress("127.0.0.1:0"),
		WithCodec("json"),
	)

	if server == nil {
		t.Fatal("failed to create server")
	}

	// 测试二进制包序列化
	testMsg := chatMessage{
		Type:    1,
		Sender:  "test-user",
		Message: "Hello, SFU!",
	}

	// 使用默认的二进制打包器
	buf, err := server.defaultMarshalNetPacket(messageTypeChat, &testMsg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if len(buf) == 0 {
		t.Fatal("marshaled buffer should not be empty")
	}

	// 反序列化
	handler, payload, err := server.defaultUnmarshalNetPacket(buf)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if handler == nil {
		t.Fatal("handler should not be nil")
	}

	if payload == nil {
		t.Fatal("payload should not be nil")
	}
}

// TestSFUIntegration_ConcurrentOperations 测试并发操作
func TestSFUIntegration_ConcurrentOperations(t *testing.T) {
	server := NewServer(
		WithAddress("127.0.0.1:0"),
	)

	if server == nil {
		t.Fatal("failed to create server")
	}

	// 并发注册消息处理器
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(msgType NetMessageType) {
			RegisterServerMessageHandler(server, msgType, func(_ SessionID, msg *chatMessage) error {
				return nil
			})
			done <- true
		}(NetMessageType(i + 100))
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	// 验证所有处理器都已注册
	for i := 0; i < 10; i++ {
		handler := server.GetMessageHandler(NetMessageType(i + 100))
		if handler == nil {
			t.Fatalf("handler %d should be registered", i+100)
		}
	}
}

// TestSFUIntegration_ErrorHandling 测试错误处理
func TestSFUIntegration_ErrorHandling(t *testing.T) {
	// 测试发送消息到不存在的会话
	server := NewServer(
		WithAddress("127.0.0.1:0"),
	)

	err := server.SendMessage("non-existent-session", 1, &chatMessage{})
	if err == nil {
		t.Fatal("expected error when sending to non-existent session")
	}

	// 测试广播（不应该 panic，即使没有会话）
	server.Broadcast(1, &chatMessage{})
	// 如果没有 panic，说明测试通过
}
