// useVAD.ts - VAD (Voice Activity Detection) 语音活动检测
// Task-F-03: VAD语音活动检测

import { ref, readonly, onUnmounted } from 'vue';
import { useAudioCapture } from './useAudioCapture';

/**
 * VAD 配置
 */
export interface VADOptions {
  /** 语音检测灵敏度 (0-3)，0最不敏感，3最敏感 */
  sensitivity?: number;
  /** 语音开始阈值 (0-1) */
  speechStartThreshold?: number;
  /** 语音结束阈值 (0-1) */
  speechEndThreshold?: number;
  /** 静音超时时间(ms)，超过此时间未检测到语音则判定为语音结束 */
  silenceTimeout?: number;
  /** 最大语音时长(ms)，超过此时间强制判定为语音结束 */
  maxSpeechDuration?: number;
  /** 启用状态 */
  enabled?: boolean;
  /** 语音开始回调 */
  onSpeechStart?: () => void;
  /** 语音结束回调 */
  onSpeechEnd?: (duration: number) => void;
  /** 语音中回调（用于实时处理） */
  onSpeechInProgress?: (volume: number) => void;
}

/**
 * VAD 状态
 */
export enum VADState {
  IDLE = 'idle',
  LISTENING = 'listening',
  SPEECH_START = 'speech_start',
  SPEAKING = 'speaking',
  SPEECH_END = 'speech_end',
}

/**
 * VAD 统计信息
 */
export interface VADStats {
  /** 总检测次数 */
  totalDetections: number;
  /** 语音开始次数 */
  speechStartCount: number;
  /** 语音结束次数 */
  speechEndCount: number;
  /** 平均语音时长 */
  averageSpeechDuration: number;
  /** 最后一次语音时长 */
  lastSpeechDuration: number;
}

/**
 * VAD 语音活动检测服务
 * 使用 WebRTC VAD 风格的能量检测实现
 */
export function useVAD(options: VADOptions = {}) {
  const {
    sensitivity = 2,
    speechStartThreshold = 0.5,
    speechEndThreshold = 0.2,
    silenceTimeout = 3000,
    maxSpeechDuration = 60000,
    enabled = true,
    onSpeechStart,
    onSpeechEnd,
    onSpeechInProgress,
  } = options;

  // VAD 状态
  const vadState = ref<VADState>(VADState.IDLE);
  const currentVolume = ref(0);
  const isSpeechActive = ref(false);
  const speechStartTime = ref<number | null>(null);

  // 音频采集
  const audioCapture = useAudioCapture({
    sampleRate: 16000,
    noiseSuppression: true,
    echoCancellation: true,
    autoGainControl: true,
  });

  // 静音检测定时器
  let silenceTimer: number | null = null;
  let maxDurationTimer: number | null = null;

  // 统计信息
  const stats = ref<VADStats>({
    totalDetections: 0,
    speechStartCount: 0,
    speechEndCount: 0,
    averageSpeechDuration: 0,
    lastSpeechDuration: 0,
  });

  // 累积语音时长（用于计算平均值）
  let totalSpeechDuration = 0;
  let speechCount = 0;

  /**
   * 灵敏度与阈值映射
   * sensitivity: 0-3
   * 0: 阈值 +0.1，最不敏感
   * 1: 标准阈值
   * 2: 阈值 -0.05
   * 3: 阈值 -0.1，最敏感
   */
  function getAdjustedThreshold(baseThreshold: number): number {
    const adjustment = (3 - sensitivity) * 0.05;
    return Math.max(0.1, Math.min(0.9, baseThreshold - adjustment));
  }

  /**
   * 初始化 VAD
   */
  async function initialize(): Promise<void> {
    if (!enabled) {
      console.log('[VAD] Disabled');
      return;
    }

    try {
      await audioCapture.initialize();
      console.log('[VAD] Initialized', {
        sensitivity,
        speechStartThreshold,
        speechEndThreshold,
        silenceTimeout,
        maxSpeechDuration,
      });
    } catch (error) {
      console.error('[VAD] Failed to initialize:', error);
      throw error;
    }
  }

  /**
   * 开始 VAD 检测
   */
  function start(): void {
    if (!enabled || !audioCapture.isInitialized.value) {
      console.warn('[VAD] Not initialized or disabled');
      return;
    }

    if (vadState.value !== VADState.IDLE) {
      console.warn('[VAD] Already started');
      return;
    }

    vadState.value = VADState.LISTENING;
    isSpeechActive.value = false;
    speechStartTime.value = null;

    // 清除之前的定时器
    clearTimers();

    // 开始音频采集
    audioCapture.start();
    audioCapture.onVolumeChange(handleVolumeChange);

    console.log('[VAD] Started listening');
  }

  /**
   * 停止 VAD 检测
   */
  function stop(): void {
    if (vadState.value === VADState.IDLE) {
      return;
    }

    vadState.value = VADState.IDLE;
    isSpeechActive.value = false;

    // 清除定时器
    clearTimers();

    // 停止音频采集
    audioCapture.stop();

    console.log('[VAD] Stopped');
  }

  /**
   * 处理音量变化
   */
  function handleVolumeChange(volume: number): void {
    if (!enabled || vadState.value === VADState.IDLE) {
      return;
    }

    currentVolume.value = volume;

    // 通知实时回调
    if (isSpeechActive.value && onSpeechInProgress) {
      onSpeechInProgress(volume);
    }

    const adjustedStartThreshold = getAdjustedThreshold(speechStartThreshold);
    const adjustedEndThreshold = getAdjustedThreshold(speechEndThreshold);

    switch (vadState.value) {
      case VADState.LISTENING:
        // 监听中，检测是否开始说话
        if (volume >= adjustedStartThreshold) {
          handleSpeechStart();
        }
        break;

      case VADState.SPEAKING:
        // 说话中，检测是否结束
        if (volume < adjustedEndThreshold) {
          // 开始静音计时
          startSilenceTimer();
        } else {
          // 有声音，取消静音计时
          cancelSilenceTimer();
        }

        // 检查最大语音时长
        if (speechStartTime.value) {
          const elapsed = Date.now() - speechStartTime.value;
          if (elapsed >= maxSpeechDuration) {
            console.log('[VAD] Max speech duration reached');
            handleSpeechEnd();
          }
        }
        break;
    }
  }

  /**
   * 处理语音开始
   */
  function handleSpeechStart(): void {
    if (isSpeechActive.value) {
      return;
    }

    // 清除静音定时器
    cancelSilenceTimer();

    vadState.value = VADState.SPEECH_START;
    isSpeechActive.value = true;
    speechStartTime.value = Date.now();

    // 更新统计
    stats.value.speechStartCount++;
    stats.value.totalDetections++;

    console.log('[VAD] Speech started at', new Date().toISOString());

    // 触发回调
    if (onSpeechStart) {
      onSpeechStart();
    }

    // 短暂延迟后进入SPEAKING状态
    setTimeout(() => {
      if (vadState.value === VADState.SPEECH_START) {
        vadState.value = VADState.SPEAKING;
      }
    }, 100);
  }

  /**
   * 处理语音结束
   */
  function handleSpeechEnd(): void {
    if (!isSpeechActive.value) {
      return;
    }

    vadState.value = VADState.SPEECH_END;
    isSpeechActive.value = false;

    // 计算语音时长
    const duration = speechStartTime.value ? Date.now() - speechStartTime.value : 0;

    // 更新统计
    stats.value.speechEndCount++;
    stats.value.lastSpeechDuration = duration;
    totalSpeechDuration += duration;
    speechCount++;
    stats.value.averageSpeechDuration = totalSpeechDuration / speechCount;

    // 重置状态
    speechStartTime.value = null;

    // 清除定时器
    clearTimers();

    console.log('[VAD] Speech ended, duration:', duration, 'ms');

    // 触发回调
    if (onSpeechEnd) {
      onSpeechEnd(duration);
    }

    // 延迟后回到LISTENING状态
    setTimeout(() => {
      if (vadState.value === VADState.SPEECH_END) {
        vadState.value = VADState.LISTENING;
      }
    }, 100);
  }

  /**
   * 开始静音定时器
   */
  function startSilenceTimer(): void {
    if (silenceTimer !== null) {
      return;
    }

    silenceTimer = window.setTimeout(() => {
      console.log('[VAD] Silence timeout reached');
      handleSpeechEnd();
      silenceTimer = null;
    }, silenceTimeout);
  }

  /**
   * 取消静音定时器
   */
  function cancelSilenceTimer(): void {
    if (silenceTimer !== null) {
      clearTimeout(silenceTimer);
      silenceTimer = null;
    }
  }

  /**
   * 清除所有定时器
   */
  function clearTimers(): void {
    cancelSilenceTimer();

    if (maxDurationTimer !== null) {
      clearTimeout(maxDurationTimer);
      maxDurationTimer = null;
    }
  }

  /**
   * 强制触发语音结束
   */
  function forceEndSpeech(): void {
    if (isSpeechActive.value) {
      handleSpeechEnd();
    }
  }

  /**
   * 重置统计
   */
  function resetStats(): void {
    stats.value = {
      totalDetections: 0,
      speechStartCount: 0,
      speechEndCount: 0,
      averageSpeechDuration: 0,
      lastSpeechDuration: 0,
    };
    totalSpeechDuration = 0;
    speechCount = 0;
  }

  /**
   * 清理资源
   */
  function cleanup(): void {
    stop();
    audioCapture.cleanup();
    resetStats();

    console.log('[VAD] Cleaned up');
  }

  // 组件卸载时清理
  onUnmounted(() => {
    cleanup();
  });

  return {
    // 状态
    vadState: readonly(vadState),
    currentVolume: readonly(currentVolume),
    isSpeechActive: readonly(isSpeechActive),
    stats: readonly(stats),

    // 方法
    initialize,
    start,
    stop,
    forceEndSpeech,
    resetStats,
    cleanup,

    // 音量回调
    onVolumeChange: audioCapture.onVolumeChange,
  };
}

/**
 * WebRTC VAD 兼容接口
 * 预留接口用于集成真实的 WebRTC VAD
 */
export function createWebRTCVAD() {
  // 这是一个占位实现
  // 实际可以使用 pion/webrtc 或类似的 WebRTC VAD 实现
  return {
    async initialize(): Promise<void> {
      console.log('[WebRTC VAD] Placeholder initialization');
    },

    async processAudio(audioData: Int16Array): Promise<{ speech: boolean; probability: number }> {
      // 简化实现
      const sum = Math.abs(audioData.reduce((acc, val) => acc + val, 0));
      const average = sum / audioData.length;
      const probability = Math.min(1, average / 3000);

      return {
        speech: probability > 0.5,
        probability,
      };
    },

    setSensitivity(sensitivity: number): void {
      console.log('[WebRTC VAD] Sensitivity set to', sensitivity);
    },
  };
}

export default useVAD;
