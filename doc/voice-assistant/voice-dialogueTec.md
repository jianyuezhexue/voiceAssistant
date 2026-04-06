# 语音对话功能技术架构文档

**版本**: v1.5
**创建日期**: 2026-04-06
**修订日期**: 2026-04-06
**状态**: 待架构专家评审（第4轮）
**关联PRD**: voice-assistant-prd-v1.0.html

---

## 目录

1. [系统架构概述](#1-系统架构概述)
2. [模块职责划分](#2-模块职责划分)
   2.4 [LLM API调用方案](#24-llm-api调用方案-qwen-api)
   2.5 [Go WebRTC 库推荐](#25-go-webrtc-库推荐)
3. [系统架构图](#3-系统架构图)
4. [前端架构设计](#4-前端架构设计)
5. [后端架构设计](#5-后端架构设计)
6. [通信协议定义](#6-通信协议定义)
   6.4 [全双工通信通道设计](#64-全双工通信通道设计)
   6.5 [UDP乱序处理方案](#65-udp乱序处理方案)
7. [状态机设计](#7-状态机设计)
8. [打断机制实现](#8-打断机制实现)
9. [API接口设计](#9-api接口设计)
10. [数据流设计](#10-数据流设计)
11. [性能指标](#11-性能指标)
12. [异常处理](#12-异常处理)
13. [部署架构](#13-部署架构)
14. [版本记录](#14-版本记录)

---

## 1. 系统架构概述

### 1.1 架构设计目标

- **低延迟**: 端到端延迟 < 2秒
- **高实时性**: 打断响应时间 < 300ms
- **稳定性**: 识别延迟 < 500ms
- **可扩展性**: 支持后续多模态扩展

### 1.2 技术选型

| 模块 | 技术方案 | 说明 |
|------|---------|------|
| 语音识别 (ASR) | CosyVoice Small / SenseVoice Small | 本地部署，流式输出 |
| 语音合成 (TTS) | Qwen3-TTS | 本地部署，流式合成 |
| 语音活动检测 (VAD) | WebRTC VAD | 内置于浏览器 |
| 音频传输 | WebRTC DataChannel | UDP传输，保证实时性 |
| 文字传输 | WebSocket | 双向实时通信 |
| 后端框架 | Go + Gin | 高性能HTTP服务 |
| 前端框架 | Vue 3 + TypeScript | 响应式UI |
| 模型服务 | 独立容器部署 | 与后端解耦 |

### 1.3 系统组件

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client (Browser)                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  Wake Word  │  │   Audio     │  │   TTS       │              │
│  │  Detector   │  │   Player    │  │   Player    │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│  ┌─────────────────────────────────────────────────────────┐      │
│  │              VoiceDialogueService                       │      │
│  │  - AudioCapture (getUserMedia)                        │      │
│  │  - VAD (WebRTC VAD)                                    │      │
│  │  - StateMachine                                        │      │
│  │  - InterruptHandler                                    │      │
│  └─────────────────────────────────────────────────────────┘      │
│  ┌─────────────┐  ┌─────────────┐                                │
│  │  WebSocket  │  │  DataChannel│                                │
│  │  Client     │  │  Client     │                                │
│  └─────────────┘  └─────────────┘                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ WebSocket (文字+控制)
                              │ DataChannel (音频流)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         Backend Service                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │   LLM       │  │   ASR       │  │   TTS       │              │
│  │   Service   │  │   Service   │  │   Service   │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│  ┌─────────────────────────────────────────────────────────┐      │
│  │              VoiceDialogueHandler                       │      │
│  │  - SessionManager                                      │      │
│  │  - AudioBuffer                                         │      │
│  │  - StreamRouter                                        │      │
│  └─────────────────────────────────────────────────────────┘      │
│  ┌─────────────┐  ┌─────────────┐                                │
│  │  WebSocket  │  │  WebRTC     │                                │
│  │  Handler    │  │  Handler    │                                │
│  └─────────────┘  └─────────────┘                                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ gRPC / HTTP
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Model Services (Containers)                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  CosyVoice  │  │  Qwen3-TTS  │  │  LLM        │              │
│  │  (ASR)      │  │             │  │  Service    │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

---

## 2. 模块职责划分

### 2.1 前端模块职责

| 模块 | 职责 | 技术实现 |
|------|------|---------|
| AudioCapture | 音频采集、降噪、AEC | WebRTC getUserMedia + AudioContext |
| WakeWordDetector | 唤醒词"小爱同学"检测 | 音频特征匹配 / 后端检测 |
| VADClient | 语音活动检测 | WebRTC VAD |
| VoiceDialogueService | 核心语音服务编排 | TypeScript Class |
| StateMachine | 状态机管理 | XState / 自定义状态机 |
| InterruptHandler | 打断处理 | 事件监听 + 状态同步 |
| WebSocketClient | 文字通信 | 原生WebSocket |
| DataChannelClient | 音频流传输 | WebRTC DataChannel |
| TTSPlayer | TTS音频播放 | Web Audio API |

### 2.2 后端模块职责

| 模块 | 职责 | 技术实现 |
|------|------|---------|
| VoiceDialogueHandler | 语音对话请求处理 | Gin WebSocket Handler |
| SessionManager | 会话管理、状态维护 | Go Struct + Redis |
| AudioBuffer | 音频数据缓冲 | Ring Buffer |
| StreamRouter | 流式数据路由 | Channel Select |
| ASRService | 语音识别服务封装 | CosyVoice / SenseVoice Client |
| TTSService | 语音合成服务封装 | Qwen3-TTS Client |
| LLMService | 大模型服务封装 | Qwen API (dashscope) |
| WebRTCSignaling | WebRTC信令处理 | WebSocket Signaling |

### 2.3 模型服务职责

| 服务 | 职责 | 部署方式 |
|------|------|---------|
| CosyVoice Small | 语音识别，流式输出 | Docker容器 |
| SenseVoice Small | 备选ASR引擎 | Docker容器 |
| Qwen3-TTS | 语音合成，流式输出 | Docker容器 |
| LLM Service | 对话生成 | Qwen API (dashscope) |

### 2.4 LLM API调用方案 (Qwen API)

#### 2.4.1 配置说明

LLM服务通过Qwen API（阿里云百炼平台）调用，不再使用本地vLLM部署。

**环境变量配置**:
| 变量名 | 说明 | 示例值 |
|-------|------|-------|
| DASHSCOPE_API_KEY | 阿里云百炼API密钥 | sk-e692504205e74522b45710e1c25065ad |
| LLM_MODEL | 模型名称 | qwen-plus |

#### 2.4.2 Go代码实现

```go
// component/llm/client.go
package llm

import (
    "context"
    "github.com/qwenlm/qwen-go"
)

// NewChatModel 创建Qwen聊天模型实例
apiKey := "sk-e692504205e74522b45710e1c25065ad"
modelName := "qwen-plus"
chatModel, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
    BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
    APIKey:      apiKey,
    Timeout:     0,  // 0表示无超时限制
    Model:       modelName,
    MaxTokens:   of(2048),
    Temperature: of(float32(0.7)),
    TopP:        of(float32(0.7)),
})

// 流式对话
stream, err := chatModel.Stream(ctx, &qwen.ChatRequest{
    Messages: []qwen.Message{
        {Role: "user", Content: "你好"},
    },
})

// 处理流式响应
for {
    resp, err := stream.Recv()
    if err == io.EOF {
        break
    }
    fmt.Println(resp.Content)
}
```

#### 2.4.3 API调用特点

| 特性 | 说明 |
|-----|------|
| 调用方式 | HTTP REST API |
| BaseURL | https://dashscope.aliyuncs.com/compatible-mode/v1 |
| 模型 | qwen-plus |
| 认证方式 | API Key (环境变量配置) |
| 超时处理 | API超时5s后降级回复"服务繁忙，请稍后再试" |
| 无需GPU | 使用外部API服务，无需本地GPU资源 |

### 2.5 Go WebRTC 库推荐

#### 2.5.1 主流选择: pion/webrtc

推荐使用 `github.com/pion/webrtc/v3`，这是目前最成熟、使用最广泛的纯 Go WebRTC 实现。

**优势**:
- 纯 Go 实现，易于集成和部署
- API 设计友好，文档完善
- 活跃的社区和持续的版本更新
- 支持 DataChannel、SDP 协商、ICE Candidate 处理

**安装**:
```bash
go get github.com/pion/webrtc/v3
```

**后端 WebRTC 配置示例**:
```go
import "github.com/pion/webrtc/v3"

// 创建 PeerConnection
config := &webrtc.Configuration{
    ICEServers: []ICEServer{
        {
            URLs: []string{"stun:stun.l.google.com:19302"},
        },
    },
}

peerConnection, err := webrtc.NewPeerConnection(config)
if err != nil {
    log.Fatal(err)
}
defer peerConnection.Close()

// 创建 DataChannel
dc, err := peerConnection.CreateDataChannel("voice-audio", &webrtc.DataChannelInit{
    Ordered:           false,        // UDP模式，不保证顺序
    MaxRetransmits:   0,            // 不重传
})
if err != nil {
    log.Fatal(err)
}

// 设置 DataChannel 回调
dc.OnMessage(func(msg webrtc.DataChannelMessage) {
    if msg.IsString {
        // 处理字符串消息（控制命令）
        log.Printf("DC String: %s", string(msg.Data))
    } else {
        // 处理二进制消息（音频数据）
        log.Printf("DC Binary: %d bytes", len(msg.Data))
    }
})
```

#### 2.5.2 其他备选库

| 库 | 说明 | 适用场景 |
|----|------|---------|
| `github.com/pion/webrtc/v3` | 纯 Go 实现，主流选择 | 通用 WebRTC 应用 |
| `github.com/ggreal/pion/webrtc` | pion 分支，可能有特定优化 | 需要特定功能时 |
| `github.com/meinside/libwebrtc-go` | 绑定 libwebrtc C++ 库 | 追求极致性能 |

#### 2.5.3 pion/webrtc vs 原生 libwebrtc

| 特性 | pion/webrtc | 原生 libwebrtc |
|------|------------|----------------|
| 语言 | 纯 Go | C++ |
| 集成难度 | 低 | 高 |
| 性能 | 良好 | 极佳 |
| 跨平台 | 简单（Go 天然跨平台） | 需要编译 C++ |
| 维护成本 | 低（社区维护） | 高（需同步官方） |

**推荐**: 对于 VoiceAssistant 项目，优先使用 `pion/webrtc`，其在保持良好性能的同时大大降低了集成复杂度。

#### 2.5.4 后端 DataChannel 服务端实现

```go
// component/webrtc/datachannel.go
package webrtc

import (
    "github.com/pion/webrtc/v3"
    "github.com/pion/webrtc/v3/pkg/media"
)

type AudioDataChannel struct {
    peerConnection *webrtc.PeerConnection
    dataChannel    *webrtc.DataChannel
    audioBuffer    *AudioBuffer  // 参见 6.5 节
}

// 创建 WebRTC 服务端
func NewAudioDataChannel() (*AudioDataChannel, error) {
    config := &webrtc.Configuration{
        ICEServers: []ICEServer{
            {URLs: []string{"stun:stun.l.google.com:19302"}},
        },
    }

    pc, err := webrtc.NewPeerConnection(config)
    if err != nil {
        return nil, err
    }

    // 创建 DataChannel（服务端接收端）
    dc, err := pc.CreateDataChannel("voice-audio", &webrtc.DataChannelInit{
        Ordered: false,     // UDP模式
    })
    if err != nil {
        pc.Close()
        return nil, err
    }

    return &AudioDataChannel{
        peerConnection: pc,
        dataChannel:    dc,
        audioBuffer:    NewAudioBuffer(),
    }, nil
}

// 发送音频数据
func (dc *AudioDataChannel) SendAudio(packet *AudioPacket) error {
    // 序列化音频包
    data, err := json.Marshal(packet)
    if err != nil {
        return err
    }

    // 通过 DataChannel 发送
    return dc.dataChannel.Send(data)
}

// 发送 TTS 音频流
func (dc *AudioDataChannel) SendTTSAudio(pcmData []byte, timestamp int64, isLast bool) error {
    packet := &AudioPacket{
        Sequence:   dc.audioBuffer.NextSequence(),
        Timestamp:  timestamp,
        Data:       pcmData,
        SampleRate: 24000,
        IsLast:     isLast,
    }
    return dc.SendAudio(packet)
}
```

---

## 3. 系统架构图

### 3.1 整体架构图

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              CLIENT (BROWSER)                             │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                        Vue 3 Application                           │  │
│  │  ┌──────────────────────────────────────────────────────────────┐  │  │
│  │  │                    VoiceDialogueComponent                   │  │  │
│  │  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐   │  │  │
│  │  │  │ 监听中 │ │ 识别中 │ │ 思考中 │ │ 回复中 │ │ 播放中 │   │  │  │
│  │  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘   │  │  │
│  │  │       ▲         ▲         ▲         ▲         ▲          │  │  │
│  │  │       └─────────┴─────────┴─────────┴─────────┘          │  │  │
│  │  │                    State Machine                          │  │  │
│  │  └──────────────────────────────────────────────────────────────┘  │  │
│  │  ┌──────────────────────────────────────────────────────────────┐  │  │
│  │  │                     VoiceService                            │  │  │
│  │  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐  │  │  │
│  │  │  │AudioCapture │ │WakeWordDetector│ │InterruptHandler │  │  │  │
│  │  │  └─────────────┘ └─────────────┘ └─────────────────────┘  │  │  │
│  │  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐  │  │  │
│  │  │  │VADClient    │ │TTSPlayer    │ │DialogueStore(Pinia)│  │  │  │
│  │  │  └─────────────┘ └─────────────┘ └─────────────────────┘  │  │  │
│  │  └──────────────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────┐ ┌─────────────┐                                   │  │
│  │  │WebSocket    │ │DataChannel  │                                   │  │
│  │  │Client       │ │Client       │                                   │  │
│  │  └─────────────┘ └─────────────┘                                   │  │
│  └────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────┘
           │                                    │
           │ WebSocket (Text/Control)           │ DataChannel (Audio)
           │ ws://host:8080/ws/voice             │ UDP-like
           ▼                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                            BACKEND (Go + Gin)                            │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                      VoiceDialogueHandler                         │  │
│  │  ┌────────────────┐  ┌────────────────┐  ┌────────────────────┐  │  │
│  │  │ onTextMessage  │  │ onAudioStream  │  │ onTTSStream        │  │  │
│  │  └────────────────┘  └────────────────┘  └────────────────────┘  │  │
│  │  ┌────────────────────────────────────────────────────────────┐  │  │
│  │  │                    SessionManager                          │  │  │
│  │  │  - user_id, state, audio_buffer, context                  │  │  │
│  │  └────────────────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│           │                    │                    │
│           ▼                    ▼                    ▼
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐
│  │   ASRService   │  │   LLMService   │  │   TTSService   │
│  │  (CosyVoice)   │  │  (OpenAI API)  │  │  (Qwen3-TTS)   │
│  └────────────────┘  └────────────────┘  └────────────────┘
│           │                    │                    │
└───────────│────────────────────│────────────────────│──────────────────
            │ gRPC               │ HTTP               │ gRPC
            ▼                    ▼                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                         MODEL SERVICES (Docker)                          │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐               │
│  │  CosyVoice   │   │  Qwen API    │   │  Qwen3-TTS   │               │
│  │  Small       │   │ (dashscope)  │   │              │               │
│  │  :8001      │   │   外部服务     │   │   :8003      │               │
│  └──────────────┘   └──────────────┘   └──────────────┘               │
└──────────────────────────────────────────────────────────────────────────┘
```

### 3.2 前端音频架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     Browser Audio Architecture                   │
│                                                                  │
│  ┌───────────────┐                                              │
│  │ Microphone    │ ─── getUserMedia() ───▶ MediaStream         │
│  └───────────────┘                                              │
│          │                                                       │
│          ▼                                                       │
│  ┌───────────────┐     ┌───────────────┐     ┌───────────────┐ │
│  │ AudioContext  │────▶│  Noise        │────▶│  Voice         │ │
│  │ (分析节点)     │     │  Suppression  │     │  Detector      │ │
│  └───────────────┘     └───────────────┘     │  (VAD)         │ │
│          │                                      └───────────────┘ │
│          │                                               │        │
│          ▼                                               ▼        │
│  ┌───────────────┐                              ┌───────────────┐ │
│  │ MediaRecorder │ ◀─── AudioChunks ────────── │  Wake Word    │ │
│  │ (for ASR)     │                              │  Detector     │ │
│  └───────────────┘                              └───────────────┘ │
│          │                                               │        │
│          ▼                                               ▼        │
│  ┌───────────────┐     ┌───────────────┐     ┌───────────────┐ │
│  │ DataChannel   │────▶│  WebSocket   │     │  StateMachine │ │
│  │ (Audio to     │     │  (Text/Cmd)  │     │               │ │
│  │  Backend)     │     │              │     │               │ │
│  └───────────────┘     └───────────────┘     └───────────────┘ │
│          │                    │                      │          │
└──────────│────────────────────│──────────────────────│──────────┘
           │                    │                      │
           ▼                    ▼                      ▼
      [Backend]           [Backend]             [Frontend]
      Audio API           Text/Control          UI Update
```

### 3.3 后端服务架构

```
┌─────────────────────────────────────────────────────────────────┐
│                    Backend Service Architecture                  │
│                                                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    Gin HTTP Server                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐   │  │
│  │  │ GET /health │  │WS /ws/voice │  │ /api/v1/*       │   │  │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘   │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                   │
│                              ▼                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              VoiceDialogueHandler (WebSocket)             │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │                   Session Manager                    │  │  │
│  │  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐    │  │  │
│  │  │  │Session1│ │Session2│ │Session3│ │SessionN│    │  │  │
│  │  │  │(user1) │ │(user2) │ │(user3) │ │(userN) │    │  │  │
│  │  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘    │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │                  Stream Multiplexer                 │  │  │
│  │  │   ASR ──▶ LLM ──▶ TTS ──▶ DataChannel              │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│           │                │                │                    │
│           ▼                ▼                ▼                    │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────┐           │
│  │ ASR Client  │   │ LLM Client  │   │ TTS Client  │           │
│  │ (CosyVoice) │   │ (Qwen API)  │   │ (Qwen3-TTS) │           │
│  └─────────────┘   └─────────────┘   └─────────────┘           │
│           │                │                │                    │
└───────────│────────────────│────────────────│────────────────────┘
            │                │                │
            ▼                ▼                ▼
      ┌───────────┐    ┌───────────┐    ┌───────────┐
      │CosyVoice  │    │Qwen API   │    │Qwen3-TTS │
      │:8001      │    │(外部)      │    │:8003     │
      └───────────┘    └───────────┘    └───────────┘
```

---

## 4. 前端架构设计

### 4.1 前端目录结构

```
frontend/src/
├── services/
│   ├── ws.ts                    # WebSocket 服务
│   ├── datachannel.ts           # DataChannel 服务
│   └── api.ts                   # REST API 服务
├── stores/
│   └── voice.ts                 # Pinia 语音状态管理
├── composables/
│   ├── useAudioCapture.ts       # 音频采集 composable
│   ├── useVAD.ts                # VAD composable
│   ├── useWakeWord.ts           # 唤醒词 composable
│   └── useVoiceDialogue.ts      # 语音对话 composable
├── components/
│   └── voice/
│       ├── VoiceButton.vue      # 语音按钮组件
│       ├── VoiceWaveform.vue    # 波形动画组件
│       ├── VoiceStatus.vue      # 状态显示组件
│       └── DialogueBubble.vue   # 对话气泡组件
├── types/
│   └── voice.ts                 # 语音相关类型定义
└── pages/
    └── VoicePage.vue            # 语音对话页面
```

### 4.2 前端核心类型定义

```typescript
// voice.ts

// ==================== 枚举定义 ====================

/**
 * 语音对话状态枚举
 */
export enum VoiceState {
  /** 初始状态/监听中 */
  LISTENING = 'listening',
  /** 识别中 - 检测到语音正在识别 */
  RECOGNIZING = 'recognizing',
  /** 思考中 - 等待AI回复 */
  THINKING = 'thinking',
  /** 回复中 - AI正在回复（文字） */
  RESPONDING = 'responding',
  /** 播放中 - TTS正在播放 */
  PLAYING = 'playing',
  /** 错误状态 */
  ERROR = 'error',
  /** 空闲状态 */
  IDLE = 'idle'
}

/**
 * 打断来源枚举
 */
export enum InterruptSource {
  USER_SPEECH = 'user_speech',   // 用户新话语
  USER_CLICK = 'user_click',      // 用户点击停止
  SERVER_CMD = 'server_cmd',     // 服务器命令
  TIMEOUT = 'timeout'             // 超时
}

/**
 * 消息类型枚举
 */
export enum MessageType {
  /** ASR识别结果 */
  ASR_RESULT = 'asr_result',
  /** ASR识别完成 */
  ASR_COMPLETE = 'asr_complete',
  /** LLM回复文字 */
  LLM_TEXT = 'llm_text',
  /** LLM回复完成 */
  LLM_COMPLETE = 'llm_complete',
  /** TTS音频数据 */
  TTS_AUDIO = 'tts_audio',
  /** TTS完成 */
  TTS_COMPLETE = 'tts_complete',
  /** 状态更新 */
  STATE_UPDATE = 'state_update',
  /** 错误信息 */
  ERROR = 'error',
  /** 打断命令 */
  INTERRUPT = 'interrupt',
  /** 心跳 */
  PING = 'ping',
  /** 心跳响应 */
  PONG = 'pong'
}

// ==================== 核心接口定义 ====================

/**
 * 音频配置
 */
export interface AudioConfig {
  /** 采样率 */
  sampleRate: number;
  /** 声道数 */
  channels: number;
  /** 采样位数 */
  bitDepth: number;
  /** 音频块大小(ms) */
  chunkDuration: number;
  /** 是否启用降噪 */
  noiseSuppression: boolean;
  /** 是否启用回声消除 */
  echoCancellation: boolean;
  /** 是否启用自动增益 */
  autoGainControl: boolean;
}

/**
 * 语音消息结构
 */
export interface VoiceMessage {
  /** 消息ID */
  id: string;
  /** 消息类型 */
  type: MessageType;
  /** 会话ID */
  sessionId: string;
  /** 消息内容 */
  data?: unknown;
  /** 时间戳 */
  timestamp: number;
}

/**
 * ASR识别结果
 */
export interface ASRResult {
  /** 识别文本 */
  text: string;
  /** 是否为最终结果 */
  isFinal: boolean;
  /** 置信度 */
  confidence?: number;
  /** 开始时间 */
  startTime?: number;
  /** 结束时间 */
  endTime?: number;
}

/**
 * LLM回复结构
 */
export interface LLMResponse {
  /** 回复文本 */
  text: string;
  /** 是否为流式片段 */
  isChunk: boolean;
  /** 是否完成 */
  isComplete: boolean;
  /** 完整回复（isComplete时） */
  fullText?: string;
}

/**
 * TTS音频结构
 */
export interface TTSAudio {
  /** 音频数据 (PCM) */
  data: ArrayBuffer;
  /** 是否为最后一片 */
  isLast: boolean;
  /** 音频时间戳 */
  timestamp?: number;
}

/**
 * 打断事件
 */
export interface InterruptEvent {
  /** 打断来源 */
  source: InterruptSource;
  /** 打断原因 */
  reason?: string;
  /** 时间戳 */
  timestamp: number;
}

/**
 * 语音会话状态
 */
export interface VoiceSession {
  /** 会话ID */
  id: string;
  /** 用户ID */
  userId: string;
  /** 当前状态 */
  state: VoiceState;
  /** 识别文本 */
  recognizedText: string;
  /** AI回复文本 */
  responseText: string;
  /** 创建时间 */
  createdAt: number;
  /** 最后活跃时间 */
  lastActiveAt: number;
  /** 是否已打断 */
  isInterrupted: boolean;
}

/**
 * WebSocket消息（前端发送）
 */
export interface WSClientMessage {
  /** 消息类型 */
  type: MessageType;
  /** 会话ID */
  sessionId?: string;
  /** 消息数据 */
  data?: unknown;
  /** 时间戳 */
  timestamp: number;
}

/**
 * WebSocket消息（前端接收）
 */
export interface WSServerMessage {
  /** 消息类型 */
  type: MessageType;
  /** 会话ID */
  sessionId: string;
  /** 消息数据 */
  data: unknown;
  /** 时间戳 */
  timestamp: number;
}

/**
 * DataChannel消息
 */
export interface DCMessage {
  /** 消息类型 */
  type: 'audio' | 'control';
  /** 音频数据或控制命令 */
  payload: ArrayBuffer | string;
  /** 时间戳 */
  timestamp: number;
}

/**
 * 唤醒词检测配置
 */
export interface WakeWordConfig {
  /** 唤醒词 */
  keyword: string;
  /** 灵敏度 */
  sensitivity: number;
  /** 检测阈值 */
  threshold: number;
  /** 是否启用 */
  enabled: boolean;
}

/**
 * VAD配置
 */
export interface VADConfig {
  /** 语音检测灵敏度 (0-3) */
  sensitivity: number;
  /** 语音开始阈值 */
  speechStartThreshold: number;
  /** 语音结束阈值 */
  speechEndThreshold: number;
  /** 静音超时时间(ms) */
  silenceTimeout: number;
  /** 最大语音时长(ms) */
  maxSpeechDuration: number;
}

/**
 * 性能指标
 */
export interface PerformanceMetrics {
  /** 端到端延迟(ms) */
  e2eLatency: number;
  /** 打断响应时间(ms) */
  interruptLatency: number;
  /** 识别延迟(ms) */
  asrLatency: number;
  /** TTS延迟(ms) */
  ttsLatency: number;
}
```

### 4.3 前端核心服务接口

```typescript
// services/voiceService.ts

import { VoiceState, VoiceMessage, ASRResult, LLMResponse, TTSAudio } from '../types';

/**
 * 语音对话服务接口
 */
export interface IVoiceService {
  /** 连接服务器 */
  connect(): Promise<void>;
  /** 断开连接 */
  disconnect(): void;
  /** 开始录音 */
  startRecording(): Promise<void>;
  /** 停止录音 */
  stopRecording(): void;
  /** 打断当前流程 */
  interrupt(): void;
  /** 发送文字消息 */
  sendText(text: string): void;
  /** 获取当前状态 */
  getState(): VoiceState;
  /** 是否已连接 */
  isConnected(): boolean;

  // 事件订阅
  onStateChange(callback: (state: VoiceState) => void): () => void;
  onASRResult(callback: (result: ASRResult) => void): () => void;
  onLLMResponse(callback: (response: LLMResponse) => void): () => void;
  onTTSAudio(callback: (audio: TTSAudio) => void): () => void;
  onError(callback: (error: Error) => void): () => void;
}

/**
 * 音频采集服务接口
 */
export interface IAudioCapture {
  /** 初始化音频环境 */
  initialize(): Promise<void>;
  /** 开始采集 */
  start(): void;
  /** 停止采集 */
  stop(): void;
  /** 获取音频流 */
  getStream(): MediaStream | null;
  /** 设置音量阈值回调 */
  onVolumeChange(callback: (volume: number) => void): () => void;
  /** 获取当前音量 (0-1) */
  getCurrentVolume(): number;
}
```

### 4.4 唤醒词检测架构决策

**推荐方案**: 前端独立方案

**设计原则**: 唤醒词检测仅在前端完成，唤醒成功后再与后端建立连接。减少后端验证耗时，提升用户体验。

```
┌─────────────────────────────────────────────────────────────────┐
│                    唤醒词检测架构 (前端独立)                       │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                     Browser (前端)                        │    │
│  │                                                         │    │
│  │   状态1: Always-on 监听模式                              │    │
│  │   ┌─────────────────────────────────────────────────┐  │    │
│  │   │  AudioCapture (麦克风持续采集)                    │  │    │
│  │   │  ↓                                              │  │    │
│  │   │  WakeWordDetector (MFCC + DTW 实时检测)           │  │    │
│  │   │  ↓ 唤醒词检测到                                   │  │    │
│  │   │  播放唤醒成功音效                                 │  │    │
│  │   └─────────────────────────────────────────────────┘  │    │
│  │                                                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                  │
│                              │ 唤醒成功，建立连接                  │
│                              ↓                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │                     Backend (后端)                        │    │
│  │                                                         │    │
│  │   状态2: WebSocket + DataChannel 连接已建立              │    │
│  │   状态3: 正常语音对话流程                                 │    │
│  │                                                         │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

| 层级 | 实现方式 | 职责 | 优点 |
|------|---------|------|------|
| 前端检测 | 音频特征匹配 (MFCC + DTW) | 持续轻量监听，检测唤醒词 | 低功耗、隐私保护、快速响应、无后端依赖 |

**前端检测流程**:
1. **Always-on 监听**: 麦克风持续采集音频（降噪处理后），功耗极低
2. **特征提取**: 每帧提取 MFCC 音频特征
3. **DTW 匹配**: 与预设唤醒词"小爱同学"模板做动态时间规整匹配
4. **阈值判断**: 匹配分数超过阈值（0.75）时，触发唤醒
5. **唤醒成功**: 播放本地唤醒成功音效，状态切换为 LISTENING
6. **建立连接**: 唤醒成功后，前端主动建立 WebSocket + DataChannel 连接
7. **连接成功提示**: 连接建立成功后，Alert 弹窗提醒"连接成功，开启对话"
8. **对话结束**: 用户主动结束对话时，关闭 WebSocket + DataChannel，Alert 弹窗提醒"对话已结束，连接已关闭"

**后端职责变化**:
- 唤醒阶段: **无后端参与**，纯前端独立完成
- 对话阶段: WebSocket + DataChannel 连接建立后，后端正常处理 ASR/LLM/TTS

**连接生命周期**:
```
┌─────────────────────────────────────────────────────────────────┐
│                      连接生命周期管理                               │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  唤醒成功                                                         │
│    │                                                              │
│    ▼                                                              │
│  建立 WebSocket + DataChannel ─────→ Alert("连接成功，开启对话")   │
│    │                                                              │
│    ▼                                                              │
│  语音对话流程 (可多轮)                                            │
│    │                                                              │
│    ├── 用户主动结束 ──→ 关闭连接 ──→ Alert("对话已结束，连接已关闭")│
│    │                                                              │
│    └── 超时自动结束 ──→ 关闭连接                                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Alert 提示设计**:
| 场景 | Alert 内容 | 样式 |
|------|-----------|------|
| 连接建立成功 | "连接成功，开启对话" | 成功类型（绿色），3秒自动关闭 |
| 连接关闭 | "对话已结束，连接已关闭" | 信息类型（蓝色），3秒自动关闭 |

**唤醒词配置**:
```yaml
wake_word:
  keyword: "小爱同学"
  frontend:
    enabled: true
    sample_rate: 16000        # 采样率
    chunk_duration: 100ms     # 帧长
    threshold: 0.75           # 匹配阈值
    audio_format: PCM_16bit   # 音频格式
  connection:
    # 唤醒成功后再建立这些连接
    websocket: "ws://host:8080/ws/voice"
    datachannel_label: "voice-audio"
```

**性能优势**:
| 对比项 | 前端+后端混合 | 前端独立 (优化后) |
|--------|--------------|------------------|
| 唤醒响应时间 | 约 500-800ms | 约 100-200ms |
| 后端资源消耗 | 需要承载唤醒验证 | 无 |
| 依赖性 | 依赖后端在线 | 无后端依赖 |
| 隐私性 | 音频需上传后端 | 音频保留本地 |

---

## 5. 后端架构设计

### 5.1 后端目录结构

```
backend/
├── api/
│   ├── voice/
│   │   ├── handler.go          # WebSocket Handler
│   │   ├── middleware.go        # 认证、限流中间件
│   │   └── validator.go         # 请求参数校验
│   └── common/
│       ├── response.go          # 统一响应结构
│       └── errors.go            # 错误定义
├── component/
│   ├── llm/
│   │   └── client.go           # Qwen API HTTP Client
│   ├── asr/
│   │   └── client.go           # CosyVoice gRPC Client
│   ├── tts/
│   │   └── client.go           # Qwen3-TTS gRPC Client
│   ├── webrtc/
│   │   └── datachannel.go      # WebRTC DataChannel 服务端封装
│   └── redis/
│       └── redis.go            # Redis 客户端封装
├── domain/
│   └── voice/
│       ├── model.go            # 枚举: VoiceState, MessageType, InterruptSource
│       ├── session.go          # Session 实体
│       ├── aggregate.go        # VoiceDialogue 聚合根
│       ├── event.go            # 领域事件定义
│       ├── repository.go       # 仓储接口: ISessionRepository
│       └── rules.go            # 业务规则: 状态转换、打断规则
├── logic/
│   └── voice/
│       ├── dialogue.go         # 语音对话流程编排
│       └── session.go          # 会话管理逻辑
├── config/
│   └── config.go              # 配置结构体定义
├── router/
│   └── router.go              # 路由注册
├── main.go                    # 应用入口
└── go.mod                    # Go模块
```

### 5.1.1 各层职责定义

| 层级 | 目录 | 职责 | 原则 |
|------|------|------|------|
| API | `api/` | HTTP/WebSocket 处理、参数校验、路由入口 | 只做入口，**不含业务逻辑** |
| Component | `component/` | 封装外部依赖客户端 (LLM/ASR/TTS/Redis/WebRTC) | **纯技术封装，无业务判断** |
| Domain | `domain/` | 业务领域模型、枚举、实体、聚合根、仓储接口、业务规则 | **包含业务规则，依赖接口而非实现** |
| Logic | `logic/` | 业务用例编排，依赖注入 component 和 domain 接口 | **实现业务逻辑**，组合 component 实现用例 |

### 5.1.2 Component vs Logic 区分

| 归入 Component | 归入 Logic |
|---------------|------------|
| `component/llm/client.go` | `logic/voice/dialogue.go` |
| 只做 HTTP/gRPC 调用 | 编排业务逻辑 |
| 封装第三方SDK | 实现领域规则 |
| 无状态 | 有状态 |
| 接口签名: `SendChat(ctx, msgs) (stream, error)` | 接口签名: `StartDialogue(ctx, sessionID) (err)` |
| **不关心业务场景** | **理解业务语义** |

### 5.1.3 Domain 模型构建

```
domain/voice/
│
├── model.go          # 值对象和枚举
│   ├── VoiceState    # LISTENING/RECOGNIZING/THINKING/...
│   ├── MessageType   # asr_result/llm_text/tts_audio/...
│   └── InterruptSource
│
├── session.go        # Session 实体
│   ├── ID, UserID, State
│   ├── RecognizedText, ResponseText
│   ├── CreatedAt, LastActiveAt
│   └── AddContext(), ClearContext()
│
├── aggregate.go      # VoiceDialogue 聚合根 (核心!)
│   ├── ID
│   ├── Session       # 聚合内唯一实体
│   ├── StateHistory # 状态转换历史
│   │
│   ├── 业务方法 (内聚领域逻辑)
│   │   ├── HandleWakeWordDetected()
│   │   ├── HandleSpeechStarted()
│   │   ├── HandleSpeechEnded(text)
│   │   ├── HandleInterrupt(source)
│   │   ├── CanTransitionTo(targetState)  // 状态机规则
│   │   └── CanInterrupt()                 // 打断规则
│   │
│   └── 领域事件发布
│       ├── PublishWakeWordDetected()
│       ├── PublishASRCompleted(text)
│       └── PublishInterruptTriggered(source)
│
├── event.go         # 领域事件
│   ├── WakeWordDetectedEvent
│   ├── SpeechRecognizedEvent
│   ├── LLMResponseStartedEvent
│   ├── TTSPlaybackStartedEvent
│   └── InterruptEvent
│
├── repository.go    # 仓储接口
│   └── interface ISessionRepository
│       ├── Save(ctx, session)
│       ├── Get(ctx, id)
│       ├── Update(ctx, session)
│       └── Delete(ctx, id)
│
└── rules.go        # 业务不变式
    ├── 状态转换规则: IDLE→LISTENING, LISTENING→RECOGNIZING, etc.
    ├── 打断规则: PLAYING时可被打断, IDLE时不可打断
    └── 超时规则: 30s无操作回到IDLE
```

### 5.2 后端核心类型定义

```go
// domain/voice/types.go

package voice

import (
    "time"
)

// VoiceState 语音状态
type VoiceState int

const (
    StateIdle       VoiceState = iota // 空闲
    StateListening                    // 监听中
    StateRecognizing                  // 识别中
    StateThinking                      // 思考中
    StateResponding                    // 回复中
    StatePlaying                       // 播放中
    StateError                         // 错误
)

// String 返回状态字符串
func (s VoiceState) String() string {
    switch s {
    case StateIdle:
        return "idle"
    case StateListening:
        return "listening"
    case StateRecognizing:
        return "recognizing"
    case StateThinking:
        return "thinking"
    case StateResponding:
        return "responding"
    case StatePlaying:
        return "playing"
    case StateError:
        return "error"
    default:
        return "unknown"
    }
}

// MessageType 消息类型
type MessageType string

const (
    MsgTypeASRResult   MessageType = "asr_result"    // ASR识别结果
    MsgTypeASRComplete MessageType = "asr_complete"   // ASR识别完成
    MsgTypeLLMText     MessageType = "llm_text"       // LLM回复文字
    MsgTypeLLMComplete MessageType = "llm_complete"   // LLM回复完成
    MsgTypeTTSAudio    MessageType = "tts_audio"      // TTS音频
    MsgTypeTTSComplete MessageType = "tts_complete"   // TTS完成
    MsgTypeStateUpdate MessageType = "state_update"   // 状态更新
    MsgTypeError       MessageType = "error"          // 错误
    MsgTypeInterrupt   MessageType = "interrupt"      // 打断
    MsgTypePing        MessageType = "ping"            // 心跳
    MsgTypePong        MessageType = "pong"            // 心跳响应
)

// Session 会话
type Session struct {
    ID              string       `json:"id"`               // 会话ID
    UserID          string       `json:"user_id"`          // 用户ID
    State           VoiceState   `json:"state"`            // 当前状态
    RecognizedText  string       `json:"recognized_text"`  // 识别文本
    ResponseText    string       `json:"response_text"`    // AI回复
    Context         []string     `json:"context"`          // 对话上下文
    CreatedAt       time.Time    `json:"created_at"`       // 创建时间
    LastActiveAt    time.Time    `json:"last_active_at"`   // 最后活跃时间
    IsInterrupted   bool         `json:"is_interrupted"`   // 是否已打断
    AudioBuffer     []byte       `json:"-"`                // 音频缓冲区
}

// WSMessage WebSocket消息
type WSMessage struct {
    Type      MessageType `json:"type"`
    SessionID string      `json:"session_id"`
    Data      interface{} `json:"data"`
    Timestamp int64       `json:"timestamp"`
}

// ASRResult ASR识别结果
type ASRResult struct {
    Text       string  `json:"text"`        // 识别文本
    IsFinal    bool    `json:"is_final"`    // 是否最终结果
    Confidence float64 `json:"confidence"`   // 置信度
}

// LLMResponse LLM回复
type LLMResponse struct {
    Text      string `json:"text"`       // 回复文本
    IsChunk   bool   `json:"is_chunk"`   // 是否为片段
    IsComplete bool  `json:"is_complete"` // 是否完成
}

// TTSAudio TTS音频
type TTSAudio struct {
    Data      []byte `json:"data"`       // 音频数据
    IsLast    bool   `json:"is_last"`    // 是否最后一片
    Timestamp int64  `json:"timestamp"`  // 时间戳
}

// ErrorResponse 错误响应
type ErrorResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}
```

### 5.3 会话管理器

```go
// domain/voice/session.go

package voice

import (
    "sync"
    "time"
)

// SessionManager 会话管理器
type SessionManager struct {
    sessions map[string]*Session
    mu       sync.RWMutex
}

// NewSessionManager 创建会话管理器
func NewSessionManager() *SessionManager {
    return &SessionManager{
        sessions: make(map[string]*Session),
    }
}

// Create 创建新会话
func (sm *SessionManager) Create(userID string) *Session {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    session := &Session{
        ID:            generateSessionID(),
        UserID:        userID,
        State:         StateIdle,
        CreatedAt:     time.Now(),
        LastActiveAt:  time.Now(),
        Context:       make([]string, 0),
    }
    sm.sessions[session.ID] = session
    return session
}

// Get 获取会话
func (sm *SessionManager) Get(sessionID string) (*Session, bool) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    session, ok := sm.sessions[sessionID]
    return session, ok
}

// Update 更新会话状态
func (sm *SessionManager) Update(sessionID string, state VoiceState) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    session, ok := sm.sessions[sessionID]
    if !ok {
        return ErrSessionNotFound
    }

    session.State = state
    session.LastActiveAt = time.Now()
    return nil
}

// Delete 删除会话
func (sm *SessionManager) Delete(sessionID string) {
    sm.mu.Lock()
    defer sm.mu.Unlock()

    delete(sm.sessions, sessionID)
}

// GetByUserID 根据用户ID获取活跃会话
func (sm *SessionManager) GetByUserID(userID string) *Session {
    sm.mu.RLock()
    defer sm.mu.RUnlock()

    for _, session := range sm.sessions {
        if session.UserID == userID && session.State != StateIdle {
            return session
        }
    }
    return nil
}

// generateSessionID 生成会话ID
func generateSessionID() string {
    return fmt.Sprintf("vs_%d_%d", time.Now().UnixNano(), rand.Int63())
}
```

---

## 6. 通信协议定义

### 6.1 WebSocket消息格式

#### 6.1.1 消息结构

```typescript
// 统一消息结构
interface WSMessage {
  type: MessageType;      // 消息类型
  sessionId: string;     // 会话ID
  data: unknown;         // 消息数据
  timestamp: number;     // 时间戳(ms)
}
```

#### 6.1.2 消息类型定义

| 消息类型 | 方向 | 数据结构 | 说明 |
|---------|------|---------|------|
| `asr_result` | Server→Client | `{text: string, isFinal: boolean}` | ASR识别结果 |
| `asr_complete` | Server→Client | `{text: string}` | ASR识别完成 |
| `llm_text` | Server→Client | `{text: string, isChunk: boolean}` | LLM回复片段 |
| `llm_complete` | Server→Client | `{text: string, fullText: string}` | LLM回复完成 |
| `tts_started` | Server→Client | `{timestamp: number}` | TTS音频流开始（DataChannel） |
| `tts_complete` | Server→Client | `{}` | TTS播放完成 |
| `state_update` | Server→Client | `{state: VoiceState}` | 状态更新 |
| `error` | Server→Client | `{code: number, message: string}` | 错误信息 |
| `interrupt` | Bidirectional | `{source: string}` | 打断命令 |
| `start` | Client→Server | `{mode: "wake_word" \| "manual"}` | 开始录音 |
| `stop` | Client→Server | `{}` | 停止录音 |
| `ping` | Client→Server | `{}` | 心跳 |
| `pong` | Server→Client | `{}` | 心跳响应 |

#### 6.1.3 消息流示例

**开始语音对话流程:**

```json
// 1. 客户端发送开始命令
{"type": "start", "sessionId": "vs_123456", "data": {"mode": "wake_word"}, "timestamp": 1712398400000}

// 2. 服务端返回状态更新
{"type": "state_update", "sessionId": "vs_123456", "data": {"state": "listening"}, "timestamp": 1712398400010}

// 3. 服务端返回ASR识别结果(流式)
{"type": "asr_result", "sessionId": "vs_123456", "data": {"text": "今天天气", "isFinal": false}, "timestamp": 1712398401000}
{"type": "asr_result", "sessionId": "vs_123456", "data": {"text": "今天天气怎么样", "isFinal": true}, "timestamp": 1712398401500}

// 4. 服务端返回ASR完成
{"type": "asr_complete", "sessionId": "vs_123456", "data": {"text": "今天天气怎么样"}, "timestamp": 1712398401510}

// 5. 服务端返回状态更新(思考中)
{"type": "state_update", "sessionId": "vs_123456", "data": {"state": "thinking"}, "timestamp": 1712398401520}

// 6. 服务端返回LLM回复(流式)
{"type": "llm_text", "sessionId": "vs_123456", "data": {"text": "今天", "isChunk": true}, "timestamp": 1712398402000}
{"type": "llm_text", "sessionId": "vs_123456", "data": {"text": "今天天气晴朗", "isChunk": true}, "timestamp": 1712398402500}

// 7. 服务端返回LLM完成
{"type": "llm_complete", "sessionId": "vs_123456", "data": {"text": "今天天气晴朗，适合外出。", "fullText": "今天天气晴朗，适合外出。"}, "timestamp": 1712398403000}

// 8. 服务端返回状态更新(播放中) + TTS音频通过DataChannel传输
{"type": "state_update", "sessionId": "vs_123456", "data": {"state": "playing"}, "timestamp": 1712398403010}

// 9. TTS音频通过DataChannel传输 (tts_started通知前端开始接收)
{"type": "tts_started", "sessionId": "vs_123456", "data": {"timestamp": 1712398403100}, "timestamp": 1712398403100}

// 10. 服务端返回TTS完成
{"type": "tts_complete", "sessionId": "vs_123456", "data": {}, "timestamp": 1712398405000}

// 11. 服务端返回状态更新(空闲)
{"type": "state_update", "sessionId": "vs_123456", "data": {"state": "idle"}, "timestamp": 1712398405010}
```

**打断流程:**

```json
// 1. 用户开始说话，客户端发送打断命令
{"type": "interrupt", "sessionId": "vs_123456", "data": {"source": "user_speech"}, "timestamp": 1712398404000}

// 2. 服务端返回状态更新(重新开始识别)
{"type": "state_update", "sessionId": "vs_123456", "data": {"state": "recognizing"}, "timestamp": 1712398404010}
```

### 6.2 DataChannel消息格式

**重要说明**: 统一音频传输路径
- **ASR输入音频** → DataChannel (前端 → 后端)
- **TTS输出音频** → DataChannel (后端 → 前端)
- **文字/控制消息** → WebSocket

```typescript
// DataChannel用于传输音频流（ASR输入 + TTS输出）

// ASR输入音频消息 (Client → Server)
interface DCASRMessage {
  type: 'asr_audio';               // 消息类型
  data: ArrayBuffer;                // PCM音频数据 (16bit 16kHz单声道)
  timestamp: number;                // 时间戳
}

// TTS输出音频消息 (Server → Client)
interface DCTTSMessage {
  type: 'tts_audio';               // 消息类型
  data: ArrayBuffer;                // PCM音频数据 (16bit 24kHz单声道)
  timestamp: number;                // 时间戳
  isLast: boolean;                  // 是否最后一片
}

// DataChannel控制消息
interface DCControlMessage {
  type: 'control';
  action: 'play' | 'pause' | 'stop' | 'flush';
  data?: unknown;
  timestamp: number;
}
```

### 6.3 音频数据格式

| 用途 | 格式 | 采样率 | 声道数 | 传输通道 |
|------|------|--------|-------|---------|
| ASR输入 | PCM 16bit | 16kHz | 单声道 | DataChannel |
| TTS输出 | PCM 16bit | 24kHz | 单声道 | DataChannel |
| 文字/控制 | JSON | - | - | WebSocket |

### 6.4 全双工通信通道设计

#### 6.4.1 双通道架构概述

系统采用双通道全双工通信机制，实现前端与后端之间的实时双向数据传输。

```
┌─────────────────────────────────────────────────────────────┐
│                    双通道全双工架构                          │
├─────────────────────────────────────────────────────────────┤
│  通道1: WebSocket (文字+控制)                               │
│    - 用途: ASR识别结果、LLM回复片段、状态更新、控制指令      │
│    - 特性: 可靠传输、按序到达、JSON格式                      │
│                                                             │
│  通道2: DataChannel (音频流)                                │
│    - 用途: 前端→后端(ASR音频), 后端→前端(TTS音频)           │
│    - 特性: UDP-like低延迟、可能乱序/丢包                    │
└─────────────────────────────────────────────────────────────┘
```

#### 6.4.2 通道职责划分

| 通道 | 传输内容 | 方向 | 特性 |
|------|---------|------|------|
| WebSocket | ASR结果、LLM文本片段、状态更新、控制指令 | 双向 | 可靠、有序、JSON |
| DataChannel | ASR音频流、TTS音频流 | 双向 | 低延迟、UDP模式 |

**关键设计点**:
1. **控制面 vs 数据面分离**: WebSocket 负责控制面，DataChannel 负责数据面
2. **传输效率优化**: 音频大数据走 DataChannel，避免 WebSocket 协议开销
3. **全双工通信**: 前后端可同时发送数据，前端可在播放TTS时同时发送下一轮ASR音频

#### 6.4.3 全双工时序说明

**真正的全双工**:
- 前端可以在播放TTS时，同时发送下一轮ASR音频
- 后端可以同时处理ASR输入和TTS输出

**打断检测机制**:
- 新音频开始时，DataChannel 发送 `audio_start` 标记
- 后端收到 `audio_start` 后立即中断当前TTS合成
- 前端本地同时停止TTS播放，实现零延迟响应

**并行处理**:
```
时间线 ───────────────────────────────────────────────────────────▶

前端:    |---录音---|---播放TTS---|---录音---|---播放TTS---|
              │         ▲            │         ▲
              │         │            │         │
              ▼         │            ▼         │
         DataChannel  DataChannel  DataChannel  DataChannel
              │         │            │         │
              ▼         │            ▼         │
后端:    |---ASR--|---TTS---|---ASR--|---TTS---|
              │                   │
              └─────── 可并行 ─────┘
```

#### 6.4.4 子流设计

DataChannel 支持一对多子流（subStream），用于区分不同类型的音频流：

```typescript
// 子流类型定义
type SubStreamType = 'asr_audio' | 'tts_audio' | 'audio_control';

// 扩展 DCAudioMessage 支持子流
interface DCAudioMessage {
  type: 'audio';
  subStream: SubStreamType;      // 子流类型
  timestamp: number;              // 采集/播放时间戳 (ms)
  sequence: number;              // 序列号，递增
  sampleRate: number;            // 采样率
  data: ArrayBuffer;             // PCM 数据
  isLast: boolean;               // 是否最后一片
}

// audio_start 控制消息
interface DCAudioStartMessage {
  type: 'control';
  action: 'audio_start';
  subStream: SubStreamType;
  timestamp: number;
}
```

#### 6.4.5 心跳机制

通过 WebSocket ping/pong 维持连接，定期发送心跳检测连接状态：

```typescript
// 前端心跳发送
class VoiceWebSocket {
  private pingInterval: number = 30000;  // 30秒

  startHeartbeat() {
    setInterval(() => {
      if (this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({
          type: 'ping',
          timestamp: Date.now()
        }));
      }
    }, this.pingInterval);
  }

  // 接收 pong 响应
  private handleMessage(msg: WSServerMessage) {
    if (msg.type === 'pong') {
      console.log('[WS] Heartbeat OK');
    }
  }
}
```

```go
// 后端心跳处理
func (h *VoiceHandler) HandlePing(sessionID string) {
    h.sendToClient(sessionID, &WSMessage{
        Type:      MsgTypePong,
        Timestamp: time.Now().UnixMilli(),
    })
}
```

---

### 6.5 UDP乱序处理方案

#### 6.5.1 问题分析

DataChannel 在 UDP模式下（`Ordered: false`）不保证顺序，可能出现：

| 问题场景 | 影响 |
|---------|------|
| TTS音频包乱序 | 播放杂音/音频错乱 |
| ASR音频包乱序 | 识别结果错误 |

#### 6.5.2 解决方案：时间戳+序列号机制

```typescript
// 音频消息增加元数据
interface DCAudioMessage {
  type: 'audio';
  timestamp: number;      // 采集/播放时间戳 (ms)
  sequence: number;       // 序列号，递增
  sampleRate: number;     // 采样率
  data: ArrayBuffer;      // PCM 数据
  isLast: boolean;        // 是否最后一片
}

// 接收端缓冲池
class AudioBuffer {
  private buffer: Map<number, DCAudioMessage> = new Map();
  private baseSequence: number = 0;
  private playheadTime: number = 0;
  private flushInterval: number = 50;  // 刷新间隔(ms)

  // 乱序处理：等待乱序包归位
  addPacket(msg: DCAudioMessage) {
    if (msg.sequence < this.baseSequence) {
      // 丢弃过期包
      return;
    }
    this.buffer.set(msg.sequence, msg);
    this.scheduleFlush();
  }

  private scheduleFlush() {
    // 使用 requestAnimationFrame 或 setTimeout 调度播放
    requestAnimationFrame(() => this.flush());
  }

  private flush() {
    // 按序列号顺序播放
    while (this.buffer.has(this.baseSequence)) {
      const msg = this.buffer.get(this.baseSequence);
      this.playAudio(msg);
      this.buffer.delete(this.baseSequence);
      this.baseSequence++;
    }
  }

  private playAudio(msg: DCAudioMessage) {
    // 播放音频数据
    this.audioContext.decodeAudioData(msg.data.slice(0))
      .then(buffer => this.playBuffer(buffer));
  }
}
```

#### 6.5.3 乱序容忍策略

| 场景 | 策略 | 配置 |
|------|------|------|
| TTS播放 | `Ordered: false` + 序列号，超过50ms未到的包直接丢弃 | 避免卡顿 |
| ASR输入 | 允许短暂缓冲（100ms窗口），超时未到重发 | 平衡延迟和完整性 |

**Jitter Buffer 配置**:
```typescript
interface JitterBufferConfig {
  maxBufferTime: number;     // 最大缓冲时间 (ms)
  latePacketTolerance: number; // 晚到包容忍时间 (ms)
}

// TTS 播放配置（低延迟优先）
const ttsJitterConfig: JitterBufferConfig = {
  maxBufferTime: 50,         // 50ms缓冲
  latePacketTolerance: 50,    // 超过50ms直接丢弃
};

// ASR 输入配置（完整性优先）
const asrJitterConfig: JitterBufferConfig = {
  maxBufferTime: 100,        // 100ms缓冲
  latePacketTolerance: 100,  // 等待100ms
};
```

#### 6.5.4 后端 Go 实现

```go
// AudioPacket 音频包结构
type AudioPacket struct {
    Sequence   uint32    `json:"sequence"`
    Timestamp  int64     `json:"timestamp"`   // 相对时间戳
    Data       []byte    `json:"data"`        // PCM
    SampleRate int       `json:"sampleRate"`
    IsLast     bool      `json:"isLast"`
}

// AudioBuffer 音频包缓冲管理器
type AudioBuffer struct {
    packets   map[uint32]*AudioPacket
    baseSeq   uint32
    lock      sync.Mutex
    maxSize   int                      // 最大缓冲数量
}

// NewAudioBuffer 创建音频缓冲
func NewAudioBuffer() *AudioBuffer {
    return &AudioBuffer{
        packets: make(map[uint32]*AudioPacket),
        baseSeq: 0,
        maxSize: 100,
    }
}

// Add 添加数据包
func (b *AudioBuffer) Add(pkt *AudioPacket) bool {
    b.lock.Lock()
    defer b.lock.Unlock()

    // 丢弃过期包
    if pkt.Sequence < b.baseSeq {
        return false
    }

    // 缓冲已满，强制刷新
    if len(b.packets) >= b.maxSize {
        b.flushLocked(nil)
    }

    b.packets[pkt.Sequence] = pkt
    return true
}

// Flush 刷新缓冲区，按序列号顺序处理
func (b *AudioBuffer) Flush(fn func(*AudioPacket)) {
    b.lock.Lock()
    defer b.lock.Unlock()
    b.flushLocked(fn)
}

func (b *AudioBuffer) flushLocked(fn func(*AudioPacket)) {
    for {
        pkt, ok := b.packets[b.baseSeq]
        if !ok {
            break
        }
        if fn != nil {
            fn(pkt)
        }
        delete(b.packets, b.baseSeq)
        b.baseSeq++
    }
}

// NextSequence 获取下一个序列号
func (b *AudioBuffer) NextSequence() uint32 {
    b.lock.Lock()
    defer b.lock.Unlock()
    return b.baseSeq + uint32(len(b.packets))
}

// TTSBuffer TTS专用缓冲（低延迟策略）
type TTSBuffer struct {
    *AudioBuffer
    latePacketWindow time.Duration // 晚到包窗口
}

// NewTTSBuffer 创建TTS缓冲
func NewTTSBuffer(latePacketWindowMs int) *TTSBuffer {
    return &TTSBuffer{
        AudioBuffer:      NewAudioBuffer(),
        latePacketWindow: time.Duration(latePacketWindowMs) * time.Millisecond,
    }
}

// AddWithTimeout 添加包，超过窗口期则丢弃
func (b *TTSBuffer) AddWithTimeout(pkt *AudioPacket, now int64) bool {
    b.lock.Lock()
    defer b.lock.Unlock()

    // 检查是否超时
    packetTime := pkt.Timestamp
    if now-packetTime > int64(b.latePacketWindow/time.Millisecond) {
        // 包已过期，丢弃
        return false
    }

    return b.Add(pkt)
}
```

#### 6.5.5 前端 TypeScript 实现

```typescript
// AudioBufferManager 音频缓冲管理器
class AudioBufferManager {
  private buffers: Map<string, AudioBuffer> = new Map();
  private audioContext: AudioContext;

  constructor(audioContext: AudioContext) {
    this.audioContext = audioContext;
  }

  // 获取或创建缓冲
  getOrCreateBuffer(streamId: string): AudioBuffer {
    if (!this.buffers.has(streamId)) {
      this.buffers.set(streamId, new AudioBuffer(this.audioContext, streamId));
    }
    return this.buffers.get(streamId)!;
  }

  // 处理收到的音频包
  handleAudioPacket(packet: DCAudioMessage) {
    const buffer = this.getOrCreateBuffer(packet.subStream);
    buffer.addPacket(packet);
  }

  // 清理缓冲
  clearBuffer(streamId: string) {
    const buffer = this.buffers.get(streamId);
    if (buffer) {
      buffer.clear();
      this.buffers.delete(streamId);
    }
  }
}

// AudioBuffer 单一音频流缓冲
class AudioBuffer {
  private buffer: Map<number, DCAudioMessage> = new Map();
  private baseSequence: number = 0;
  private audioContext: AudioContext;
  private sourceNode: AudioBufferSourceNode | null = null;

  constructor(audioContext: AudioContext, private streamId: string) {
    this.audioContext = audioContext;
  }

  addPacket(msg: DCAudioMessage) {
    if (msg.sequence < this.baseSequence) {
      // 丢弃过期包
      return;
    }

    this.buffer.set(msg.sequence, msg);

    // 尝试播放连续序列
    this.tryFlush();
  }

  private tryFlush() {
    while (this.buffer.has(this.baseSequence)) {
      const msg = this.buffer.get(this.baseSequence)!;
      this.playPacket(msg);
      this.buffer.delete(this.baseSequence);
      this.baseSequence++;
    }
  }

  private playPacket(msg: DCAudioMessage) {
    // 解码并播放
    this.audioContext.decodeAudioData(msg.data.slice(0))
      .then(audioBuffer => {
        const source = this.audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(this.audioContext.destination);
        source.start();
      })
      .catch(err => {
        console.error(`[AudioBuffer] Failed to decode audio:`, err);
      });
  }

  clear() {
    this.buffer.clear();
    this.baseSequence = 0;
    if (this.sourceNode) {
      this.sourceNode.stop();
      this.sourceNode = null;
    }
  }
}
```

---

## 7. 状态机设计

### 7.1 前端语音状态机

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Voice State Machine                             │
│                                                                          │
│                           ┌──────────────┐                              │
│                           │     IDLE     │                              │
│                           │    (空闲)     │                              │
│                           └──────┬───────┘                              │
│                                  │                                       │
│                    start()       │       stop()                         │
│                    wake_word     │       timeout                         │
│                                  ▼                                       │
│                           ┌──────────────┐                              │
│              ┌───────────▶│  LISTENING   │◀──────────────┐              │
│              │            │   (监听中)    │               │              │
│              │            └──────┬───────┘               │              │
│              │                   │                       │              │
│              │         VAD检测到  │  说话                   │              │
│              │         语音开始  │                       │              │
│              │                   ▼                       │              │
│              │            ┌──────────────┐                │              │
│    interrupt │            │RECOGNIZING   │                │ interrupt    │
│              │            │   (识别中)    │                │              │
│              │            └──────┬───────┘                │              │
│              │                   │                       │              │
│              │          VAD检测到│语音结束                 │              │
│              │                   │                       │              │
│              │                   ▼                       │              │
│              │            ┌──────────────┐                │              │
│              │            │   THINKING   │────────────────┘              │
│              │            │   (思考中)    │   interrupt (用户新说话)        │
│              │            └──────┬───────┘                                │
│              │                   │                                        │
│              │         LLM开始  │回复                                     │
│              │                   ▼                                        │
│              │            ┌──────────────┐                                │
│              │            │  RESPONDING  │                                │
│              │            │   (回复中)    │                                │
│              │            └──────┬───────┘                                │
│              │                   │                                        │
│              │         TTS开始  │合成                                     │
│              │                   ▼                                        │
│              │            ┌──────────────┐                                │
│              │            │   PLAYING    │                                │
│              │            │   (播放中)    │                                │
│              │            └──────┬───────┘                                │
│              │                   │                                        │
│              │         TTS播放   │完成                                     │
│              │                   │                                        │
│              │                   ▼                                        │
│              │            ┌──────────────┐                                │
│              └────────────│    IDLE     │◀─────────────────────────────┘
│   interrupt              │   (空闲)     │         error recovery
│   (error)                └──────────────┘
│                                                                          │
│  ═══════════════════════════════════════════════════════════════════   │
│                                                                          │
│  Events:                                                                 │
│    - start()              : 开始语音对话                                  │
│    - stop()               : 停止录音                                      │
│    - speech_start         : VAD检测到语音开始                             │
│    - speech_end           : VAD检测到语音结束                             │
│    - llm_start            : LLM开始生成回复                               │
│    - llm_end              : LLM回复完成                                   │
│    - tts_start            : TTS开始播放                                   │
│    - tts_end              : TTS播放完成                                   │
│    - interrupt(source)    : 打断事件                                      │
│    - timeout              : 超时事件                                      │
│    - error                : 错误事件                                      │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 7.2 状态转换矩阵

| 当前状态 | 事件 | 下一状态 | 动作 |
|---------|------|---------|------|
| IDLE | start() | LISTENING | 开始录音，发送start消息 |
| LISTENING | speech_start | RECOGNIZING | 切换到识别状态 |
| LISTENING | stop() | IDLE | 停止录音 |
| LISTENING | timeout | IDLE | 停止录音，回到空闲 |
| LISTENING | interrupt | IDLE | 停止录音，回到空闲 |
| RECOGNIZING | speech_end | THINKING | 发送识别完成，启动LLM |
| RECOGNIZING | interrupt | LISTENING | 重新开始识别 |
| RECOGNIZING | error | IDLE | 报告错误，回到空闲 |
| THINKING | llm_start | RESPONDING | 开始接收LLM回复 |
| THINKING | interrupt | LISTENING | 取消LLM请求，重新识别 |
| THINKING | error | IDLE | 报告错误 |
| RESPONDING | tts_start | PLAYING | 开始TTS播放 |
| RESPONDING | interrupt | LISTENING | 停止TTS，重新识别 |
| RESPONDING | llm_end | PLAYING | 直接进入播放状态 |
| RESPONDING | error | IDLE | 报告错误 |
| PLAYING | tts_end | IDLE | 播放完成，回到空闲 |
| PLAYING | interrupt | LISTENING | 停止TTS，重新识别 |
| PLAYING | error | IDLE | 报告错误 |
| ANY | error | IDLE | 错误恢复 |

### 7.3 后端状态管理

```go
// 状态转换规则在SessionManager中实现

// Transition 状态转换
func (sm *SessionManager) Transition(sessionID string, newState VoiceState) error {
    session, ok := sm.Get(sessionID)
    if !ok {
        return ErrSessionNotFound
    }

    // 验证状态转换合法性
    if !isValidTransition(session.State, newState) {
        return ErrInvalidStateTransition
    }

    session.State = newState
    session.LastActiveAt = time.Now()
    return nil
}

// isValidTransition 验证状态转换
func isValidTransition(from, to VoiceState) bool {
    validTransitions := map[VoiceState][]VoiceState{
        StateIdle:       {StateListening},
        StateListening:  {StateRecognizing, StateIdle},
        StateRecognizing: {StateThinking, StateListening, StateIdle},
        StateThinking:   {StateResponding, StateListening, StateIdle},
        StateResponding: {StatePlaying, StateListening, StateIdle},
        StatePlaying:    {StateIdle, StateListening},
        StateError:      {StateIdle},
    }

    allowed, ok := validTransitions[from]
    if !ok {
        return false
    }

    for _, s := range allowed {
        if s == to {
            return true
        }
    }
    return false
}
```

---

## 8. 打断机制实现

### 8.1 打断场景分类

| 场景 | 触发条件 | 打断来源 | 处理方式 |
|------|---------|---------|---------|
| AI播放中被打断 | 用户新说话 | user_speech | 立即停止TTS，清空缓冲区 |
| AI思考中被打断 | 用户新说话 | user_speech | 取消LLM请求，终止TTS合成 |
| AI回复中被打断 | 用户新说话 | user_speech | 停止LLM流式输出 |
| 用户主动中断 | 用户点击停止 | user_click | 停止所有流程 |
| 服务器打断 | 服务器发送interrupt | server_cmd | 根据服务器指令处理 |

### 8.2 打断处理流程

```
┌─────────────────────────────────────────────────────────────────┐
│                        Interrupt Flow                            │
│                                                                  │
│  ┌─────────────┐                                               │
│  │   打断触发   │                                               │
│  └──────┬──────┘                                               │
│         │                                                      │
│         ▼                                                      │
│  ┌─────────────┐                                               │
│  │ 记录打断事件 │  source, timestamp, current_state             │
│  └──────┬──────┘                                               │
│         │                                                      │
│         ▼                                                      │
│  ┌─────────────┐     ┌─────────────┐                          │
│  │ 停止音频采集 │────▶│ 停止TTS播放 │                          │
│  └──────┬──────┘     └──────┬──────┘                          │
│         │                   │                                   │
│         ▼                   ▼                                   │
│  ┌─────────────┐     ┌─────────────┐                          │
│  │ 取消ASR请求  │     │ 清空缓冲区   │                          │
│  └──────┬──────┘     └──────┬──────┘                          │
│         │                   │                                   │
│         └─────────┬─────────┘                                   │
│                   ▼                                            │
│           ┌─────────────┐                                        │
│           │ 发送打断命令 │  WebSocket                            │
│           │ 到服务端    │                                        │
│           └──────┬──────┘                                        │
│                  │                                              │
│                  ▼                                              │
│          ┌─────────────┐                                        │
│          │ 重置状态机   │                                        │
│          │ 回到LISTENING│                                        │
│          └─────────────┘                                        │
│                  │                                              │
│                  ▼                                              │
│          ┌─────────────┐                                        │
│          │ 继续音频采集 │                                        │
│          │ 等待用户说话 │                                        │
│          └─────────────┘                                        │
└─────────────────────────────────────────────────────────────────┘
```

### 8.3 打断响应时间优化

- **目标**: 打断响应 < 300ms
- **优化策略**:
  1. 前端本地立即停止TTS播放（0ms延迟）
  2. 并行发送打断命令到服务端
  3. 服务端收到打断后立即取消正在进行的请求
  4. 使用context.WithTimeout控制取消延迟

```typescript
// 前端打断处理示例
class InterruptHandler {
  private voiceService: IVoiceService;
  private ttsPlayer: TTSPlayer;

  async handleInterrupt(source: InterruptSource): Promise<void> {
    const startTime = performance.now();

    // 1. 立即停止本地TTS播放（零延迟响应）
    this.ttsPlayer.stop();
    this.ttsPlayer.clearBuffer();

    // 2. 立即更新UI状态
    this.voiceService.updateState(VoiceState.LISTENING);

    // 3. 发送打断命令到服务端
    this.voiceService.sendInterrupt(source);

    // 4. 重新开始音频采集
    this.audioCapture.start();

    const latency = performance.now() - startTime;
    console.log(`[Interrupt] Handled in ${latency}ms`);

    // 记录性能指标
    this.reportMetrics({ interruptLatency: latency });
  }
}
```

### 8.4 服务端打断处理

```go
// 打断处理
func (h *VoiceHandler) HandleInterrupt(sessionID string, source string) error {
    session, ok := h.sessionManager.Get(sessionID)
    if !ok {
        return ErrSessionNotFound
    }

    // 标记会话已被打断
    session.IsInterrupted = true

    // 取消当前正在进行的请求
    switch session.State {
    case StateThinking:
        // 取消LLM请求
        if session.llmCancel != nil {
            session.llmCancel()
        }
    case StateResponding, StatePlaying:
        // 取消TTS请求
        if session.ttsCancel != nil {
            session.ttsCancel()
        }
        // 停止向DataChannel发送音频
        if session.dc != nil {
            session.dc.SendControl("stop", nil)
        }
    }

    // 重置会话状态
    session.State = StateListening
    session.RecognizedText = ""
    session.ResponseText = ""

    return nil
}
```

---

## 9. API接口设计

### 9.1 WebSocket连接

```
Endpoint: ws://host:port/ws/voice
Protocol: WebSocket
Authentication: X-Token header
```

### 9.2 REST API

#### 9.2.1 语音配置

```
GET /api/v1/voice/config

Response:
{
  "code": 0,
  "data": {
    "asr": {
      "provider": "cosyvoice",
      "model": "small",
      "sample_rate": 16000
    },
    "tts": {
      "provider": "qwen3-tts",
      "model": "small",
      "sample_rate": 24000
    },
    "vad": {
      "enabled": true,
      "sensitivity": 2
    },
    "wake_word": {
      "enabled": true,
      "keyword": "小爱同学"
    }
  }
}
```

#### 9.2.2 语音历史

```
GET /api/v1/voice/history?page=1&page_size=20

Response:
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "vs_xxx",
        "user_text": "今天天气怎么样",
        "ai_text": "今天天气晴朗",
        "duration": 3500,
        "created_at": "2026-04-06T10:00:00Z"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 9.3 WebSocket消息API

#### 9.3.1 客户端发送消息

| 消息类型 | 方向 | 数据结构 | 说明 |
|---------|------|---------|------|
| start | Client→Server | `{mode: "wake_word" \| "manual"}` | 开始语音对话 |
| stop | Client→Server | `{}` | 停止语音对话 |
| interrupt | Client→Server | `{source: string}` | 发送打断信号 |
| ping | Client→Server | `{}` | 心跳 |

#### 9.3.2 服务端推送消息

| 消息类型 | 方向 | 数据结构 | 说明 |
|---------|------|---------|------|
| asr_result | Server→Client | `{text: string, isFinal: boolean}` | ASR识别结果 |
| asr_complete | Server→Client | `{text: string}` | ASR识别完成 |
| llm_text | Server→Client | `{text: string, isChunk: boolean}` | LLM回复片段 |
| llm_complete | Server→Client | `{text: string}` | LLM回复完成 |
| tts_started | Server→Client | `{timestamp: number}` | TTS音频流开始（通过DataChannel传输） |
| tts_complete | Server→Client | `{}` | TTS播放完成 |
| state_update | Server→Client | `{state: string}` | 状态更新 |
| error | Server→Client | `{code: number, message: string}` | 错误信息 |
| pong | Server→Client | `{}` | 心跳响应 |

### 9.4 DataChannel API

```
Label: voice-audio
Protocol: UDP-like (WebRTC DataChannel)

传输内容:
  - ASR输入音频: Client → Server (PCM 16bit 16kHz单声道)
  - TTS输出音频: Server → Client (PCM 16bit 24kHz单声道)
控制: {type: "control", action: "stop", timestamp: number}
```

---

## 10. 数据流设计

### 10.1 完整语音对话数据流

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                              完整数据流                                          │
│                                                                                 │
│  USER SPEECH                                                                     │
│      │                                                                            │
│      ▼                                                                            │
│  ┌────────────────────────────────────────────────────────────────────────────┐   │
│  │                            FRONTEND                                         │   │
│  │                                                                            │   │
│  │  [Microphone] ──▶ [AudioContext] ──▶ [NoiseSuppression]                   │   │
│  │                                          │                                 │   │
│  │                                          ▼                                 │   │
│  │                                   [VAD Detector]                            │   │
│  │                                          │                                 │   │
│  │                                          │ isSpeaking=true                 │   │
│  │                                          ▼                                 │   │
│  │                              [State: LISTENING → RECOGNIZING]               │   │
│  │                                          │                                 │   │
│  │                                          ▼                                 │   │
│  │                              [AudioChunk × N] ──▶ [WebSocket]              │   │
│  │                                          │                                 │   │
│  └──────────────────────────────────────────│─────────────────────────────────┘   │
│                                             │                                      │
└─────────────────────────────────────────────│──────────────────────────────────────┘
                                              │
                                              ▼
┌──────────────────────────────────────────────────────────────────────────────────┐
│                               BACKEND                                             │
│                                                                                    │
│  [WebSocket] ──▶ [Session Manager] ──▶ [ASR Service]                              │
│                                              │                                      │
│                                              │ streaming results                    │
│                                              ▼                                      │
│                                       [CosyVoice ASR]                               │
│                                              │                                      │
│                                              │ final text                           │
│                                              ▼                                      │
│                                       [LLM Service]                                 │
│                                              │                                      │
│                                              │ streaming text                       │
│                                              ▼                                      │
│  [WebSocket] ◀──────────────────────────────┴──────────────────────────────────── │
│  (asr_result, llm_text)                                                               │
│                                                                                    │
│                                              │                                      │
│                                              │ full text                            │
│                                              ▼                                      │
│                                       [TTS Service]                                │
│                                              │                                      │
│                                              │ streaming audio                      │
│                                              ▼                                      │
│  [DataChannel] ◀────────────────────────────┴───────────────────────────────────── │
│  (tts_audio)                                                                              │
│                                                                                    │
└──────────────────────────────────────────────────────────────────────────────────┘
                                              │
                                              ▼
┌──────────────────────────────────────────────────────────────────────────────────┐
│                               FRONTEND (TTS Playback)                              │
│                                                                                    │
│  [DataChannel] ──▶ [Audio Buffer] ──▶ [Web Audio API] ──▶ [Speaker]                │
│                                                                                    │
│  [WebSocket] ──▶ (tts_complete) ──▶ [State: PLAYING → IDLE]                        │
│                                                                                    │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### 10.2 音频数据流

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                           Audio Data Flow                                       │
│                                                                                 │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐            │
│  │  Microphone   │         │   Browser    │         │   Backend    │            │
│  │  (getUserMedia)│         │  AudioContext │         │  ASR Service │            │
│  └──────┬───────┘         └──────┬───────┘         └──────┬───────┘            │
│         │                        │                        │                     │
│         │ PCM 16kHz              │ PCM 16kHz              │                      │
│         │ 16bit mono             │ 16bit mono             │                      │
│         ▼                        ▼                        ▼                     │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐            │
│  │ MediaStream  │────────▶│ ScriptProcessor │──────▶│ WebSocket    │            │
│  │ AudioTrack   │         │ /AudioWorklet   │       │ Binary       │            │
│  └──────────────┘         └──────────────┘         └──────────────┘            │
│                                                              │                     │
│                                                              ▼                     │
│                                                       ┌──────────────┐            │
│                                                       │ CosyVoice    │            │
│                                                       │ Small        │            │
│                                                       └──────────────┘            │
│                                                                                 │
│  ───────────────────────────────────────────────────────────────────────────    │
│                                                                                 │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐            │
│  │  LLM Service │         │   Backend    │         │   Browser    │            │
│  │              │         │  TTS Service │         │  Audio API   │            │
│  └──────┬───────┘         └──────┬───────┘         └──────┬───────┘            │
│         │                        │                        │                     │
│         │ text                    │ PCM 24kHz             │ PCM 24kHz          │
│         │                         │ 16bit mono            │ 16bit mono         │
│         ▼                        ▼                        ▼                     │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐            │
│  │ Qwen3-TTS    │────────▶│ DataChannel  │────────▶│ AudioBuffer  │            │
│  │              │         │ (UDP-like)   │         │ SourceNode   │            │
│  └──────────────┘         └──────────────┘         └──────┬───────┘            │
│                                                            │                     │
│                                                            ▼                     │
│                                                     ┌──────────────┐            │
│                                                     │  Speaker     │            │
│                                                     └──────────────┘            │
└────────────────────────────────────────────────────────────────────────────────┘
```

### 10.3 文字数据流

```
┌────────────────────────────────────────────────────────────────────────────────┐
│                           Text Data Flow                                        │
│                                                                                 │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐            │
│  │  User Speech │         │   Frontend   │         │   Backend    │            │
│  └──────┬───────┘         └──────┬───────┘         └──────┬───────┘            │
│         │                        │                        │                     │
│         │ audio                  │ websocket             │                     │
│         ▼                        ▼                        ▼                     │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐            │
│  │ ASR Engine   │────────▶│ WS Message   │────────▶│ Session      │            │
│  │ (CosyVoice) │  text   │ asr_result   │  text   │ Manager      │            │
│  └──────────────┘         └──────────────┘         └──────┬───────┘            │
│                                                            │                     │
│                                                            ▼                     │
│                                                      ┌──────────────┐            │
│                                                      │ LLM Service  │            │
│                                                      │ (Streaming)  │            │
│                                                      └──────┬───────┘            │
│                                                            │                     │
│                                                            │ llm_text            │
│                                                            ▼                     │
│                                                      ┌──────────────┐            │
│                                                      │ WS Message   │            │
│                                                      │ llm_text     │            │
│                                                      └──────┬───────┘            │
│                                                            │                     │
│                                                            ▼                     │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐            │
│  │  AI Reply    │         │   Frontend   │         │   Backend    │            │
│  │  Display     │◀────────│ Vue Component│◀────────│ WS Message   │            │
│  │              │  text   │              │  text   │ llm_complete │            │
│  └──────────────┘         └──────────────┘         └──────────────┘            │
└────────────────────────────────────────────────────────────────────────────────┘
```

---

## 11. 性能指标

### 11.1 延迟指标

| 指标 | 目标值 | 测量方式 |
|------|-------|---------|
| 端到端延迟 | < 2000ms | 用户说话结束 → TTS播放开始 |
| 打断响应时间 | < 300ms | 打断信号发送 → TTS停止播放 |
| ASR识别延迟 | < 500ms | 语音结束 → 文字输出 |
| TTS合成延迟 | < 300ms | 文字输入 → 首音频包输出 |
| 首音频延迟 | < 800ms | 语音结束 → 听到TTS音频 |

### 11.1.1 Pipeline并行说明

端到端延迟 < 2s 的关键在于**流式架构下的Pipeline并行处理**：

```
时间线 (ms)
0        200     400     600     800     1000    1200    1400    1600    1800    2000
|        |        |        |        |        |        |        |        |        |
├────────┴────────┴────────┴────────┴────────┴────────┴────────┴────────┴────────┤
│                         Pipeline 并行处理                                          │
│                                                                                   │
│  ASR (流式):    |██████████████████████████████████████|                          │
│                      ↓ ASR首结果(200ms)                                          │
│  LLM (流式):         |████████████████████████████████████████████|              │
│                            ↓ LLM首token(400ms)                                    │
│  TTS (流式):              |████████████████████████████████████████████|          │
│                                  ↓ TTS首音频(600ms)                               │
│  播放:                        |████████████████████████████████|                   │
│                                      ↓ 播放开始(约600ms)                           │
└───────────────────────────────────────────────────────────────────────────────────┘
```

**关键优化点**:

1. **ASR → LLM 并行**: ASR一旦完成（识别到语音结束），立即启动LLM，无需等待完整句子
2. **LLM → TTS 并行**: LLM开始输出文字后，立即启动TTS合成（流式到流式）
3. **TTS 首包输出**: LLM生成少量文字后TTS即可开始合成，首音频包延迟约 400-600ms

**真正的端到端延迟计算**:
- 旧方式: ASR延迟(500ms) + LLM延迟(1000ms) + TTS延迟(300ms) = 1800ms (串行累加)
- 新方式: LLM首token到TTS首音频包 = **400-600ms** (流式并行)

**实际端到端延迟分解**:
- 用户语音结束 → ASR检测到结束: ~200ms (VAD延迟)
- ASR → LLM 启动: ~50ms
- LLM 首token输出: ~200-400ms
- TTS 首音频输出: ~100-200ms
- **总计**: ~550-850ms (远低于2000ms目标)

### 11.2 性能优化策略

1. **ASR流式输出**: 边识别边发送，减少等待时间
2. **LLM流式输出**: 逐词/逐句输出，加快首响应
3. **TTS流式合成**: 流式输出音频，无需等待完整合成
4. **并行处理**: ASR→LLM可并行启动（流式）
5. **音频预缓冲**: TTS播放前预缓冲200ms，减少卡顿
6. **打断优先级**: 打断信号优先处理

### 11.3 性能监控

```typescript
interface PerformanceMonitor {
  // 记录各阶段耗时
  recordMetric(name: string, duration: number): void;

  // 获取当前性能数据
  getMetrics(): PerformanceMetrics;

  // 上报性能数据
  reportMetrics(): void;
}

// 性能指标采集点
const METRICS = {
  ASR_LATENCY: 'asr_latency',           // ASR识别延迟
  LLM_LATENCY: 'llm_latency',            // LLM响应延迟
  TTS_LATENCY: 'tts_latency',           // TTS合成延迟
  E2E_LATENCY: 'e2e_latency',           // 端到端延迟
  INTERRUPT_LATENCY: 'interrupt_latency', // 打断响应延迟
};
```

---

## 12. 异常处理

### 12.1 异常场景分类

| 异常类型 | 错误码 | 说明 | 处理策略 |
|---------|-------|------|---------|
| 网络断开 | 1001 | WebSocket断开 | 自动重连，显示提示 |
| ASR失败 | 2001 | 语音识别服务不可用 | 回退到手动输入 |
| LLM失败 | 3001 | LLM API不可用/超时 | API超时5s → 降级回复"服务繁忙，请稍后再试" |
| TTS失败 | 4001 | 语音合成服务不可用 | 显示文字回复 |
| 麦克风权限 | 5001 | 麦克风权限被拒绝 | 引导用户开启权限 |
| 会话超时 | 6001 | 长时间无响应 | 自动结束会话 |
| 服务端错误 | 9001 | 服务器内部错误 | 显示错误，提示重试 |

### 12.2 异常处理流程

```typescript
// 统一异常处理
class VoiceErrorHandler {
  handleError(error: VoiceError): ErrorAction {
    switch (error.code) {
      case 1001: // 网络断开
        return this.handleNetworkError(error);
      case 2001: // ASR失败
        return this.handleASRError(error);
      case 3001: // LLM失败
        return this.handleLLMError(error);
      case 4001: // TTS失败
        return this.handleTTSError(error);
      case 5001: // 权限问题
        return this.handlePermissionError(error);
      default:
        return this.handleGenericError(error);
    }
  }

  private handleNetworkError(error: VoiceError): ErrorAction {
    // 断开连接，提示用户，等待重连
    this.voiceService.disconnect();
    this.ui.showToast('网络连接断开，正在重连...');
    return { retry: true, retryDelay: 3000 };
  }

  private handleASRError(error: VoiceError): ErrorAction {
    // 停止当前识别，回退到手动输入模式
    this.voiceService.stopRecording();
    this.ui.showFallbackInput();
    return { retry: false };
  }

  private handleTTSError(error: VoiceError): ErrorAction {
    // 停止TTS，改为显示文字
    this.ttsPlayer.stop();
    this.ui.showTextResponse(this.currentResponse);
    return { retry: false };
  }
}
```

### 12.3 重试机制

```go
// 后端重试策略
type RetryConfig struct {
    MaxRetries     int
    RetryDelay     time.Duration
    BackoffMultiplier float64
}

// 重试示例
func withRetry(fn func() error, config RetryConfig) error {
    var lastErr error
    delay := config.RetryDelay

    for i := 0; i < config.MaxRetries; i++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
            time.Sleep(delay)
            delay = time.Duration(float64(delay) * config.BackoffMultiplier)
        }
    }
    return lastErr
}
```

### 12.4 熔断器配置

```go
// 熔断器配置 - 使用gobreaker库
import "github.com/sony/gobreaker"

// 模型服务熔断器配置
type CircuitBreakerConfig struct {
    Name:        string        // 熔断器名称
    MaxRequests: uint32        // 半开状态下最大请求数
    Interval:    time.Duration // 统计周期
    Timeout:     time.Duration // 熔断开启后的恢复超时
}

var asrCircuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "asr-service",
    MaxRequests: 3,             // 半开状态下最多3个请求
    Interval:    10 * time.Second,  // 10秒内恢复一次统计
    Timeout:     30 * time.Second,  // 熔断开启30秒后尝试恢复
})

var llmCircuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "llm-service",      // Qwen API (dashscope)
    MaxRequests: 2,
    Interval:    10 * time.Second,
    Timeout:     60 * time.Second,  // API恢复时间
})

var ttsCircuitBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
    Name:        "tts-service",
    MaxRequests: 3,
    Interval:    10 * time.Second,
    Timeout:     30 * time.Second,
})

// 使用熔断器包装服务调用
func callWithCircuitBreaker[T any](
    cb *gobreaker.CircuitBreaker,
    fn func() (T, error),
) (T, error) {
    result, err := cb.Execute(func() (interface{}, error) {
        return fn()
    })
    if err != nil {
        return *new(T), err
    }
    return result.(T), nil
}
```

### 12.5 模型服务健康检查

```yaml
# docker-compose.yml 健康检查配置
services:
  backend:
    # ... 其他配置
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  cosyvoice:
    # ... 其他配置
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8001/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

  qwen3-tts:
    # ... 其他配置
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8003/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

# 注意：LLM使用Qwen API（外部服务），无需健康检查配置
```

### 12.6 服务降级策略

| 服务 | 正常行为 | 降级行为 | 触发条件 |
|------|---------|---------|---------|
| ASR | 语音识别 | 回退到手动文本输入 | 连续失败3次 / 熔断开启 |
| LLM | AI对话 | 返回"服务繁忙，请稍后再试" | 响应超时5s / 熔断开启 |
| TTS | 语音合成 | 显示文字回复（不播放音频） | 连续失败3次 / 熔断开启 |
| 全部不可用 | - | 显示错误提示，提示用户检查网络 | 所有服务均不可达 |

---

## 13. 部署架构

### 13.1 Docker容器布局

```
┌─────────────────────────────────────────────────────────────────┐
│                        Docker Compose                            │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │  frontend   │  │   backend   │  │    mysql    │              │
│  │  :3000      │  │   :8080     │  │   :3306     │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                                                                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐              │
│  │ cosyvoice   │  │  qwen3-tts  │  │   redis     │              │
│  │  :8001      │  │   :8003     │  │   :6379     │              │
│  └─────────────┘  └─────────────┘  └─────────────┘              │
│                                                                  │
│  ════════════════════════════════════════════════════════════   │
│                        外部服务 (不通过Docker部署)                  │
│  ┌─────────────┐                                                 │
│  │ Qwen API    │  (dashscope - 阿里云百炼)                        │
│  │  (qwen-plus)│                                                 │
│  └─────────────┘                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 13.2 Docker配置

#### 13.2.1 后端 Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -o server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY config/ ./config/
EXPOSE 8080
CMD ["./server"]
```

#### 13.2.2 docker-compose.yml (Voice Dialogue部分)

```yaml
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      - CONFIG_PATH=/app/config/config.yaml
      - ASR_HOST=cosyvoice:8001
      - TTS_HOST=qwen3-tts:8003
      # LLM使用Qwen API（外部服务），通过配置获取API Key和BaseURL
      - DASHSCOPE_API_KEY=${DASHSCOPE_API_KEY}
      - LLM_MODEL=qwen-plus
    depends_on:
      cosyvoice:
        condition: service_healthy
      qwen3-tts:
        condition: service_healthy
      mysql:
        condition: service_started
      redis:
        condition: service_started
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 2G
        reservations:
          memory: 1G
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  cosyvoice:
    image: cosyvoice/small:latest
    ports:
      - "8001:8001"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
          cpus: '2'
        reservations:
          memory: 2G
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8001/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

  qwen3-tts:
    image: qwen3-tts/small:latest
    ports:
      - "8003:8003"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 4G
          cpus: '2'
        reservations:
          memory: 2G
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8003/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

  # 注意：LLM使用Qwen API（外部服务），无需在docker-compose中部署

  mysql:
    image: mysql:8.0
    ports:
      - "3306:3306"
    environment:
      - MYSQL_ROOT_PASSWORD=your-password
      - MYSQL_DATABASE=voice_assistant
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 1G
    volumes:
      - mysql_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    restart: unless-stopped
    deploy:
      resources:
        limits:
          memory: 512M
    volumes:
      - redis_data:/data

volumes:
  mysql_data:
  redis_data:
```

### 13.3 端口映射

| 服务 | 容器端口 | 主机端口 | 协议 | 容器间通信端口 |
|------|---------|---------|------|---------------|
| frontend | 3000 | 3000 | HTTP | - |
| backend | 8080 | 8080 | HTTP/WSS | - |
| mysql | 3306 | 3306 | TCP | 3306 |
| redis | 6379 | 6379 | TCP | 6379 |
| cosyvoice | 8001 | 8001 | gRPC | 8001 |
| qwen3-tts | 8003 | 8003 | gRPC | 8003 |

**注意**: LLM使用Qwen API（外部服务），无需在Docker中部署。backend通过环境变量`DASHSCOPE_API_KEY`配置API认证。

---

## 14. 版本记录

| 版本 | 日期 | 作者 | 变更内容 |
|------|------|------|---------|
| v1.0 | 2026-04-06 | Business Architect | 初始版本，基于PRD v1.0设计 |
| v1.1 | 2026-04-06 | Business Architect | 架构专家评审修订：<br>- 问题1: 统一TTS音频传输路径为DataChannel<br>- 问题2: 修正LLM容器间通信端口配置<br>- 问题3: 增加Pipeline并行处理说明<br>- 问题4: 补充GPU和资源限制配置<br>- 问题5: 增加熔断器和健康检查配置<br>- 问题6: 明确DataChannel音频格式<br>- 问题7: 补充唤醒词检测架构决策 |
| v1.2 | 2026-04-06 | Business Architect | 架构专家评审修订（第3轮）：<br>- 移除vLLM本地部署，改用Qwen API（dashscope）调用qwen-plus模型<br>- 删除docker-compose.yml中的llm服务定义及相关GPU配置<br>- 更新backend环境变量：LLM_HOST改为DASHSCOPE_API_KEY和LLM_MODEL<br>- 更新系统架构图，标注LLM为外部服务<br>- 更新异常处理：LLM失败处理策略改为API超时降级回复<br>- 更新端口映射表格，移除llm服务<br>- 更新熔断器配置注释 |
| v1.4 | 2026-04-06 | Business Architect | 优化唤醒词检测架构：<br>- 改为前端独立方案，移除后端验证步骤<br>- 唤醒成功后建立WebSocket+DataChannel连接<br>- 连接成功后Alert提醒"连接成功，开启对话"<br>- 用户结束对话后Alert提醒"对话已结束，连接已关闭" |
| v1.5 | 2026-04-06 | Business Architect | 后端分层规范重构：<br>- 移除service层，统一使用logic层做业务编排<br>- 更新目录结构：api/component/domain/logic/config/router<br>- 明确各层职责：api(入口)、component(技术封装)、domain(领域模型)、logic(业务编排)<br>- Component vs Logic区分：component只做SDK调用，logic实现业务逻辑 |
| v1.3 | 2026-04-06 | Business Architect | 补充技术架构设计：<br>- 新增2.5节：Go WebRTC库推荐（pion/webrtc）<br>- 新增6.4节：全双工通信通道设计（双通道架构、WebSocket+DataChannel）<br>- 新增6.5节：UDP乱序处理方案（时间戳+序列号机制） |

---

## 附录

### A.1 参考文献

- [CosyVoice GitHub](https://github.com/Sxdlj/CosyVoice)
- [SenseVoice](https://www.modelscope.cn/models/iic/sensevoice)
- [Qwen3-TTS](https://qwenlm.github.io/)
- [Qwen (DashScope API)](https://dashscope.console.aliyun.com/)
- [WebRTC VAD](https://chromium.googlesource.com/external/webrtc/+/master/modules/audio_processing/vad/)
- [Web Audio API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Audio_API)

### A.2 术语表

| 术语 | 说明 |
|------|------|
| ASR | Automatic Speech Recognition，自动语音识别 |
| TTS | Text-to-Speech，文本转语音 |
| VAD | Voice Activity Detection，语音活动检测 |
| AEC | Acoustic Echo Cancellation，回声消除 |
| WSS | WebSocket Secure，WebSocket安全协议 |
| DataChannel | WebRTC数据通道，用于传输任意数据 |
| Wake Word | 唤醒词，用于激活语音助手 |
| E2E | End-to-End，端到端 |
| API | Application Programming Interface，应用程序编程接口 |
| DashScope | 阿里云百炼平台，提供Qwen等大模型API服务 |

---

**文档状态**: 待架构专家评审（第3轮）
**评审轮次**: 第3轮
