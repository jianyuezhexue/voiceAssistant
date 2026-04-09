package voice

import (
	"context"
	"log"
	"sync"

	"voice-assistant/backend/component/llm"
	"voice-assistant/backend/component/tts"
	"voice-assistant/backend/component/webrtc"
	"voice-assistant/backend/domain/voice"
)

// VoiceDialogueLogic 语音对话逻辑
type VoiceDialogueLogic struct {
	// 组件客户端
	llmClient *llm.Client
	ttsClient *tts.Client
	dcServer  *webrtc.DataChannelHandler

	// 会话管理
	sessionManager *voice.SessionManager

	// 流式路由
	streamRouter *StreamRouter

	// 取消函数存储
	cancelFuncs map[string]context.CancelFunc
	mu          sync.RWMutex

	// 回调函数
	onStateChange func(sessionID string, state voice.VoiceState)
	onASRResult   func(sessionID string, result *voice.ASRResult)
	onLLMResponse func(sessionID string, resp *voice.LLMResponse)
	onTTSAudio    func(sessionID string, audio *voice.TTSAudio)
	onError       func(sessionID string, err error)
}

// VoiceDialogueLogicConfig 语音对话逻辑配置
type VoiceDialogueLogicConfig struct {
	// 注意：LLMClient 已移除，请使用 llm.GetClient() 获取单例
	TTSClient      *tts.Client
	DCServer       *webrtc.DataChannelHandler
	SessionManager *voice.SessionManager
}

// NewVoiceDialogueLogic 创建语音对话逻辑
func NewVoiceDialogueLogic(config *VoiceDialogueLogicConfig) *VoiceDialogueLogic {
	// 使用单例获取 llm.Client（惰性加载）
	llmClient := llm.GetClient("", "", "")

	logic := &VoiceDialogueLogic{
		llmClient:      llmClient,
		ttsClient:      config.TTSClient,
		dcServer:       config.DCServer,
		sessionManager: config.SessionManager,
		streamRouter:   NewStreamRouter(),
		cancelFuncs:    make(map[string]context.CancelFunc),
	}

	// 设置流式路由回调
	logic.streamRouter.SetOnLLMResponse(func(resp *llm.LLMResponse) {
		// LLM响应通过路由处理
	})

	return logic
}

// SetCallbacks 设置回调函数
func (l *VoiceDialogueLogic) SetCallbacks(callbacks *VoiceDialogueCallbacks) {
	l.onStateChange = callbacks.OnStateChange
	l.onASRResult = callbacks.OnASRResult
	l.onLLMResponse = callbacks.OnLLMResponse
	l.onTTSAudio = callbacks.OnTTSAudio
	l.onError = callbacks.OnError
}

// VoiceDialogueCallbacks 回调函数集合
type VoiceDialogueCallbacks struct {
	OnStateChange func(sessionID string, state voice.VoiceState)
	OnASRResult   func(sessionID string, result *voice.ASRResult)
	OnLLMResponse func(sessionID string, resp *voice.LLMResponse)
	OnTTSAudio    func(sessionID string, audio *voice.TTSAudio)
	OnError       func(sessionID string, err error)
}

// HandleWakeWordDetected 处理唤醒词检测
func (l *VoiceDialogueLogic) HandleWakeWordDetected(ctx context.Context, sessionID string) error {
	session, err := l.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	// 状态切换到 LISTENING
	session.UpdateState(voice.StateListening)
	l.notifyStateChange(sessionID, voice.StateListening)

	log.Printf("[VoiceDialogue] Wake word detected for session: %s", sessionID)
	return nil
}

// HandleSpeechStarted 处理语音开始
func (l *VoiceDialogueLogic) HandleSpeechStarted(ctx context.Context, sessionID string) error {
	session, err := l.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	// 取消之前的处理流程
	l.cancelAndCleanup(sessionID)

	// 创建新的上下文
	newCtx, cancel := context.WithCancel(ctx)
	l.storeCancelFunc(sessionID, cancel)
	_ = newCtx // newCtx stored in session for later use

	// 状态切换到 RECOGNIZING
	session.UpdateState(voice.StateRecognizing)
	session.RecognizedText = ""
	l.notifyStateChange(sessionID, voice.StateRecognizing)

	log.Printf("[VoiceDialogue] Speech started for session: %s", sessionID)
	return nil
}

// HandleSpeechEnded 处理语音结束
func (l *VoiceDialogueLogic) HandleSpeechEnded(ctx context.Context, sessionID string, text string) error {
	session, err := l.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	// 保存识别文本
	session.RecognizedText = text
	session.AddContext(text)

	log.Printf("[VoiceDialogue] Speech ended for session: %s, text: %s", sessionID, text)

	// 通知 ASR 完成
	if l.onASRResult != nil {
		l.onASRResult(sessionID, voice.NewASRResult(text, true, 1.0))
	}

	// 启动 LLM 处理
	return l.startLLMProcessing(ctx, sessionID, text)
}

// HandleInterrupt 处理打断
func (l *VoiceDialogueLogic) HandleInterrupt(ctx context.Context, sessionID string, source voice.InterruptSource) error {
	session, err := l.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	log.Printf("[VoiceDialogue] Interrupt for session: %s, source: %s", sessionID, source.String())

	// 取消当前处理流程
	l.cancelAndCleanup(sessionID)

	// 根据来源和当前状态处理打断
	switch source {
	case voice.InterruptUserSpeech, voice.InterruptUserClick:
		// 用户主动打断，切换到 LISTENING 状态
		session.SetInterrupted(true)
		session.UpdateState(voice.StateListening)
		session.RecognizedText = ""
		session.ResponseText = ""
		l.notifyStateChange(sessionID, voice.StateListening)

	case voice.InterruptServerCmd:
		// 服务器命令打断，根据状态处理
		switch session.State {
		case voice.StatePlaying:
			// TTS 播放中被打断，停止 TTS
			if l.ttsClient != nil {
				l.ttsClient.Interrupt()
			}
			session.UpdateState(voice.StateListening)
			l.notifyStateChange(sessionID, voice.StateListening)

		case voice.StateThinking, voice.StateResponding:
			// LLM 处理中被打断，取消请求
			session.UpdateState(voice.StateListening)
			l.notifyStateChange(sessionID, voice.StateListening)

		default:
			session.UpdateState(voice.StateListening)
			l.notifyStateChange(sessionID, voice.StateListening)
		}

	case voice.InterruptTimeout:
		// 超时打断，回到 IDLE
		session.SetInterrupted(true)
		session.UpdateState(voice.StateIdle)
		l.notifyStateChange(sessionID, voice.StateIdle)
	}

	return nil
}

// HandleAudioStream 处理音频流
func (l *VoiceDialogueLogic) HandleAudioStream(ctx context.Context, sessionID string, audioData []byte) error {
	session, err := l.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	// 忽略非识别状态的音频
	if session.State != voice.StateRecognizing && session.State != voice.StateListening {
		return nil
	}

	// 如果还没开始识别，切换到识别状态
	if session.State == voice.StateListening {
		session.UpdateState(voice.StateRecognizing)
		l.notifyStateChange(sessionID, voice.StateRecognizing)
	}

	// TODO: 发送到 ASR 处理
	// 这里需要根据实际的音频处理流程来实现

	return nil
}

// startLLMProcessing 启动 LLM 处理
func (l *VoiceDialogueLogic) startLLMProcessing(ctx context.Context, sessionID string, text string) error {
	session, err := l.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	// 创建新上下文
	newCtx, cancel := context.WithCancel(ctx)
	l.storeCancelFunc(sessionID, cancel)

	// 状态切换到 THINKING
	session.UpdateState(voice.StateThinking)
	session.ResponseText = ""
	l.notifyStateChange(sessionID, voice.StateThinking)

	// 构建消息列表
	messages := l.buildMessages(session, text)

	// 启动 LLM 流式处理
	go func() {
		if err := l.processLLMStream(newCtx, sessionID, messages); err != nil {
			log.Printf("[VoiceDialogue] LLM processing error: %v", err)
			l.notifyError(sessionID, err)
		}
	}()

	return nil
}

// processLLMStream 处理 LLM 流式响应
func (l *VoiceDialogueLogic) processLLMStream(ctx context.Context, sessionID string, messages []llm.Message) error {
	session, err := l.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	// 检查是否已中断
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 启动 LLM 流式对话
	llmChan, err := l.llmClient.StreamChat(ctx, messages)
	if err != nil {
		return err
	}

	// 状态切换到 RESPONDING
	session.UpdateState(voice.StateResponding)
	l.notifyStateChange(sessionID, voice.StateResponding)

	fullText := ""
	firstChunk := true

	// 处理 LLM 响应流
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case resp, ok := <-llmChan:
			if !ok {
				// 流关闭，检查是否完成
				break
			}

			if resp.Error != nil {
				return resp.Error
			}

			if resp.IsComplete {
				// LLM 回复完成
				session.ResponseText = fullText
				session.AddContext(fullText)

				// 通知 LLM 完成
				if l.onLLMResponse != nil {
					l.onLLMResponse(sessionID, voice.NewLLMResponse("", false, true, fullText))
				}

				// 启动 TTS
				if fullText != "" {
					return l.startTTSProcessing(ctx, sessionID, fullText)
				}

				// 无文本，切换到 LISTENING
				session.UpdateState(voice.StateListening)
				l.notifyStateChange(sessionID, voice.StateListening)
				return nil
			}

			if resp.IsChunk && resp.Text != "" {
				fullText += resp.Text

				// 首字符输出时立即启动 TTS (Pipeline 并行)
				if firstChunk && l.ttsClient != nil {
					firstChunk = false
					// 异步启动 TTS
					go func() {
						ttsCtx, ttsCancel := context.WithCancel(ctx)
						l.storeCancelFunc(sessionID+"-tts", ttsCancel)
						if err := l.startTTSProcessing(ttsCtx, sessionID, fullText); err != nil {
							log.Printf("[VoiceDialogue] TTS processing error: %v", err)
						}
					}()
				}

				// 通知 LLM 文本片段
				if l.onLLMResponse != nil {
					l.onLLMResponse(sessionID, voice.NewLLMResponse(resp.Text, true, false, fullText))
				}
			}
		}

		// 检查是否处理完毕
		if fullText != "" && session.ResponseText == fullText {
			break
		}
	}

	return nil
}

// startTTSProcessing 启动 TTS 处理
func (l *VoiceDialogueLogic) startTTSProcessing(ctx context.Context, sessionID string, text string) error {
	session, err := l.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	// 状态切换到 PLAYING
	session.UpdateState(voice.StatePlaying)
	l.notifyStateChange(sessionID, voice.StatePlaying)

	// 启动 TTS 流式合成
	audioChan, err := l.ttsClient.StreamSynthesize(ctx, text)
	if err != nil {
		return err
	}

	// 处理 TTS 音频流
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case audio, ok := <-audioChan:
				if !ok {
					// TTS 完成
					session.UpdateState(voice.StateListening)
					l.notifyStateChange(sessionID, voice.StateListening)
					return
				}

				// 发送 TTS 音频
				if l.dcServer != nil {
					ttsAudio := voice.NewTTSAudio(audio.Data, audio.Timestamp, audio.IsLast)
					if err := l.dcServer.SendTTSAudio(audio.Data); err != nil {
						log.Printf("[VoiceDialogue] Failed to send TTS audio: %v", err)
					}
					if l.onTTSAudio != nil {
						l.onTTSAudio(sessionID, ttsAudio)
					}
				}
			}
		}
	}()

	return nil
}

// buildMessages 构建 LLM 消息列表
func (l *VoiceDialogueLogic) buildMessages(session *voice.Session, text string) []llm.Message {
	messages := make([]llm.Message, 0)

	// 添加系统提示
	messages = append(messages, llm.Message{
		Role:    "system",
		Content: "你是一个智能语音助手，请用简洁的语言回答用户的问题。",
	})

	// 添加对话历史
	for _, msg := range session.Context {
		// 简单处理：交替添加用户和助手消息
		// 实际应用中需要更好的历史管理
		_ = msg
	}

	// 添加当前用户输入
	messages = append(messages, llm.Message{
		Role:    "user",
		Content: text,
	})

	return messages
}

// cancelAndCleanup 取消并清理
func (l *VoiceDialogueLogic) cancelAndCleanup(sessionID string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if cancel, ok := l.cancelFuncs[sessionID]; ok {
		cancel()
		delete(l.cancelFuncs, sessionID)
	}

	// 取消 TTS
	if cancel, ok := l.cancelFuncs[sessionID+"-tts"]; ok {
		cancel()
		delete(l.cancelFuncs, sessionID+"-tts")
	}

	// 中断 TTS
	if l.ttsClient != nil {
		l.ttsClient.Interrupt()
	}
}

// storeCancelFunc 存储取消函数
func (l *VoiceDialogueLogic) storeCancelFunc(sessionID string, cancel context.CancelFunc) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cancelFuncs[sessionID] = cancel
}

// notifyStateChange 通知状态变化
func (l *VoiceDialogueLogic) notifyStateChange(sessionID string, state voice.VoiceState) {
	if l.onStateChange != nil {
		l.onStateChange(sessionID, state)
	}
}

// notifyError 通知错误
func (l *VoiceDialogueLogic) notifyError(sessionID string, err error) {
	if l.onError != nil {
		l.onError(sessionID, err)
	}
}

// Close 关闭
func (l *VoiceDialogueLogic) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 取消所有进行中的处理
	for _, cancel := range l.cancelFuncs {
		cancel()
	}
	l.cancelFuncs = make(map[string]context.CancelFunc)

	return nil
}

// GetSessionManager 获取会话管理器
func (l *VoiceDialogueLogic) GetSessionManager() *voice.SessionManager {
	return l.sessionManager
}

// CreateSession 创建新会话
func (l *VoiceDialogueLogic) CreateSession(userID string) (*voice.Session, error) {
	return l.sessionManager.Create(userID)
}

// GetSession 获取会话
func (l *VoiceDialogueLogic) GetSession(sessionID string) (*voice.Session, error) {
	return l.sessionManager.Get(sessionID)
}

// DeleteSession 删除会话
func (l *VoiceDialogueLogic) DeleteSession(sessionID string) error {
	// 清理相关取消函数
	l.mu.Lock()
	if cancel, ok := l.cancelFuncs[sessionID]; ok {
		cancel()
		delete(l.cancelFuncs, sessionID)
	}
	l.mu.Unlock()

	return l.sessionManager.Delete(sessionID)
}
