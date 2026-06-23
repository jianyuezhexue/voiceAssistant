// voice.ts - 语音状态管理 (Pinia Store)
// Task-F-04: 语音状态机 - 状态管理

import { defineStore } from 'pinia';
import { ref, computed, readonly } from 'vue';
import { VoiceState } from '../types';
import type { VoiceSession, StateTransitionEvent } from '../types';

/**
 * 语音状态 Store
 * 管理语音对话的状态、历史和会话信息
 */
export const useVoiceStore = defineStore('voice', () => {
  // ==================== 状态 ====================

  /** 当前状态 */
  const state = ref<VoiceState>(VoiceState.IDLE);

  /** 语音会话 */
  const session = ref<VoiceSession | null>(null);

  /** 状态转换历史 */
  const stateHistory = ref<StateTransitionEvent[]>([]);

  /** 是否已连接 */
  const isConnected = ref(false);

  /** 最后一次错误 */
  const lastError = ref<string | null>(null);

  /** 对话消息列表 */
  const messages = ref<Array<{
    id: string;
    role: 'user' | 'assistant';
    content: string;
    timestamp: number;
  }>>([]);

  /** 识别文本（实时字幕） */
  const recognizedText = ref<string>('');

  /** AI 回复文本（流式） */
  const responseText = ref<string>('');

  // ==================== 计算属性 ====================

  /** 是否处于活动状态 */
  const isActive = computed(() => {
    return state.value !== VoiceState.IDLE && state.value !== VoiceState.ERROR;
  });

  /** 是否正在录音 */
  const isRecording = computed(() => {
    return state.value === VoiceState.LISTENING || state.value === VoiceState.RECOGNIZING;
  });

  /** 是否正在播放 */
  const isPlaying = computed(() => {
    return state.value === VoiceState.PLAYING || state.value === VoiceState.RESPONDING;
  });

  /** 获取状态显示文本 */
  const stateDisplayText = computed(() => {
    switch (state.value) {
      case VoiceState.IDLE:
        return '等待唤醒';
      case VoiceState.LISTENING:
        return '倾听中';
      case VoiceState.RECOGNIZING:
        return '识别中';
      case VoiceState.THINKING:
        return '思考中';
      case VoiceState.RESPONDING:
        return '回复中';
      case VoiceState.PLAYING:
        return '播放中';
      case VoiceState.ERROR:
        return '发生错误';
      default:
        return '未知状态';
    }
  });

  // ==================== 状态转换规则 ====================

  /**
   * 允许的状态转换映射
   */
  const allowedTransitions: Record<VoiceState, VoiceState[]> = {
    [VoiceState.IDLE]: [VoiceState.LISTENING],
    [VoiceState.LISTENING]: [VoiceState.RECOGNIZING, VoiceState.IDLE],
    [VoiceState.RECOGNIZING]: [VoiceState.THINKING, VoiceState.LISTENING],
    [VoiceState.THINKING]: [VoiceState.RESPONDING, VoiceState.LISTENING],
    [VoiceState.RESPONDING]: [VoiceState.PLAYING, VoiceState.LISTENING],
    [VoiceState.PLAYING]: [VoiceState.LISTENING, VoiceState.IDLE],
    [VoiceState.ERROR]: [VoiceState.IDLE, VoiceState.LISTENING],
  };

  /**
   * 检查是否允许转换到目标状态
   */
  function canTransitionTo(targetState: VoiceState): boolean {
    const allowed = allowedTransitions[state.value];
    return allowed ? allowed.includes(targetState) : false;
  }

  // ==================== 方法 ====================

  /**
   * 设置状态
   */
  function setState(newState: VoiceState, reason?: string): boolean {
    if (state.value === newState) {
      return false;
    }

    // 检查是否允许转换
    if (!canTransitionTo(newState)) {
      console.warn(`[VoiceStore] Invalid transition: ${state.value} -> ${newState}`);
      // 允许从任意状态转到 LISTENING（打断场景）
      if (newState !== VoiceState.LISTENING) {
        return false;
      }
    }

    // 记录转换历史
    const event: StateTransitionEvent = {
      from: state.value,
      to: newState,
      timestamp: Date.now(),
      reason,
    };
    stateHistory.value.push(event);

    // 保持历史在合理大小
    if (stateHistory.value.length > 100) {
      stateHistory.value = stateHistory.value.slice(-50);
    }

    // 更新状态
    state.value = newState;

    // 更新会话的最后活跃时间
    if (session.value) {
      session.value.lastActiveAt = Date.now();
      session.value.state = newState;
    }

    console.log(`[VoiceStore] State: ${event.from} -> ${event.to}`, reason ? `(${reason})` : '');

    return true;
  }

  /**
   * 开始对话
   */
  function startDialogue(): void {
    // 创建新会话
    session.value = {
      id: generateSessionId(),
      userId: 'local-user',
      state: VoiceState.LISTENING,
      recognizedText: '',
      responseText: '',
      createdAt: Date.now(),
      lastActiveAt: Date.now(),
      isInterrupted: false,
    };

    // 重置消息
    messages.value = [];
    recognizedText.value = '';
    responseText.value = '';
    lastError.value = null;

    // 设置状态
    setState(VoiceState.LISTENING, '用户开始对话');

    console.log('[VoiceStore] Dialogue started, session:', session.value.id);
  }

  /**
   * 结束对话
   */
  function endDialogue(): void {
    setState(VoiceState.IDLE, '用户结束对话');

    if (session.value) {
      session.value.isInterrupted = true;
    }

    console.log('[VoiceStore] Dialogue ended');
  }

  /**
   * 更新识别文本
   */
  function updateRecognizedText(text: string): void {
    recognizedText.value = text;

    if (session.value) {
      session.value.recognizedText = text;
    }
  }

  /**
   * 更新回复文本
   */
  function updateResponseText(text: string, isChunk: boolean = false): void {
    if (isChunk) {
      responseText.value += text;
    } else {
      responseText.value = text;
    }

    if (session.value) {
      session.value.responseText = responseText.value;
    }
  }

  /**
   * 添加消息
   */
  function addMessage(role: 'user' | 'assistant', content: string): void {
    messages.value.push({
      id: generateMessageId(),
      role,
      content,
      timestamp: Date.now(),
    });

    // 如果是用户消息，更新识别文本
    if (role === 'user') {
      recognizedText.value = content;
    }

    // 如果是AI消息，更新回复文本
    if (role === 'assistant') {
      responseText.value = content;
    }
  }

  /**
   * 设置连接状态
   */
  function setConnected(connected: boolean): void {
    isConnected.value = connected;
    console.log('[VoiceStore] Connection status:', connected);
  }

  /**
   * 设置错误
   */
  function setError(error: string): void {
    lastError.value = error;
    setState(VoiceState.ERROR, error);
    console.error('[VoiceStore] Error:', error);
  }

  /**
   * 清除错误
   */
  function clearError(): void {
    if (state.value === VoiceState.ERROR) {
      setState(VoiceState.IDLE, '错误已清除');
    }
    lastError.value = null;
  }

  /**
   * 打断当前流程
   */
  function interrupt(): void {
    if (session.value) {
      session.value.isInterrupted = true;
    }

    setState(VoiceState.LISTENING, '被打断');

    console.log('[VoiceStore] Interrupted');
  }

  /**
   * 获取状态历史
   */
  function getStateHistory(): StateTransitionEvent[] {
    return [...stateHistory.value];
  }

  /**
   * 重置状态
   */
  function reset(): void {
    state.value = VoiceState.IDLE;
    session.value = null;
    stateHistory.value = [];
    messages.value = [];
    recognizedText.value = '';
    responseText.value = '';
    lastError.value = null;
    isConnected.value = false;

    console.log('[VoiceStore] Reset');
  }

  // ==================== 工具函数 ====================

  /**
   * 生成会话ID
   */
  function generateSessionId(): string {
    return `session_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * 生成消息ID
   */
  function generateMessageId(): string {
    return `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  return {
    // 状态（只读）
    state: readonly(state),
    session: readonly(session),
    stateHistory: readonly(stateHistory),
    isConnected: readonly(isConnected),
    lastError: readonly(lastError),
    messages: readonly(messages),
    recognizedText: readonly(recognizedText),
    responseText: readonly(responseText),

    // 计算属性
    isActive,
    isRecording,
    isPlaying,
    stateDisplayText,

    // 方法
    setState,
    canTransitionTo,
    startDialogue,
    endDialogue,
    updateRecognizedText,
    updateResponseText,
    addMessage,
    setConnected,
    setError,
    clearError,
    interrupt,
    getStateHistory,
    reset,
  };
});

export default useVoiceStore;
