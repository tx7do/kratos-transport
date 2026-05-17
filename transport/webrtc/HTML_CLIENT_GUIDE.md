# WebRTC SFU HTML 测试客户端使用指南

## 📖 概述

`test_client.html` 是一个功能完整的浏览器端 WebRTC SFU 测试客户端，支持：

- ✅ **音视频通话** - 实时音频和视频传输
- ✅ **数据通道** - 低延迟二进制/文本消息
- ✅ **设备选择** - 选择摄像头和麦克风
- ✅ **多路视频** - 同时显示多个远程视频流
- ✅ **SFU 架构** - 通过服务器转发媒体流

---

## 🚀 快速开始

### 1. 启动 SFU 服务器

```bash
cd kratos-transport\transport\webrtc\cmd\sfu_server
go run main.go
```

服务器将在 `http://localhost:9999/signal` 启动。

### 2. 打开测试页面

在浏览器中打开：
```
file:///D:/GoProject/kratos-transport/transport/webrtc/test_client.html
```

或者使用本地服务器：
```bash
# 使用 Python
python -m http.server 8080

# 使用 Node.js
npx http-server .

# 然后访问
http://localhost:8080/test_client.html
```

### 3. 连接测试

1. **配置连接**：
   - Signal URL: `http://127.0.0.1:9999/signal`（默认）
   - Authorization: （可选）Bearer token

2. **选择媒体设备**：
   - ✅ Enable Audio - 启用音频
   - ☑️ Enable Video - 启用视频
   - 从下拉列表选择摄像头/麦克风

3. **点击 "🔗 Connect"**
   - 浏览器会请求摄像头/麦克风权限
   - 本地视频会显示在 "Local Video" 区域
   - 日志显示连接状态

4. **发送消息**（可选）：
   - 修改 Payload 内容
   - 点击 "📤 Send Message"

5. **查看远程视频**：
   - 当其他客户端连接时，他们的视频会自动显示
   - 每个远程流有独立的视频元素

6. **断开连接**：
   - 点击 "❌ Disconnect"
   - 摄像头/麦克风指示灯会关闭

---

## 🎨 界面说明

### 🔧 Connection Settings（连接设置）
- **Signal URL**: SFU 服务器的信令端点
- **Authorization**: 认证 Token（可选）
- **DataChannel Label**: 数据通道标签（默认 "kratos"）

### 📹 Media Settings（媒体设置）
- **Enable Audio**: 是否采集音频
- **Enable Video**: 是否采集视频
- **Audio Device**: 选择麦克风
- **Video Device**: 选择摄像头

### 💬 Data Channel Messages（数据通道消息）
- **Packet Mode**: 
  - `binary`: 4字节类型头 + 负载（高效）
  - `text`: JSON 格式 `{"type":n,"payload":"..."}`
- **Message Type**: 消息类型 ID（uint32）
- **Payload**: 消息内容（JSON 或纯文本）

### 🎥 Video Streams（视频流）
- **Local Video**: 本地摄像头预览（静音）
- **Remote Videos**: 远程视频流网格显示

---

## 🧪 测试场景

### 场景 1: 单客户端数据通道测试

1. 只勾选 "Enable Audio"（不需要视频）
2. 点击 Connect
3. 修改 Payload 为测试消息
4. 点击 Send Message
5. 在服务器日志中查看接收到的消息

### 场景 2: 双客户端音视频通话

**客户端 A**：
1. 勾选 Enable Audio 和 Enable Video
2. 选择摄像头
3. 点击 Connect
4. 允许浏览器权限请求

**客户端 B**（另一个浏览器窗口）：
1. 重复上述步骤
2. 两个客户端的视频会互相显示

**预期结果**：
- 每个客户端看到自己的本地视频
- 每个客户端看到对方的远程视频
- 音频双向传输（注意防止回声）

### 场景 3: 多客户端会议（SFU 优势）

打开 3-5 个浏览器窗口，全部连接到服务器：

**预期结果**：
- 每个客户端看到其他所有客户端的视频
- SFU 服务器负责媒体流转发
- 带宽优化：每个客户端只需上传 1 路，下载 N-1 路

### 场景 4: 设备切换测试

1. 连接后，在 Media Settings 中切换设备
2. 断开重连
3. 验证新设备生效

---

## 🔍 调试技巧

### 查看日志

日志区域显示所有关键事件：
```
[2024-01-01T12:00:00.000Z] Media devices enumerated
[2024-01-01T12:00:05.000Z] Local media stream obtained: audio=true, video=true
[2024-01-01T12:00:05.100Z] Added audio track: Default Audio Device
[2024-01-01T12:00:05.200Z] Added video track: Integrated Camera
[2024-01-01T12:00:06.000Z] ice state: checking
[2024-01-01T12:00:07.000Z] ice state: connected
[2024-01-01T12:00:07.100Z] data channel open
[2024-01-01T12:00:07.200Z] remote description set, signaling complete
[2024-01-01T12:00:10.000Z] Received remote video track from stream-abc123
[2024-01-01T12:00:10.100Z] Created video element for stream stream-abc123
```

### 浏览器开发者工具

按 F12 打开开发者工具：

1. **Console 标签**：查看详细错误信息
2. **Network 标签**：监控信令 HTTP 请求
3. **WebRTC Internals**（Chrome）：
   - 地址栏输入：`chrome://webrtc-internals`
   - 查看详细的 RTP 统计、ICE 候选者等

### 常见问题排查

#### ❌ 无法获取摄像头权限
- 检查浏览器权限设置
- 确保使用 HTTPS 或 localhost
- 尝试其他浏览器

#### ❌ 连接失败
- 确认服务器正在运行
- 检查 Signal URL 是否正确
- 查看浏览器 Console 的错误信息
- 检查防火墙/代理设置

#### ❌ 看不到远程视频
- 确认对方已启用视频
- 检查 ICE 连接状态（应为 "connected"）
- 查看日志是否有 "Received remote track" 消息
- 尝试刷新页面重连

#### ❌ 音频回声
- 佩戴耳机
- 降低扬声器音量
- 启用浏览器的回声消除（通常自动启用）

---

## 📊 性能监控

### 带宽估算

| 配置 | 上行带宽 | 下行带宽（每客户端） |
|------|---------|-------------------|
| 仅音频 | ~50 Kbps | ~50 Kbps × (N-1) |
| 音频 + 480p 视频 | ~500 Kbps | ~500 Kbps × (N-1) |
| 音频 + 720p 视频 | ~1.5 Mbps | ~1.5 Mbps × (N-1) |
| 音频 + 1080p 视频 | ~3 Mbps | ~3 Mbps × (N-1) |

*N = 客户端总数*

### CPU 使用率

- **编码**：主要消耗在本地（H.264/VP8 编码）
- **解码**：取决于远程流数量
- **SFU 服务器**：仅转发，开销较低

---

## 🔧 高级配置

### 修改视频分辨率

编辑 `test_client.html`，找到 `constraints` 部分：

```javascript
video: {
  deviceId: videoDeviceEl.value ? { exact: videoDeviceEl.value } : undefined,
  width: { ideal: 1920 },  // 改为 1920
  height: { ideal: 1080 }  // 改为 1080
}
```

### 添加 TURN 服务器

修改 `RTCPeerConnection` 配置：

```javascript
pc = new RTCPeerConnection({ 
  iceServers: [
    { urls: "stun:stun.l.google.com:19302" },
    {
      urls: "turn:your-turn-server.com:3478",
      username: "user",
      credential: "pass"
    }
  ]
});
```

### 自定义消息协议

修改 `buildBinaryPacket` 和 `parseBinaryPacket` 函数以适配你的协议。

---

## 📝 与 Go 客户端对比

| 功能 | HTML 客户端 | Go 客户端 |
|------|------------|----------|
| 音视频采集 | ✅ 浏览器 API | ⚠️ 需要额外库 |
| 数据通道 | ✅ 完整支持 | ✅ 完整支持 |
| 设备选择 | ✅ GUI 界面 | ❌ 命令行 |
| 可视化 | ✅ 实时视频 | ❌ 无界面 |
| 自动化测试 | ❌ 手动操作 | ✅ 可编程 |
| 跨平台 | ✅ 任何浏览器 | ✅ Go 支持的平台 |

**建议**：
- **开发调试**：使用 HTML 客户端（直观）
- **自动化测试**：使用 Go 客户端（可编程）
- **生产环境**：根据需求选择

---

## 🎯 下一步

1. **集成到你的应用**：参考此页面的代码实现 WebRTC 功能
2. **自定义 UI**：根据你的需求调整样式和布局
3. **添加功能**：
   - 屏幕共享
   - 文件传输
   - 聊天室管理
   - 录制功能

---

## 📚 相关资源

- [WebRTC API 文档](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API)
- [Pion WebRTC](https://pion.ly/)
- [SFU 架构详解](https://webrtc.org/getting-started/server-side)
- [本项目 README](./README.md)

---

## ⚠️ 注意事项

1. **HTTPS 要求**：生产环境必须使用 HTTPS 才能访问摄像头/麦克风
2. **浏览器兼容性**：推荐使用 Chrome/Edge/Firefox 最新版
3. **权限管理**：首次连接需要用户授权媒体设备
4. **资源释放**：断开时务必停止媒体轨道，否则摄像头指示灯不会关闭
5. **并发限制**：浏览器对同时打开的摄像头数量有限制（通常 1-2 个）
