// ==================== 通用 API 类型 ====================

export interface Todo {
  id: number;
  title: string;
  completed: boolean;
  status: 'pending' | 'in_progress' | 'completed';
  created_at: string;
  updated_at: string;
}

export interface Knowledge {
  id: number;
  title: string;
  content: string;
  summary?: string;
  created_at: string;
  updated_at: string;
}

export interface ApiResponse<T> {
  code: number;
  data: T;
  message: string;
}

export interface ASRMessage {
  text?: string;
  error?: string;
  type?: 'transcript' | 'todo' | 'knowledge';
  data?: Todo | Knowledge;
}

// ==================== 语音对话类型（重新导出） ====================

export {
  VoiceState,
  InterruptSource,
  MessageType
} from './voice';

export type {
  AudioConfig,
  VoiceMessage,
  ASRResult,
  LLMResponse,
  TTSAudio,
  InterruptEvent,
  VoiceSession,
  WSClientMessage,
  WSServerMessage,
  DCMessage,
  WakeWordConfig,
  VADConfig,
  PerformanceMetrics,
  StateTransitionEvent,
  StateTransitionRule
} from './voice';
