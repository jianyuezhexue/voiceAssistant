package voice

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"voice-assistant/backend/component/tts"
	"voice-assistant/backend/domain/voice"
)

// CircuitBreakerStateType 熔断器状态
type CircuitBreakerStateType int32

const (
	CircuitBreakerClosed CircuitBreakerStateType = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

func (s CircuitBreakerStateType) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "closed"
	case CircuitBreakerOpen:
		return "open"
	case CircuitBreakerHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker 简单的熔断器实现
type CircuitBreaker struct {
	name                 string
	maxRequests          int32
	timeout              time.Duration
	consecutiveFailures  int32
	consecutiveSuccesses int32
	state                CircuitBreakerStateType
	lastStateChange      time.Time
	mu                   sync.RWMutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(name string, maxRequests int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:        name,
		maxRequests: int32(maxRequests),
		timeout:     timeout,
		state:       CircuitBreakerClosed,
	}
}

// State 获取当前状态
func (cb *CircuitBreaker) State() CircuitBreakerStateType {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// RecordSuccess 记录成功
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerHalfOpen:
		cb.consecutiveSuccesses++
		if cb.consecutiveSuccesses >= cb.maxRequests {
			cb.setStateLocked(CircuitBreakerClosed)
		}
	case CircuitBreakerClosed:
		cb.consecutiveFailures = 0
	}
}

// RecordFailure 记录失败
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerClosed:
		cb.consecutiveFailures++
		if cb.consecutiveFailures >= 3 {
			cb.setStateLocked(CircuitBreakerOpen)
		}
	case CircuitBreakerHalfOpen:
		cb.setStateLocked(CircuitBreakerOpen)
	}
}

// setStateLocked 设置状态（需要持有锁）
func (cb *CircuitBreaker) setStateLocked(state CircuitBreakerStateType) {
	cb.state = state
	cb.lastStateChange = time.Now()
	if state == CircuitBreakerClosed {
		cb.consecutiveFailures = 0
		cb.consecutiveSuccesses = 0
	} else if state == CircuitBreakerHalfOpen {
		cb.consecutiveSuccesses = 0
	}
	log.Printf("[CircuitBreaker] %s state changed to %s", cb.name, state.String())
}

// Allow 是否允许请求
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerHalfOpen:
		return cb.consecutiveSuccesses < cb.maxRequests
	case CircuitBreakerOpen:
		// 检查超时后切换到半开
		if time.Since(cb.lastStateChange) > cb.timeout {
			cb.setStateLocked(CircuitBreakerHalfOpen)
			return true
		}
		return false
	}
	return false
}

// Execute 执行函数并自动处理熔断
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Allow() {
		return errors.New("circuit breaker is open")
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
	} else {
		cb.RecordSuccess()
	}
	return err
}

// 熔断器实例
var (
	asrCircuitBreaker = NewCircuitBreaker("asr-service", 3, 30*time.Second)
	llmCircuitBreaker = NewCircuitBreaker("llm-service", 3, 60*time.Second)
	ttsCircuitBreaker = NewCircuitBreaker("tts-service", 3, 30*time.Second)
)

// InterruptHandler 打断处理逻辑
type InterruptHandler struct {
	// 组件客户端
	ttsClient *tts.Client

	// 会话管理器
	sessionManager *voice.SessionManager

	// 取消函数存储
	cancelFuncs map[string]context.CancelFunc
	mu          sync.RWMutex

	// 状态回调
	onStateChange func(sessionID string, state voice.VoiceState)
	onError       func(sessionID string, err error)
}

// NewInterruptHandler 创建打断处理器
func NewInterruptHandler(config *InterruptHandlerConfig) *InterruptHandler {
	return &InterruptHandler{
		ttsClient:      config.TTSClient,
		sessionManager: config.SessionManager,
		cancelFuncs:    make(map[string]context.CancelFunc),
	}
}

// InterruptHandlerConfig 打断处理器配置
type InterruptHandlerConfig struct {
	TTSClient      *tts.Client
	SessionManager *voice.SessionManager
}

// SetCallbacks 设置回调函数
func (h *InterruptHandler) SetCallbacks(callbacks *InterruptCallbacks) {
	h.onStateChange = callbacks.OnStateChange
	h.onError = callbacks.OnError
}

// InterruptCallbacks 回调函数集合
type InterruptCallbacks struct {
	OnStateChange func(sessionID string, state voice.VoiceState)
	OnError       func(sessionID string, err error)
}

// Handle 统一的打断处理入口
func (h *InterruptHandler) Handle(ctx context.Context, sessionID string, source voice.InterruptSource) error {
	session, err := h.sessionManager.Get(sessionID)
	if err != nil {
		return err
	}

	log.Printf("[InterruptHandler] Handle interrupt: session=%s, source=%s, currentState=%s",
		sessionID, source.String(), session.State.String())

	switch source {
	case voice.InterruptUserSpeech:
		// 用户说话打断
		return h.handleUserSpeechInterrupt(ctx, session, sessionID)

	case voice.InterruptUserClick:
		// 用户点击打断
		return h.handleUserClickInterrupt(ctx, session, sessionID)

	case voice.InterruptServerCmd:
		// 服务器命令打断
		return h.handleServerCmdInterrupt(ctx, session, sessionID)

	case voice.InterruptTimeout:
		// 超时打断
		return h.handleTimeoutInterrupt(ctx, session, sessionID)

	default:
		return errors.New("unknown interrupt source")
	}
}

// handleUserSpeechInterrupt 用户说话打断
func (h *InterruptHandler) handleUserSpeechInterrupt(ctx context.Context, session *voice.Session, sessionID string) error {
	// 取消当前处理流程
	h.cancelSession(sessionID)

	switch session.State {
	case voice.StatePlaying:
		// TTS 播放中打断 → 停止 TTS + 切换到 LISTENING
		if h.ttsClient != nil {
			h.ttsClient.Interrupt()
		}
		session.UpdateState(voice.StateListening)
		session.SetInterrupted(true)
		session.RecognizedText = ""
		session.ResponseText = ""
		h.notifyStateChange(sessionID, voice.StateListening)

	case voice.StateThinking, voice.StateResponding:
		// LLM 思考/回复中打断 → 取消 LLM 请求 + 切换到 LISTENING
		session.UpdateState(voice.StateListening)
		session.SetInterrupted(true)
		session.ResponseText = ""
		h.notifyStateChange(sessionID, voice.StateListening)

	case voice.StateRecognizing:
		// ASR 识别中打断 → 切换到 LISTENING
		session.UpdateState(voice.StateListening)
		session.SetInterrupted(true)
		session.RecognizedText = ""
		h.notifyStateChange(sessionID, voice.StateListening)

	default:
		// 其他状态，切换到 LISTENING
		session.UpdateState(voice.StateListening)
		session.SetInterrupted(true)
		h.notifyStateChange(sessionID, voice.StateListening)
	}

	return nil
}

// handleUserClickInterrupt 用户点击打断
func (h *InterruptHandler) handleUserClickInterrupt(ctx context.Context, session *voice.Session, sessionID string) error {
	// 取消当前处理流程
	h.cancelSession(sessionID)

	// 停止所有处理
	if h.ttsClient != nil {
		h.ttsClient.Interrupt()
	}

	// 切换到 IDLE 状态
	session.SetInterrupted(true)
	session.UpdateState(voice.StateIdle)
	session.RecognizedText = ""
	session.ResponseText = ""
	h.notifyStateChange(sessionID, voice.StateIdle)

	return nil
}

// handleServerCmdInterrupt 服务器命令打断
func (h *InterruptHandler) handleServerCmdInterrupt(ctx context.Context, session *voice.Session, sessionID string) error {
	// 取消当前处理流程
	h.cancelSession(sessionID)

	switch session.State {
	case voice.StatePlaying:
		// TTS 播放中打断 → 停止 TTS + 切换到 LISTENING
		if h.ttsClient != nil {
			h.ttsClient.Interrupt()
		}
		session.UpdateState(voice.StateListening)
		session.SetInterrupted(true)
		h.notifyStateChange(sessionID, voice.StateListening)

	case voice.StateThinking, voice.StateResponding:
		// LLM 思考/回复中打断 → 取消 LLM + 切换到 LISTENING
		session.UpdateState(voice.StateListening)
		session.SetInterrupted(true)
		session.ResponseText = ""
		h.notifyStateChange(sessionID, voice.StateListening)

	case voice.StateRecognizing:
		// ASR 识别中打断 → 切换到 LISTENING
		session.UpdateState(voice.StateListening)
		session.SetInterrupted(true)
		session.RecognizedText = ""
		h.notifyStateChange(sessionID, voice.StateListening)

	default:
		session.SetInterrupted(true)
	}

	return nil
}

// handleTimeoutInterrupt 超时打断
func (h *InterruptHandler) handleTimeoutInterrupt(ctx context.Context, session *voice.Session, sessionID string) error {
	// 取消当前处理流程
	h.cancelSession(sessionID)

	// 停止所有处理
	if h.ttsClient != nil {
		h.ttsClient.Interrupt()
	}

	// 切换到 IDLE 状态
	session.SetInterrupted(true)
	session.UpdateState(voice.StateIdle)
	h.notifyStateChange(sessionID, voice.StateIdle)

	return nil
}

// cancelSession 取消会话的所有处理
func (h *InterruptHandler) cancelSession(sessionID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if cancel, ok := h.cancelFuncs[sessionID]; ok {
		cancel()
	}

	if cancel, ok := h.cancelFuncs[sessionID+"-llm"]; ok {
		cancel()
	}

	if cancel, ok := h.cancelFuncs[sessionID+"-tts"]; ok {
		cancel()
	}
}

// storeCancelFunc 存储取消函数
func (h *InterruptHandler) storeCancelFunc(sessionID string, cancel context.CancelFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cancelFuncs[sessionID] = cancel
}

// notifyStateChange 通知状态变化
func (h *InterruptHandler) notifyStateChange(sessionID string, state voice.VoiceState) {
	if h.onStateChange != nil {
		h.onStateChange(sessionID, state)
	}
}

// notifyError 通知错误
func (h *InterruptHandler) notifyError(sessionID string, err error) {
	if h.onError != nil {
		h.onError(sessionID, err)
	}
}

// CircuitBreakerState 熔断器状态
type CircuitBreakerState struct {
	ASR string `json:"asr"`
	LLM string `json:"llm"`
	TTS string `json:"tts"`
}

// GetCircuitBreakerState 获取熔断器状态
func (h *InterruptHandler) GetCircuitBreakerState() CircuitBreakerState {
	return CircuitBreakerState{
		ASR: asrCircuitBreaker.State().String(),
		LLM: llmCircuitBreaker.State().String(),
		TTS: ttsCircuitBreaker.State().String(),
	}
}

// IsServiceAvailable 检查服务是否可用
func (h *InterruptHandler) IsServiceAvailable(service string) bool {
	var cb *CircuitBreaker
	switch service {
	case "asr":
		cb = asrCircuitBreaker
	case "llm":
		cb = llmCircuitBreaker
	case "tts":
		cb = ttsCircuitBreaker
	default:
		return false
	}

	state := cb.State()
	return state == CircuitBreakerClosed || state == CircuitBreakerHalfOpen
}

// ExecuteWithCircuitBreaker 使用熔断器执行
func ExecuteWithCircuitBreaker(cb *CircuitBreaker, name string, fn func() error) error {
	if !cb.Allow() {
		err := errors.New("circuit breaker is open")
		log.Printf("[CircuitBreaker] %s execution blocked", name)
		return err
	}

	err := fn()
	if err != nil {
		cb.RecordFailure()
		log.Printf("[CircuitBreaker] %s execution error: %v", name, err)
	} else {
		cb.RecordSuccess()
	}

	return err
}
