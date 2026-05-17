package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	pionwebrtc "github.com/pion/webrtc/v4"
	"github.com/tx7do/kratos-transport/transport/webrtc"
)

// 示例：SFU 架构的音视频会议服务器
func main() {
	fmt.Println("=== WebRTC SFU Server Example ===")

	// 配置 STUN/TURN 服务器（用于 NAT 穿透）
	webrtcConfig := pionwebrtc.Configuration{
		ICEServers: []pionwebrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
			// 如果有 TURN 服务器，可以添加：
			// {
			// 	URLs:       []string{"turn:your-turn-server.com:3478"},
			// 	Username:   "user",
			// 	Credential: "pass",
			// },
		},
	}

	// 创建支持媒体的 WebRTC API
	settingEngine := pionwebrtc.SettingEngine{}
	settingEngine.SetICETimeouts(5*time.Second, 5*time.Second, 2*time.Second)

	mediaEngine := &pionwebrtc.MediaEngine{}

	// 注册视频编解码器
	if err := mediaEngine.RegisterCodec(pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{
			MimeType:     pionwebrtc.MimeTypeH264,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
			RTCPFeedback: nil,
		},
		PayloadType: 96,
	}, pionwebrtc.RTPCodecTypeVideo); err != nil {
		log.Printf("register H264 codec error: %s", err)
	}

	if err := mediaEngine.RegisterCodec(pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{
			MimeType:     pionwebrtc.MimeTypeVP8,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 96,
	}, pionwebrtc.RTPCodecTypeVideo); err != nil {
		log.Printf("register VP8 codec error: %s", err)
	}

	// 注册音频编解码器
	if err := mediaEngine.RegisterCodec(pionwebrtc.RTPCodecParameters{
		RTPCodecCapability: pionwebrtc.RTPCodecCapability{
			MimeType:     pionwebrtc.MimeTypeOpus,
			ClockRate:    48000,
			Channels:     2,
			SDPFmtpLine:  "minptime=10;useinbandfec=1",
			RTCPFeedback: nil,
		},
		PayloadType: 111,
	}, pionwebrtc.RTPCodecTypeAudio); err != nil {
		log.Printf("register Opus codec error: %s", err)
	}

	api := pionwebrtc.NewAPI(
		pionwebrtc.WithMediaEngine(mediaEngine),
		pionwebrtc.WithSettingEngine(settingEngine),
	)

	// 创建 WebRTC 服务器
	server := webrtc.NewServer(
		webrtc.WithAddress(":9999"),
		webrtc.WithPath("/signal"),
		webrtc.WithWebRTCAPI(api),
		webrtc.WithWebRTCConfiguration(webrtcConfig),
		webrtc.WithDataChannelLabel("kratos"),
		webrtc.WithAllowAnyDataChannelLabel(true),
		webrtc.WithSocketConnectHandler(func(sessionId webrtc.SessionID, queries url.Values, connect bool) {
			if connect {
				log.Printf("Client connected: %s", sessionId)
			} else {
				log.Printf("Client disconnected: %s", sessionId)
			}
		}),
	)

	if server == nil {
		log.Fatal("failed to create server")
	}

	// 注册消息处理器（用于数据通道信令）
	webrtc.RegisterServerMessageHandler(server, 1, func(sessionId webrtc.SessionID, msg *TestMessage) error {
		log.Printf("Received message from %s: %+v", sessionId, msg)

		// 广播消息给所有客户端
		server.Broadcast(1, msg)

		return nil
	})

	// 启动服务器
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatalf("server error: %s", err)
		}
	}()

	log.Println("WebRTC SFU server started on :9999")
	log.Println("Signal endpoint: http://localhost:9999/signal")
	log.Println("Press Ctrl+C to stop")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Stop(shutdownCtx); err != nil {
		log.Printf("server stop error: %s", err)
	}

	log.Println("Server stopped")
}

// TestMessage 测试消息结构
type TestMessage struct {
	Type      int    `json:"type"`
	Sender    string `json:"sender"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}
