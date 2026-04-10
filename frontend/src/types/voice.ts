// ==================== 语音对话类型定义 ====================

/**
 * 语音对话状态枚举
 */
export enum VoiceState {
  /** 初始状态/监听中 */
  LISTENING = 'listening',
  /** 识别中 - 检测到语音正在识别 */
  RECOGNIZING = 'recognizing',
  /** 思考中 - 等待AI回复 */
  THINKING = 'thinking',
  /** 回复中 - AI正在回复（文字） */
  RESPONDING = 'responding',
  /** 播放中 - TTS正在播放 */
  PLAYING = 'playing',
  /** 错误状态 */
  ERROR = 'error',
  /** 空闲状态 */
  IDLE = 'idle'
}

/**
 * 打断来源枚举
 */
export enum InterruptSource {
  USER_SPEECH = 'user_speech',
  USER_CLICK = 'user_click',
  SERVER_CMD = 'server_cmd',
  TIMEOUT = 'timeout'
}

/**
 * 消息类型枚举
 * 统一格式: type + data 结构
 */
export enum MessageType {
  // ========== 客户端发送 ==========
  /** 用户发送文本 */
  USER_TEXT = 'user_text',
  /** 用户发送音频 (base64) */
  USER_AUDIO = 'user_audio',

  // ========== 服务端下发 ==========
  /** ASR 实时识别结果 */
  ASR_RESULT = 'asr_result',
  /** ASR 识别完成 */
  ASR_COMPLETE = 'asr_complete',
  /** LLM 流式文本 */
  LLM_TEXT = 'llm_text',
  /** LLM 回复完成 */
  LLM_COMPLETE = 'llm_complete',
  /** TTS 音频数据 (base64) */
  TTS_AUDIO = 'tts_audio',
  /** TTS 播放完成 */
  TTS_COMPLETE = 'tts_complete',
  /** 状态更新 */
  STATE_UPDATE = 'state_update',
  /** 错误消息 */
  ERROR = 'error',
  /** 打断通知 */
  INTERRUPT = 'interrupt',

  // ========== 心跳 ==========
  PING = 'ping',
  PONG = 'pong'
}

// ==================== 接口定义 ====================

/**
 * 音频配置
 */
export interface AudioConfig {
  sampleRate: number;
  channels: number;
  bitDepth: number;
  chunkDuration: number;
  noiseSuppression: boolean;
  echoCancellation: boolean;
  autoGainControl: boolean;
}

/**
 * 语音消息结构
 */
export interface VoiceMessage {
  id: string;
  type: MessageType;
  sessionId: string;
  data?: unknown;
  timestamp: number;
}

/**
 * ASR识别结果
 */
export interface ASRResult {
  text: string;
  isFinal: boolean;
  confidence?: number;
  startTime?: number;
  endTime?: number;
}

/**
 * LLM回复结构
 */
export interface LLMResponse {
  text: string;
  isChunk: boolean;
  isComplete: boolean;
  fullText?: string;
}

/**
 * TTS音频结构
 */
export interface TTSAudio {
  data: ArrayBuffer;
  isLast: boolean;
  timestamp?: number;
}

/**
 * 打断事件
 */
export interface InterruptEvent {
  source: InterruptSource;
  reason?: string;
  timestamp: number;
}

/**
 * 语音会话状态
 */
export interface VoiceSession {
  id: string;
  userId: string;
  state: VoiceState;
  recognizedText: string;
  responseText: string;
  createdAt: number;
  lastActiveAt: number;
  isInterrupted: boolean;
}

/**
 * 用户文本消息数据
 */
export interface UserTextData {
  text: string;
}

/**
 * 用户音频消息数据
 */
export interface UserAudioData {
  /** base64 编码的音频数据 */
  audio: string;
  /** 音频格式: webm, wav, pcm 等 */
  format: string;
  /** 是否为最后一片 (流式传输用) */
  isLast?: boolean;
}

/**
 * WebSocket消息（前端发送）
 * 统一结构: { type, sessionId, data, timestamp }
 */
export interface WSClientMessage {
  type: MessageType;
  sessionId?: string;
  /** 根据 type 不同，data 类型不同:
   * - USER_TEXT: UserTextData
   * - USER_AUDIO: UserAudioData
   * - INTERRUPT: { reason?: string }
   * - PING: undefined
   */
  data?: unknown;
  timestamp: number;
}

/**
 * WebSocket消息（前端接收）
 * 统一结构: { type, sessionId, data, timestamp }
 */
export interface WSServerMessage {
  type: MessageType;
  sessionId: string;
  text: string;
  /** 根据 type 不同，data 类型不同:
   * - ASR_RESULT: ASRResult
   * - ASR_COMPLETE: ASRResult
   * - LLM_TEXT: LLMResponse
   * - LLM_COMPLETE: LLMResponse
   * - TTS_AUDIO: { audio: string, isLast: boolean }
   * - TTS_COMPLETE: undefined
   * - STATE_UPDATE: VoiceState
   * - ERROR: { code: string, message: string }
   * - INTERRUPT: InterruptEvent
   * - PONG: undefined
   */
  data: unknown;
  timestamp: number;
}

/**
 * DataChannel消息
 */
export interface DCMessage {
  type: 'audio' | 'control';
  payload: ArrayBuffer | string;
  timestamp: number;
}

/**
 * 唤醒词检测配置
 */
export interface WakeWordConfig {
  keyword: string;
  sensitivity: number;
  threshold: number;
  enabled: boolean;
}

/**
 * VAD配置
 */
export interface VADConfig {
  sensitivity: number;
  speechStartThreshold: number;
  speechEndThreshold: number;
  silenceTimeout: number;
  maxSpeechDuration: number;
}

/**
 * 性能指标
 */
export interface PerformanceMetrics {
  e2eLatency: number;
  interruptLatency: number;
  asrLatency: number;
  ttsLatency: number;
}

/**
 * 状态转换事件
 */
export interface StateTransitionEvent {
  /** 源状态 */
  from: VoiceState;
  /** 目标状态 */
  to: VoiceState;
  /** 时间戳 */
  timestamp: number;
  /** 触发原因 */
  reason?: string;
}

/**
 * 状态转换规则
 */
export type StateTransitionRule = {
  [key in VoiceState]?: VoiceState[];
};

// ==================== 导出所有类型 ====================
// 注意: AudioConfig, VoiceMessage 等接口已在上面直接导出
// 此处仅导出那些没有直接 export 的类型别名
