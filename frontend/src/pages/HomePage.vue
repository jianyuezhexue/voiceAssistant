<template>
  <div class="home-page">
    <!-- Warm Cream Background -->
    <div class="ambient-bg">
      <div class="cream-gradient"></div>
    </div>

    <!-- Main Content -->
    <div class="main-content">
      <!-- Chat Interface -->
      <section class="chat-section">
        <div class="chat-card">
          <!-- Chat Header -->
          <div class="chat-header">
            <div class="header-left">
              <div class="ai-avatar">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" />
                </svg>
              </div>
              <div class="header-info">
                <h2>语音助手</h2>
                <div class="status">
                  <span class="status-dot" :class="{ active: !isRecording, listening: isWakeWordListening, thinking: isLoading }"></span>
                  <span>{{ isLoading ? '思考中...' : (isRecording ? '正在聆听...' : (isWakeWordListening ? '等待唤醒...' : '随时待命')) }}</span>
                </div>
              </div>
            </div>
            <div class="header-actions">
              <button class="icon-btn" title="设置">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path
                    d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <circle cx="12" cy="12" r="3" />
                </svg>
              </button>
            </div>
          </div>

          <!-- Messages Area -->
          <div class="chat-messages" ref="messagesContainer">
            <TransitionGroup name="message" tag="div" class="messages-inner">
              <div v-for="(msg, index) in messages" :key="msg.id" class="chat-message" :class="msg.role"
                :style="{ '--delay': `${index * 0.05}s` }">
                <div class="message-avatar" :class="msg.role">
                  <svg v-if="msg.role === 'assistant'" viewBox="0 0 24 24" fill="none" stroke="currentColor"
                    stroke-width="1.5">
                    <path
                      d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                  </svg>
                  <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                </div>
                <div class="message-content">
                  <div class="message-bubble" :class="msg.role">
                    <template v-if="msg.thinking">
                      <div class="thinking-dots">
                        <span class="dot"></span>
                        <span class="dot"></span>
                        <span class="dot"></span>
                      </div>
                    </template>
                    <p v-else>{{ msg.content }}</p>
                  </div>
                  <div class="message-meta">
                    <span class="time">{{ formatTime(msg.createdAt) }}</span>
                  </div>
                </div>
              </div>
            </TransitionGroup>

            <!-- Empty State Hint -->
            <div v-if="messages.length === 0" class="empty-state">
              <div class="empty-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                  <path d="M12 2a3 3 0 00-3 3v7a3 3 0 006 0V5a3 3 0 00-3-3zM19 10v2a7 7 0 01-14 0v-2M12 19v3M8 22h8" />
                </svg>
              </div>
              <p>开始对话吧</p>
            </div>
          </div>

          <!-- Input Area -->
          <div class="input-section" :class="{ recording: isRecording }">

            <!-- Input Row: input + voice button + send button -->
            <div class="input-row">
              <!-- Text Input -->
              <div v-show="!isRecording" class="input-container" @click="focusInput">
                <input ref="inputRef" v-model="textInput" type="text" class="chat-input" placeholder="输入消息..."
                  @keyup.enter="sendTextMessage" />
                <div class="input-glow"></div>
              </div>

              <!-- Listening Indicator (shown when recording) -->
              <div v-show="isRecording" class="listening-indicator">
                <span class="listening-label">正在聆听...</span>
              </div>

              <!-- Voice Button (toggles recording) -->
              <button class="action-btn voice-btn"
                :class="{ recording: isRecording }"
                @click="toggleRecording"
                :title="isRecording ? '结束录音' : '开始录音'">
                <svg v-if="!isRecording" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 1a3 3 0 00-3 3v8a3 3 0 006 0V4a3 3 0 00-3-3z" />
                  <path d="M19 10v2a7 7 0 01-14 0v-2" />
                  <line x1="12" y1="19" x2="12" y2="23" />
                  <line x1="8" y1="23" x2="16" y2="23" />
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path
                    d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
                  <path d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2" />
                </svg>
              </button>

              <!-- Send Button (sends text message) -->
              <button class="action-btn send-btn" @click="sendTextMessage" :disabled="!textInput.trim()" title="发送消息">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </section>

    </div>
  </div>
</template>


<script setup lang="ts">
import { ref, onUnmounted, onMounted, nextTick, watch } from 'vue';
import { voiceWS } from '../services/ws';
import { getSessionId } from '../utils/session';
import type { WSServerMessage } from '../types';
import { MessageType, VoiceState } from '../types';

// 定义组件名称，供 KeepAlive 使用
defineOptions({
  name: 'home'
});

// ==================== 状态管理 ====================
const textInput = ref('');
const messages = ref<{ role: 'user' | 'assistant'; content: string; id: number; thinking?: boolean; createdAt: number }[]>([
  { role: 'assistant', content: '你好！我是语音助手，请问有什么可以帮助你的吗？', id: Date.now(), createdAt: Date.now() }
]);
const isRecording = ref(false);
const isLoading = ref(false);
const isWakeWordListening = ref(false);
const wakeWordDetected = ref(false);

const messagesContainer = ref<HTMLElement | null>(null);
const inputRef = ref<HTMLInputElement | null>(null);

let messageIdCounter = Date.now();

/**
 * Float32 转 Int16 PCM
 * @param float32Array Float32 音频数据
 * @returns Int16Array PCM 数据
 */
function float32ToInt16(float32Array: Float32Array): Int16Array {
  const int16Array = new Int16Array(float32Array.length);
  for (let i = 0; i < float32Array.length; i++) {
    const s = Math.max(-1, Math.min(1, float32Array[i]));
    int16Array[i] = s < 0 ? s * 0x8000 : s * 0x7FFF;
  }
  return int16Array;
}

// ==================== 音频录制相关 ====================
let audioContext: AudioContext | null = null;
let audioStream: MediaStream | null = null;
let analyser: AnalyserNode | null = null;
let animationFrame: number | null = null;
let audioProcessor: ScriptProcessorNode | null = null;
let audioSource: MediaStreamAudioSourceNode | null = null;

// 音频缓冲区管理
const AUDIO_BUFFER_SIZE = 3200; // 200ms @ 16kHz = 3200 samples
// const AUDIO_BUFFER_SIZE = 320; // 200ms @ 16kHz = 3200 samples
let audioBuffer: Int16Array[] = [];
let audioBufferTotalSamples = 0;

/**
 * 开始录音
 */
async function startRecording(): Promise<void> {
  try {
    console.log('[AudioRecorder] Starting recording...');

    // 检查 WebSocket 连接
    if (!voiceWS.isConnected()) {
      console.log('[AudioRecorder] WebSocket not connected, connecting...');
      voiceWS.connect();
    }

    // 获取麦克风权限（已启用浏览器降噪）
    audioStream = await navigator.mediaDevices.getUserMedia({
      audio: {
        noiseSuppression: true,
        echoCancellation: true,
        autoGainControl: true,
        sampleRate: 16000
      }
    });

    // 创建 AudioContext（16kHz 采样率）
    audioContext = new AudioContext({ sampleRate: 16000 });
    console.log('[AudioRecorder] Actual AudioContext sampleRate:', audioContext.sampleRate);

    // 创建分析器（用于可视化）
    analyser = audioContext.createAnalyser();
    analyser.fftSize = 256;

    // 创建音频源和处理器
    audioSource = audioContext.createMediaStreamSource(audioStream);
    audioProcessor = audioContext.createScriptProcessor(4096, 1, 1);

    // 重置音频缓冲区
    audioBuffer = [];
    audioBufferTotalSamples = 0;

    // 实时处理音频帧
    audioProcessor.onaudioprocess = (event) => {
      if (!isRecording.value) return
      const inputData = event.inputBuffer.getChannelData(0)
      const actualRate = audioContext!.sampleRate
      let pcmData: Int16Array
      if (actualRate !== 16000) {
        const ratio = actualRate / 16000
        const newLen = Math.round(inputData.length / ratio)
        const resampled = new Float32Array(newLen)
        for (let i = 0; i < newLen; i++) {
          const pos = i * ratio
          const idx = Math.floor(pos)
          const frac = pos - idx
          resampled[i] = idx + 1 < inputData.length
            ? inputData[idx] * (1 - frac) + inputData[idx + 1] * frac
            : inputData[idx]
        }
        pcmData = float32ToInt16(resampled)
      } else {
        pcmData = float32ToInt16(inputData)
      }
      audioBuffer.push(pcmData)
      audioBufferTotalSamples += pcmData.length
      if (audioBufferTotalSamples >= AUDIO_BUFFER_SIZE) {
        sendAccumulatedAudio(false)
      }
    };

    // 连接节点
    audioSource.connect(audioProcessor);
    audioProcessor.connect(analyser);
    analyser.connect(audioContext.destination);

    // 开始音频级别可视化
    updateAudioLevel();

    isRecording.value = true;
    console.log('[AudioRecorder] Recording started');

  } catch (error) {
    console.error('[AudioRecorder] Failed to start recording:', error);
    stopRecording();
  }
}

/**
 * 发送累积的音频数据
 * @param isLast 是否为最后一片（语音结束）
 */
function sendAccumulatedAudio(isLast: boolean = false): void {
  if (audioBuffer.length === 0 && !isLast) return;

  // 合并所有缓冲区
  const totalLength = audioBufferTotalSamples;
  const mergedBuffer = new Int16Array(totalLength);
  let offset = 0;
  for (const chunk of audioBuffer) {
    mergedBuffer.set(chunk, offset);
    offset += chunk.length;
  }

  // 发送音频数据（流式传输）
  voiceWS.sendAudio(mergedBuffer.buffer, 'pcm', isLast);
  console.log(`[AudioRecorder] Sent ${audioBuffer.length} voice frames, ${totalLength} samples, isLast=${isLast}`);

  // 重置缓冲区
  audioBuffer = [];
  audioBufferTotalSamples = 0;
}

/**
 * 停止录音
 */
function stopRecording(): void {
  console.log('[AudioRecorder] Stopping recording...');

  // 发送剩余缓冲区并通知后端音频结束（即使缓冲为空也必须发，触发 ASR 会话关闭）
  sendAccumulatedAudio(true);

  // 断开音频处理器连接
  if (audioSource) {
    audioSource.disconnect();
    audioSource = null;
  }

  if (audioProcessor) {
    audioProcessor.disconnect();
    audioProcessor = null;
  }

  if (animationFrame) {
    cancelAnimationFrame(animationFrame);
    animationFrame = null;
  }

  if (analyser) {
    analyser.disconnect();
    analyser = null;
  }

  if (audioContext) {
    audioContext.close();
    audioContext = null;
  }

  if (audioStream) {
    audioStream.getTracks().forEach(track => track.stop());
    audioStream = null;
  }

  // 重置缓冲区
  audioBuffer = [];
  audioBufferTotalSamples = 0;

  isRecording.value = false;
  console.log('[AudioRecorder] Recording stopped');
}

/**
 * 更新音频级别可视化
 */
function updateAudioLevel(): void {
  if (!analyser || !isRecording.value) return;

  const dataArray = new Uint8Array(analyser.frequencyBinCount);
  analyser.getByteFrequencyData(dataArray);

  // TODO: 这里可以添加音频波形可视化

  animationFrame = requestAnimationFrame(updateAudioLevel);
}

/**
 * 切换录音状态
 */
async function toggleRecording(): Promise<void> {
  if (isRecording.value) {
    stopRecording();
  } else {
    await startRecording();
  }
}

// ==================== 唤醒词检测相关 ====================
const WAKE_WORD_REGEX = /小爱同学/;
let wakeWordRecognition: any = null;

/**
 * 初始化唤醒词识别
 */
function initWakeWordRecognition(): any {
  const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;
  if (!SpeechRecognition) {
    console.warn('[WakeWord] Web Speech API not supported in this browser');
    return null;
  }

  const recognition = new SpeechRecognition();
  recognition.continuous = true;
  recognition.interimResults = true;
  recognition.lang = 'zh-CN';

  recognition.onstart = () => {
    console.log('[WakeWord] Recognition started');
    isWakeWordListening.value = true;
  };

  recognition.onresult = (event: any) => {
    let interimTranscript = '';
    let finalTranscript = '';

    for (let i = event.resultIndex; i < event.results.length; i++) {
      const transcript = event.results[i][0].transcript;
      if (event.results[i].isFinal) {
        finalTranscript += transcript;
      } else {
        interimTranscript += transcript;
      }
    }

    const combinedTranscript = (finalTranscript + interimTranscript).toLowerCase();
    console.log('[WakeWord] Heard:', combinedTranscript);

    // 检测唤醒词
    if (WAKE_WORD_REGEX.test(combinedTranscript)) {
      console.log('[WakeWord] Wake word detected!');
      handleWakeWordDetected();
    }
  };

  recognition.onerror = (event: any) => {
    console.error('[WakeWord] Recognition error:', event.error);
    if (event.error === 'not-allowed' || event.error === 'service-not-allowed') {
      console.warn('[WakeWord] Microphone permission denied');
      isWakeWordListening.value = false;
    }
  };

  recognition.onend = () => {
    console.log('[WakeWord] Recognition ended');
    isWakeWordListening.value = false;
    // 如果没有被唤醒词触发，且不在录音状态，则重启监听
    if (!wakeWordDetected.value && !isRecording.value) {
      setTimeout(() => {
        if (wakeWordRecognition && !isRecording.value) {
          try {
            wakeWordRecognition.start();
          } catch (error) {
            console.warn('[WakeWord] Could not restart recognition');
          }
        }
      }, 1000);
    }
  };

  return recognition;
}

/**
 * 停止唤醒词检测
 */
function stopWakeWordDetection(): void {
  if (wakeWordRecognition) {
    try {
      wakeWordRecognition.stop();
    } catch (error) {
      // 忽略停止错误
    }
    wakeWordRecognition = null;
  }
  isWakeWordListening.value = false;
  console.log('[WakeWord] Wake word detection stopped');
}

/**
 * 唤醒词触发处理
 */
function handleWakeWordDetected(): void {
  // 防止重复触发
  if (wakeWordDetected.value) return;
  wakeWordDetected.value = true;

  console.log('[WakeWord] Wake word detected, stopping detection...');

  // 停止唤醒词检测
  stopWakeWordDetection();

  // 显示提示消息
  messages.value.push({
    role: 'assistant',
    content: '听到你叫我了，正在打开语音输入...',
    id: ++messageIdCounter,
    createdAt: Date.now()
  });
  scrollToBottom();

  // 立即开始录音
  startRecording();
}

// ==================== WebSocket 消息处理 ====================
/**
 * 处理 WebSocket 消息
 * 根据 MessageType 枚举维护对话消息列表
 */
function handleWSMessage(message: WSServerMessage): void {
  console.log('[WebSocket] Received message:', message.type, message);

  switch (message.type) {
    // ========== 状态更新 ==========
    case MessageType.STATE_UPDATE:
      if (message.data) {
        const state = message.data as VoiceState;
        console.log('[WebSocket] State update:', state);
        // TODO: 根据 state 更新 UI 状态（如显示「思考中...」等）
      }
      break;

    // ========== ASR 实时识别结果（流式） ==========
    case MessageType.ASR_RESULT:
      if (message.data) {
        const asrData = message.data as { text?: string; isFinal?: boolean };
        console.log('[WebSocket] ASR result:', asrData.text);
        // 实时识别结果可以显示临时文本或更新已有消息
        // TODO: 如果需要显示「识别中...」的临时状态，可以在这里实现
      }
      break;

    // ========== ASR 识别完成（用户语音识别文字） ==========
    case MessageType.ASR_COMPLETE:
      if (message.text) {
        console.log('[WebSocket] ASR complete:', message.text);
        messages.value.push({
          role: 'user',
          content: message.text,
          id: ++messageIdCounter,
          createdAt: Date.now()
        });
        messages.value.push({
          role: 'assistant',
          content: '思考中...',
          id: ++messageIdCounter,
          thinking: true,
          createdAt: Date.now()
        });
        isLoading.value = true;
        scrollToBottom();
      }
      break;

    // ========== LLM 流式文本（打字机效果） ==========
    case MessageType.LLM_TEXT:
      if (message.text) {
        console.log('[WebSocket] LLM chunk:', message.text);
        // 查找并更新"思考中"占位消息
        const thinkingIdx = messages.value.findIndex(m => m.thinking);
        if (thinkingIdx !== -1) {
          messages.value[thinkingIdx] = {
            ...messages.value[thinkingIdx],
            role: 'assistant',
            content: message.text,
            thinking: false
          };
        } else {
          // 如果没有思考中消息，直接添加
          messages.value.push({
            role: 'assistant',
            content: message.text,
            id: ++messageIdCounter,
            createdAt: Date.now()
          });
        }
        scrollToBottom();
      }
      break;

    // ========== LLM 回复完成（大模型回复文字） ==========
    case MessageType.LLM_COMPLETE:
      if (message.text) {
        console.log('[WebSocket] LLM complete:', message.text);
        // 查找并替换"思考中"占位消息
        const thinkingIdx = messages.value.findIndex(m => m.thinking);
        if (thinkingIdx !== -1) {
          messages.value[thinkingIdx] = {
            ...messages.value[thinkingIdx],
            role: 'assistant',
            content: message.text,
            thinking: false
          };
        } else {
          // 如果没有思考中消息，直接添加
          messages.value.push({
            role: 'assistant',
            content: message.text,
            id: ++messageIdCounter,
            createdAt: Date.now()
          });
        }
        isLoading.value = false;
        scrollToBottom();
      }
      break;

    // ========== TTS 音频数据 ==========
    case MessageType.TTS_AUDIO:
      if (message.data instanceof ArrayBuffer) {
        console.log('[WebSocket] Received TTS audio (binary)');
        // TODO: 播放 TTS 音频
      } else if (message.data) {
        // 处理 base64 格式的音频数据
        const ttsData = message.data as { audio?: string; isLast?: boolean };
        if (ttsData?.audio) {
          console.log('[WebSocket] Received TTS audio (base64)');
          // TODO: 解码 base64 并播放音频
        }
      }
      break;

    // ========== TTS 播放完成 ==========
    case MessageType.TTS_COMPLETE:
      console.log('[WebSocket] TTS complete');
      // TODO: 更新 UI 状态，表示播放完成
      break;

    // ========== 错误消息 ==========
    case MessageType.ERROR:
      if (message.data) {
        const errorData = message.data as { code?: string; message?: string };
        console.error('[WebSocket] Error:', errorData);
        // 替换思考中消息为错误提示
        const errThinkingIdx = messages.value.findIndex(m => m.thinking);
        if (errThinkingIdx !== -1) {
          messages.value[errThinkingIdx] = {
            ...messages.value[errThinkingIdx],
            role: 'assistant',
            content: `抱歉，出错了: ${errorData?.message || '未知错误'}`
          };
        } else {
          messages.value.push({
            role: 'assistant',
            content: `抱歉，出错了: ${errorData?.message || '未知错误'}`,
            id: ++messageIdCounter,
            createdAt: Date.now()
          });
        }
        isLoading.value = false;
        scrollToBottom();
      }
      break;

    // ========== 打断通知 ==========
    case MessageType.INTERRUPT:
      if (message.data) {
        const interruptData = message.data as { source?: string; reason?: string };
        console.log('[WebSocket] Interrupted:', interruptData);
        // TODO: 处理打断逻辑（如停止播放 TTS）
      }
      break;

    // ========== 心跳响应 ==========
    case MessageType.PONG:
      // 心跳响应已在 ws.ts 中处理，这里忽略
      break;

    default:
      console.log('[WebSocket] Unhandled message type:', message.type);
  }
}

// ==================== 文字消息相关 ====================
/**
 * 发送文字消息
 */
async function sendTextMessage(): Promise<void> {
  if (!textInput.value.trim() || isLoading.value) return;

  const userMessage = textInput.value.trim();
  textInput.value = '';

  // 添加用户消息
  messages.value.push({
    role: 'user',
    content: userMessage,
    id: ++messageIdCounter,
    createdAt: Date.now()
  });

  // 添加"思考中"占位消息
  const thinkingId = ++messageIdCounter;
  messages.value.push({
    role: 'assistant',
    content: '思考中...',
    id: thinkingId,
    thinking: true,
    createdAt: Date.now()
  });
  scrollToBottom();

  // 通过 WebSocket 发送
  isLoading.value = true;
  try {
    voiceWS.sendText(userMessage);
    console.log('[WebSocket] Sent text message:', userMessage);
  } catch (error) {
    console.error('[WebSocket] Failed to send text:', error);
    // 发送失败，移除思考中消息并显示错误
    const idx = messages.value.findIndex(m => m.id === thinkingId);
    if (idx !== -1) {
      messages.value[idx] = {
        ...messages.value[idx],
        role: 'assistant',
        content: '抱歉，发送消息失败，请稍后重试。'
      };
    }
    isLoading.value = false;
    scrollToBottom();
  }
}

// ==================== 工具函数 ====================
function formatTime(timestamp: number): string {
  return new Date(timestamp).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
}

function scrollToBottom(): void {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
    }
  });
}

function focusInput(): void {
  inputRef.value?.focus();
}

watch(messages, scrollToBottom, { deep: true });

// ==================== 生命周期管理 ====================
onMounted(async () => {
  console.log('[HomePage] Component mounted');

  // 1. 生成 sessionId 并连接 WebSocket
  const sessionId = await getSessionId();
  voiceWS.setSessionId(sessionId);
  voiceWS.connect(sessionId);
  voiceWS.onMessage(handleWSMessage);

  voiceWS.onConnect(() => {
    console.log('[WebSocket] Connected to voice service');
  });

  voiceWS.onDisconnect(() => {
    console.log('[WebSocket] Disconnected from voice service');
  });

  voiceWS.onError((error) => {
    console.error('[WebSocket] Error:', error);
  });

});

onUnmounted(() => {
  console.log('[HomePage] Component unmounting');

  // 停止录音
  if (isRecording.value) {
    stopRecording();
  }

  // 停止唤醒词检测
  stopWakeWordDetection();

  // 断开 WebSocket
  voiceWS.disconnect();
});
</script>


<style scoped>
.home-page {
  min-height: calc(100vh - 140px);
  position: relative;
  overflow: hidden;
}

/* Warm Cream Background */
.ambient-bg {
  position: absolute;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  overflow: hidden;
}

.cream-gradient {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse at 20% 20%, rgba(251, 146, 60, 0.08) 0%, transparent 50%),
    radial-gradient(ellipse at 80% 80%, rgba(249, 115, 22, 0.06) 0%, transparent 50%),
    radial-gradient(ellipse at 50% 50%, rgba(254, 215, 170, 0.05) 0%, transparent 70%);
}

/* Main Content */
.main-content {
  position: relative;
  z-index: 1;
  max-width: 900px;
  margin: 0 auto;
  padding: 24px 24px 60px;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(24px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* Chat Section */
.chat-section {
  margin-bottom: 48px;
  animation: fadeInUp 0.8s ease-out 0.1s backwards;
  margin-top: 0;
}

.chat-card {
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(24px);
  -webkit-backdrop-filter: blur(24px);
  border-radius: 28px;
  border: 1px solid var(--color-border);
  box-shadow:
    0 4px 24px rgba(249, 115, 22, 0.06),
    0 12px 48px rgba(249, 115, 22, 0.08);
  overflow: hidden;
}

/* Chat Header */
.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--color-border);
  background: rgba(255, 255, 255, 0.8);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.ai-avatar {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 4px 16px rgba(249, 115, 22, 0.3);
}

.ai-avatar svg {
  width: 24px;
  height: 24px;
}

.header-info h2 {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 4px;
  letter-spacing: -0.01em;
}

.status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-muted);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #d1d5db;
  transition: all 0.3s ease;
}

.status-dot.active {
  background: var(--color-primary);
  box-shadow: 0 0 8px var(--color-primary);
  animation: glow-pulse 2s ease-in-out infinite;
}

.status-dot.listening {
  background: #22c55e;
  box-shadow: 0 0 8px #22c55e;
  animation: listening-pulse 1.5s ease-in-out infinite;
}

.status-dot.thinking {
  background: #f59e0b;
  box-shadow: 0 0 8px #f59e0b;
  animation: thinking-pulse 1s ease-in-out infinite;
}

@keyframes thinking-pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.5;
    transform: scale(0.85);
  }
}

/* Thinking dots animation */
.thinking-dots {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 0;
}

.thinking-dots .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
  opacity: 0.4;
  animation: dot-bounce 1.4s ease-in-out infinite;
}

.thinking-dots .dot:nth-child(2) {
  animation-delay: 0.2s;
}

.thinking-dots .dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes dot-bounce {
  0%, 80%, 100% {
    opacity: 0.4;
    transform: scale(0.8);
  }
  40% {
    opacity: 1;
    transform: scale(1.1);
  }
}

@keyframes listening-pulse {

  0%,
  100% {
    opacity: 1;
    transform: scale(1);
  }

  50% {
    opacity: 0.6;
    transform: scale(0.9);
  }
}

.header-actions {
  display: flex;
  gap: 8px;
}

.icon-btn {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  border: 1px solid var(--color-border);
  background: rgba(255, 255, 255, 0.8);
  color: var(--color-text-muted);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.icon-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: rgba(249, 115, 22, 0.05);
}

.icon-btn svg {
  width: 20px;
  height: 20px;
}

/* Messages Area */
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  min-height: 500px;
  max-height: 60vh;
  scroll-behavior: smooth;
}

.messages-inner {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.chat-message {
  display: flex;
  gap: 14px;
  max-width: 80%;
  opacity: 0;
  transform: translateY(12px);
  animation: message-in 0.4s cubic-bezier(0.16, 1, 0.3, 1) forwards;
  animation-delay: var(--delay);
}

@keyframes message-in {
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.chat-message.user {
  flex-direction: row-reverse;
  align-self: flex-end;
  margin-left: auto;
}

.chat-message.assistant {
  align-self: flex-start;
}

.message-avatar {
  width: 40px;
  height: 40px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.message-avatar.assistant {
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  color: white;
  box-shadow: 0 4px 14px rgba(249, 115, 22, 0.25);
}

.message-avatar.user {
  background: linear-gradient(135deg, var(--color-primary-dark) 0%, var(--color-primary) 100%);
  color: white;
  box-shadow: 0 4px 14px rgba(249, 115, 22, 0.25);
}

.message-avatar svg {
  width: 20px;
  height: 20px;
}

.message-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.message-bubble {
  padding: 16px 20px;
  border-radius: 20px;
  font-size: 15px;
  line-height: 1.6;
  position: relative;
  word-break: break-word;
}

.message-bubble.assistant {
  background: linear-gradient(135deg, #ffffff 0%, #fff7ed 100%);
  border: 1px solid var(--color-border);
  box-shadow:
    0 2px 8px rgba(0, 0, 0, 0.04),
    0 8px 24px rgba(0, 0, 0, 0.04);
  border-radius: 20px 20px 20px 6px;
}

.message-bubble.user {
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  color: white;
  box-shadow:
    0 4px 14px rgba(249, 115, 22, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 0.15);
  border-radius: 20px 20px 6px 20px;
}

.message-bubble p {
  margin: 0;
}

.message-meta {
  padding: 0 4px;
}

.message-meta .time {
  font-size: 11px;
  color: var(--color-text-muted);
}

/* Empty State */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.empty-icon {
  width: 80px;
  height: 80px;
  border-radius: 24px;
  background: linear-gradient(135deg, rgba(249, 115, 22, 0.1) 0%, rgba(249, 115, 22, 0.05) 100%);
  border: 1px solid rgba(249, 115, 22, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  margin-bottom: 16px;
}

.empty-icon svg {
  width: 36px;
  height: 36px;
  opacity: 0.6;
}

.empty-state p {
  font-size: 14px;
  color: var(--color-text-muted);
  margin: 0;
}

/* Input Section */
.input-section {
  padding: 16px 24px 24px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0) 0%, rgba(255, 247, 237, 0.95) 20%);
  transition: all 0.3s ease;
}

.input-section.recording {
  background: linear-gradient(180deg, rgba(249, 115, 22, 0.02) 0%, rgba(249, 115, 22, 0.06) 100%);
}

.input-section.calibrating {
  background: linear-gradient(180deg, rgba(249, 115, 22, 0.02) 0%, rgba(249, 115, 22, 0.08) 100%);
}

/* Calibration Panel */
.calibration-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 20px 24px;
  background: rgba(249, 115, 22, 0.04);
  border: 2px solid rgba(249, 115, 22, 0.2);
  border-radius: 16px;
  margin-bottom: 8px;
}

.calibration-wave {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 32px;
}

.cw-bar {
  display: block;
  width: 4px;
  background: var(--color-primary);
  border-radius: 2px;
  animation: cw-bounce 1.2s ease-in-out infinite;
  animation-delay: calc(var(--i) * 0.15s);
  opacity: 0.7;
}

.cw-bar:nth-child(1) { height: 12px; }
.cw-bar:nth-child(2) { height: 20px; }
.cw-bar:nth-child(3) { height: 28px; }
.cw-bar:nth-child(4) { height: 20px; }
.cw-bar:nth-child(5) { height: 12px; }

@keyframes cw-bounce {
  0%, 100% { transform: scaleY(0.5); opacity: 0.4; }
  50% { transform: scaleY(1.3); opacity: 1; }
}

.calibration-info {
  text-align: center;
}

.calibration-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
  margin: 0 0 4px;
}

.calibration-sub {
  font-size: 12px;
  color: var(--color-text-muted);
  margin: 0;
}

.calibration-track {
  width: 100%;
  height: 4px;
  background: rgba(249, 115, 22, 0.15);
  border-radius: 2px;
  overflow: hidden;
}

.calibration-fill {
  height: 100%;
  background: linear-gradient(90deg, var(--color-primary), var(--color-primary-light));
  border-radius: 2px;
  transition: width 0.3s ease;
}

.calibration-fade-enter-active,
.calibration-fade-leave-active {
  transition: all 0.3s ease;
}

.calibration-fade-enter-from,
.calibration-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Input Row Layout */
.input-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.input-container {
  flex: 1;
  position: relative;
}

.chat-input {
  width: 100%;
  padding: 16px 20px;
  border: 2px solid var(--color-border);
  border-radius: 16px;
  font-size: 15px;
  font-family: var(--font-sans);
  background: rgba(255, 255, 255, 0.98);
  outline: none;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  color: var(--color-text);
}

.chat-input::placeholder {
  color: var(--color-text-muted);
}

.chat-input:focus {
  border-color: var(--color-primary);
  background: white;
  box-shadow:
    0 0 0 4px rgba(249, 115, 22, 0.08),
    0 4px 20px rgba(249, 115, 22, 0.1);
}

.input-glow {
  position: absolute;
  inset: -2px;
  border-radius: 18px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  opacity: 0;
  z-index: -1;
  filter: blur(8px);
  transition: opacity 0.25s ease;
}

.chat-input:focus+.input-glow {
  opacity: 0.15;
}

/* Action Buttons (voice and send) */
.action-btn {
  width: 52px;
  height: 52px;
  border: none;
  border-radius: 16px;
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  color: white;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow:
    0 4px 14px rgba(249, 115, 22, 0.3),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
  flex-shrink: 0;
}

.action-btn:hover:not(:disabled) {
  transform: translateY(-2px) scale(1.02);
  box-shadow:
    0 6px 20px rgba(249, 115, 22, 0.4),
    inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.action-btn:active:not(:disabled) {
  transform: translateY(0) scale(0.98);
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.action-btn svg {
  width: 22px;
  height: 22px;
}

/* Voice Button Recording State */
.voice-btn.recording {
  background: linear-gradient(135deg, var(--color-primary-dark) 0%, var(--color-primary) 100%);
  animation: recording-pulse 1.5s ease-in-out infinite;
}

@keyframes recording-pulse {

  0%,
  100% {
    transform: scale(1);
  }

  50% {
    transform: scale(1.08);
  }
}

/* Listening Indicator */
.listening-indicator {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 16px 20px;
  background: rgba(249, 115, 22, 0.05);
  border: 2px solid rgba(249, 115, 22, 0.2);
  border-radius: 16px;
}

.listening-icon {
  position: relative;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.listening-ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 2px solid rgba(249, 115, 22, 0.4);
  animation: ring-expand 2s ease-out infinite;
}

.listening-ring.ring-1 {
  animation-delay: 0s;
}

.listening-ring.ring-2 {
  animation-delay: 0.4s;
}

.listening-ring.ring-3 {
  animation-delay: 0.8s;
}

@keyframes ring-expand {
  0% {
    transform: scale(1);
    opacity: 0.6;
  }

  100% {
    transform: scale(1.8);
    opacity: 0;
  }
}

.mic-icon {
  width: 24px;
  height: 24px;
  color: var(--color-primary);
  position: relative;
  z-index: 1;
}

.listening-label {
  font-size: 15px;
  font-weight: 500;
  color: var(--color-primary);
  animation: pulse-text 1.5s ease-in-out infinite;
}

@keyframes pulse-text {

  0%,
  100% {
    opacity: 1;
  }

  50% {
    opacity: 0.6;
  }
}

/* Features Section */
.features-section {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  animation: fadeInUp 0.8s ease-out 0.2s backwards;
}

.feature-card {
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-radius: 20px;
  border: 1px solid var(--color-border);
  padding: 24px;
  text-align: center;
  transition: all 0.3s ease;
}

.feature-card:hover {
  transform: translateY(-4px);
  box-shadow:
    0 8px 24px rgba(249, 115, 22, 0.1),
    0 16px 48px rgba(249, 115, 22, 0.08);
}

.feature-icon {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: linear-gradient(135deg, rgba(249, 115, 22, 0.1) 0%, rgba(249, 115, 22, 0.05) 100%);
  border: 1px solid rgba(249, 115, 22, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  margin: 0 auto 16px;
  transition: all 0.3s ease;
}

.feature-card:hover .feature-icon {
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%);
  color: white;
  box-shadow: 0 4px 16px rgba(249, 115, 22, 0.3);
}

.feature-icon svg {
  width: 26px;
  height: 26px;
}

.feature-card h3 {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 4px;
  letter-spacing: -0.01em;
}

.feature-card p {
  font-size: 13px;
  color: var(--color-text-muted);
  margin: 0;
}

/* Transitions */
.message-enter-active {
  transition: all 0.4s cubic-bezier(0.16, 1, 0.3, 1);
}

.message-leave-active {
  transition: all 0.2s ease;
}

.message-enter-from {
  opacity: 0;
  transform: translateY(12px);
}

.message-leave-to {
  opacity: 0;
  transform: scale(0.95);
}

/* Responsive */
@media (max-width: 768px) {
  .main-content {
    padding: 24px 16px 40px;
  }

  .features-section {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .feature-card {
    display: flex;
    align-items: center;
    gap: 16px;
    text-align: left;
    padding: 16px 20px;
  }

  .feature-icon {
    margin: 0;
    flex-shrink: 0;
  }

  .chat-messages {
    min-height: 400px;
    max-height: 50vh;
    padding: 16px;
  }

  .chat-message {
    max-width: 88%;
  }

  .input-section {
    padding: 12px 16px 20px;
  }

  .action-btn {
    width: 48px;
    height: 48px;
  }
}

@media (prefers-reduced-motion: reduce) {

  .status-dot,
  .listening-ring,
  .chat-message,
  .chat-section,
  .features-section,
  .feature-card {
    animation: none !important;
  }

  .chat-message,
  .chat-section,
  .features-section {
    opacity: 1 !important;
    transform: none !important;
  }

  * {
    transition-duration: 0.01ms !important;
  }
}
</style>
