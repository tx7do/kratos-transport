package webtransport

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestEndToEnd_ClientServer 用回环 QUIC 连接验证：
// 1) 客户端 CONNECT 建连成功
// 2) 客户端上行消息能被服务端 handler 收到
// 3) 服务端 SendRawData 下行消息能被客户端 handler 收到
func TestEndToEnd_ClientServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e in short mode")
	}

	// 找一个可用的本地 UDP 端口
	probe, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe udp port: %v", err)
	}
	port := probe.LocalAddr().(*net.UDPAddr).Port
	_ = probe.Close()

	srv := NewServer(
		WithAddress("127.0.0.1:"+itoa(port)),
		WithPath("/webtransport"),
	)

	upstreamCh := make(chan string, 1)
	srv.RegisterMessageHandler(MessageType(1),
		func(sessionId SessionID, payload MessagePayload) error {
			if b, ok := payload.([]byte); ok {
				upstreamCh <- string(b)
			}
			return nil
		},
		nil,
	)

	go func() {
		_ = srv.Start(context.Background())
	}()
	defer func() { _ = srv.Stop(context.Background()) }()

	// 等 QUIC 服务端就绪
	time.Sleep(200 * time.Millisecond)

	cli := NewClient(
		WithEndpoint("https://127.0.0.1:"+itoa(port)+"/webtransport"),
		WithClientTimeout(3*time.Second),
	)
	defer func() { _ = cli.Disconnect() }()

	downstreamCh := make(chan string, 1)
	cli.RegisterMessageHandler(MessageType(2),
		func(payload MessagePayload) error {
			if b, ok := payload.([]byte); ok {
				downstreamCh <- string(b)
			}
			return nil
		},
		nil,
	)

	if err := cli.Connect(); err != nil {
		t.Fatalf("client connect: %v", err)
	}

	// 上行：客户端 → 服务端（按协议封装 Message{Type,Body}）
	if err := cli.SendMessage(1, "ping"); err != nil {
		t.Fatalf("send upstream: %v", err)
	}
	select {
	case got := <-upstreamCh:
		// body 经 json codec 编码，字符串带引号
		if got != `"`+"ping"+`"` {
			t.Fatalf("upstream payload mismatch: %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting upstream message")
	}

	// 下行：服务端 → 客户端（封装 Message{Type:2} 后向在线会话广播）
	down, err := srv.marshalMessage(MessageType(2), []byte("pong"))
	if err != nil {
		t.Fatalf("marshal downstream: %v", err)
	}
	if err := srv.BroadcastRawData(down); err != nil {
		t.Fatalf("broadcast downstream: %v", err)
	}
	select {
	case got := <-downstreamCh:
		// []byte body 经 json codec 编码为 base64 字符串
		if got != `"cG9uZw=="` {
			t.Fatalf("downstream payload mismatch: %q", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting downstream message")
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
