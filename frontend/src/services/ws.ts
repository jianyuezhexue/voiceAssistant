import type { ASRMessage } from '../types';

// 开发环境直接连接后端，绕过 Vite 代理（二进制数据问题）
const WS_URL = 'ws://localhost:8080/ws/asr';

type MessageHandler = (message: ASRMessage) => void;
type ErrorHandler = (error: Event) => void;

class ASRWebSocket {
  private ws: WebSocket | null = null;
  private messageHandlers: Set<MessageHandler> = new Set();
  private errorHandlers: Set<ErrorHandler> = new Set();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 1; // 开发环境只重连一次
  private reconnectDelay = 2000;

  connect() {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return;
    }

    this.ws = new WebSocket(WS_URL);
    this.ws.binaryType = 'arraybuffer'; // 明确接收二进制数据

    this.ws.onopen = () => {
      console.log('[WS] Connected to ASR service');
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      // 后端可能返回二进制音频数据或 JSON 文本
      try {
        if (typeof event.data === 'string') {
          const message: ASRMessage = JSON.parse(event.data);
          this.messageHandlers.forEach((handler) => handler(message));
        }
        // 如果是二进制数据，暂时忽略（用于音频处理场景）
      } catch (e) {
        console.error('[WS] Failed to parse message:', e);
      }
    };

    this.ws.onerror = (error) => {
      console.error('[WS] Error:', error);
      this.errorHandlers.forEach((handler) => handler(error));
    };

    this.ws.onclose = () => {
      console.log('[WS] Connection closed');
      this.attemptReconnect();
    };
  }

  private attemptReconnect() {
    // 开发环境禁用自动重连，避免控制台刷屏
    // 实际重连应该在用户点击录音按钮时触发
    console.log('[WS] Connection failed, will retry on next recording start');
    this.disconnect();
  }

  send(data: string | ArrayBuffer) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(data);
    }
  }

  sendText(text: string) {
    this.send(text);
  }

  onMessage(handler: MessageHandler) {
    this.messageHandlers.add(handler);
    return () => this.messageHandlers.delete(handler);
  }

  onError(handler: ErrorHandler) {
    this.errorHandlers.add(handler);
    return () => this.errorHandlers.delete(handler);
  }

  disconnect() {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }

  get isConnected() {
    return this.ws?.readyState === WebSocket.OPEN;
  }
}

export const asrService = new ASRWebSocket();
