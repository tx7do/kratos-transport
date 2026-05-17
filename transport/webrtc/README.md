# WebRTC SFU Server

## 一、WebRTC 是什么

**WebRTC（Web Real‑Time Communication）** 是一套**开源、免费**的实时通信技术栈，让浏览器与设备之间**无需插件**即可直接进行低延迟音视频通话与数据传输**。

- 发起：2011 年 Google 开源
- 标准化：2021 年 1 月成为 W3C/IETF 正式标准
- 核心价值：**浏览器原生 P2P 实时通信**，延迟通常 <500ms

## 二、核心能力

### 1. MediaStream（媒体流）

- 访问摄像头、麦克风、屏幕共享
- 输出：音频 / 视频轨道流

### 2. RTCPeerConnection（P2P 连接）

- 负责两端媒体协商、加密传输、带宽自适应
- 基于 UDP，优先实时性

### 3. RTCDataChannel（数据通道）

- 传输任意二进制 / 文本数据（聊天、文件、游戏信令）
- 低延迟、可配置可靠 / 不可靠模式

## 三、关键技术

- 编解码
    - 音频：Opus（高音质、低延迟）
    - 视频：VP8/VP9、H.264
- 网络穿透（NAT Traversal）
    - STUN：获取公网地址，用于 P2P 直连
    - TURN：直连失败时中转媒体流
    - ICE：自动选最优路径
- 安全
    - 强制 SRTP 加密媒体流
    - 信令可搭配 TLS
- 音频处理
    - AEC（回声消除）、ANS（降噪）、AGC（自动增益）

## 四、SFU 架构升级

### ✨ 新增功能（v2.0）

本次改造将 WebRTC 服务器升级为 **SFU（Selective Forwarding Unit）架构**，支持：

#### 1. **音视频通话**
- ✅ 支持 H.264/VP8 视频编解码
- ✅ 支持 Opus 音频编解码
- ✅ RTP 媒体流转发
- ✅ 多客户端订阅/发布

#### 2. **P2P 数据通道**
- ✅ 低延迟二进制数据传输
- ✅ 消息类型路由
- ✅ 广播/单播支持

#### 3. **中大规模支持**
- ✅ SFU 路由器（高效媒体转发）
- ✅ 动态订阅/取消订阅
- ✅ 会话管理优化
- ✅ NAT 穿透（STUN/TURN）

### 架构说明

```
                    ┌──────────────┐
                    │   SFU Server │
                    │  (Kratos)    │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼────┐  ┌───▼────┐  ┌───▼────┐
         │Client A │  │Client B│  │Client C│
         └─────────┘  └────────┘  └────────┘
              │            │            │
              └────────────┴────────────┘
                   媒体流通过 SFU 转发
```

**工作流程**：
1. Client A 发布音视频流 → SFU Server
2. SFU Server 接收并转发 → Client B, Client C
3. Client B/C 订阅 Client A 的媒体流
4. 数据通道用于信令交换（订阅通知、聊天等）

### 使用示例

#### 启动 SFU 服务器

```bash
cd cmd/sfu_server
go run main.go
```

服务器将在 `:9999` 端口启动，信令端点：`http://localhost:9999/signal`

#### 启用媒体支持

```go
import "github.com/pion/webrtc/v4"

// 创建带媒体支持的 WebRTC API
settingEngine := webrtc.SettingEngine{}
mediaEngine := &webrtc.MediaEngine{}

// 注册编解码器
mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
    RTPCodecCapability: webrtc.RTPCodecCapability{
        MimeType:  webrtc.MimeTypeH264,
        ClockRate: 90000,
    },
    PayloadType: 96,
}, webrtc.RTPCodecTypeVideo)

mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
    RTPCodecCapability: webrtc.RTPCodecCapability{
        MimeType:  webrtc.MimeTypeOpus,
        ClockRate: 48000,
        Channels:  2,
    },
    PayloadType: 111,
}, webrtc.RTPCodecTypeAudio)

api := webrtc.NewAPI(
    webrtc.WithMediaEngine(mediaEngine),
    webrtc.WithSettingEngine(settingEngine),
)

// 创建服务器
server := webrtc.NewServer(
    webrtc.WithAddress(":9999"),
    webrtc.WithWebRTCAPI(api),
    webrtc.WithWebRTCConfiguration(webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {URLs: []string{"stun:stun.l.google.com:19302"}},
        },
    }),
)
```

#### 订阅媒体流

```go
// 服务器端：订阅其他客户端的媒体流
err := server.SubscribeToPublisher(subscriberID, publisherID)
if err != nil {
    log.Printf("subscribe error: %s", err)
}

// 客户端：添加本地媒体轨道
localTrack, _ := webrtc.NewTrackLocalStaticRTP(
    webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
    "audio-track",
    "client-id",
)
client.AddLocalTrack(localTrack)
```

## 五、Go 测试客户端

已新增一个 Go 版本测试客户端：`cmd/test_client/main.go`，用于和 `webrtc.Server` 做联调。

运行示例：

```powershell
Push-Location "D:\GoProject\kratos-transport\transport\webrtc"
go run ./cmd/test_client -signal "http://127.0.0.1:9999/signal" -type 1 -mode binary -payload '{"type":1,"sender":"go","message":"hello"}'
Pop-Location
```

常用参数：

- `-signal`：信令地址（默认 `http://127.0.0.1:9999/signal`）
- `-auth`：`Authorization` 请求头
- `-label`：DataChannel 标签（默认 `kratos`）
- `-mode`：`binary` 或 `text`
- `-type`：消息类型（`NetMessageType`）
- `-payload`：消息体（JSON 或普通字符串）

## 六、API 参考

### Server 方法

- `SubscribeToPublisher(subscriberID, publisherID)` - 订阅发布者的媒体流
- `UnsubscribeFromPublisher(subscriberID, publisherID)` - 取消订阅
- `Broadcast(messageType, message)` - 广播消息到所有客户端
- `SendMessage(sessionID, messageType, message)` - 发送消息到指定客户端

### Client 方法

- `AddLocalTrack(track)` - 添加本地媒体轨道（音频/视频）
- `RemoveLocalTrack(trackID)` - 移除本地媒体轨道
- `GetRemoteTracks()` - 获取所有远程媒体轨道
- `SendMessage(messageType, message)` - 发送消息
- `Connect()` - 连接到服务器
- `Disconnect()` - 断开连接

## 安装

```bash
go get github.com/tx7do/kratos-transport/transport/webrtc
```

## 性能指标

- **延迟**: < 100ms (局域网), < 300ms (广域网)
- **并发**: 支持 100+ 客户端（取决于服务器配置）
- **带宽**: 每个视频流约 1-5 Mbps（取决于分辨率）
- **CPU**: SFU 转发开销较低，主要消耗在编解码
