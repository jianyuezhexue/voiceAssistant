import type { DCMessage } from '../types';

// WebRTC 配置
const RTC_CONFIG: RTCConfiguration = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' }
  ]
};

// DataChannel 配置 - UDP模式
const DATACHANNEL_CONFIG: RTCDataChannelInit = {
  ordered: false, // 不保证顺序，UDP模式
  maxRetransmits: 0, // 不重传
  protocol: 'udp'
};

/**
 * DataChannel 消息类型
 */
export type DataChannelMessageType = 'audio' | 'control';

/**
 * 控制消息动作
 */
export type ControlAction = 'start' | 'stop' | 'pause' | 'resume' | 'interrupt';

type AudioReceiveCallback = (audioData: ArrayBuffer, timestamp: number) => void;
type ControlReceiveCallback = (action: ControlAction, data?: unknown) => void;
type ConnectionCallback = () => void;
type ErrorCallback = (error: Error) => void;

/**
 * WebRTC DataChannel 音频传输服务
 * 用于实时音频数据传输
 */
class DataChannelService {
  private peerConnection: RTCPeerConnection | null = null;
  private dataChannel: RTCDataChannel | null = null;
  private isConnected = false;
  private isAudioChannelOpen = false;

  // 回调
  private audioReceiveCallbacks: Set<AudioReceiveCallback> = new Set();
  private controlReceiveCallbacks: Set<ControlReceiveCallback> = new Set();
  private connectCallbacks: Set<ConnectionCallback> = new Set();
  private errorCallbacks: Set<ErrorCallback> = new Set();

  /**
   * 初始化 WebRTC 连接
   */
  async initialize(): Promise<void> {
    if (this.peerConnection) {
      console.warn('[DataChannel] Already initialized');
      return;
    }

    console.log('[DataChannel] Initializing...');

    try {
      this.peerConnection = new RTCPeerConnection(RTC_CONFIG);

      // 监听 ICE 连接状态
      this.peerConnection.oniceconnectionstatechange = () => {
        console.log(`[DataChannel] ICE state: ${this.peerConnection?.iceConnectionState}`);
      };

      // 监听 ICE 候选
      this.peerConnection.onicecandidate = (event) => {
        if (event.candidate) {
          console.log('[DataChannel] ICE candidate:', event.candidate);
        }
      };

      // 创建 DataChannel
      this.createDataChannel();

    } catch (e) {
      console.error('[DataChannel] Failed to initialize:', e);
      throw e;
    }
  }

  /**
   * 创建 DataChannel
   */
  private createDataChannel(): void {
    if (!this.peerConnection) return;

    console.log('[DataChannel] Creating data channel...');

    // 在 offer 端创建 DataChannel
    this.dataChannel = this.peerConnection.createDataChannel('voice-audio', DATACHANNEL_CONFIG);
    this.setupDataChannel();
  }

  /**
   * 设置 DataChannel 事件
   */
  private setupDataChannel(): void {
    if (!this.dataChannel) return;

    this.dataChannel.onopen = () => {
      console.log('[DataChannel] Channel opened');
      this.isAudioChannelOpen = true;
      this.isConnected = true;
      this.connectCallbacks.forEach((cb) => cb());
    };

    this.dataChannel.onclose = () => {
      console.log('[DataChannel] Channel closed');
      this.isAudioChannelOpen = false;
      this.isConnected = false;
    };

    this.dataChannel.onmessage = (event) => {
      this.handleMessage(event);
    };

    this.dataChannel.onerror = (error) => {
      console.error('[DataChannel] Channel error:', error);
      this.errorCallbacks.forEach((cb) => cb(new Error(String(error))));
    };
  }

  /**
   * 处理接收到的消息
   */
  private handleMessage(event: MessageEvent): void {
    try {
      // DataChannel 可能收到二进制或文本
      if (event.data instanceof ArrayBuffer) {
        // 音频数据
        const timestamp = Date.now();
        this.audioReceiveCallbacks.forEach((cb) => cb(event.data as ArrayBuffer, timestamp));
      } else if (typeof event.data === 'string') {
        // 控制消息
        const message: DCMessage = JSON.parse(event.data);
        if (message.type === 'control') {
          const action = message.payload as ControlAction;
          this.controlReceiveCallbacks.forEach((cb) => cb(action, message));
        }
      }
    } catch (e) {
      console.error('[DataChannel] Failed to handle message:', e);
    }
  }

  /**
   * 作为 offer 端创建连接
   */
  async createOffer(): Promise<RTCSessionDescriptionInit> {
    if (!this.peerConnection) {
      await this.initialize();
    }

    const offer = await this.peerConnection!.createOffer();
    await this.peerConnection!.setLocalDescription(offer);

    console.log('[DataChannel] Offer created');
    return offer;
  }

  /**
   * 作为 answer 端处理 offer
   */
  async handleOffer(offer: RTCSessionDescriptionInit): Promise<RTCSessionDescriptionInit> {
    if (!this.peerConnection) {
      await this.initialize();
    }

    await this.peerConnection!.setRemoteDescription(new RTCSessionDescription(offer));
    const answer = await this.peerConnection!.createAnswer();
    await this.peerConnection!.setLocalDescription(answer);

    console.log('[DataChannel] Answer created');
    return answer;
  }

  /**
   * 处理 answer
   */
  async handleAnswer(answer: RTCSessionDescriptionInit): Promise<void> {
    if (!this.peerConnection) {
      throw new Error('PeerConnection not initialized');
    }

    await this.peerConnection.setRemoteDescription(new RTCSessionDescription(answer));
    console.log('[DataChannel] Answer applied');
  }

  /**
   * 添加 ICE 候选
   */
  async addIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    if (!this.peerConnection) {
      throw new Error('PeerConnection not initialized');
    }

    await this.peerConnection.addIceCandidate(new RTCIceCandidate(candidate));
    console.log('[DataChannel] ICE candidate added');
  }

  /**
   * 发送音频数据
   * @param audioData 音频数据（ArrayBuffer）
   */
  sendAudio(audioData: ArrayBuffer): void {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      console.warn('[DataChannel] Cannot send audio, channel not ready');
      return;
    }

    try {
      this.dataChannel.send(audioData);
    } catch (e) {
      console.error('[DataChannel] Failed to send audio:', e);
    }
  }

  /**
   * 发送控制消息
   * @param action 控制动作
   * @param data 附加数据
   */
  sendControl(action: ControlAction, data?: unknown): void {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      console.warn('[DataChannel] Cannot send control, channel not ready');
      return;
    }

    try {
      const message: DCMessage = {
        type: 'control',
        payload: JSON.stringify({ action, data }),
        timestamp: Date.now()
      };
      this.dataChannel.send(JSON.stringify(message));
    } catch (e) {
      console.error('[DataChannel] Failed to send control:', e);
    }
  }

  /**
   * 发送文本消息（调试用）
   */
  sendText(text: string): void {
    if (!this.dataChannel || this.dataChannel.readyState !== 'open') {
      console.warn('[DataChannel] Cannot send text, channel not ready');
      return;
    }

    try {
      this.dataChannel.send(text);
    } catch (e) {
      console.error('[DataChannel] Failed to send text:', e);
    }
  }

  /**
   * 订阅音频接收
   */
  onAudioReceive(callback: AudioReceiveCallback): () => void {
    this.audioReceiveCallbacks.add(callback);
    return () => this.audioReceiveCallbacks.delete(callback);
  }

  /**
   * 订阅控制消息接收
   */
  onControlReceive(callback: ControlReceiveCallback): () => void {
    this.controlReceiveCallbacks.add(callback);
    return () => this.controlReceiveCallbacks.delete(callback);
  }

  /**
   * 订阅连接成功
   */
  onConnect(callback: ConnectionCallback): () => void {
    this.connectCallbacks.add(callback);
    return () => this.connectCallbacks.delete(callback);
  }

  /**
   * 订阅错误
   */
  onError(callback: ErrorCallback): () => void {
    this.errorCallbacks.add(callback);
    return () => this.errorCallbacks.delete(callback);
  }

  /**
   * 关闭连接
   */
  close(): void {
    console.log('[DataChannel] Closing...');

    if (this.dataChannel) {
      this.dataChannel.close();
      this.dataChannel = null;
    }

    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }

    this.isConnected = false;
    this.isAudioChannelOpen = false;
  }

  /**
   * 获取连接状态
   */
  getIsConnected(): boolean {
    return this.isConnected && this.dataChannel?.readyState === 'open';
  }
}

// 导出单例
export const dataChannelService = new DataChannelService();

export default dataChannelService;
