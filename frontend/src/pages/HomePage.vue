<script setup lang="ts">
import { ref, onUnmounted, nextTick, watch, onMounted } from 'vue';
import { voiceWS as asrService } from '../services/ws';
import type { ASRMessage } from '../types';

const textInput = ref('');
const messages = ref<{ role: 'user' | 'assistant'; content: string; id: number }[]>([
  { role: 'assistant', content: '你好！我是语音助手，请问有什么可以帮助你的吗？', id: Date.now() }
]);
const isRecording = ref(false);
const isWakeWordListening = ref(false);
const wakeWordDetected = ref(false);

// Wake word detection constants
const WAKE_WORD = '小爱同学';
const WAKE_WORD_REGEX = /小爱同学/;

// Web Speech API recognition
let wakeWordRecognition: any = null;

let mediaRecorder: MediaRecorder | null = null;
let audioContext: AudioContext | null = null;
let analyser: AnalyserNode | null = null;
let animationFrame: number | null = null;
let messageIdCounter = Date.now();

const messagesContainer = ref<HTMLElement | null>(null);
const inputRef = ref<HTMLInputElement | null>(null);

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight;
    }
  });
}

watch(messages, scrollToBottom, { deep: true });

function sendTextMessage() {
  if (!textInput.value.trim()) return;

  messages.value.push({
    role: 'user',
    content: textInput.value,
    id: ++messageIdCounter
  });

  scrollToBottom();

  setTimeout(() => {
    messages.value.push({
      role: 'assistant',
      content: `收到: "${textInput.value}"，正在处理中...`,
      id: ++messageIdCounter
    });
    scrollToBottom();
  }, 600);

  textInput.value = '';
}

async function toggleRecording() {
  if (isRecording.value) {
    stopRecording();
  } else {
    await startRecording();
  }
}

async function startRecording() {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true });

    audioContext = new AudioContext();
    analyser = audioContext.createAnalyser();
    const source = audioContext.createMediaStreamSource(stream);
    source.connect(analyser);
    analyser.fftSize = 256;

    updateAudioLevel();

    // Connect WebSocket in background, don't wait
    asrService.connect();

    mediaRecorder = new MediaRecorder(stream, {
      mimeType: 'audio/webm;codecs=opus',
    });

    mediaRecorder.ondataavailable = async (event) => {
      if (event.data.size > 0) {
        const arrayBuffer = await event.data.arrayBuffer();
        asrService.send(arrayBuffer);
      }
    };

    mediaRecorder.onstop = () => {
      stream.getTracks().forEach((track) => track.stop());
    };

    mediaRecorder.start(100);

    isRecording.value = true;

    asrService.onMessage(handleASRMessage);
  } catch (e) {
    console.error('Failed to start recording:', e);
    isRecording.value = false;
  }
}

function stopRecording() {
  if (mediaRecorder && mediaRecorder.state !== 'inactive') {
    mediaRecorder.stop();
  }

  if (animationFrame) {
    cancelAnimationFrame(animationFrame);
    animationFrame = null;
  }

  if (audioContext) {
    audioContext.close();
    audioContext = null;
  }

  asrService.disconnect();

  isRecording.value = false;
}

function updateAudioLevel() {
  if (!analyser || !isRecording.value) return;

  const dataArray = new Uint8Array(analyser.frequencyBinCount);
  analyser.getByteFrequencyData(dataArray);

  animationFrame = requestAnimationFrame(updateAudioLevel);
}

function handleASRMessage(message: ASRMessage) {
  if (message.text) {
    messages.value.push({
      role: 'user',
      content: message.text,
      id: ++messageIdCounter
    });

    scrollToBottom();

    setTimeout(() => {
      messages.value.push({
        role: 'assistant',
        content: `收到: "${message.text}"，这是自动回复。`,
        id: ++messageIdCounter
      });
      scrollToBottom();
    }, 800);
  }
}

function focusInput() {
  inputRef.value?.focus();
}

onUnmounted(() => {
  if (isRecording.value) {
    stopRecording();
  }
  stopWakeWordDetection();
});

// ============================================
// Wake Word Detection (Auto Listen on Load)
// ============================================

function initWakeWordRecognition() {
  // Check if Web Speech API is supported
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

    // Check for wake word
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
    // Restart listening if still in wake word mode and not recording
    if (!wakeWordDetected.value && !isRecording.value) {
      setTimeout(() => {
        if (wakeWordRecognition && !isRecording.value) {
          try {
            wakeWordRecognition.start();
          } catch (e) {
            console.warn('[WakeWord] Could not restart recognition');
          }
        }
      }, 1000);
    }
  };

  return recognition;
}

async function startWakeWordDetection() {
  // Request microphone permission first
  try {
    await navigator.mediaDevices.getUserMedia({ audio: true });
  } catch (e) {
    console.warn('[WakeWord] Microphone permission denied');
    return;
  }

  wakeWordRecognition = initWakeWordRecognition();
  if (wakeWordRecognition) {
    try {
      wakeWordRecognition.start();
    } catch (e) {
      console.error('[WakeWord] Failed to start recognition:', e);
    }
  }
}

function stopWakeWordDetection() {
  if (wakeWordRecognition) {
    try {
      wakeWordRecognition.stop();
    } catch (e) {
      // Ignore errors when stopping
    }
    wakeWordRecognition = null;
  }
  isWakeWordListening.value = false;
}

function handleWakeWordDetected() {
  // Guard: prevent duplicate execution
  if (wakeWordDetected.value) return;
  wakeWordDetected.value = true;

  // Stop wake word detection
  stopWakeWordDetection();

  // Add system message
  messages.value.push({
    role: 'assistant',
    content: '听到你叫我了，正在打开语音输入...',
    id: ++messageIdCounter
  });
  scrollToBottom();

  // Immediately start voice recording (removed 800ms delay)
  startRecording();
}

// Auto-start wake word detection when page loads
onMounted(() => {
  // Small delay to ensure page is fully loaded
  setTimeout(() => {
    startWakeWordDetection();
  }, 1000);
});
</script>

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
                  <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
                </svg>
              </div>
              <div class="header-info">
                <h2>语音助手</h2>
                <div class="status">
                  <span class="status-dot" :class="{ active: !isRecording, listening: isWakeWordListening }"></span>
                  <span>{{ isRecording ? '正在聆听...' : (isWakeWordListening ? '等待唤醒...' : '随时待命') }}</span>
                </div>
              </div>
            </div>
            <div class="header-actions">
              <button class="icon-btn" title="设置">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/>
                  <circle cx="12" cy="12" r="3"/>
                </svg>
              </button>
            </div>
          </div>

          <!-- Messages Area -->
          <div class="chat-messages" ref="messagesContainer">
            <TransitionGroup name="message" tag="div" class="messages-inner">
              <div
                v-for="(msg, index) in messages"
                :key="msg.id"
                class="chat-message"
                :class="msg.role"
                :style="{ '--delay': `${index * 0.05}s` }"
              >
                <div class="message-avatar" :class="msg.role">
                  <svg v-if="msg.role === 'assistant'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/>
                  </svg>
                  <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <path d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/>
                  </svg>
                </div>
                <div class="message-content">
                  <div class="message-bubble" :class="msg.role">
                    <p>{{ msg.content }}</p>
                  </div>
                  <div class="message-meta">
                    <span class="time">{{ new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) }}</span>
                  </div>
                </div>
              </div>
            </TransitionGroup>

            <!-- Empty State Hint -->
            <div v-if="messages.length === 0" class="empty-state">
              <div class="empty-icon">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
                  <path d="M12 2a3 3 0 00-3 3v7a3 3 0 006 0V5a3 3 0 00-3-3zM19 10v2a7 7 0 01-14 0v-2M12 19v3M8 22h8"/>
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
                <input
                  ref="inputRef"
                  v-model="textInput"
                  type="text"
                  class="chat-input"
                  placeholder="输入消息..."
                  @keyup.enter="sendTextMessage"
                />
                <div class="input-glow"></div>
              </div>

              <!-- Listening Indicator (shown when recording) -->
              <div v-show="isRecording" class="listening-indicator">
                <span class="listening-label">正在聆听...</span>
              </div>

              <!-- Voice Button (toggles recording) -->
              <button
                class="action-btn voice-btn"
                :class="{ recording: isRecording }"
                @click="toggleRecording"
                :title="isRecording ? '结束录音' : '开始录音'"
              >
                <svg v-if="!isRecording" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 1a3 3 0 00-3 3v8a3 3 0 006 0V4a3 3 0 00-3-3z"/>
                  <path d="M19 10v2a7 7 0 01-14 0v-2"/>
                  <line x1="12" y1="19" x2="12" y2="23"/>
                  <line x1="8" y1="23" x2="16" y2="23"/>
                </svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z"/>
                  <path d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2"/>
                </svg>
              </button>

              <!-- Send Button (sends text message) -->
              <button
                class="action-btn send-btn"
                @click="sendTextMessage"
                :disabled="!textInput.trim()"
                title="发送消息"
              >
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
                </svg>
              </button>
            </div>
          </div>
        </div>
      </section>

    </div>
  </div>
</template>

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

@keyframes listening-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.6; transform: scale(0.9); }
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

.chat-input:focus + .input-glow {
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
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.08); }
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

.listening-ring.ring-1 { animation-delay: 0s; }
.listening-ring.ring-2 { animation-delay: 0.4s; }
.listening-ring.ring-3 { animation-delay: 0.8s; }

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
  0%, 100% { opacity: 1; }
  50% { opacity: 0.6; }
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
