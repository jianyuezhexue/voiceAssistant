<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue';
import { todoApi, knowledgeApi } from './services/api';
import { asrService } from './services/ws';
import type { Todo, Knowledge, ASRMessage } from './types';

// State
const todos = ref<Todo[]>([]);
const knowledgeItems = ref<Knowledge[]>([]);
const transcriptText = ref('');
const isRecording = ref(false);
const meetingTitle = ref('开始新的会议');
const meetingStatus = ref<'idle' | 'recording' | 'ended'>('idle');
const audioLevel = ref(0);
const isInitialLoading = ref(true); // 控制初始加载状态
const loadingProgress = ref(0); // 加载进度

// MediaRecorder for audio capture
let mediaRecorder: MediaRecorder | null = null;
let audioContext: AudioContext | null = null;
let analyser: AnalyserNode | null = null;
let animationFrame: number | null = null;

// Computed
const statusText = computed(() => {
  switch (meetingStatus.value) {
    case 'idle':
      return '准备就绪';
    case 'recording':
      return '正在录音...';
    case 'ended':
      return '会议已结束';
    default:
      return '';
  }
});

// Load initial data - 后台异步加载，不阻塞页面渲染
async function loadData() {
  loadingProgress.value = 10;
  try {
    const [todoData, knowledgeData] = await Promise.all([
      todoApi.list(),
      knowledgeApi.list(),
    ]);
    loadingProgress.value = 80;
    todos.value = todoData || [];
    knowledgeItems.value = knowledgeData || [];
    loadingProgress.value = 100;
  } catch (e) {
    console.error('Failed to load data:', e);
  } finally {
    // 延迟一小段时间让用户感知加载完成
    setTimeout(() => {
      isInitialLoading.value = false;
    }, 300);
  }
}

// Toggle recording
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

    // Setup audio analysis for level visualization
    audioContext = new AudioContext();
    analyser = audioContext.createAnalyser();
    const source = audioContext.createMediaStreamSource(stream);
    source.connect(analyser);
    analyser.fftSize = 256;

    // Start visual feedback
    updateAudioLevel();

    // Setup MediaRecorder
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

    // Send audio data every 100ms
    mediaRecorder.start(100);

    // Connect to ASR WebSocket
    asrService.connect();

    isRecording.value = true;
    meetingStatus.value = 'recording';
    meetingTitle.value = `会议 ${new Date().toLocaleTimeString()}`;

    // Listen for ASR messages
    asrService.onMessage(handleASRMessage);
  } catch (e) {
    console.error('Failed to start recording:', e);
    alert('无法访问麦克风，请检查权限设置');
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
  meetingStatus.value = 'ended';
  audioLevel.value = 0;
}

function updateAudioLevel() {
  if (!analyser || !isRecording.value) return;

  const dataArray = new Uint8Array(analyser.frequencyBinCount);
  analyser.getByteFrequencyData(dataArray);

  const average = dataArray.reduce((a, b) => a + b, 0) / dataArray.length;
  audioLevel.value = Math.min(average / 128, 1);

  animationFrame = requestAnimationFrame(updateAudioLevel);
}

function handleASRMessage(message: ASRMessage) {
  if (message.text) {
    transcriptText.value += message.text + ' ';
  }

  if (message.type === 'todo' && message.data) {
    const todo = message.data as Todo;
    const existingIndex = todos.value.findIndex((t) => t.id === todo.id);
    if (existingIndex >= 0) {
      todos.value[existingIndex] = todo;
    } else {
      todos.value.unshift(todo);
    }
  }

  if (message.type === 'knowledge' && message.data) {
    const knowledge = message.data as Knowledge;
    knowledgeItems.value.unshift(knowledge);
  }
}

function endMeeting() {
  stopRecording();
  meetingTitle.value = '会议已结束';
}

async function toggleTodoComplete(todo: Todo) {
  try {
    const updated = await todoApi.update(todo.id, {
      completed: !todo.completed,
      status: todo.completed ? 'pending' : 'completed',
    });
    const index = todos.value.findIndex((t) => t.id === todo.id);
    if (index >= 0) {
      todos.value[index] = updated;
    }
  } catch (e) {
    console.error('Failed to update todo:', e);
  }
}

async function deleteTodo(id: number) {
  try {
    await todoApi.delete(id);
    todos.value = todos.value.filter((t) => t.id !== id);
  } catch (e) {
    console.error('Failed to delete todo:', e);
  }
}

async function deleteKnowledge(id: number) {
  try {
    await knowledgeApi.delete(id);
    knowledgeItems.value = knowledgeItems.value.filter((k) => k.id !== id);
  } catch (e) {
    console.error('Failed to delete knowledge:', e);
  }
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

onMounted(() => {
  // 先渲染页面，后台加载数据
  // 使用 requestAnimationFrame 确保页面先渲染完成
  requestAnimationFrame(() => {
    loadData();
  });
});

onUnmounted(() => {
  stopRecording();
});
</script>

<template>
  <div class="app-container">
    <!-- 初始加载进度条 -->
    <div v-if="isInitialLoading" class="loading-overlay">
      <div class="loading-content">
        <div class="loading-spinner"></div>
        <p class="loading-text">正在加载...</p>
        <div class="loading-bar-container">
          <div class="loading-bar" :style="{ width: loadingProgress + '%' }"></div>
        </div>
      </div>
    </div>

    <!-- Background effects -->
    <div class="gradient-orb gradient-orb-1"></div>
    <div class="gradient-orb gradient-orb-2"></div>
    <div class="noise-overlay"></div>

    <!-- Main Layout -->
    <div class="main-layout">
      <!-- Left Sidebar - Todos -->
      <aside class="sidebar sidebar-left animate-slide-in-left">
        <div class="sidebar-header">
          <div class="flex items-center gap-2">
            <svg
              class="w-5 h-5 text-[var(--color-accent-cyan)]"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
              />
            </svg>
            <h2 class="sidebar-title">待办事项</h2>
          </div>
          <span class="badge">{{ todos.length }}</span>
        </div>

        <div class="sidebar-content">
          <div v-if="todos.length === 0" class="empty-state">
            <svg
              class="w-12 h-12 mx-auto mb-3 opacity-30"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.5"
                d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
              />
            </svg>
            <p class="text-[var(--color-text-muted)] text-sm">暂无待办事项</p>
            <p class="text-[var(--color-text-muted)] text-xs mt-1">
              开始说话自动提取
            </p>
          </div>

          <div class="todo-list">
            <div
              v-for="todo in todos"
              :key="todo.id"
              class="todo-item glass-card"
            >
              <button
                class="todo-checkbox"
                :class="{ completed: todo.completed }"
                @click="toggleTodoComplete(todo)"
              >
                <svg
                  v-if="todo.completed"
                  class="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2.5"
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              </button>
              <div class="todo-content">
                <span
                  class="todo-title"
                  :class="{ completed: todo.completed }"
                >
                  {{ todo.title }}
                </span>
                <span class="todo-status" :class="todo.status">
                  {{ todo.status === 'completed' ? '已完成' : '待处理' }}
                </span>
              </div>
              <button class="todo-delete" @click="deleteTodo(todo.id)">
                <svg
                  class="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </aside>

      <!-- Center - Voice Area -->
      <main class="center-area">
        <!-- Header -->
        <header class="center-header">
          <div class="meeting-info">
            <h1 class="meeting-title">{{ meetingTitle }}</h1>
            <div class="status-row">
              <span class="status-dot" :class="{ active: isRecording }"></span>
              <span class="status-text">{{ statusText }}</span>
            </div>
          </div>
        </header>

        <!-- Voice Button -->
        <div class="voice-section">
          <div class="mic-wrapper">
            <!-- Audio level rings -->
            <div
              v-if="isRecording"
              class="audio-ring audio-ring-1"
              :style="{ transform: `scale(${1 + audioLevel * 0.3})` }"
            ></div>
            <div
              v-if="isRecording"
              class="audio-ring audio-ring-2"
              :style="{ transform: `scale(${1 + audioLevel * 0.5})` }"
            ></div>

            <button
              class="mic-button"
              :class="{ recording: isRecording }"
              @click="toggleRecording"
            >
              <div class="mic-inner">
                <svg
                  v-if="!isRecording"
                  class="mic-icon"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="1.5"
                    d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z"
                  />
                </svg>
                <svg
                  v-else
                  class="mic-icon"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="1.5"
                    d="M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z"
                  />
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="1.5"
                    d="M17 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2"
                  />
                </svg>
              </div>
            </button>
          </div>

          <p class="mic-hint">
            {{ isRecording ? '点击停止录音' : '点击开始说话' }}
          </p>
        </div>

        <!-- Transcript Area -->
        <div class="transcript-section glass-card">
          <div class="transcript-header">
            <span class="transcript-label">实时转写</span>
            <button
              v-if="transcriptText"
              class="clear-btn"
              @click="transcriptText = ''"
            >
              清空
            </button>
          </div>
          <div class="transcript-content">
            <p v-if="!transcriptText" class="transcript-placeholder">
              说话内容将实时显示在这里...
            </p>
            <p v-else class="transcript-text">{{ transcriptText }}</p>
          </div>
        </div>

        <!-- End Meeting Button -->
        <button
          v-if="meetingStatus === 'recording'"
          class="end-meeting-btn"
          @click="endMeeting"
        >
          <svg
            class="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"
            />
          </svg>
          结束会议
        </button>
      </main>

      <!-- Right Sidebar - Knowledge -->
      <aside class="sidebar sidebar-right animate-slide-in-right">
        <div class="sidebar-header">
          <div class="flex items-center gap-2">
            <svg
              class="w-5 h-5 text-[var(--color-accent-purple)]"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
              />
            </svg>
            <h2 class="sidebar-title">知识库</h2>
          </div>
          <span class="badge">{{ knowledgeItems.length }}</span>
        </div>

        <div class="sidebar-content">
          <div v-if="knowledgeItems.length === 0" class="empty-state">
            <svg
              class="w-12 h-12 mx-auto mb-3 opacity-30"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="1.5"
                d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"
              />
            </svg>
            <p class="text-[var(--color-text-muted)] text-sm">暂无知识内容</p>
            <p class="text-[var(--color-text-muted)] text-xs mt-1">
              开始说话自动提取
            </p>
          </div>

          <div class="knowledge-list">
            <div
              v-for="item in knowledgeItems"
              :key="item.id"
              class="knowledge-item glass-card"
            >
              <div class="knowledge-header">
                <h3 class="knowledge-title">{{ item.title }}</h3>
                <button
                  class="knowledge-delete"
                  @click="deleteKnowledge(item.id)"
                >
                  <svg
                    class="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      stroke-width="2"
                      d="M6 18L18 6M6 6l12 12"
                    />
                  </svg>
                </button>
              </div>
              <p class="knowledge-content">
                {{ item.content }}
              </p>
              <span class="knowledge-date">{{
                formatDate(item.created_at)
              }}</span>
            </div>
          </div>
        </div>
      </aside>
    </div>
  </div>
</template>

<style scoped>
.app-container {
  min-height: 100vh;
  position: relative;
  overflow: hidden;
}

.main-layout {
  display: grid;
  grid-template-columns: 320px 1fr 320px;
  gap: 24px;
  padding: 24px;
  min-height: 100vh;
  position: relative;
  z-index: 1;
}

/* Sidebars */
.sidebar {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 48px);
  position: sticky;
  top: 24px;
}

.sidebar-left {
  animation-delay: 0.1s;
}

.sidebar-right {
  animation-delay: 0.2s;
}

.sidebar-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--color-glass);
  backdrop-filter: blur(20px);
  border: 1px solid var(--color-border);
  border-radius: 12px 12px 0 0;
}

.sidebar-title {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.badge {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  padding: 2px 10px;
  border-radius: 10px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.sidebar-content {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: var(--color-glass);
  backdrop-filter: blur(20px);
  border: 1px solid var(--color-border);
  border-top: none;
  border-radius: 0 0 12px 12px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  text-align: center;
}

/* Todo Items */
.todo-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.todo-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 10px;
  transition: all 0.2s ease;
}

.todo-item:hover {
  border-color: rgba(0, 245, 212, 0.2);
}

.todo-checkbox {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  border: 2px solid var(--color-text-muted);
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  flex-shrink: 0;
  padding: 0;
}

.todo-checkbox:hover {
  border-color: var(--color-accent-cyan);
}

.todo-checkbox.completed {
  background: var(--color-accent-cyan);
  border-color: var(--color-accent-cyan);
}

.todo-checkbox.completed svg {
  color: var(--color-bg-deep);
}

.todo-content {
  flex: 1;
  min-width: 0;
}

.todo-title {
  display: block;
  font-size: 14px;
  color: var(--color-text-primary);
  line-height: 1.4;
  word-break: break-word;
}

.todo-title.completed {
  text-decoration: line-through;
  color: var(--color-text-muted);
}

.todo-status {
  display: inline-block;
  margin-top: 6px;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  background: var(--color-bg-elevated);
  color: var(--color-text-secondary);
}

.todo-status.completed {
  background: rgba(0, 245, 212, 0.15);
  color: var(--color-accent-cyan);
}

.todo-delete {
  background: transparent;
  border: none;
  padding: 4px;
  cursor: pointer;
  color: var(--color-text-muted);
  opacity: 0;
  transition: all 0.2s ease;
  border-radius: 4px;
}

.todo-item:hover .todo-delete {
  opacity: 1;
}

.todo-delete:hover {
  color: var(--color-accent-pink);
  background: rgba(255, 0, 110, 0.1);
}

/* Knowledge Items */
.knowledge-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.knowledge-item {
  padding: 16px;
  border-radius: 10px;
  transition: all 0.2s ease;
}

.knowledge-item:hover {
  border-color: rgba(157, 78, 221, 0.3);
}

.knowledge-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
}

.knowledge-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  line-height: 1.3;
}

.knowledge-delete {
  background: transparent;
  border: none;
  padding: 2px;
  cursor: pointer;
  color: var(--color-text-muted);
  opacity: 0;
  transition: all 0.2s ease;
  border-radius: 4px;
  flex-shrink: 0;
}

.knowledge-item:hover .knowledge-delete {
  opacity: 1;
}

.knowledge-delete:hover {
  color: var(--color-accent-pink);
  background: rgba(255, 0, 110, 0.1);
}

.knowledge-content {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.5;
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.knowledge-date {
  display: block;
  margin-top: 10px;
  font-size: 11px;
  color: var(--color-text-muted);
}

/* Center Area */
.center-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 20px;
}

.center-header {
  text-align: center;
  margin-bottom: 40px;
  animation: fade-in-up 0.6s ease-out forwards;
}

.meeting-title {
  font-family: var(--font-display);
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0 0 12px 0;
  letter-spacing: -0.5px;
}

.status-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.status-text {
  font-size: 14px;
  color: var(--color-text-secondary);
}

/* Voice Section */
.voice-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 40px;
}

.mic-wrapper {
  position: relative;
  width: 160px;
  height: 160px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.audio-ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 2px solid var(--color-accent-cyan);
  opacity: 0.3;
  transition: transform 0.1s ease-out;
  pointer-events: none;
}

.audio-ring-2 {
  inset: -20px;
  opacity: 0.15;
}

.mic-button {
  position: relative;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  border: none;
  background: linear-gradient(
    135deg,
    var(--color-bg-elevated) 0%,
    var(--color-bg-surface) 100%
  );
  box-shadow: 0 0 40px rgba(0, 245, 212, 0.2),
    inset 0 1px 0 rgba(255, 255, 255, 0.1);
  cursor: pointer;
  transition: all 0.3s ease;
  z-index: 1;
}

.mic-button.recording {
  background: linear-gradient(
    135deg,
    var(--color-accent-pink) 0%,
    var(--color-accent-purple) 100%
  );
  box-shadow: 0 0 50px rgba(157, 78, 221, 0.4),
    0 0 100px rgba(255, 0, 110, 0.2);
}

.mic-inner {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.mic-icon {
  width: 40px;
  height: 40px;
  color: var(--color-accent-cyan);
  transition: all 0.3s ease;
}

.mic-button.recording .mic-icon {
  color: white;
}

.mic-hint {
  margin-top: 20px;
  font-size: 14px;
  color: var(--color-text-muted);
}

/* Transcript Section */
.transcript-section {
  width: 100%;
  max-width: 600px;
  min-height: 200px;
  padding: 20px;
  animation: fade-in-up 0.6s ease-out 0.2s forwards;
  opacity: 0;
}

.transcript-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.transcript-label {
  font-family: var(--font-display);
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 1px;
}

.clear-btn {
  background: transparent;
  border: 1px solid var(--color-border);
  padding: 4px 12px;
  border-radius: 6px;
  font-size: 12px;
  color: var(--color-text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.clear-btn:hover {
  border-color: var(--color-accent-cyan);
  color: var(--color-accent-cyan);
}

.transcript-content {
  min-height: 120px;
  max-height: 300px;
  overflow-y: auto;
}

.transcript-placeholder {
  color: var(--color-text-muted);
  font-size: 15px;
  line-height: 1.6;
  font-style: italic;
}

.transcript-text {
  color: var(--color-text-primary);
  font-size: 15px;
  line-height: 1.8;
  margin: 0;
  word-break: break-word;
}

/* End Meeting Button */
.end-meeting-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 30px;
  padding: 12px 24px;
  background: transparent;
  border: 1px solid var(--color-accent-pink);
  border-radius: 8px;
  color: var(--color-accent-pink);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  animation: fade-in-up 0.4s ease-out forwards;
}

.end-meeting-btn:hover {
  background: var(--color-accent-pink);
  color: white;
  box-shadow: 0 0 20px rgba(255, 0, 110, 0.3);
}

/* Responsive */
@media (max-width: 1200px) {
  .main-layout {
    grid-template-columns: 280px 1fr 280px;
    gap: 16px;
    padding: 16px;
  }
}

@media (max-width: 1024px) {
  .main-layout {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto auto;
  }

  .sidebar {
    height: auto;
    max-height: 300px;
    position: static;
  }

  .sidebar-left {
    order: 2;
  }

  .center-area {
    order: 1;
  }

  .sidebar-right {
    order: 3;
  }
}

/* Loading Overlay */
.loading-overlay {
  position: fixed;
  inset: 0;
  background: var(--color-bg-deep);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9998;
  animation: fade-in 0.3s ease-out;
}

@keyframes fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

.loading-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.loading-spinner {
  width: 48px;
  height: 48px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-accent-cyan);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-text {
  color: var(--color-text-secondary);
  font-size: 14px;
  margin: 0;
}

.loading-bar-container {
  width: 200px;
  height: 4px;
  background: var(--color-border);
  border-radius: 2px;
  overflow: hidden;
}

.loading-bar {
  height: 100%;
  background: linear-gradient(90deg, var(--color-accent-cyan), var(--color-accent-purple));
  border-radius: 2px;
  transition: width 0.3s ease-out;
}

@media (max-width: 640px) {
  .main-layout {
    padding: 12px;
    gap: 12px;
  }

  .center-area {
    padding: 20px 12px;
  }

  .meeting-title {
    font-size: 22px;
  }

  .mic-button {
    width: 100px;
    height: 100px;
  }

  .mic-icon {
    width: 32px;
    height: 32px;
  }

  .transcript-section {
    min-height: 160px;
  }
}
</style>
