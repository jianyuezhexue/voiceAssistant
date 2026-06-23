// useVoiceDialogue.ts - 语音对话核心服务
// Task-F-04: 语音状态机 - 核心服务编排

import { ref, readonly, onUnmounted, watch } from 'vue';
import { useVoiceStore } from '../stores/voice';
import { useAudioCapture } from './useAudioCapture';
import { useWakeWord } from './useWakeWord';
import { useVAD } from './useVAD';
import { VoiceState, MessageType, ASRResult, LLMResponse, TTSAudio } from '../types';

/**
 * 语音对话配置
 */
export interface VoiceDialogueOptions {
  /** WebSocket URL */
  wsUrl?: string;
  /** 唤醒词 */
  wakeWord?: string;
  /** 是否启用唤醒词检测 */
  enableWakeWord?: boolean;
  /** 静音超时时间 */
  silenceTimeout?: number;
  /** 最大语音时长 */
  maxSpeechDuration?: number;
  /** 唤醒成功回调 */
  onWakeSuccess?: () => void;
  /** 连接成功回调 */
  onConnectSuccess?: () => void;
  /** 连接关闭回调 */
  onDisconnect?: () => void;
  /** 错误回调 */
  onError?: (error: Error) => void;
}

/**
 * 语音对话核心服务
 * 编排 AudioCapture、WakeWord、VAD 和状态管理
 */
export function useVoiceDialogue(options: VoiceDialogueOptions = {}) {
  const {
    wsUrl = 'ws://localhost:8080/ws/voice',
    wakeWord = '小爱同学',
    enableWakeWord = true,
    silenceTimeout = 3000,
    maxSpeechDuration = 60000,
    onWakeSuccess,
    onConnectSuccess,
    onDisconnect,
    onError,
  } = options;

  // Store
  const voiceStore = useVoiceStore();

  // 组件状态
  const isInitialized = ref(false);
  const isConnecting = ref(false);
  const wakeWordDetected = ref(false);

  // WebSocket 连接
  const ws = ref<WebSocket | null>(null);
  const wsReconnectTimer = ref<number | null>(null);

  // 事件回调
  const stateChangeCallbacks = new Set<(state: VoiceState) => void>();
  const asrResultCallbacks = new Set<(result: ASRResult) => void>();
  const llmResponseCallbacks = new Set<(response: LLMResponse) => void>();
  const ttsAudioCallbacks = new Set<(audio: TTSAudio) => void>();
  const errorCallbacks = new Set<(error: Error) => void>();

  // Audio Capture
  const audioCapture = useAudioCapture({
    sampleRate: 16000,
    noiseSuppression: true,
    echoCancellation: true,
    autoGainControl: true,
  });

  // Wake Word Detector
  const wakeWordDetector = useWakeWord({
    keyword: wakeWord,
    threshold: 0.75,
    sensitivity: 2,
    enabled: enableWakeWord,
    onWakeWordDetected: handleWakeWordDetected,
  });

  // VAD
  const vad = useVAD({
    sensitivity: 2,
    silenceTimeout,
    maxSpeechDuration,
    onSpeechStart: handleSpeechStart,
    onSpeechEnd: handleSpeechEnd,
    onSpeechInProgress: handleSpeechInProgress,
  });

  // ==================== 生命周期 ====================

  /**
   * 初始化语音对话服务
   */
  async function initialize(): Promise<void> {
    if (isInitialized.value) {
      console.warn('[VoiceDialogue] Already initialized');
      return;
    }

    try {
      console.log('[VoiceDialogue] Initializing...');

      // 初始化各个组件
      if (enableWakeWord) {
        await wakeWordDetector.initialize();
        wakeWordDetector.startListening();
      }

      // 监听状态变化
      watch(() => voiceStore.state, (newState, oldState) => {
        if (newState !== oldState) {
          notifyStateChange(newState);
        }
      });

      isInitialized.value = true;
      console.log('[VoiceDialogue] Initialized successfully');
    } catch (error) {
      console.error('[VoiceDialogue] Failed to initialize:', error);
      handleError(error as Error);
      throw error;
    }
  }

  /**
   * 销毁语音对话服务
   */
  function destroy(): void {
    console.log('[VoiceDialogue] Destroying...');

    // 断开连接
    disconnect();

    // 清理各个组件
    wakeWordDetector.cleanup();
    vad.cleanup();
    audioCapture.cleanup();

    // 清除定时器
    if (wsReconnectTimer.value !== null) {
      clearTimeout(wsReconnectTimer.value);
      wsReconnectTimer.value = null;
    }

    // 重置状态
    voiceStore.reset();
    isInitialized.value = false;

    console.log('[VoiceDialogue] Destroyed');
  }

  // ==================== 连接管理 ====================

  /**
   * 建立 WebSocket 连接
   */
  async function connect(): Promise<void> {
    if (ws.value?.readyState === WebSocket.OPEN) {
      console.warn('[VoiceDialogue] Already connected');
      return;
    }

    if (isConnecting.value) {
      console.warn('[VoiceDialogue] Connection in progress');
      return;
    }

    isConnecting.value = true;

    return new Promise((resolve, reject) => {
      try {
        console.log('[VoiceDialogue] Connecting to', wsUrl);

        ws.value = new WebSocket(wsUrl);
        ws.value.binaryType = 'arraybuffer';

        ws.value.onopen = () => {
          console.log('[VoiceDialogue] WebSocket connected');
          isConnecting.value = false;
          voiceStore.setConnected(true);

          // 显示连接成功 Alert
          showAlert('连接成功，开启对话', 'success');

          if (onConnectSuccess) {
            onConnectSuccess();
          }

          resolve();
        };

        ws.value.onmessage = (event) => {
          handleWSMessage(event);
        };

        ws.value.onerror = (error) => {
          console.error('[VoiceDialogue] WebSocket error:', error);
          handleError(new Error('WebSocket connection error'));
        };

        ws.value.onclose = () => {
          console.log('[VoiceDialogue] WebSocket closed');
          isConnecting.value = false;
          voiceStore.setConnected(false);

          if (onDisconnect) {
            onDisconnect();
          }

          // 显示连接关闭 Alert
          showAlert('对话已结束，连接已关闭', 'info');
        };
      } catch (error) {
        isConnecting.value = false;
        reject(error);
      }
    });
  }

  /**
   * 断开 WebSocket 连接
   */
  function disconnect(): void {
    if (ws.value) {
      ws.value.close();
      ws.value = null;
    }

    if (wsReconnectTimer.value !== null) {
      clearTimeout(wsReconnectTimer.value);
      wsReconnectTimer.value = null;
    }

    voiceStore.setConnected(false);
  }

  /**
   * 处理 WebSocket 消息
   */
  function handleWSMessage(event: MessageEvent): void {
    try {
      const message = JSON.parse(event.data);

      switch (message.type) {
        case MessageType.ASR_RESULT:
          handleASRResult(message.data as ASRResult);
          break;

        case MessageType.ASR_COMPLETE:
          handleASRComplete(message.data as ASRResult);
          break;

        case MessageType.LLM_TEXT:
          handleLLMText(message.data as LLMResponse);
          break;

        case MessageType.LLM_COMPLETE:
          handleLLMComplete(message.data as LLMResponse);
          break;

        case MessageType.TTS_AUDIO:
          handleTTSAudio(message.data as TTSAudio);
          break;

        case MessageType.TTS_COMPLETE:
          handleTTSComplete();
          break;

        case MessageType.STATE_UPDATE:
          handleStateUpdate(message.data);
          break;

        case MessageType.ERROR:
          handleServerError(message.data);
          break;

        case MessageType.PONG:
          // 心跳响应，忽略
          break;

        default:
          console.warn('[VoiceDialogue] Unknown message type:', message.type);
      }
    } catch (error) {
      console.error('[VoiceDialogue] Failed to parse message:', error);
    }
  }

  // ==================== 语音流程处理 ====================

  /**
   * 唤醒词检测成功
   */
  async function handleWakeWordDetected(): Promise<void> {
    console.log('[VoiceDialogue] Wake word detected');

    wakeWordDetected.value = true;

    // 停止唤醒词监听
    wakeWordDetector.stopListening();

    // 显示唤醒成功
    showAlert('唤醒成功', 'success');

    // 开始对话
    voiceStore.startDialogue();

    // 建立连接
    try {
      await connect();

      if (onWakeSuccess) {
        onWakeSuccess();
      }
    } catch (error) {
      handleError(error as Error);
    }
  }

  /**
   * 语音开始
   */
  function handleSpeechStart(): void {
    console.log('[VoiceDialogue] Speech started');

    if (voiceStore.state === VoiceState.LISTENING) {
      voiceStore.setState(VoiceState.RECOGNIZING, '检测到语音开始');
    }
  }

  /**
   * 语音进行中（用于实时字幕）
   */
  function handleSpeechInProgress(volume: number): void {
    // 可以发送实时音量数据到后端
    // 或者更新 UI
  }

  /**
   * 语音结束
   */
  async function handleSpeechEnd(duration: number): Promise<void> {
    console.log('[VoiceDialogue] Speech ended, duration:', duration);

    if (voiceStore.state === VoiceState.RECOGNIZING) {
      voiceStore.setState(VoiceState.THINKING, '语音结束，等待AI回复');
    }
  }

  // ==================== 消息处理 ====================

  /**
   * 处理 ASR 识别结果
   */
  function handleASRResult(result: ASRResult): void {
    console.log('[VoiceDialogue] ASR result:', result);

    voiceStore.updateRecognizedText(result.text);

    asrResultCallbacks.forEach((callback) => {
      callback(result);
    });

    // 如果是最终结果，更新状态
    if (result.isFinal) {
      voiceStore.addMessage('user', result.text);
    }
  }

  /**
   * 处理 ASR 识别完成
   */
  function handleASRComplete(result: ASRResult): void {
    console.log('[VoiceDialogue] ASR complete:', result);

    voiceStore.updateRecognizedText(result.text);
    voiceStore.addMessage('user', result.text);

    asrResultCallbacks.forEach((callback) => {
      callback(result);
    });
  }

  /**
   * 处理 LLM 文本片段
   */
  function handleLLMText(response: LLMResponse): void {
    console.log('[VoiceDialogue] LLM text chunk:', response);

    // 更新回复文本（流式）
    voiceStore.updateResponseText(response.text, true);

    // 如果是第一个字符，切换到 RESPONDING 状态
    if (voiceStore.state === VoiceState.THINKING) {
      voiceStore.setState(VoiceState.RESPONDING, '收到AI首字符');
    }

    llmResponseCallbacks.forEach((callback) => {
      callback(response);
    });
  }

  /**
   * 处理 LLM 完成
   */
  function handleLLMComplete(response: LLMResponse): void {
    console.log('[VoiceDialogue] LLM complete:', response);

    voiceStore.updateResponseText(response.fullText || response.text, false);
    voiceStore.addMessage('assistant', response.fullText || response.text);

    llmResponseCallbacks.forEach((callback) => {
      callback({ ...response, isComplete: true });
    });
  }

  /**
   * 处理 TTS 音频
   */
  function handleTTSAudio(audio: TTSAudio): void {
    // 如果是首音频，切换到 PLAYING 状态
    if (voiceStore.state === VoiceState.RESPONDING) {
      voiceStore.setState(VoiceState.PLAYING, 'TTS音频开始');
    }

    ttsAudioCallbacks.forEach((callback) => {
      callback(audio);
    });
  }

  /**
   * 处理 TTS 完成
   */
  function handleTTSComplete(): void {
    console.log('[VoiceDialogue] TTS complete');

    if (voiceStore.state === VoiceState.PLAYING) {
      voiceStore.setState(VoiceState.LISTENING, 'TTS播放完成');
    }
  }

  /**
   * 处理状态更新（从服务器）
   */
  function handleStateUpdate(data: unknown): void {
    console.log('[VoiceDialogue] State update:', data);
    // 可以处理服务器发送的状态同步
  }

  /**
   * 处理服务器错误
   */
  function handleServerError(data: unknown): void {
    console.error('[VoiceDialogue] Server error:', data);
    handleError(new Error((data as { message?: string })?.message || 'Server error'));
  }

  // ==================== 控制方法 ====================

  /**
   * 开始录音
   */
  async function startRecording(): Promise<void> {
    if (!isInitialized.value) {
      await initialize();
    }

    try {
      await vad.initialize();
      vad.start();
    } catch (error) {
      handleError(error as Error);
    }
  }

  /**
   * 停止录音
   */
  function stopRecording(): void {
    vad.stop();
  }

  /**
   * 打断当前流程
   */
  function interrupt(): void {
    console.log('[VoiceDialogue] Interrupting...');

    // 停止录音
    vad.forceEndSpeech();

    // 停止 TTS 播放
    // TODO: 发送打断命令到服务器

    // 重置状态
    voiceStore.interrupt();

    // 重新开始监听
    vad.start();
  }

  /**
   * 结束对话
   */
  function endDialogue(): void {
    console.log('[VoiceDialogue] Ending dialogue...');

    // 停止所有活动
    vad.stop();
    disconnect();

    // 结束对话
    voiceStore.endDialogue();

    // 重新开始唤醒词监听
    if (enableWakeWord) {
      wakeWordDetector.startListening();
    }
  }

  /**
   * 发送文字消息
   */
  function sendText(text: string): void {
    if (ws.value?.readyState !== WebSocket.OPEN) {
      console.warn('[VoiceDialogue] WebSocket not connected');
      return;
    }

    const message = {
      type: MessageType.ASR_COMPLETE,
      sessionId: voiceStore.session?.id,
      data: { text },
      timestamp: Date.now(),
    };

    ws.value.send(JSON.stringify(message));
  }

  // ==================== 事件订阅 ====================

  /**
   * 订阅状态变化
   */
  function onStateChange(callback: (state: VoiceState) => void): () => void {
    stateChangeCallbacks.add(callback);
    return () => {
      stateChangeCallbacks.delete(callback);
    };
  }

  /**
   * 订阅 ASR 结果
   */
  function onASRResult(callback: (result: ASRResult) => void): () => void {
    asrResultCallbacks.add(callback);
    return () => {
      asrResultCallbacks.delete(callback);
    };
  }

  /**
   * 订阅 LLM 响应
   */
  function onLLMResponse(callback: (response: LLMResponse) => void): () => void {
    llmResponseCallbacks.add(callback);
    return () => {
      llmResponseCallbacks.delete(callback);
    };
  }

  /**
   * 订阅 TTS 音频
   */
  function onTTSAudio(callback: (audio: TTSAudio) => void): () => void {
    ttsAudioCallbacks.add(callback);
    return () => {
      ttsAudioCallbacks.delete(callback);
    };
  }

  /**
   * 订阅错误
   */
  function onVoiceError(callback: (error: Error) => void): () => void {
    errorCallbacks.add(callback);
    return () => {
      errorCallbacks.delete(callback);
    };
  }

  /**
   * 通知状态变化
   */
  function notifyStateChange(state: VoiceState): void {
    stateChangeCallbacks.forEach((callback) => {
      callback(state);
    });
  }

  /**
   * 处理错误
   */
  function handleError(error: Error): void {
    console.error('[VoiceDialogue] Error:', error);
    voiceStore.setError(error.message);

    errorCallbacks.forEach((callback) => {
      callback(error);
    });

    if (onError) {
      onError(error);
    }
  }

  // ==================== 工具方法 ====================

  /**
   * 显示 Alert 弹窗
   */
  function showAlert(message: string, type: 'success' | 'info' | 'warning' | 'error' = 'info'): void {
    // 使用浏览器原生 Alert
    // 实际项目中可以替换为自定义的 toast 组件
    console.log(`[Alert][${type}] ${message}`);

    // 3秒后自动关闭（如果是自定义 toast）
    if (type === 'success' || type === 'info') {
      setTimeout(() => {
        console.log('[Alert] Auto closed');
      }, 3000);
    }
  }

  /**
   * 获取当前状态
   */
  function getState(): VoiceState {
    return voiceStore.state;
  }

  /**
   * 是否已连接
   */
  function isConnected(): boolean {
    return voiceStore.isConnected;
  }

  // ==================== 清理 ====================

  onUnmounted(() => {
    destroy();
  });

  return {
    // 状态
    isInitialized: readonly(isInitialized),
    isConnecting: readonly(isConnecting),
    wakeWordDetected: readonly(wakeWordDetected),

    // Store 状态
    state: readonly(voiceStore.state),
    isConnected: readonly(voiceStore.isConnected),
    isActive: voiceStore.isActive,
    isRecording: voiceStore.isRecording,
    isPlaying: voiceStore.isPlaying,
    stateDisplayText: voiceStore.stateDisplayText,
    recognizedText: readonly(voiceStore.recognizedText),
    responseText: readonly(voiceStore.responseText),
    messages: readonly(voiceStore.messages),

    // 方法
    initialize,
    destroy,
    connect,
    disconnect,
    startRecording,
    stopRecording,
    interrupt,
    endDialogue,
    sendText,
    getState,
    isConnected,

    // 事件订阅
    onStateChange,
    onASRResult,
    onLLMResponse,
    onTTSAudio,
    onVoiceError,

    // 组件引用（用于高级用法）
    audioCapture,
    wakeWordDetector,
    vad,
  };
}

export default useVoiceDialogue;
