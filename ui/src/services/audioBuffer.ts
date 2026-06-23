/**
 * 音频包结构
 */
export interface AudioPacket {
  /** 序列号 */
  seq: number;
  /** 时间戳（毫秒） */
  timestamp: number;
  /** 音频数据 */
  data: ArrayBuffer;
  /** 接收时间（用于过期判断） */
  receivedAt: number;
}

/**
 * 缓冲配置
 */
export interface BufferConfig {
  /** 最大缓冲时间（毫秒） */
  maxBufferTime: number;
  /** 最小缓冲包数量 */
  minPackets?: number;
  /** 最大缓冲包数量 */
  maxPackets?: number;
}

/**
 * 默认配置
 */
const DEFAULT_TTS_CONFIG: BufferConfig = {
  maxBufferTime: 50,
  minPackets: 1,
  maxPackets: 10
};

const DEFAULT_ASR_CONFIG: BufferConfig = {
  maxBufferTime: 100,
  minPackets: 2,
  maxPackets: 20
};

/**
 * 音频缓冲区类
 * 处理 UDP 乱序包的缓冲、重排序和按序播放
 */
export class AudioBuffer {
  private packets: Map<number, AudioPacket> = new Map();
  private config: BufferConfig;
  private expectedSeq = 0; // 期望的下一个序列号
  private audioContext: AudioContext | null = null;
  private isPlaying = false;
  private flushTimer: ReturnType<typeof setTimeout> | null = null;

  // 回调
  private onPlayCallbacks: Set<(data: ArrayBuffer) => void> = new Set();
  private onErrorCallbacks: Set<(error: Error) => void> = new Set();

  constructor(config: BufferConfig = DEFAULT_TTS_CONFIG) {
    this.config = config;
    this.initAudioContext();
  }

  /**
   * 初始化 AudioContext
   */
  private initAudioContext(): void {
    try {
      this.audioContext = new AudioContext();
    } catch (e) {
      console.error('[AudioBuffer] Failed to create AudioContext:', e);
    }
  }

  /**
   * 添加音频包
   * @param packet 音频包
   */
  push(packet: AudioPacket): void {
    const now = Date.now();

    // 检查是否过期
    if (now - packet.receivedAt > this.config.maxBufferTime * 3) {
      console.warn(`[AudioBuffer] Dropped expired packet seq=${packet.seq}`);
      return;
    }

    // 检查是否在缓冲范围外（太旧的包）
    const seqDiff = packet.seq - this.expectedSeq;
    if (seqDiff < -this.config.maxPackets!) {
      console.warn(`[AudioBuffer] Dropped too old packet seq=${packet.seq}, expected=${this.expectedSeq}`);
      return;
    }

    // 存入缓冲区
    this.packets.set(packet.seq, {
      ...packet,
      receivedAt: now
    });

    // 尝试按序播放
    this.tryFlush();
  }

  /**
   * 尝试按序播放缓冲区中的包
   */
  private tryFlush(): void {
    if (this.isPlaying) return;

    const toPlay: AudioPacket[] = [];
    let currentSeq = this.expectedSeq;

    // 收集连续的包
    while (toPlay.length < (this.config.maxPackets || 10)) {
      const packet = this.packets.get(currentSeq);
      if (!packet) break;

      toPlay.push(packet);
      currentSeq++;
    }

    // 如果没有连续包，尝试最旧的包（容忍一定乱序）
    if (toPlay.length === 0 && this.packets.size > 0 && this.config.minPackets! > 1) {
      const minSeq = Math.min(...this.packets.keys());
      const maxSeq = Math.max(...this.packets.keys());

      // 检查是否有足够的包可以容忍乱序播放
      if (maxSeq - minSeq < this.config.minPackets!) {
        // 收集所有包
        for (let seq = minSeq; seq <= maxSeq; seq++) {
          const packet = this.packets.get(seq);
          if (packet) toPlay.push(packet);
        }
        currentSeq = maxSeq + 1;
      }
    }

    // 检查是否满足播放条件
    if (toPlay.length < this.config.minPackets!) {
      // 安排延迟播放（防止死锁）
      if (this.flushTimer) clearTimeout(this.flushTimer);
      this.flushTimer = setTimeout(() => {
        this.isPlaying = false;
        this.tryFlush();
      }, this.config.maxBufferTime);
      return;
    }

    // 更新期望序列号
    this.expectedSeq = currentSeq;

    // 移除已播放的包
    toPlay.forEach((p) => this.packets.delete(p.seq));

    // 播放
    this.playPackets(toPlay);
  }

  /**
   * 播放音频包
   */
  private async playPackets(packets: AudioPacket[]): Promise<void> {
    if (!this.audioContext || packets.length === 0) return;

    this.isPlaying = true;

    try {
      // 确保 AudioContext 处于运行状态
      if (this.audioContext.state === 'suspended') {
        await this.audioContext.resume();
      }

      // 合并所有包的数据
      const totalLength = packets.reduce((sum, p) => sum + p.data.byteLength, 0);
      const combinedBuffer = new ArrayBuffer(totalLength);
      const combinedView = new Uint8Array(combinedBuffer);

      let offset = 0;
      for (const packet of packets) {
        combinedView.set(new Uint8Array(packet.data), offset);
        offset += packet.data.byteLength;
      }

      // 解码并播放
      // 注意：实际项目中可能使用 AudioWorklet 或直接发送解码后的 PCM 数据
      try {
        const audioBuffer = await this.audioContext.decodeAudioData(combinedBuffer);
        const source = this.audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(this.audioContext.destination);

        source.onended = () => {
          this.isPlaying = false;
          // 继续播放剩余包
          if (this.packets.size > 0) {
            this.tryFlush();
          }
        };

        source.start();
      } catch {
        // 解码失败，直接发送原始数据（可能已经是 PCM）
        this.onPlayCallbacks.forEach((cb) => cb(combinedBuffer));
        this.isPlaying = false;
      }
    } catch (e) {
      console.error('[AudioBuffer] Playback error:', e);
      this.isPlaying = false;
      this.onErrorCallbacks.forEach((cb) => cb(e as Error));
    }
  }

  /**
   * 刷新缓冲区（强制播放所有包）
   */
  flush(): void {
    if (this.flushTimer) {
      clearTimeout(this.flushTimer);
      this.flushTimer = null;
    }

    // 按序列号排序所有包
    const sortedPackets = Array.from(this.packets.values()).sort((a, b) => a.seq - b.seq);
    this.packets.clear();

    if (sortedPackets.length > 0) {
      this.expectedSeq = sortedPackets[sortedPackets.length - 1].seq + 1;
      this.playPackets(sortedPackets);
    }
  }

  /**
   * 清空缓冲区
   */
  clear(): void {
    if (this.flushTimer) {
      clearTimeout(this.flushTimer);
      this.flushTimer = null;
    }
    this.packets.clear();
    this.expectedSeq = 0;
    this.isPlaying = false;
  }

  /**
   * 获取当前缓冲状态
   */
  getStatus(): { buffered: number; expectedSeq: number; isPlaying: boolean } {
    return {
      buffered: this.packets.size,
      expectedSeq: this.expectedSeq,
      isPlaying: this.isPlaying
    };
  }

  /**
   * 订阅播放事件
   */
  onPlay(callback: (data: ArrayBuffer) => void): () => void {
    this.onPlayCallbacks.add(callback);
    return () => this.onPlayCallbacks.delete(callback);
  }

  /**
   * 订阅错误事件
   */
  onError(callback: (error: Error) => void): () => void {
    this.onErrorCallbacks.add(callback);
    return () => this.onErrorCallbacks.delete(callback);
  }

  /**
   * 销毁缓冲区
   */
  destroy(): void {
    this.clear();
    if (this.audioContext) {
      this.audioContext.close();
      this.audioContext = null;
    }
    this.onPlayCallbacks.clear();
    this.onErrorCallbacks.clear();
  }
}

/**
 * TTS 音频缓冲区（50ms 缓冲）
 */
export class TTSAudioBuffer extends AudioBuffer {
  constructor() {
    super(DEFAULT_TTS_CONFIG);
  }
}

/**
 * ASR 音频缓冲区（100ms 缓冲）
 */
export class ASRAudioBuffer extends AudioBuffer {
  constructor() {
    super(DEFAULT_ASR_CONFIG);
  }
}

/**
 * 创建 TTS 缓冲区
 */
export function createTTSBuffer(): AudioBuffer {
  return new AudioBuffer(DEFAULT_TTS_CONFIG);
}

/**
 * 创建 ASR 缓冲区
 */
export function createASRBuffer(): AudioBuffer {
  return new AudioBuffer(DEFAULT_ASR_CONFIG);
}

export default AudioBuffer;
