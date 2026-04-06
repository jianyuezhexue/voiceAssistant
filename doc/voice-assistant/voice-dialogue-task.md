# 语音对话功能开发任务清单

**版本**: v1.0
**创建日期**: 2026-04-06
**关联文档**: voice-dialogueTec.md (v1.3), voice-assistant-prd-v1.0.html

---

## 一、需求覆盖验证

### 1.1 用户操作流程与功能点映射

```
用户操作流程                              对应功能点
────────────────────────────────────────────────────────────────
1. 唤醒词触发"小爱同学"      →  [前端] WakeWordDetector (前端独立完成)
2. 系统播放唤醒成功音效      →  [前端] TTSPlayer (本地音效) + 状态切换
3. 进入"监听中"状态          →  [前端] StateMachine (LISTENING)
4. 用户开始说话              →  [前端] AudioCapture + VADClient (语音开始检测)
5. 进入"识别中"状态          →  [前端] StateMachine (RECOGNIZING) + 实时字幕
6. VAD检测到语音结束         →  [前端] VADClient (语音结束检测)
7. 进入"思考中"状态          →  [前端] StateMachine (THINKING)
8. AI生成回复文字            →  [后端] LLMService (Qwen API) + 流式输出
9. 文字显示在对话框          →  [前端] WebSocketClient 接收 + DialogueBubble
10. 进入"回复中/播放中"状态  →  [前端] StateMachine (RESPONDING/PLAYING)
11. TTS音频流式传输          →  [后端] TTSService + DataChannel
12. 前端播放TTS音频          →  [前端] TTSPlayer + 波形动画
13. 播放完成，回到监听       →  [前端] StateMachine (LISTENING)

打断流程:
────────────────────────────────────────────────────────────────
用户说话打断AI播放   →  [前端] InterruptHandler + AudioCapture新音频检测
                    →  [后端] SessionManager 中断当前TTS
                    →  [前端] TTSPlayer.stop() + 状态切回RECOGNIZING
```

### 1.2 需求完整性检查

| PRD需求 | 覆盖状态 | 实现任务 |
|---------|---------|---------|
| Always-on持续监听 | ✅ 已覆盖 | Task-F-01 音频采集 |
| 唤醒词"小爱同学"触发 | ✅ 已覆盖 | Task-F-02 唤醒词检测 |
| 点击触发语音对话 | ✅ 已覆盖 | Task-F-01 + Task-F-06 |
| WebRTC VAD检测 | ✅ 已覆盖 | Task-F-03 VAD检测 |
| 降噪+AEC | ✅ 已覆盖 | Task-F-01 音频前处理 |
| 实时ASR流式输出 | ✅ 已覆盖 | Task-B-06 ASR客户端 |
| 用户打断AI | ✅ 已覆盖 | Task-F-05 打断机制 + Task-B-09 打断处理 |
| AI回复文字→TTS播放 | ✅ 已覆盖 | Task-B-07 LLM客户端 + Task-B-08 TTS客户端 |
| 5种UI状态展示 | ✅ 已覆盖 | Task-F-04 状态机 + Task-F-09 UI组件 |
| 波形动画 | ✅ 已覆盖 | Task-F-10 波形动画 |
| 实时字幕 | ✅ 已覆盖 | Task-F-09 DialogueBubble |
| 双通道全双工 | ✅ 已覆盖 | Task-F-06 + Task-F-07 + Task-B-05 |
| UDP乱序处理 | ✅ 已覆盖 | Task-F-08 乱序缓冲 + Task-B-04 音频缓冲 |
| 熔断器+异常处理 | ✅ 已覆盖 | Task-B-10 异常处理 |

---

## 二、人工测试主流程

### 2.1 测试前准备

| 步骤 | 操作 | 预期结果 |
|------|------|---------|
| 1 | 启动所有 Docker 服务: `cd docker && docker-compose up -d` | 所有容器正常运行 |
| 2 | 打开浏览器，访问 VoiceAssistant 首页 | 页面正常加载 |
| 3 | 打开浏览器控制台，切换到 Network 标签 | Network 面板已就绪 |
| 4 | 检查 WebSocket 连接: `ws://localhost:8080/ws/voice` | 连接状态为 101 Switching Protocols |
| 5 | 检查麦克风权限 | 浏览器弹出权限请求，点击允许 |

---

### 2.2 基础语音对话测试

| 步骤 | 操作 | 预期结果 | 验证点 |
|------|------|---------|-------|
| 1 | 点击语音按钮（或说"小爱同学"） | 按钮变为录音状态，状态显示"监听中" | ✅ UI 状态切换 |
| 2 | 观察连接建立 | Alert 弹窗"连接成功，开启对话" | ✅ 连接成功提示 |
| 3 | 对麦克风说"今天天气怎么样" | 实时字幕显示识别文字，状态变为"识别中" | ✅ 实时字幕上屏 |
| 4 | 说完后等待 | 状态变为"思考中"，对话框显示用户消息 | ✅ 状态切换、对话气泡 |
| 5 | 等待 AI 生成回复 | 对话框显示 AI 回复文字，流式逐字显示 | ✅ LLM 流式输出 |
| 6 | AI 开始回复 | 状态变为"播放中"，TTS 音频播放，波形动画显示 | ✅ TTS 播放、波形动画 |
| 7 | 播放完成后 | 状态回到"监听中" | ✅ 状态恢复 |

**人工测试记录表**:
```
日期: ___________  测试人: ___________  结果: □ 通过 □ 失败

1. 唤醒触发: □  耗时: ___ms
2. 连接成功Alert: □
3. 识别延迟: □  耗时: ___ms
4. 端到端延迟: □  耗时: ___ms (目标 < 2000ms)
5. 打断响应: □  耗时: ___ms (目标 < 300ms)
```

---

### 2.3 连接关闭测试

| 步骤 | 操作 | 预期结果 | 验证点 |
|------|------|---------|-------|
| 1 | 完成一轮语音对话 | 状态回到"监听中" | - |
| 2 | 点击"结束对话"按钮 | WebSocket + DataChannel 连接关闭 | ✅ 连接关闭 |
| 3 | 观察 Alert 弹窗 | Alert 弹窗"对话已结束，连接已关闭" | ✅ 关闭提示 |
| 4 | 尝试说话 | 无响应（连接已断开） | ✅ 断连验证 |

---

### 2.4 打断功能测试

| 步骤 | 操作 | 预期结果 | 验证点 |
|------|------|---------|-------|
| 1 | 启动一轮语音对话，等待 AI 开始播放 | AI 正在播放 TTS | - |
| 2 | 在 AI 播放过程中，直接对着麦克风说话 | AI 播放立即停止 | ✅ TTS 打断 |
| 3 | 观察状态变化 | 状态切回"识别中" | ✅ 状态正确切换 |
| 4 | 等待新的 AI 回复生成并播放 | 新回复正常播放 | ✅ 新对话继续 |
| 5 | 点击页面"停止"按钮 | 所有流程停止，状态回到初始 | ✅ 主动中断 |

**打断响应时间测试**:
```
打断触发时刻: ___________
TTS 停止时刻: ___________
打断延迟: ___________ms (目标 < 300ms)
```

---

### 2.5 多轮对话测试

| 步骤 | 操作 | 预期结果 | 验证点 |
|------|------|---------|-------|
| 1 | 完成第一轮对话，状态回到"监听中" | - | - |
| 2 | 继续说"那明天呢？" | 正确结合上一轮上下文 | ✅ 对话上下文 |
| 3 | 观察 AI 回复是否基于"今天天气"+"明天" | AI 回答明天天气 | ✅ 上下文理解 |
| 4 | 连续进行 5 轮对话 | 每轮正常完成，无累积延迟 | ✅ 多轮稳定性 |

---

### 2.6 异常场景测试

| 步骤 | 操作 | 预期结果 | 验证点 |
|------|------|---------|-------|
| 1 | 拔掉网线，触发语音对话 | 显示"网络断开"提示 | ✅ 断网检测 |
| 2 | 恢复网络，点击"重试" | 恢复正常工作 | ✅ 断网恢复 |
| 3 | 故意不说话，等待 30 秒 | 超时回到初始状态 | ✅ 超时处理 |
| 4 | 拒绝麦克风权限 | 引导用户开启权限提示 | ✅ 权限引导 |
| 5 | 服务端关闭 LLM 服务，发送语音 | 显示"服务暂时不可用" | ✅ 降级策略 |

---

### 2.7 性能指标测试

| 指标 | 测试方法 | 目标值 | 实际值记录 |
|------|---------|--------|-----------|
| 端到端延迟 | 从用户说完到最后 TTS 播放 | < 2000ms | ___ms |
| 打断响应时间 | 从用户开始说话到 TTS 停止 | < 300ms | ___ms |
| ASR 识别延迟 | 从用户说完到收到 asr_complete | < 500ms | ___ms |
| TTS 首音频延迟 | 从 LLM 开始输出到首音频播放 | < 800ms | ___ms |

**测试命令**:
```bash
# 查看 WebSocket 消息时间戳计算延迟
# 在浏览器控制台执行:
const times = [];
ws.onmessage = (e) => times.push({type: e.data.type, t: Date.now()});
```

---

### 2.8 兼容性测试

| 环境 | 测试内容 | 结果 |
|------|---------|------|
| Chrome 最新版 | 完整流程 | □ |
| Firefox 最新版 | 完整流程 | □ |
| Safari 最新版 | 完整流程 | □ |
| 移动端 Chrome | 完整流程 | □ |
| 移动端 Safari | 完整流程 | □ |

---

### 2.9 测试签名

| 项目 | 签名 | 日期 |
|------|------|------|
| 功能测试 | ___________ | ___________ |
| 性能测试 | ___________ | ___________ |
| 兼容性测试 | ___________ | ___________ |
| 最终验收 | ___________ | ___________ |

---

## 三、前端开发任务

### Task-F-01: 音频采集服务
**文件**: `frontend/src/composables/useAudioCapture.ts`

#### 功能点
- [ ] `initialize()` - 初始化 WebRTC AudioContext
- [ ] `start()` - 开始采集麦克风音频
- [ ] `stop()` - 停止采集
- [ ] `getStream()` - 获取 MediaStream
- [ ] `getCurrentVolume()` - 获取当前音量 (0-1)
- [ ] `onVolumeChange(callback)` - 音量变化回调
- [ ] 降噪处理 (noiseSuppression: true)
- [ ] 回声消除 (echoCancellation: true)
- [ ] 自动增益 (autoGainControl: true)
- [ ] 音频格式: PCM 16bit 16kHz 单声道

#### 单元测试要点
- [ ] 测试 `initialize()` 正常初始化
- [ ] 测试 `initialize()` 麦克风权限被拒绝时的错误处理
- [ ] 测试 `start()` / `stop()` 正常开始/停止
- [ ] 测试 `getCurrentVolume()` 返回值范围 (0-1)
- [ ] 测试 `onVolumeChange` 回调被正确触发
- [ ] 测试降噪/echo cancellation/AGC 配置正确应用

---

### Task-F-02: 唤醒词检测服务
**文件**: `frontend/src/composables/useWakeWord.ts`

#### 功能点
- [ ] 唤醒词配置加载 (`keyword: "小爱同学"`)
- [ ] MFCC 特征提取 (每帧 100ms)
- [ ] DTW 模板匹配算法
- [ ] 匹配分数计算
- [ ] 阈值判断 (threshold: 0.75)
- [ ] 唤醒成功音效播放 (本地音频)
- [ ] 唤醒成功后建立 WebSocket 连接
- [ ] 唤醒成功后建立 DataChannel 连接
- [ ] 连接成功后 Alert 弹窗 "连接成功，开启对话"
- [ ] **纯前端独立完成，无需后端验证**

#### 单元测试要点
- [ ] 测试唤醒词配置正确加载
- [ ] 测试 MFCC 特征提取输出维度正确
- [ ] 测试 DTW 匹配分数计算
- [ ] 测试阈值判断逻辑 (分数 >= 0.75 → 触发)
- [ ] 测试唤醒成功时音效播放
- [ ] 测试唤醒成功后 WebSocket 连接建立
- [ ] 测试唤醒成功后 DataChannel 连接建立
- [ ] 测试连接成功 Alert 弹窗显示
- [ ] 测试低分时正确抑制误触发
- [ ] 测试唤醒响应时间 < 200ms

---

### Task-F-03: VAD 语音活动检测
**文件**: `frontend/src/composables/useVAD.ts`

#### 功能点
- [ ] WebRTC VAD 初始化
- [ ] 灵敏度配置 (sensitivity: 0-3)
- [ ] 语音开始检测 (speechStartThreshold)
- [ ] 语音结束检测 (speechEndThreshold)
- [ ] 静音超时检测 (silenceTimeout: 3000ms)
- [ ] 最大语音时长检测 (maxSpeechDuration: 60000ms)
- [ ] `onSpeechStart` 回调
- [ ] `onSpeechEnd` 回调
- [ ] 状态上报到 StateMachine

#### 单元测试要点
- [ ] 测试 VAD 灵敏度设置正确
- [ ] 测试静音超时触发 `onSpeechEnd`
- [ ] 测试最大语音时长到达时触发 `onSpeechEnd`
- [ ] 测试 `onSpeechStart` 回调正确触发
- [ ] 测试 `onSpeechEnd` 回调正确触发
- [ ] 测试连续静音场景下的行为

---

### Task-F-04: 语音状态机
**文件**: `frontend/src/composables/useVoiceDialogue.ts` + `frontend/src/stores/voice.ts`

#### 功能点
- [ ] 7种状态定义: IDLE/LISTENING/RECOGNIZING/THINKING/RESPONDING/PLAYING/ERROR
- [ ] 状态转换规则:
  - IDLE → LISTENING (开始录音)
  - LISTENING → RECOGNIZING (检测到语音)
  - RECOGNIZING → THINKING (语音结束)
  - THINKING → RESPONDING (收到LLM首字符)
  - RESPONDING → PLAYING (TTS开始)
  - PLAYING → LISTENING (播放完成)
  - 任意状态 → ERROR (异常)
  - 任意状态 → LISTENING (打断)
- [ ] `getState()` 获取当前状态
- [ ] `onStateChange(callback)` 状态变化监听
- [ ] 状态历史记录

#### 单元测试要点
- [ ] 测试初始状态为 IDLE
- [ ] 测试 IDLE → LISTENING 转换
- [ ] 测试 LISTENING → RECOGNIZING 转换
- [ ] 测试完整流程转换: IDLE→LISTENING→RECOGNIZING→THINKING→RESPONDING→PLAYING→LISTENING
- [ ] 测试打断时任意状态 → LISTENING
- [ ] 测试 ERROR 状态恢复逻辑
- [ ] 测试 `onStateChange` 回调在每次状态变化时被调用
- [ ] 测试非法状态转换被拒绝

---

### Task-F-05: 打断处理机制
**文件**: `frontend/src/composables/useInterruptHandler.ts`

#### 功能点
- [ ] 打断来源枚举: USER_SPEECH / USER_CLICK / SERVER_CMD / TIMEOUT
- [ ] 新语音检测触发打断 (AudioCapture 音量突变)
- [ ] 用户点击停止按钮触发打断
- [ ] 服务器命令打断接收
- [ ] 超时打断 (30s 无响应)
- [ ] `interrupt(source)` 方法
- [ ] 打断时停止 TTS 播放
- [ ] 打断时取消当前 ASR/LLM 请求
- [ ] 打断后状态重置为 LISTENING

#### 单元测试要点
- [ ] 测试 USER_SPEECH 打断触发
- [ ] 测试 USER_CLICK 打断触发
- [ ] 测试打断后 TTS 播放停止
- [ ] 测试打断后状态正确重置为 LISTENING
- [ ] 测试打断响应时间 < 300ms
- [ ] 测试打断时正在进行的请求被正确取消

---

### Task-F-06: WebSocket 通信服务
**文件**: `frontend/src/services/ws.ts`

#### 功能点
- [ ] `connect()` - 建立 WebSocket 连接 (ws://host:8080/ws/voice)
- [ ] `disconnect()` - 断开连接
- [ ] `isConnected()` - 连接状态查询
- [ ] `send(message)` - 发送消息
- [ ] `onMessage(callback)` - 消息接收
- [ ] `onOpen(callback)` - 连接建立
- [ ] `onClose(callback)` - 连接关闭
- [ ] `onError(callback)` - 错误处理
- [ ] 心跳机制 (ping/pong, 30秒间隔)
- [ ] 重连机制 (连接断开时自动重连)
- [ ] 消息队列 (离线时缓存消息)

#### 单元测试要点
- [ ] 测试 `connect()` 成功建立连接
- [ ] 测试 `connect()` 连接失败时的错误处理
- [ ] 测试 `send()` 消息发送
- [ ] 测试 `onMessage` 回调接收消息
- [ ] 测试心跳 ping 正确发送
- [ ] 测试 pong 响应正确处理
- [ ] 测试断线重连逻辑
- [ ] 测试消息队列缓存和重发

---

### Task-F-06.5: 连接生命周期管理
**文件**: `frontend/src/composables/useVoiceDialogue.ts`

#### 功能点
- [ ] 连接建立成功后 Alert 弹窗 "连接成功，开启对话"
- [ ] 用户主动结束对话时关闭 WebSocket + DataChannel
- [ ] 连接关闭后 Alert 弹窗 "对话已结束，连接已关闭"
- [ ] 超时自动关闭连接
- [ ] Alert 弹窗样式 (3秒自动关闭)

#### 单元测试要点
- [ ] 测试连接成功时 Alert 显示
- [ ] 测试连接关闭时 Alert 显示
- [ ] 测试 Alert 3秒自动关闭
- [ ] 测试用户主动结束流程
- [ ] 测试超时自动结束流程

---

### Task-F-07: DataChannel 音频传输服务
**文件**: `frontend/src/services/datachannel.ts`

#### 功能点
- [ ] WebRTC PeerConnection 创建
- [ ] DataChannel 创建 (label: "voice-audio")
- [ ] `Ordered: false` 配置 (UDP模式)
- [ ] `MaxRetransmits: 0` 配置 (不重传)
- [ ] `sendAudio(audioData)` - 发送 ASR 音频
- [ ] `onAudioReceive(callback)` - 接收 TTS 音频
- [ ] `sendControl(action)` - 发送控制命令 (play/stop/flush/audio_start)
- [ ] `onControlReceive(callback)` - 接收控制命令
- [ ] 音频子流支持 (asr_audio / tts_audio / audio_control)

#### 单元测试要点
- [ ] 测试 DataChannel 创建成功
- [ ] 测试 `Ordered: false` 配置正确
- [ ] 测试 `sendAudio` 发送二进制数据
- [ ] 测试 `onAudioReceive` 接收 TTS 音频
- [ ] 测试 `sendControl` 发送控制命令
- [ ] 测试子流类型区分 (asr_audio vs tts_audio)

---

### Task-F-08: UDP 乱序缓冲处理
**文件**: `frontend/src/services/audioBuffer.ts`

#### 功能点
- [ ] `AudioBuffer` 类实现
- [ ] 序列号管理 (sequence 递增)
- [ ] 时间戳记录 (timestamp ms)
- [ ] 乱序包缓冲 (Map<sequence, packet>)
- [ ] `addPacket(packet)` - 添加数据包
- [ ] 过期包丢弃 (sequence < baseSequence)
- [ ] `flush()` - 按序列号顺序播放
- [ ] Jitter Buffer 配置:
  - TTS: maxBufferTime=50ms, latePacketTolerance=50ms
  - ASR: maxBufferTime=100ms, latePacketTolerance=100ms
- [ ] `AudioBufferManager` 多流管理

#### 单元测试要点
- [ ] 测试序列号递增正确
- [ ] 测试乱序包被正确缓冲
- [ ] 测试过期包被丢弃
- [ ] 测试 flush 按顺序播放
- [ ] 测试 TTS 低延迟策略 (50ms 窗口)
- [ ] 测试 ASR 完整优先策略 (100ms 窗口)
- [ ] 测试多流缓冲隔离

---

### Task-F-09: UI 组件开发

#### Task-F-09.1: VoiceButton 语音按钮
**文件**: `frontend/src/components/voice/VoiceButton.vue`

- [ ] 圆形按钮设计 (64px)
- [ ] 默认/录音/禁用 状态样式
- [ ] 渐变橙色背景 (primary gradient)
- [ ] 悬停动画 (scale 1.05)
- [ ] 点击涟漪效果
- [ ] 状态同步 (idle/listening/recording)

**单元测试要点**:
- [ ] 测试按钮点击事件触发
- [ ] 测试录音状态样式正确应用
- [ ] 测试禁用状态点击无响应

#### Task-F-09.2: VoiceWaveform 波形动画
**文件**: `frontend/src/components/voice/VoiceWaveform.vue`

- [ ] Canvas 绘制波形
- [ ] 录音时实时音量可视化
- [ ] AI 播放时 TTS 波形动画
- [ ] 平滑动画过渡 (60fps)
- [ ] 状态联动 (listening/playing)

**单元测试要点**:
- [ ] 测试波形在录音状态正确显示
- [ ] 测试 AI 播放时动画效果
- [ ] 测试动画帧率 (60fps)

#### Task-F-09.3: VoiceStatus 状态显示
**文件**: `frontend/src/components/voice/VoiceStatus.vue`

- [ ] 5种状态文字显示
- [ ] 状态对应图标/动画
- [ ] 状态切换过渡效果

**单元测试要点**:
- [ ] 测试 5 种状态文字正确显示
- [ ] 测试状态切换时 UI 更新

#### Task-F-09.4: DialogueBubble 对话气泡
**文件**: `frontend/src/components/voice/DialogueBubble.vue`

- [ ] 用户消息气泡 (右侧，橙色)
- [ ] AI 消息气泡 (左侧，白色)
- [ ] 流式文字逐字显示
- [ ] 时间戳显示
- [ ] 实时字幕模式 (ASR 识别结果实时上屏)

**单元测试要点**:
- [ ] 测试用户/AI 气泡样式区分
- [ ] 测试流式文字逐字显示效果
- [ ] 测试实时字幕模式

---

### Task-F-10: 前端类型定义
**文件**: `frontend/src/types/voice.ts`

- [ ] VoiceState 枚举
- [ ] InterruptSource 枚举
- [ ] MessageType 枚举
- [ ] AudioConfig 接口
- [ ] VoiceMessage 接口
- [ ] ASRResult 接口
- [ ] LLMResponse 接口
- [ ] TTSAudio 接口
- [ ] InterruptEvent 接口
- [ ] VoiceSession 接口
- [ ] WSClientMessage / WSServerMessage 接口
- [ ] DCMessage 接口
- [ ] WakeWordConfig / VADConfig 接口
- [ ] PerformanceMetrics 接口

---

## 三、后端开发任务

### Task-B-01: 项目结构与配置
**文件**: `backend/`

#### 功能点
- [ ] 目录结构创建 (api/, component/, domain/, logic/, config/, router/)
- [ ] Go module 初始化 (go mod init)
- [ ] pion/webrtc 依赖安装
- [ ] qwen-go SDK 依赖安装
- [ ] Gin 框架依赖
- [ ] 配置管理 (config.go)
- [ ] 环境变量配置:
  - DASHSCOPE_API_KEY
  - LLM_MODEL (qwen-plus)
  - COSYVOICE_HOST
  - QWEN3_TTS_HOST
  - STUN_SERVER

---

### Task-B-02: 后端类型定义
**文件**: `backend/domain/voice/types.go`

#### 功能点
- [ ] VoiceState 枚举 (Idle/Listening/Recognizing/Thinking/Responding/Playing/Error)
- [ ] MessageType 字符串
- [ ] Session 结构体
- [ ] WSMessage 结构体
- [ ] ASRResult 结构体
- [ ] LLMResponse 结构体
- [ ] TTSAudio 结构体
- [ ] ErrorResponse 结构体

#### 单元测试要点
- [ ] 测试 VoiceState.String() 返回正确字符串
- [ ] 测试 Session 结构体序列化/反序列化

---

### Task-B-03: SessionManager 会话管理
**文件**: `backend/domain/voice/session.go`

#### 功能点
- [ ] `NewSessionManager()` 创建管理器
- [ ] `Create(userID)` 创建新会话
- [ ] `Get(sessionID)` 获取会话
- [ ] `Update(sessionID, state)` 更新状态
- [ ] `Delete(sessionID)` 删除会话
- [ ] `GetByUserID(userID)` 获取用户活跃会话
- [ ] `GetAll()` 获取所有会话
- [ ] 会话超时自动删除 (30min)
- [ ] 并发安全 (sync.RWMutex)

#### 单元测试要点
- [ ] 测试 Create 创建会话
- [ ] 测试 Get 获取存在的会话
- [ ] 测试 Get 获取不存在的会话返回错误
- [ ] 测试 Update 更新状态
- [ ] 测试 Delete 删除会话
- [ ] 测试 GetByUserID 正确返回用户活跃会话
- [ ] 测试并发安全 (多个 goroutine 同时访问)

---

### Task-B-04: AudioBuffer 音频缓冲管理
**文件**: `backend/component/audio/buffer.go`

#### 功能点
- [ ] AudioPacket 结构体 (Sequence, Timestamp, Data, SampleRate, IsLast)
- [ ] AudioBuffer 结构体
- [ ] `NewAudioBuffer()` 创建缓冲
- [ ] `Add(packet)` 添加包
- [ ] `Flush(fn)` 按序刷新
- [ ] `NextSequence()` 获取下一个序列号
- [ ] `GetBaseSequence()` 获取基础序列号
- [ ] 过期包丢弃逻辑
- [ ] TTSBuffer 低延迟缓冲 (latePacketWindow: 50ms)
- [ ] `AddWithTimeout()` 超时丢弃

#### 单元测试要点
- [ ] 测试 Add 添加包
- [ ] 测试乱序包被正确缓冲
- [ ] 测试 Flush 按序列号顺序处理
- [ ] 测试过期包被丢弃
- [ ] 测试 NextSequence 递增正确
- [ ] 测试 TTSBuffer 超时丢弃逻辑
- [ ] 测试高并发下 buffer 安全

---

### Task-B-05: WebRTC DataChannel 服务端
**文件**: `backend/component/webrtc/datachannel.go`

#### 功能点
- [ ] `NewAudioDataChannel()` 创建
- [ ] PeerConnection 配置 (ICEServers: stun.l.google.com:19302)
- [ ] DataChannel 创建 (Ordered: false, MaxRetransmits: 0)
- [ ] `SendAudio(packet)` 发送音频
- [ ] `SendTTSAudio(pcmData, timestamp, isLast)` 发送 TTS
- [ ] `OnMessage(func)` 接收消息回调
- [ ] 二进制消息处理 (音频数据)
- [ ] 字符串消息处理 (控制命令)
- [ ] 子流支持 (asr_audio / tts_audio)

#### 单元测试要点
- [ ] 测试 PeerConnection 创建成功
- [ ] 测试 DataChannel 配置 Ordered: false
- [ ] 测试 SendAudio 发送二进制数据
- [ ] 测试 OnMessage 二进制消息回调
- [ ] 测试 OnMessage 字符串消息回调

---

### Task-B-06: ASR 客户端封装
**文件**: `backend/component/asr/client.go`

#### 功能点
- [ ] ASR 客户端连接 CosyVoice/SenseVoice
- [ ] `StreamRecognize(audioStream)` 流式识别
- [ ] 音频格式: PCM 16bit 16kHz
- [ ] 识别结果流式返回
- [ ] `asr_result` 消息 (text, isFinal)
- [ ] `asr_complete` 消息 (text)
- [ ] 错误处理和重连

#### 单元测试要点
- [ ] 测试 CosyVoice 连接成功
- [ ] 测试流式识别返回结果
- [ ] 测试 isFinal 标志正确设置
- [ ] 测试连接失败错误处理

---

### Task-B-07: LLM 客户端封装 (Qwen API)
**文件**: `backend/component/llm/client.go`

#### 功能点
- [ ] Qwen API 客户端初始化
- [ ] 配置: BaseURL, APIKey, Model, MaxTokens, Temperature, TopP
- [ ] 环境变量读取 (DASHSCOPE_API_KEY, LLM_MODEL)
- [ ] `StreamChat(messages)` 流式对话
- [ ] `llm_text` 消息 (text, isChunk)
- [ ] `llm_complete` 消息 (text, fullText)
- [ ] 超时处理 (5s 降级)
- [ ] 错误处理

#### 单元测试要点
- [ ] 测试 Qwen API 客户端初始化
- [ ] 测试流式对话返回
- [ ] 测试 isChunk 正确标识片段
- [ ] 测试 fullText 累积完整回复
- [ ] 测试超时降级回复
- [ ] 测试 API Key 无效错误处理

---

### Task-B-08: TTS 客户端封装
**文件**: `backend/component/tts/client.go`

#### 功能点
- [ ] TTS 客户端连接 Qwen3-TTS
- [ ] 音频格式: PCM 16bit 24kHz
- [ ] `StreamSynthesize(text)` 流式合成
- [ ] `tts_started` 消息 (timestamp)
- [ ] `tts_audio` DataChannel 传输
- [ ] `tts_complete` 消息
- [ ] 中断支持 (立即停止)
- [ ] 错误处理

#### 单元测试要点
- [ ] 测试 Qwen3-TTS 连接成功
- [ ] 测试流式合成音频输出
- [ ] 测试 tts_started 正确发送
- [ ] 测试 tts_audio 通过 DataChannel 发送
- [ ] 测试 tts_complete 正确发送
- [ ] 测试中断时立即停止合成

---

### Task-B-09: 语音对话 WebSocket 处理
**文件**: `backend/api/voice/dialogue.go`

#### 功能点
- [ ] WebSocket 握手升级 (Gin)
- [ ] `onTextMessage` 处理文字消息
- [ ] `onAudioStream` 处理音频流
- [ ] `onTTSStream` 处理 TTS 流
- [ ] 消息类型路由分发
- [ ] 会话状态同步
- [ ] 心跳处理 (ping/pong)
- [ ] 断开连接清理

#### 单元测试要点
- [ ] 测试 WebSocket 握手成功
- [ ] 测试 ping 消息响应 pong
- [ ] 测试文字消息路由分发
- [ ] 测试音频流处理
- [ ] 测试断开时资源清理

---

### Task-B-10: 打断处理与异常管理
**文件**: `backend/logic/voice/dialogue.go`

#### 功能点
- [ ] `HandleInterrupt(sessionID, source)` 打断处理
- [ ] 5种打断场景:
  - AI 播放中被打断 → 停止 TTS
  - AI 思考中被打断 → 取消 LLM 请求
  - AI 回复中被打断 → 取消 LLM 请求
  - 用户主动中断 → 停止所有流程
  - 服务器命令打断 → 执行打断
- [ ] `state_update` 消息发送
- [ ] 熔断器配置 (gobreaker)
- [ ] 错误码定义 (1001-1999)

#### 单元测试要点
- [ ] 测试播放中打断停止 TTS
- [ ] 测试思考中打断取消 LLM
- [ ] 测试打断后状态正确切换为 LISTENING
- [ ] 测试熔断器触发和恢复
- [ ] 测试连续失败熔断开启

---

### Task-B-11: 全双工通信时序控制
**文件**: `backend/logic/voice/stream.go`

#### 功能点
- [ ] ASR → LLM → TTS Pipeline 并行
- [ ] ASR 流式完成立即启动 LLM
- [ ] LLM 首字符输出立即启动 TTS
- [ ] 并发会话管理
- [ ] StreamRouter 流式路由

#### 单元测试要点
- [ ] 测试 ASR 完成后 LLM 立即启动
- [ ] 测试 LLM 首字符后 TTS 立即启动
- [ ] 测试 Pipeline 并行处理
- [ ] 测试多会话并发处理

---

### Task-B-12: 健康检查与降级策略
**文件**: `backend/logic/health.go`

#### 功能点
- [ ] `/health` 端点
- [ ] 各服务健康状态检测 (ASR/TTS/LLM)
- [ ] 服务降级配置:
  - LLM 超时 5s → 降级回复
  - ASR 失败 → 提示"请重试"
  - TTS 失败 → 提示"服务繁忙"
- [ ] 降级回复内容模板

---

### Task-B-13: 后端单元测试框架
**文件**: `backend/*_test.go`

#### 单元测试覆盖率要求
- [ ] SessionManager: 90%+
- [ ] AudioBuffer: 90%+
- [ ] LLM Client: 80%+ (Mock Qwen API)
- [ ] TTS Client: 80%+ (Mock Qwen3-TTS)
- [ ] ASR Client: 80%+ (Mock CosyVoice)
- [ ] InterruptHandler: 90%+

---

## 四、集成测试任务

### Task-I-01: 前后端 WebSocket 集成
- [ ] 前端连接后端 WebSocket 成功
- [ ] 前端发送 start 消息，后端正确响应 state_update
- [ ] 前后端消息格式正确解析

### Task-I-02: DataChannel 音频传输集成
- [ ] 前端 DataChannel 连接到后端
- [ ] ASR 音频流从前端发送到后端
- [ ] TTS 音频流从后端发送到前端

### Task-I-03: 端到端语音对话流程
- [ ] 唤醒 → 识别 → 回复 → 播放 完整流程
- [ ] 端到端延迟 < 2s

### Task-I-04: 打断功能集成
- [ ] AI 播放中用户说话打断
- [ ] 打断响应时间 < 300ms

### Task-I-05: 异常场景集成
- [ ] 网络断开提示
- [ ] ASR 识别失败提示
- [ ] LLM 服务不可用降级
- [ ] TTS 服务不可用降级

---

## 五、部署任务

### Task-D-01: Docker 配置
- [ ] backend/Dockerfile
- [ ] cosyvoice/Dockerfile
- [ ] qwen3-tts/Dockerfile
- [ ] docker-compose.yml 更新

### Task-D-02: 环境配置
- [ ] 环境变量文档
- [ ] 敏感信息管理

---

## 六、任务依赖关系

```
Task-F-01 (音频采集) ─────┬────→ Task-F-03 (VAD)
                          │
                          └────→ Task-F-05 (打断)
                          │
Task-F-02 (唤醒词) ───────────────┐
                                  │
Task-F-03 (VAD) ──────────────────┴────→ Task-F-04 (状态机)
                                  │
Task-F-04 (状态机) ────────────────────┴──────→ Task-F-09 (UI)
                                  │
Task-F-06 (WebSocket) ────────────┴──────┐
                                          │
Task-F-07 (DataChannel) ──────────────────┴────→ Task-B-05 (WebRTC)
                                          │
Task-B-03 (Session) ───────────────────────┤
                                          │
Task-B-04 (AudioBuffer) ───────────────────┤
                                          │
Task-B-06 (ASR) ◀──────┐                   │
                        │                   │
Task-B-07 (LLM) ────────┼──── Pipeline ────┼────→ Task-B-08 (TTS)
                        │                   │
                        └───────────────────┘
                                          │
Task-B-09 (WS Handler) ◀──────────────────┘
```

---

## 七、版本记录

| 版本 | 日期 | 变更内容 |
|------|------|---------|
| v1.2 | 2026-04-06 | 新增开发计划：8阶段划分、任务类型定义、代办详情与交付标准、核对清单 |
| v1.1 | 2026-04-06 | 统一后端分层规范：移除service层，改用logic层做业务编排；更新目录结构为api/component/domain/logic/config/router |
| v1.0 | 2026-04-06 | 初始任务清单，基于 voice-dialogueTec.md v1.3 |

---

## 八、开发计划

### 8.1 任务类型定义

| 任务类型 | 说明 | 交付物 |
|---------|------|--------|
| **目录结构创建** | 初始化项目目录和基础文件 | 目录结构、go.mod、配置文件 |
| **类型定义** | 定义枚举、结构体、接口 | .go 类型文件 |
| **组件封装** | 封装外部依赖客户端 | component/ 下各 client.go |
| **领域模型** | 构建业务领域模型、聚合根 | domain/ 下各 .go 文件 |
| **逻辑编排** | 实现业务用例流程 | logic/ 下各 .go 文件 |
| **接口实现** | 实现 WebSocket/HTTP Handler | api/ 下 handler.go |
| **单元测试** | 编写各模块单元测试 | *_test.go |
| **集成测试** | 端到端流程测试 | 集成测试报告 |
| **人工测试** | 完整功能验证 | 测试记录表 |

### 8.2 开发阶段划分

| 阶段 | 任务 | 预计工时 | 依赖 |
|------|------|---------|------|
| **Phase 1: 基础设施** | Task-B-01, Task-B-02 | 4h | - |
| **Phase 2: 核心领域模型** | Task-B-03, Task-B-04 | 6h | Phase 1 |
| **Phase 3: 组件封装** | Task-B-05, Task-B-06, Task-B-07, Task-B-08 | 12h | Phase 1 |
| **Phase 4: 逻辑编排** | Task-B-09, Task-B-10, Task-B-11 | 8h | Phase 2, 3 |
| **Phase 5: 接口层** | Task-B-12 | 4h | Phase 4 |
| **Phase 6: 测试** | Task-B-13, Task-I-* | 8h | Phase 4 |
| **总计** | | **42h** | |

---

## 九、代办详情与交付标准

### Phase 1: 基础设施

#### Task-B-01: 项目结构与配置
**文件**: `backend/`

| 项目 | 内容 |
|------|------|
| **任务类型** | 目录结构创建 |
| **交付标准** | 目录结构完整，go.mod 初始化，依赖安装完成 |
| **交付物** | 目录结构、go.mod、config/config.go |

**开发任务详情**:
```
1. 创建目录结构
   backend/
   ├── api/voice/ & api/common/
   ├── component/llm/, asr/, tts/, webrtc/, redis/
   ├── domain/voice/
   ├── logic/voice/
   ├── config/
   └── router/

2. 初始化 go mod
   go mod init voice-assistant/backend

3. 安装依赖
   go get github.com/gin-gonic/gin
   go get github.com/pion/webrtc/v3
   go get github.com/redis/go-redis/v9
   go get github.com/qwenlm/qwen-go

4. 创建 config/config.go
   - DASHSCOPE_API_KEY
   - LLM_MODEL
   - COSYVOICE_HOST
   - QWEN3_TTS_HOST
   - STUN_SERVER
   - REDIS_ADDR
```

---

#### Task-B-02: 后端类型定义
**文件**: `backend/domain/voice/model.go`, `backend/domain/voice/types.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 类型定义 |
| **交付标准** | VoiceState, MessageType, InterruptSource 等枚举定义完整，结构体序列化正确 |
| **交付物** | domain/voice/model.go, domain/voice/types.go |

**开发任务详情**:
```
1. 定义枚举 (VoiceState)
   - IDLE = 0
   - LISTENING = 1
   - RECOGNIZING = 2
   - THINKING = 3
   - RESPONDING = 4
   - PLAYING = 5
   - ERROR = 6

2. 定义枚举 (MessageType)
   - asr_result, asr_complete
   - llm_text, llm_complete
   - tts_started, tts_audio, tts_complete
   - state_update, error, interrupt, ping, pong

3. 定义枚举 (InterruptSource)
   - USER_SPEECH, USER_CLICK, SERVER_CMD, TIMEOUT

4. 定义结构体
   - Session: ID, UserID, State, RecognizedText, ResponseText, Context, CreatedAt, LastActiveAt, IsInterrupted
   - WSMessage: Type, SessionID, Data, Timestamp
   - ASRResult: Text, IsFinal, Confidence
   - LLMResponse: Text, IsChunk, IsComplete
   - TTSAudio: Data, IsLast, Timestamp
   - ErrorResponse: Code, Message
```

---

### Phase 2: 核心领域模型

#### Task-B-03: Session 实体与仓储接口
**文件**: `backend/domain/voice/session.go`, `backend/domain/voice/repository.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 领域模型 |
| **交付标准** | Session 实体完整，ISessionRepository 接口定义正确，session.go 单元测试覆盖率 90%+ |
| **交付物** | domain/voice/session.go, domain/voice/repository.go, domain/voice/session_test.go |

**开发任务详情**:
```go
// domain/voice/session.go
type Session struct {
    ID              string
    UserID          string
    State           VoiceState
    RecognizedText  string
    ResponseText    string
    Context         []string
    CreatedAt       time.Time
    LastActiveAt    time.Time
    IsInterrupted   bool
}

func (s *Session) AddContext(text string)
func (s *Session) ClearContext()
func (s *Session) UpdateState(state VoiceState)
func (s *Session) SetInterrupted(interrupted bool)

// domain/voice/repository.go
type ISessionRepository interface {
    Save(ctx context.Context, session *Session) error
    Get(ctx context.Context, id string) (*Session, error)
    Update(ctx context.Context, session *Session) error
    Delete(ctx context.Context, id string) error
    GetByUserID(ctx context.Context, userID string) (*Session, error)
}
```

**单元测试要点**:
- [ ] Create 创建会话
- [ ] Get 获取存在的会话
- [ ] Get 获取不存在的会话返回错误
- [ ] Update 更新状态
- [ ] Delete 删除会话
- [ ] GetByUserID 正确返回用户活跃会话
- [ ] 并发安全测试

---

#### Task-B-04: AudioBuffer 音频缓冲管理
**文件**: `backend/component/audio/buffer.go`, `backend/component/audio/buffer_test.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 组件封装 + 单元测试 |
| **交付标准** | 乱序处理正确，Jitter Buffer 配置正确，单元测试覆盖率 90%+ |
| **交付物** | component/audio/buffer.go, component/audio/buffer_test.go |

**开发任务详情**:
```go
// AudioPacket 音频包结构
type AudioPacket struct {
    Sequence   uint32
    Timestamp  int64
    Data       []byte
    SampleRate int
    IsLast     bool
}

// AudioBuffer 缓冲管理器
type AudioBuffer struct {
    packets map[uint32]*AudioPacket
    baseSeq uint32
    lock    sync.Mutex
    maxSize int
}

func NewAudioBuffer() *AudioBuffer
func (b *AudioBuffer) Add(pkt *AudioPacket) bool
func (b *AudioBuffer) Flush(fn func(*AudioPacket))
func (b *AudioBuffer) NextSequence() uint32
func (b *AudioBuffer) GetBaseSequence() uint32

// TTSBuffer 低延迟缓冲
type TTSBuffer struct {
    *AudioBuffer
    latePacketWindow time.Duration
}

func NewTTSBuffer(latePacketWindowMs int) *TTSBuffer
func (b *TTSBuffer) AddWithTimeout(pkt *AudioPacket, now int64) bool
```

**逻辑编排事项**:
- TTS 播放使用 TTSBuffer，latePacketWindow = 50ms
- ASR 输入使用 AudioBuffer，maxBufferTime = 100ms

**单元测试要点**:
- [ ] Add 添加包
- [ ] 乱序包被正确缓冲
- [ ] Flush 按序列号顺序处理
- [ ] 过期包被丢弃
- [ ] NextSequence 递增正确
- [ ] TTSBuffer 超时丢弃逻辑
- [ ] 高并发下 buffer 安全

---

### Phase 3: 组件封装

#### Task-B-05: WebRTC DataChannel 服务端
**文件**: `backend/component/webrtc/datachannel.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 组件封装 |
| **交付标准** | PeerConnection 创建成功，DataChannel 配置 Ordered:false，音频收发正确 |
| **交付物** | component/webrtc/datachannel.go |

**开发任务详情**:
```go
// AudioDataChannel WebRTC DataChannel 服务端
type AudioDataChannel struct {
    peerConnection *webrtc.PeerConnection
    dataChannel    *webrtc.DataChannel
    audioBuffer    *AudioBuffer
}

func NewAudioDataChannel() (*AudioDataChannel, error)
func (dc *AudioDataChannel) SendAudio(packet *AudioPacket) error
func (dc *AudioDataChannel) SendTTSAudio(pcmData []byte, timestamp int64, isLast bool) error
func (dc *AudioDataChannel) OnMessage(func(msg webrtc.DataChannelMessage))
func (dc *AudioDataChannel) Close() error
```

**逻辑编排事项**:
- 创建时设置 ICEServers = stun:stun.l.google.com:19302
- DataChannel 创建时设置 Ordered: false, MaxRetransmits: 0

---

#### Task-B-06: ASR 客户端封装
**文件**: `backend/component/asr/client.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 组件封装 |
| **交付标准** | CosyVoice 连接成功，流式识别正常，单元测试覆盖率 80%+ |
| **交付物** | component/asr/client.go, component/asr/client_test.go |

**开发任务详情**:
```go
// ASRClient ASR 客户端
type ASRClient struct {
    host string
    conn *grpc.ClientConn
    client cosyvoice.CosyVoiceClient
}

func NewASRClient(host string) (*ASRClient, error)
func (c *ASRClient) StreamRecognize(ctx context.Context, audioStream io.Reader) (<-chan *ASRResult, error)
func (c *ASRClient) Close() error
```

**接口签名** (供 logic 层调用):
```go
// ISpeechRecognizer 语音识别接口
type ISpeechRecognizer interface {
    StreamRecognize(ctx context.Context, audioStream io.Reader) (<-chan *ASRResult, error)
    Close() error
}
```

---

#### Task-B-07: LLM 客户端封装 (Qwen API)
**文件**: `backend/component/llm/client.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 组件封装 |
| **交付标准** | Qwen API 连接成功，流式对话正常，超时降级正确，单元测试覆盖率 80%+ |
| **交付物** | component/llm/client.go, component/llm/client_test.go |

**开发任务详情**:
```go
// LLMClient LLM 客户端
type LLMClient struct {
    apiKey string
    model  string
    client *http.Client
}

func NewLLMClient(apiKey, model string) *LLMClient
func (c *LLMClient) StreamChat(ctx context.Context, messages []Message) (<-chan *LLMResponse, error)
func (c *LLMClient) Close() error
```

**接口签名** (供 logic 层调用):
```go
// IChatModel 聊天模型接口
type IChatModel interface {
    StreamChat(ctx context.Context, messages []Message) (<-chan *LLMResponse, error)
    Close() error
}
```

**降级策略**:
- API 超时 5s → 返回降级回复 "抱歉，服务繁忙，请稍后再试"

---

#### Task-B-08: TTS 客户端封装
**文件**: `backend/component/tts/client.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 组件封装 |
| **交付标准** | Qwen3-TTS 连接成功，流式合成正常，中断支持正确，单元测试覆盖率 80%+ |
| **交付物** | component/tts/client.go, component/tts/client_test.go |

**开发任务详情**:
```go
// TTSClient TTS 客户端
type TTSClient struct {
    host string
    conn *grpc.ClientConn
    client qwen3tts.Qwen3TTSClient
}

func NewTTSClient(host string) (*TTSClient, error)
func (c *TTSClient) StreamSynthesize(ctx context.Context, text string) (<-chan *TTSAudio, error)
func (c *TTSClient) Interrupt()
func (c *TTSClient) Close() error
```

**接口签名** (供 logic 层调用):
```go
// ISpeechSynthesizer 语音合成接口
type ISpeechSynthesizer interface {
    StreamSynthesize(ctx context.Context, text string) (<-chan *TTSAudio, error)
    Interrupt()
    Close() error
}
```

---

### Phase 4: 逻辑编排

#### Task-B-09: 语音对话 WebSocket 处理
**文件**: `backend/api/voice/handler.go`, `backend/logic/voice/dialogue.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 接口实现 + 逻辑编排 |
| **交付标准** | WebSocket 握手成功，消息路由正确，状态同步正确 |
| **交付物** | api/voice/handler.go, logic/voice/dialogue.go |

**开发任务详情**:

**api/voice/handler.go** (接口层):
```go
type VoiceHandler struct {
    dialogueLogic *logic.VoiceDialogueLogic
}

func (h *VoiceHandler) HandleWS(c *gin.Context)
func (h *VoiceHandler) onTextMessage(sessionID string, msg *WSMessage)
func (h *VoiceHandler) onAudioStream(sessionID string, audioData []byte)
func (h *VoiceHandler) onTTSStream(sessionID string, ttsData []byte)
func (h *VoiceHandler) HandlePing(sessionID string)
```

**logic/voice/dialogue.go** (逻辑编排):
```go
type VoiceDialogueLogic struct {
    llmClient   *component.LLMClient
    asrClient   *component.ASRClient
    ttsClient   *component.TTSClient
    dcServer    *component.AudioDataChannel
    sessionRepo domain.ISessionRepository
}

func NewVoiceDialogueLogic(...) *VoiceDialogueLogic
func (l *VoiceDialogueLogic) HandleWakeWordDetected(ctx context.Context, sessionID string) error
func (l *VoiceDialogueLogic) HandleSpeechStarted(ctx context.Context, sessionID string) error
func (l *VoiceDialogueLogic) HandleSpeechEnded(ctx context.Context, sessionID, text string) error
func (l *VoiceDialogueLogic) HandleInterrupt(ctx context.Context, sessionID string, source domain.InterruptSource) error
```

**逻辑编排事项**:
- ASR 完成后立即启动 LLM (Pipeline 并行)
- LLM 首字符输出后立即启动 TTS (Pipeline 并行)
- 所有组件通过接口注入，不直接依赖实现

---

#### Task-B-10: 打断处理与异常管理
**文件**: `backend/logic/voice/interrupt.go`, `backend/logic/voice/circuitbreaker.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 逻辑编排 |
| **交付标准** | 5 种打断场景正确处理，熔断器配置正确，单元测试覆盖率 90%+ |
| **交付物** | logic/voice/interrupt.go, logic/voice/interrupt_test.go |

**开发任务详情**:
```go
// InterruptHandler 打断处理
type InterruptHandler struct {
    ttsClient   *component.TTSClient
    llmClient   *component.LLMClient
    sessionRepo domain.ISessionRepository
}

func (h *InterruptHandler) HandleInterrupt(ctx context.Context, sessionID string, source InterruptSource) error

// 打断场景:
// 1. AI 播放中打断 → ttsClient.Interrupt() + 状态切 LISTENING
// 2. AI 思考中打断 → 取消 LLM 请求 + 状态切 LISTENING
// 3. AI 回复中打断 → 取消 LLM 请求 + 状态切 LISTENING
// 4. 用户主动中断 → 停止所有 + 状态切 IDLE
// 5. 服务器命令打断 → 执行对应打断逻辑
```

**熔断器配置** (使用 gobreaker):
```go
type CircuitBreakerConfig struct {
    FailureThreshold int           // 失败次数阈值: 5
    RecoveryTimeout   time.Duration // 恢复超时: 30s
}

// ASR/LLM/TTS 各服务配置独立熔断器
```

---

#### Task-B-11: 全双工通信时序控制
**文件**: `backend/logic/voice/stream.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 逻辑编排 |
| **交付标准** | Pipeline 并行正确，多会话并发处理正确 |
| **交付物** | logic/voice/stream.go |

**开发任务详情**:
```go
// StreamRouter 流式路由
type StreamRouter struct {
    asrInput  chan *AudioPacket
    llmOutput chan *LLMResponse
    ttsInput  chan *TTSAudio
}

func (r *StreamRouter) RouteASR(ctx context.Context, packet *AudioPacket)
func (r *StreamRouter) RouteLLM(ctx context.Context, resp *LLMResponse)
func (r *StreamRouter) RouteTTS(ctx context.Context, audio *TTSAudio)
```

**Pipeline 并行时序**:
```
用户说完话 → ASR识别 → LLM生成 → TTS合成 → 播放
              ↓
         LLM首字符 → TTS首音频 (并行)

前端播放TTS时，可同时发送下一轮ASR音频 (全双工)
```

---

#### Task-B-12: 健康检查端点
**文件**: `backend/api/health.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 接口实现 |
| **交付标准** | /health 端点正常，各服务健康状态检测正确 |
| **交付物** | api/health.go |

**开发任务详情**:
```go
func HealthCheck(c *gin.Context) {
    // 检测 ASR/TTS/LLM 服务状态
    // 返回降级状态
    // 配置降级回复模板
}
```

---

### Phase 6: 测试

#### Task-B-13: 后端单元测试框架
**文件**: `backend/*_test.go`

| 项目 | 内容 |
|------|------|
| **任务类型** | 单元测试 |
| **交付标准** | 各模块覆盖率达标 (SessionManager 90%+, AudioBuffer 90%+, 组件 80%+) |
| **交付物** | 各 *_test.go 文件 |

**覆盖率要求**:
| 模块 | 覆盖率 |
|------|--------|
| domain/voice/session.go | 90%+ |
| component/audio/buffer.go | 90%+ |
| component/llm/client.go | 80%+ (Mock Qwen API) |
| component/tts/client.go | 80%+ (Mock Qwen3-TTS) |
| component/asr/client.go | 80%+ (Mock CosyVoice) |
| logic/voice/interrupt.go | 90%+ |

---

## 十、交付核对清单

### Phase 1 交付 (基础设施)
- [ ] 目录结构创建完成
- [ ] go.mod 初始化完成
- [ ] 依赖安装完成
- [ ] config/config.go 定义完整
- [ ] domain/voice/model.go 枚举定义完整
- [ ] domain/voice/types.go 结构体定义完整

### Phase 2 交付 (核心领域模型)
- [ ] Session 实体定义完整
- [ ] ISessionRepository 接口定义完整
- [ ] AudioBuffer 乱序处理正确
- [ ] SessionManager 单元测试覆盖率 90%+
- [ ] AudioBuffer 单元测试覆盖率 90%+

### Phase 3 交付 (组件封装)
- [ ] WebRTC DataChannel 创建成功
- [ ] ASR Client 流式识别正常
- [ ] LLM Client 流式对话正常
- [ ] TTS Client 流式合成正常
- [ ] 各组件单元测试覆盖率 80%+

### Phase 4 交付 (逻辑编排)
- [ ] WebSocket Handler 消息路由正确
- [ ] VoiceDialogueLogic 业务编排正确
- [ ] 打断处理 5 种场景正确
- [ ] 熔断器配置正确
- [ ] Pipeline 并行时序正确
- [ ] /health 端点正常

### Phase 5 交付 (测试)
- [ ] 所有单元测试通过
- [ ] 覆盖率达标
- [ ] 集成测试通过
- [ ] 人工测试通过

---

## 十一、任务依赖关系图

```
Phase 1 ───────────────────────────────────────────────┐
├── Task-B-01: 项目结构                                 │
└── Task-B-02: 类型定义                                 │
                                                            │
Phase 2 ───────────────────────────────────────────────┘
├── Task-B-03: Session 实体                             │
│   └── 依赖: Task-B-02                                 │
└── Task-B-04: AudioBuffer                             │
    └── 依赖: Task-B-02                                 │
                                                            │
Phase 3 ───────────────────────────────────────────────┘
├── Task-B-05: WebRTC DataChannel                      │
│   └── 依赖: Task-B-01                                 │
├── Task-B-06: ASR Client                              │
│   └── 依赖: Task-B-01                                 │
├── Task-B-07: LLM Client                              │
│   └── 依赖: Task-B-01                                 │
└── Task-B-08: TTS Client                              │
    └── 依赖: Task-B-01                                 │
                                                            │
Phase 4 ───────────────────────────────────────────────┐
├── Task-B-09: WebSocket Handler                        │
│   └── 依赖: Task-B-03, Task-B-05, Task-B-06/07/08   │
├── Task-B-10: 打断处理                                │
│   └── 依赖: Task-B-03, Task-B-07, Task-B-08          │
└── Task-B-11: Pipeline 并行                           │
    └── 依赖: Task-B-09, Task-B-10                      │
                                                            │
Phase 5 ───────────────────────────────────────────────┘
└── Task-B-12: 健康检查                                 │
    └── 依赖: Task-B-06/07/08                          │
                                                            │
Phase 6 ───────────────────────────────────────────────┐
├── Task-B-13: 单元测试                                 │
│   └── 依赖: Phase 1-4 完成                            │
└── Task-I-*: 集成测试                                 │
    └── 依赖: Phase 1-5 完成                            │
```
