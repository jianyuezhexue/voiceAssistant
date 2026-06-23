/**
 * TTSPlayer
 * 基于 MediaSource Extensions 的流式 TTS 播放器，逐片接收 base64 mp3
 * 数据并实时播放，关闭时自动调用 endOfStream。
 *
 * 用法：
 *   const player = new TTSPlayer()
 *   player.start('mp3')
 *   player.feed(base64Chunk)         // 多次
 *   player.finish()                  // tts_complete 时调用
 *   player.stop()                    // 中途打断
 */

const MIME_MAP: Record<string, string> = {
  mp3: 'audio/mpeg',
  mpeg: 'audio/mpeg',
  wav: 'audio/wav',
  pcm: 'audio/wav', // PCM 通常需要外层 wav 容器，此处仅占位
  ogg: 'audio/ogg'
};

function base64ToUint8Array(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

function concatChunks(chunks: Uint8Array[]): Uint8Array {
  const total = chunks.reduce((sum, c) => sum + c.byteLength, 0);
  const out = new Uint8Array(total);
  let offset = 0;
  for (const c of chunks) {
    out.set(c, offset);
    offset += c.byteLength;
  }
  return out;
}

export class TTSPlayer {
  private audio: HTMLAudioElement;
  private mediaSource: MediaSource | null = null;
  private sourceBuffer: SourceBuffer | null = null;
  private pending: Uint8Array[] = [];
  private updating = false;
  private finalized = false;
  private mime = 'audio/mpeg';
  private objectUrl: string | null = null;

  constructor() {
    this.audio = new Audio();
    this.audio.autoplay = true;
    this.audio.preload = 'auto';
    this.audio.crossOrigin = 'anonymous';
  }

  /** 开始一段新的 TTS 会话；如果已有播放将先停止。 */
  start(format: string = 'mp3'): void {
    this.stop();
    const mime = MIME_MAP[format.toLowerCase()] || 'audio/mpeg';
    if (typeof MediaSource === 'undefined' || !MediaSource.isTypeSupported(mime)) {
      console.warn('[TTSPlayer] MediaSource 不支持类型', mime, '将退化为整段播放');
      this.mime = mime;
      return;
    }
    this.mime = mime;
    this.mediaSource = new MediaSource();
    this.objectUrl = URL.createObjectURL(this.mediaSource);
    this.audio.src = this.objectUrl;

    this.mediaSource.addEventListener('sourceopen', this.onSourceOpen, { once: true });
  }

  /** 推入一片 base64 编码的音频数据。 */
  feed(base64: string): void {
    if (!base64) return;
    const chunk = base64ToUint8Array(base64);
    this.pending.push(chunk);
    this.flush();
  }

  /** 标记本段 TTS 已结束，等待 SourceBuffer 排空后关闭流。 */
  finish(): void {
    this.finalized = true;
    this.tryEndOfStream();
  }

  /** 强制停止播放并释放资源（用于打断或组件卸载）。 */
  stop(): void {
    this.pending = [];
    this.updating = false;
    this.finalized = false;

    if (this.sourceBuffer && this.mediaSource?.readyState === 'open') {
      try {
        this.sourceBuffer.abort();
      } catch {
        /* noop */
      }
    }
    this.sourceBuffer = null;

    if (this.mediaSource && this.mediaSource.readyState === 'open') {
      try {
        this.mediaSource.endOfStream();
      } catch {
        /* noop */
      }
    }
    this.mediaSource = null;

    try {
      this.audio.pause();
    } catch {
      /* noop */
    }
    this.audio.removeAttribute('src');
    this.audio.load();

    if (this.objectUrl) {
      URL.revokeObjectURL(this.objectUrl);
      this.objectUrl = null;
    }
  }

  /** SourceOpen 后绑定 SourceBuffer 并尝试播放。 */
  private onSourceOpen = (): void => {
    if (!this.mediaSource) return;
    try {
      this.sourceBuffer = this.mediaSource.addSourceBuffer(this.mime);
      this.sourceBuffer.mode = 'sequence';
      this.sourceBuffer.addEventListener('updateend', this.onUpdateEnd);
    } catch (e) {
      console.error('[TTSPlayer] addSourceBuffer 失败:', e);
      return;
    }
    this.flush();
    // 触发一次 play()，需要在用户手势上下文中可能被阻止；失败时静默
    this.audio.play().catch((e) => {
      console.warn('[TTSPlayer] audio.play 被阻止:', e?.message || e);
    });
  };

  /** 一次 append 完成，处理下一批。 */
  private onUpdateEnd = (): void => {
    this.updating = false;
    this.flush();
    this.tryEndOfStream();
  };

  /** 把 pending 中的数据合并后写入 SourceBuffer。 */
  private flush(): void {
    if (!this.sourceBuffer || this.updating || this.pending.length === 0) return;
    const merged = concatChunks(this.pending);
    this.pending = [];
    try {
      this.updating = true;
      // Uint8Array<ArrayBufferLike> 的 .buffer 可能是 SharedArrayBuffer，
      // 此处显式断言为 BufferSource 满足 SourceBuffer 的签名。
      this.sourceBuffer.appendBuffer(merged as unknown as BufferSource);
    } catch (e) {
      console.error('[TTSPlayer] appendBuffer 失败:', e);
      this.updating = false;
    }
  }

  /** 当 finalized 且无待写数据时，关闭 MediaSource。 */
  private tryEndOfStream(): void {
    if (
      !this.finalized ||
      this.updating ||
      this.pending.length > 0 ||
      !this.mediaSource ||
      this.mediaSource.readyState !== 'open'
    ) {
      return;
    }
    try {
      this.mediaSource.endOfStream();
    } catch (e) {
      console.warn('[TTSPlayer] endOfStream 失败:', e);
    }
  }
}

export const ttsPlayer = new TTSPlayer();

export default ttsPlayer;
