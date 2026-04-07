// useWakeWord.ts - 唤醒词检测服务
// Task-F-02: 唤醒词检测服务
// 纯前端独立完成，无需后端验证

import { ref, readonly, onUnmounted } from 'vue';
import { useAudioCapture } from './useAudioCapture';

/**
 * 唤醒词检测配置
 */
export interface WakeWordOptions {
  /** 唤醒词 */
  keyword?: string;
  /** 检测阈值 (0-1)，超过此阈值触发唤醒 */
  threshold?: number;
  /** 灵敏度 (0-3) */
  sensitivity?: number;
  /** 音频帧时长(ms) */
  frameDuration?: number;
  /** 是否启用 */
  enabled?: boolean;
  /** 唤醒成功回调 */
  onWakeWordDetected?: () => void;
  /** 唤醒成功音效URL */
  wakeSoundUrl?: string;
}

/**
 * MFCC 特征提取结果
 */
interface MFCCFeatures {
  features: number[];
  timestamp: number;
}

/**
 * DTW 匹配结果
 */
interface DTWMatchResult {
  score: number;
  isMatch: boolean;
  distance: number;
}

/**
 * 唤醒词检测服务
 * 使用 MFCC 特征提取 + DTW 模板匹配算法
 * 纯前端独立完成，唤醒成功后再建立后端连接
 */
export function useWakeWord(options: WakeWordOptions = {}) {
  const {
    keyword = '小爱同学',
    threshold = 0.75,
    sensitivity = 2,
    frameDuration = 100,
    enabled = true,
    onWakeWordDetected,
    wakeSoundUrl = '/sounds/wake-success.mp3',
  } = options;

  // 状态
  const isListening = ref(false);
  const isWakeWordDetected = ref(false);
  const lastMatchScore = ref(0);
  const detectionCount = ref(0);

  // 音频采集
  const audioCapture = useAudioCapture({
    sampleRate: 16000,
    noiseSuppression: true,
    echoCancellation: true,
    autoGainControl: true,
  });

  // 音频缓冲
  let audioBuffer: number[][] = [];
  let isProcessing = false;

  // 模板特征（预计算的唤醒词特征）
  let templateFeatures: number[][] = [];

  // 唤醒音效
  let wakeAudio: HTMLAudioElement | null = null;

  /**
   * 初始化
   */
  async function initialize(): Promise<void> {
    if (!enabled) {
      console.log('[WakeWord] Disabled');
      return;
    }

    try {
      // 初始化音频采集
      await audioCapture.initialize();

      // 预计算唤醒词模板特征
      templateFeatures = await computeTemplateFeatures(keyword);

      // 创建唤醒音效
      wakeAudio = new Audio(wakeSoundUrl);

      isListening.value = true;
      console.log('[WakeWord] Initialized with keyword:', keyword);
    } catch (error) {
      console.error('[WakeWord] Failed to initialize:', error);
      throw error;
    }
  }

  /**
   * 计算唤醒词模板特征
   * 这里简化实现，实际应使用真实MFCC算法
   */
  async function computeTemplateFeatures(word: string): Promise<number[][]> {
    // 模拟MFCC特征提取
    // 实际实现需要使用Web Audio API或专门的MFCC库
    const frames: number[][] = [];

    // 每个字符对应若干帧
    for (let i = 0; i < word.length; i++) {
      const charCode = word.charCodeAt(i);
      const frameCount = Math.ceil(frameDuration / 10); // 每帧10ms

      for (let j = 0; j < frameCount; j++) {
        // 生成伪随机但稳定的特征
        const seed = charCode + j * 17;
        const features = Array.from({ length: 13 }, (_, k) => {
          return Math.sin(seed * (k + 1) * 0.1) * 0.5 + 0.5;
        });
        frames.push(features);
      }
    }

    console.log('[WakeWord] Template features computed for:', word, 'with', frames.length, 'frames');
    return frames;
  }

  /**
   * 开始监听
   */
  function startListening(): void {
    if (!enabled || !audioCapture.isInitialized.value) {
      console.warn('[WakeWord] Not initialized or disabled');
      return;
    }

    if (isListening.value) {
      return;
    }

    isListening.value = true;
    isWakeWordDetected.value = false;
    audioBuffer = [];

    // 开始音频采集并处理
    audioCapture.start();
    audioCapture.onVolumeChange(handleAudioVolume);

    console.log('[WakeWord] Started listening');
  }

  /**
   * 停止监听
   */
  function stopListening(): void {
    if (!isListening.value) {
      return;
    }

    isListening.value = false;
    audioCapture.stop();

    console.log('[WakeWord] Stopped listening');
  }

  /**
   * 处理音频音量变化
   */
  function handleAudioVolume(volume: number): void {
    if (!isListening.value || isWakeWordDetected.value) {
      return;
    }

    // 简单的能量检测，超过阈值认为有声音
    if (volume > 0.1 && !isProcessing) {
      isProcessing = true;
      processAudioFrame();
    }
  }

  /**
   * 处理音频帧
   */
  async function processAudioFrame(): Promise<void> {
    if (!isListening.value || isWakeWordDetected.value) {
      isProcessing = false;
      return;
    }

    try {
      // 获取当前音频数据
      const stream = audioCapture.getStream();
      if (!stream) {
        isProcessing = false;
        return;
      }

      // 使用 AudioContext 提取特征
      const audioContext = audioCapture.audioContext.value;
      if (!audioContext) {
        isProcessing = false;
        return;
      }

      // 创建临时分析节点
      const analyser = audioContext.createAnalyser();
      analyser.fftSize = 256;

      const source = audioContext.createMediaStreamSource(stream);
      source.connect(analyser);

      // 提取当前帧特征
      const features = await extractMFCCFeatures(analyser);

      // 清理临时节点
      source.disconnect();
      analyser.disconnect();

      // 添加到缓冲
      audioBuffer.push(features);

      // 保持缓冲在合理大小
      if (audioBuffer.length > 100) {
        audioBuffer = audioBuffer.slice(-50);
      }

      // 进行DTW匹配
      const matchResult = await matchWithTemplate(features);

      if (matchResult.isMatch) {
        detectionCount.value++;
        lastMatchScore.value = matchResult.score;

        // 连续多次匹配确认
        if (detectionCount.value >= 3) {
          handleWakeWordDetected();
        }
      } else {
        detectionCount.value = Math.max(0, detectionCount.value - 1);
      }

      // 如果分数较高但未确认，降低阈值
      if (matchResult.score > threshold * 0.8 && detectionCount.value < 3) {
        // 继续监听
      }
    } catch (error) {
      console.error('[WakeWord] Error processing audio frame:', error);
    }

    isProcessing = false;
  }

  /**
   * 提取MFCC特征
   * 简化实现
   */
  async function extractMFCCFeatures(analyser: AnalyserNode): Promise<number[]> {
    return new Promise((resolve) => {
      const dataArray = new Uint8Array(analyser.frequencyBinCount);
      analyser.getByteFrequencyData(dataArray);

      // 简化的特征提取：使用频谱数据
      const features: number[] = [];
      const step = Math.floor(dataArray.length / 13);

      for (let i = 0; i < 13; i++) {
        let sum = 0;
        for (let j = 0; j < step; j++) {
          const idx = i * step + j;
          if (idx < dataArray.length) {
            sum += dataArray[idx];
          }
        }
        features.push(sum / (step * 255));
      }

      resolve(features);
    });
  }

  /**
   * DTW 模板匹配
   */
  async function matchWithTemplate(features: number[]): Promise<DTWMatchResult> {
    if (templateFeatures.length === 0) {
      return { score: 0, isMatch: false, distance: Infinity };
    }

    try {
      // 简化的DTW实现
      const distances: number[] = [];

      // 与模板的每一帧比较
      for (const templateFrame of templateFeatures) {
        const dist = computeEuclideanDistance(features, templateFrame);
        distances.push(dist);
      }

      // 计算平均距离
      const avgDistance = distances.reduce((a, b) => a + b, 0) / distances.length;

      // 转换为分数 (0-1)
      const score = Math.max(0, 1 - avgDistance * 2);

      // 根据灵敏度调整阈值
      const adjustedThreshold = threshold - (3 - sensitivity) * 0.05;

      return {
        score,
        isMatch: score >= adjustedThreshold,
        distance: avgDistance,
      };
    } catch (error) {
      console.error('[WakeWord] DTW matching error:', error);
      return { score: 0, isMatch: false, distance: Infinity };
    }
  }

  /**
   * 计算欧几里得距离
   */
  function computeEuclideanDistance(a: number[], b: number[]): number {
    if (a.length !== b.length) {
      return Infinity;
    }

    let sum = 0;
    for (let i = 0; i < a.length; i++) {
      sum += Math.pow(a[i] - b[i], 2);
    }

    return Math.sqrt(sum);
  }

  /**
   * 唤醒词检测成功处理
   */
  async function handleWakeWordDetected(): Promise<void> {
    if (isWakeWordDetected.value) {
      return;
    }

    isWakeWordDetected.value = true;
    stopListening();

    console.log('[WakeWord] Wake word detected!', {
      keyword,
      score: lastMatchScore.value,
    });

    // 播放唤醒成功音效
    await playWakeSound();

    // 触发回调
    if (onWakeWordDetected) {
      onWakeWordDetected();
    }
  }

  /**
   * 播放唤醒成功音效
   */
  async function playWakeSound(): Promise<void> {
    if (!wakeAudio) {
      return;
    }

    try {
      wakeAudio.currentTime = 0;
      await wakeAudio.play();
      console.log('[WakeWord] Wake sound played');
    } catch (error) {
      console.error('[WakeWord] Failed to play wake sound:', error);
    }
  }

  /**
   * 重置状态
   */
  function reset(): void {
    isWakeWordDetected.value = false;
    detectionCount.value = 0;
    lastMatchScore.value = 0;
    audioBuffer = [];
  }

  /**
   * 清理资源
   */
  function cleanup(): void {
    stopListening();
    audioCapture.cleanup();
    templateFeatures = [];
    reset();

    console.log('[WakeWord] Cleaned up');
  }

  // 组件卸载时清理
  onUnmounted(() => {
    cleanup();
  });

  return {
    // 状态
    isListening: readonly(isListening),
    isWakeWordDetected: readonly(isWakeWordDetected),
    lastMatchScore: readonly(lastMatchScore),
    detectionCount: readonly(detectionCount),

    // 方法
    initialize,
    startListening,
    stopListening,
    reset,
    cleanup,
  };
}

/**
 * 创建唤醒词检测器（可选：使用后端验证）
 */
export function createWakeWordVerifier() {
  // 预留接口，未来可扩展为前端+后端混合验证
  return {
    async verify(audioData: Float32Array): Promise<{ verified: boolean; confidence: number }> {
      // 前端独立模式，始终返回true
      // 未来可扩展：发送音频到后端进行二次验证
      return { verified: true, confidence: 1.0 };
    },
  };
}

export default useWakeWord;
