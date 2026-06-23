// useAudioCapture.ts - 音频采集服务
// Task-F-01: 音频采集服务

import { ref, onUnmounted, readonly } from 'vue';

/**
 * 音频采集配置选项
 */
export interface AudioCaptureOptions {
  /** 采样率 */
  sampleRate?: number;
  /** 启用降噪 */
  noiseSuppression?: boolean;
  /** 启用回声消除 */
  echoCancellation?: boolean;
  /** 启用自动增益控制 */
  autoGainControl?: boolean;
  /** 音量变化回调 */
  onVolumeChange?: (volume: number) => void;
}

/**
 * 音频采集服务
 * 负责麦克风音频采集、降噪处理、音量检测
 */
export function useAudioCapture(options: AudioCaptureOptions = {}) {
  const {
    sampleRate = 16000,
    noiseSuppression = true,
    echoCancellation = true,
    autoGainControl = true,
    onVolumeChange,
  } = options;

  // 内部状态
  const audioContext = ref<AudioContext | null>(null);
  const mediaStream = ref<MediaStream | null>(null);
  const analyser = ref<AnalyserNode | null>(null);
  const sourceNode = ref<MediaStreamAudioSourceNode | null>(null);
  const processorNode = ref<ScriptProcessorNode | null>(null);
  const isInitialized = ref(false);
  const isRecording = ref(false);
  const currentVolume = ref(0);

  // 音量变化回调集合
  const volumeCallbacks = new Set<(volume: number) => void>();

  // 添加音量回调
  if (onVolumeChange) {
    volumeCallbacks.add(onVolumeChange);
  }

  // 音量检测动画帧ID
  let animationFrameId: number | null = null;

  /**
   * 初始化音频采集
   * 获取麦克风权限并创建音频处理图
   */
  async function initialize(): Promise<void> {
    if (isInitialized.value) {
      console.warn('[AudioCapture] Already initialized');
      return;
    }

    try {
      // 请求麦克风权限
      mediaStream.value = await navigator.mediaDevices.getUserMedia({
        audio: {
          channelCount: 1,
          sampleRate: { ideal: sampleRate },
          noiseSuppression,
          echoCancellation,
          autoGainControl,
          // 音频格式: PCM 16bit
          bitsPerSample: 16,
        } as MediaTrackConstraints,
      });

      // 创建 AudioContext
      audioContext.value = new AudioContext({ sampleRate });

      // 创建分析节点用于音量检测
      analyser.value = audioContext.value.createAnalyser();
      analyser.value.fftSize = 256;
      analyser.value.smoothingTimeConstant = 0.3;

      // 创建媒体流源
      sourceNode.value = audioContext.value.createMediaStreamSource(mediaStream.value);

      // 连接节点: source -> analyser
      sourceNode.value.connect(analyser.value);

      isInitialized.value = true;
      console.log('[AudioCapture] Initialized successfully', {
        sampleRate,
        noiseSuppression,
        echoCancellation,
        autoGainControl,
      });
    } catch (error) {
      console.error('[AudioCapture] Failed to initialize:', error);
      throw error;
    }
  }

  /**
   * 开始音频采集
   */
  function start(): void {
    if (!isInitialized.value || !mediaStream.value) {
      console.warn('[AudioCapture] Not initialized, call initialize() first');
      return;
    }

    if (isRecording.value) {
      console.warn('[AudioCapture] Already recording');
      return;
    }

    isRecording.value = true;
    updateVolumeLoop();

    console.log('[AudioCapture] Started');
  }

  /**
   * 停止音频采集
   */
  function stop(): void {
    if (!isRecording.value) {
      return;
    }

    isRecording.value = false;

    // 取消动画帧
    if (animationFrameId !== null) {
      cancelAnimationFrame(animationFrameId);
      animationFrameId = null;
    }

    console.log('[AudioCapture] Stopped');
  }

  /**
   * 音量检测循环
   */
  function updateVolumeLoop(): void {
    if (!isRecording.value || !analyser.value) {
      return;
    }

    const dataArray = new Uint8Array(analyser.value.frequencyBinCount);
    analyser.value.getByteFrequencyData(dataArray);

    // 计算音量 (0-1)
    const sum = dataArray.reduce((acc, val) => acc + val, 0);
    const average = sum / dataArray.length;
    currentVolume.value = Math.min(1, average / 128);

    // 通知所有回调
    volumeCallbacks.forEach((callback) => {
      callback(currentVolume.value);
    });

    // 继续检测
    animationFrameId = requestAnimationFrame(updateVolumeLoop);
  }

  /**
   * 获取 MediaStream
   */
  function getStream(): MediaStream | null {
    return mediaStream.value;
  }

  /**
   * 获取当前音量 (0-1)
   */
  function getCurrentVolume(): number {
    return currentVolume.value;
  }

  /**
   * 注册音量变化回调
   * @returns 取消订阅函数
   */
  function subscribeVolumeChange(callback: (volume: number) => void): () => void {
    volumeCallbacks.add(callback);
    return () => {
      volumeCallbacks.delete(callback);
    };
  }

  /**
   * 清理资源
   */
  function cleanup(): void {
    stop();

    // 断开节点连接
    if (sourceNode.value) {
      sourceNode.value.disconnect();
      sourceNode.value = null;
    }

    if (analyser.value) {
      analyser.value.disconnect();
      analyser.value = null;
    }

    if (processorNode.value) {
      processorNode.value.disconnect();
      processorNode.value = null;
    }

    // 关闭 AudioContext
    if (audioContext.value) {
      audioContext.value.close();
      audioContext.value = null;
    }

    // 停止媒体流
    if (mediaStream.value) {
      mediaStream.value.getTracks().forEach((track) => track.stop());
      mediaStream.value = null;
    }

    volumeCallbacks.clear();
    isInitialized.value = false;

    console.log('[AudioCapture] Cleaned up');
  }

  // 组件卸载时自动清理
  onUnmounted(() => {
    cleanup();
  });

  return {
    // 只读状态
    isInitialized: readonly(isInitialized),
    isRecording: readonly(isRecording),
    currentVolume: readonly(currentVolume),

    // 方法
    initialize,
    start,
    stop,
    getStream,
    getCurrentVolume,
    subscribeVolumeChange,
    cleanup,
  };
}

/**
 * 创建音频处理节点（用于更高级的音频处理）
 */
export function createAudioProcessor(
  audioContext: AudioContext,
  mediaStream: MediaStream,
  options: {
    bufferSize?: number;
    onAudioData?: (data: Float32Array) => void;
  } = {}
): ScriptProcessorNode {
  const { bufferSize = 4096, onAudioData } = options;

  const processor = audioContext.createScriptProcessor(bufferSize, 1, 1);

  processor.onaudioprocess = (event) => {
    if (!onAudioData) return;

    const inputBuffer = event.inputBuffer;
    const inputData = inputBuffer.getChannelData(0);
    onAudioData(inputData);
  };

  const source = audioContext.createMediaStreamSource(mediaStream);
  source.connect(processor);
  processor.connect(audioContext.destination);

  return processor;
}

export default useAudioCapture;
