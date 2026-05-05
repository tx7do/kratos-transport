# WebRTC Server

## 一、WebRTC 是什么

**WebRTC（Web Real‑Time Communication）** 是一套**开源、免费**的实时通信技术栈，让浏览器与设备之间**无需插件**即可直接进行*
*低延迟音视频通话与数据传输**。

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

## 四、Go 测试客户端

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

## 安装

```bash
go get github.com/tx7do/kratos-transport/transport/webrtc
```
