import { ref, readonly } from 'vue';

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
 * 打断事件结构
 */
export interface InterruptEvent {
  source: InterruptSource;
  reason?: string;
  timestamp: number;
}

/**
 * 打断处理器返回接口
 */
export interface InterruptHandler {
  /** 当前打断事件（只读） */
  lastInterrupt: readonly InterruptEvent | null;
  /** 是否正在打断 */
  isInterrupting: readonly boolean;
  /** 触发打断 */
  interrupt: (source: InterruptSource, reason?: string) => void;
  /** 注册打断钩子 */
  onInterrupt: (callback: (event: InterruptEvent) => void) => () => void;
  /** 重置打断状态 */
  reset: () => void;
}

/**
 * 音频播放控制器接口（用于停止 TTS）
 */
export interface AudioController {
  stop: () => void;
  pause?: () => void;
}

/**
 * 请求取消控制器接口（用于取消 ASR/LLM 请求）
 */
export interface RequestCanceller {
  cancel: () => void;
}

/**
 * 打断处理 composable
 * 统一管理各种打断来源，协调停止 TTS、取消请求、重置状态
 */
export function useInterruptHandler(): InterruptHandler {
  // 打断事件
  const lastInterrupt = ref<InterruptEvent | null>(null);
  // 打断中标志
  const isInterrupting = ref(false);
  // 打断回调集合
  const interruptCallbacks = new Set<(event: InterruptEvent) => void>();
  // 外部控制器引用
  let audioController: AudioController | null = null;
  let asrCanceller: RequestCanceller | null = null;
  let llmCanceller: RequestCanceller | null = null;

  /**
   * 触发打断
   * @param source 打断来源
   * @param reason 打断原因（可选）
   */
  const interrupt = (source: InterruptSource, reason?: string): void => {
    if (isInterrupting.value) {
      return;
    }

    const event: InterruptEvent = {
      source,
      reason,
      timestamp: Date.now()
    };

    isInterrupting.value = true;
    lastInterrupt.value = event;

    console.log(`[InterruptHandler] Interrupt triggered: ${source}`, reason || '');

    // 1. 停止 TTS 播放
    if (audioController) {
      try {
        audioController.stop();
        console.log('[InterruptHandler] TTS playback stopped');
      } catch (e) {
        console.error('[InterruptHandler] Failed to stop TTS:', e);
      }
    }

    // 2. 取消 ASR 请求
    if (asrCanceller) {
      try {
        asrCanceller.cancel();
        console.log('[InterruptHandler] ASR request cancelled');
      } catch (e) {
        console.error('[InterruptHandler] Failed to cancel ASR:', e);
      }
    }

    // 3. 取消 LLM 请求
    if (llmCanceller) {
      try {
        llmCanceller.cancel();
        console.log('[InterruptHandler] LLM request cancelled');
      } catch (e) {
        console.error('[InterruptHandler] Failed to cancel LLM:', e);
      }
    }

    // 4. 通知所有监听器
    interruptCallbacks.forEach((callback) => {
      try {
        callback(event);
      } catch (e) {
        console.error('[InterruptHandler] Callback error:', e);
      }
    });

    // 5. 重置打断标志（延迟清除，允许状态机处理）
    setTimeout(() => {
      isInterrupting.value = false;
    }, 100);
  };

  /**
   * 注册打断回调
   * @param callback 打断回调函数
   * @returns 取消注册函数
   */
  const onInterrupt = (callback: (event: InterruptEvent) => void): () => void => {
    interruptCallbacks.add(callback);
    return () => interruptCallbacks.delete(callback);
  };

  /**
   * 重置打断状态
   */
  const reset = (): void => {
    lastInterrupt.value = null;
    isInterrupting.value = false;
    console.log('[InterruptHandler] Reset');
  };

  /**
   * 设置音频控制器（用于停止 TTS）
   */
  const setAudioController = (controller: AudioController | null): void => {
    audioController = controller;
  };

  /**
   * 设置 ASR 请求取消器
   */
  const setASRCanceller = (canceller: RequestCanceller | null): void => {
    asrCanceller = canceller;
  };

  /**
   * 设置 LLM 请求取消器
   */
  const setLLMCanceller = (canceller: RequestCanceller | null): void => {
    llmCanceller = canceller;
  };

  return {
    lastInterrupt: readonly(lastInterrupt),
    isInterrupting: readonly(isInterrupting),
    interrupt,
    onInterrupt,
    reset,
    // 内部方法，供 voice dialogue store 调用
    _setAudioController: setAudioController,
    _setASRCanceller: setASRCanceller,
    _setLLMCanceller: setLLMCanceller
  } as InterruptHandler & {
    _setAudioController: typeof setAudioController;
    _setASRCanceller: typeof setASRCanceller;
    _setLLMCanceller: typeof setLLMCanceller;
  };
}

/**
 * 创建独立使用的打断处理器
 * 用于需要在组件级别处理打断的场景
 */
export function createInterruptHandler() {
  return useInterruptHandler();
}

export default useInterruptHandler;
