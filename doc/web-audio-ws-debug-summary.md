# Web Audio + WebSocket 实时语音识别调试经验总结

## 背景

VoiceAssistant 项目接入阿里云 NLS 实时语音识别（SpeechTranscription），前端通过浏览器 Web Audio API 采集麦克风音频，经 WebSocket 流式传输到 Go 后端，后端转发到 NLS 服务完成识别。调试过程中遇到多个典型问题，记录如下。

---

## 一、后端 ASR 生命周期管理

### 问题：IDLE_TIMEOUT + "use of closed network connection"

NLS SDK 在无活动一段时间后触发 `IDLE_TIMEOUT`，`onTaskFailed` 回调被调用，但若不清理状态则后续 `SendAudio` 会写已关闭的连接，导致 panic 或 error。

**修复：**

```go
func (r *RealTimeASR) onTaskFailed(text string, _ any) {
    r.mu.Lock()
    r.isRunning = false
    r.st = nil
    r.mu.Unlock()
    r.safeSend(ASRResult{Type: ResultError, Text: text})
    r.closeResultChan()
}
```

关键点：
- `onTaskFailed` 必须清空 `isRunning` 和 `st`，否则 `SendAudio` 仍会尝试写入
- `ResultChan` 用 `atomic.Bool` 保证只关闭一次，防止 panic
- `chat.go` 中所有 `asrClient` 的读写都要加 `sync.Mutex`

### 问题：isLast=true 从未发出

前端 `onaudioprocess` 每帧产生的 samples 数量超过了缓冲区阈值，缓冲区会在录音过程中持续清空，导致 `stopRecording` 时缓冲区已空，`isLast` 被 `if` 条件跳过。

**修复：** `stopRecording` 无论缓冲区是否为空，都必须发出 `isLast=true`，让 NLS 服务知道音频输入已结束，否则服务会等到超时才给出最终结果。

---

## 二、前端音频采集

### 问题：识别准确率极低

原因：前端叠加了多层音频前置过滤（VAD 静音检测、能量阈值过滤、fvad-wasm 语音活动检测），这些过滤器在说话的起始音（爆破音、摩擦音）和边缘帧判断上误差很大，大量有效语音帧被丢弃，送到后端的音频残缺不全。

**修复：去掉全部前置音频过滤，原始 PCM 直接发送。** 让 NLS 服务端做语音判断，它的算法远比客户端 VAD 准确。

> 教训：不要在客户端做语音活动检测再决定是否发送，NLS 流式识别本身就能处理静音段。

### 问题：重采样质量差（最近邻 → 线性插值）

原始代码用最近邻取整做降采样，高频细节损失严重。

**修复：改用线性插值：**

```ts
for (let i = 0; i < newLen; i++) {
  const pos = i * ratio;
  const idx = Math.floor(pos);
  const frac = pos - idx;
  resampled[i] = idx + 1 < input.length
    ? input[idx] * (1 - frac) + input[idx + 1] * frac
    : input[idx];
}
```

---

## 三、ScriptProcessorNode → AudioWorkletNode 迁移

### 为什么要迁移

`ScriptProcessorNode` 在 Web Audio API 中已被标记废弃（运行在主线程，容易与 UI 竞争，造成丢帧）。`AudioWorkletNode` 在独立音频线程运行，更稳定。

### 实现方案

**`public/audio-processor.js`（Worklet 处理器）：**

```js
class PCMProcessor extends AudioWorkletProcessor {
  process(inputs) {
    const ch = inputs[0]?.[0];
    if (ch?.length) this.port.postMessage(ch.slice());
    return true;
  }
}
registerProcessor('pcm-processor', PCMProcessor);
```

**主线程（Vue 组件）：**

```ts
await audioContext.audioWorklet.addModule('/audio-processor.js');
const workletNode = new AudioWorkletNode(audioContext, 'pcm-processor');

workletNode.port.onmessage = (event) => {
  // 接收 Float32Array，重采样，累积到 1600 samples（100ms）后发送
};

audioSource.connect(workletNode);
workletNode.connect(analyser);
```

### 缓冲策略

AudioWorklet 每帧固定 128 samples（约 2.67ms @ 48kHz）。不能每帧都发 WebSocket，频率太高（~375次/秒）。

**最佳实践：主线程累积 1600 samples（100ms @ 16kHz）后统一发送。**

| 缓冲大小 | 时长 | 评价 |
|---------|------|------|
| 320 | 20ms | 包频率过高 |
| 1600 | 100ms | 推荐，平衡延迟与包数 |
| 3200 | 200ms | 可用，略增端到端延迟 |

---

## 四、WebSocket 消息格式

### base64 编码性能

原始实现用 `String.fromCharCode(...bytes)` 一次性展开整个 buffer，超过 ~65000 字节时会栈溢出。

**修复：分块处理：**

```ts
function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  const chunkSize = 8192;
  let binary = '';
  for (let i = 0; i < bytes.length; i += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize));
  }
  return btoa(binary);
}
```

### 消息结构

前端发送：
```json
{
  "type": "user_audio",
  "sessionId": "xxx",
  "data": {
    "audio": "<base64 PCM>",
    "format": "pcm",
    "isLast": false
  }
}
```

后端返回（ASR 中间结果）：
```json
{ "type": "asr_result", "sessionId": "xxx", "text": "八百标兵..." }
```

后端返回（ASR 完整句子）：
```json
{ "type": "asr_complete", "sessionId": "xxx", "text": "八百标兵奔北坡" }
```

> 前端读取字段用 `message.text`，不是 `message.data.text`。

---

## 五、NLS 参数调优

```yaml
asr:
  maxSentenceSilence: 1500   # 句子间静音阈值(ms)，调大可提高长句准确率
  vocabularyId: ""            # 热词表ID，可在阿里云控制台配置
  customizationId: ""         # 自定义语言模型ID
```

- `maxSentenceSilence` 默认 800ms，调到 1500ms 对自然说话节奏更友好
- 热词表对专有名词识别帮助很大（人名、产品名等）
- `EnableIntermediateResult=true` 开启中间结果，可实时展示识别进度

---

## 六、经验总结

1. **音频不要在客户端过滤** — 交给 NLS 服务端处理，客户端 VAD 精度远不如服务端
2. **isLast 必须发出** — 无论缓冲区是否为空，停止录音时都要发 `isLast=true`
3. **ASR 错误必须清理状态** — `onTaskFailed` 后要重置 `isRunning/st`，不然下次发包会写关闭的连接
4. **AudioWorklet 优于 ScriptProcessorNode** — 音频线程独立，不受主线程阻塞影响
5. **缓冲区大小影响延迟不影响准确率** — NLS 处理的是连续字节流，包边界无关
6. **线性插值优于最近邻** — 降采样质量对识别有影响，用线性插值
