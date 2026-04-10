import type { WSClientMessage, WSServerMessage } from '../types';
import { MessageType, VoiceState } from '../types';

/**
 * ArrayBuffer 转 Base64
 */
function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

// 开发环境直接连接后端
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:2400/api/v1/chat/ws';

// 心跳配置
const HEARTBEAT_INTERVAL = 30000; // 30秒
const HEARTBEAT_TIMEOUT = 5000; // 5秒内未响应则重连
const MAX_RECONNECT_ATTEMPTS = 5;
const RECONNECT_DELAY = 3000; // 3秒后重连

type MessageHandler = (message: WSServerMessage) => void;
type StateHandler = (state: VoiceState) => void;
type ErrorHandler = (error: Event) => void;
type ConnectHandler = () => void;
type DisconnectHandler = () => void;

/**
 * WebSocket 语音通信服务
 * 支持心跳机制和自动重连
 */
class VoiceWebSocket {
  private ws: WebSocket | null = null;
  private sessionId: string | null = null;
  private reconnectAttempts = 0;
  private heartbeatTimer: ReturnType<typeof setInterval> | null = null;
  private heartbeatTimeoutTimer: ReturnType<typeof setTimeout> | null = null;
  private isManualDisconnect = false;

  // 事件处理器
  private messageHandlers: Set<MessageHandler> = new Set();
  private stateHandlers: Set<StateHandler> = new Set();
  private errorHandlers: Set<ErrorHandler> = new Set();
  private connectHandlers: Set<ConnectHandler> = new Set();
  private disconnectHandlers: Set<DisconnectHandler> = new Set();

  /**
   * 建立 WebSocket 连接
   */
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      console.log('[WS] Already connected');
      return;
    }

    this.isManualDisconnect = false;
    console.log(`[WS] Connecting to ${WS_URL}...`);

    try {
      this.ws = new WebSocket(WS_URL);
      this.ws.binaryType = 'arraybuffer';

      this.ws.onopen = this.handleOpen.bind(this);
      this.ws.onmessage = this.handleMessage.bind(this);
      this.ws.onerror = this.handleError.bind(this);
      this.ws.onclose = this.handleClose.bind(this);
    } catch (e) {
      console.error('[WS] Failed to create WebSocket:', e);
      this.scheduleReconnect();
    }
  }

  /**
   * 处理连接打开
   */
  private handleOpen(): void {
    console.log('[WS] Connected to voice service');
    this.reconnectAttempts = 0;
    this.connectHandlers.forEach((handler) => handler());
    this.startHeartbeat();
  }

  /**
   * 处理接收消息
   */
  private handleMessage(event: MessageEvent): void {
    try {
      // 处理二进制数据（音频）
      if (event.data instanceof ArrayBuffer) {
        this.handleBinaryMessage(event.data);
        return;
      }

      // 处理文本消息（JSON）
      if (typeof event.data === 'string') {
        const message: WSServerMessage = JSON.parse(event.data);
        this.handleTextMessage(message);
      }
    } catch (e) {
      console.error('[WS] Failed to parse message:', e);
    }
  }

  /**
   * 处理文本消息
   */
  private handleTextMessage(message: WSServerMessage): void {
    console.log('[WS] Received:', message.type, message.sessionId);

    // 处理心跳响应
    if (message.type === MessageType.PONG) {
      this.handleHeartbeatResponse();
      return;
    }

    // 处理状态更新
    if (message.type === MessageType.STATE_UPDATE) {
      const state = message.data as VoiceState;
      this.stateHandlers.forEach((handler) => handler(state));
    }

    // 通知所有消息监听器
    this.messageHandlers.forEach((handler) => handler(message));
  }

  /**
   * 处理二进制消息（音频数据）
   */
  private handleBinaryMessage(data: ArrayBuffer): void {
    // 二进制数据直接转发给消息处理器
    const message: WSServerMessage = {
      type: MessageType.TTS_AUDIO,
      sessionId: this.sessionId || '',
      text: '',
      data: data,
      timestamp: Date.now()
    };
    this.messageHandlers.forEach((handler) => handler(message));
  }

  /**
   * 处理错误
   */
  private handleError(error: Event): void {
    console.error('[WS] Error:', error);
    this.errorHandlers.forEach((handler) => handler(error));
  }

  /**
   * 处理连接关闭
   */
  private handleClose(event: CloseEvent): void {
    console.log(`[WS] Connection closed: ${event.code} - ${event.reason}`);
    this.stopHeartbeat();
    this.disconnectHandlers.forEach((handler) => handler());

    if (!this.isManualDisconnect) {
      this.scheduleReconnect();
    }
  }

  /**
   * 启动心跳
   */
  private startHeartbeat(): void {
    this.stopHeartbeat();

    this.heartbeatTimer = setInterval(() => {
      this.sendHeartbeat();
    }, HEARTBEAT_INTERVAL);
  }

  /**
   * 停止心跳
   */
  private stopHeartbeat(): void {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer);
      this.heartbeatTimer = null;
    }
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer);
      this.heartbeatTimeoutTimer = null;
    }
  }

  /**
   * 发送心跳
   */
  private sendHeartbeat(): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      console.warn('[WS] Cannot send heartbeat, not connected');
      return;
    }

    const message: WSClientMessage = {
      type: MessageType.PING,
      sessionId: this.sessionId || undefined,
      timestamp: Date.now()
    };

    this.ws.send(JSON.stringify(message));

    // 启动心跳超时计时器
    this.heartbeatTimeoutTimer = setTimeout(() => {
      console.warn('[WS] Heartbeat timeout, reconnecting...');
      this.ws?.close();
    }, HEARTBEAT_TIMEOUT);
  }

  /**
   * 处理心跳响应
   */
  private handleHeartbeatResponse(): void {
    if (this.heartbeatTimeoutTimer) {
      clearTimeout(this.heartbeatTimeoutTimer);
      this.heartbeatTimeoutTimer = null;
    }
  }

  /**
   * 安排重连
   */
  private scheduleReconnect(): void {
    if (this.isManualDisconnect) return;

    if (this.reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
      console.error('[WS] Max reconnect attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = RECONNECT_DELAY * this.reconnectAttempts;

    console.log(`[WS] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

    setTimeout(() => {
      this.connect();
    }, delay);
  }

  /**
   * 断开连接
   */
  disconnect(): void {
    this.isManualDisconnect = true;
    this.stopHeartbeat();

    if (this.ws) {
      this.ws.close(1000, 'Manual disconnect');
      this.ws = null;
    }
  }

  /**
   * 发送消息
   */
  send(message: WSClientMessage): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      console.warn('[WS] Cannot send, not connected');
      return;
    }

    // 如果没有 sessionId，使用当前 sessionId
    if (!message.sessionId && this.sessionId) {
      message.sessionId = this.sessionId;
    }

    this.ws.send(JSON.stringify(message));
  }

  /**
   * 发送文本消息
   * 统一使用 JSON 格式: { type: 'user_text', data: { text: '...' } }
   */
  sendText(text: string): void {
    const message: WSClientMessage = {
      type: MessageType.USER_TEXT,
      sessionId: this.sessionId || undefined,
      data: { text },
      timestamp: Date.now()
    };
    this.send(message);
  }

  /**
   * 发送音频数据
   * 统一使用 JSON 格式: { type: 'user_audio', data: { audio: 'base64...', format: 'webm' } }
   */
  sendAudio(audioData: ArrayBuffer, format: string = 'webm', isLast: boolean = true): void {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      console.warn('[WS] Cannot send audio, not connected');
      return;
    }

    // 将 ArrayBuffer 转为 base64
    const base64Audio = arrayBufferToBase64(audioData);

    const message: WSClientMessage = {
      type: MessageType.USER_AUDIO,
      sessionId: this.sessionId || undefined,
      data: {
        audio: base64Audio,
        format,
        isLast
      },
      timestamp: Date.now()
    };

    this.ws.send(JSON.stringify(message));
    console.log(`[WS] Sent audio: ${format}, ${audioData.byteLength} bytes`);
  }

  /**
   * 发送打断命令
   */
  sendInterrupt(reason?: string): void {
    const message: WSClientMessage = {
      type: MessageType.INTERRUPT,
      sessionId: this.sessionId || undefined,
      data: { reason },
      timestamp: Date.now()
    };
    this.send(message);
  }

  /**
   * 设置当前会话 ID
   */
  setSessionId(sessionId: string): void {
    this.sessionId = sessionId;
  }

  /**
   * 获取当前会话 ID
   */
  getSessionId(): string | null {
    return this.sessionId;
  }

  /**
   * 是否已连接
   */
  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  /**
   * 获取连接状态
   */
  get readyState(): number {
    return this.ws?.readyState ?? WebSocket.CLOSED;
  }

  // ==================== 事件订阅 ====================

  /**
   * 订阅消息
   */
  onMessage(handler: MessageHandler): () => void {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  /**
   * 订阅状态更新
   */
  onStateChange(handler: StateHandler): () => void {
    this.stateHandlers.add(handler);
    return () => this.stateHandlers.delete(handler);
  }

  /**
   * 订阅错误
   */
  onError(handler: ErrorHandler): () => void {
    this.errorHandlers.add(handler);
    return () => this.errorHandlers.delete(handler);
  }

  /**
   * 订阅连接成功
   */
  onConnect(handler: ConnectHandler): () => void {
    this.connectHandlers.add(handler);
    return () => this.connectHandlers.delete(handler);
  }

  /**
   * 订阅断开连接
   */
  onDisconnect(handler: DisconnectHandler): () => void {
    this.disconnectHandlers.add(handler);
    return () => this.disconnectHandlers.delete(handler);
  }
}

// 导出单例
export const voiceWS = new VoiceWebSocket();

export default voiceWS;
