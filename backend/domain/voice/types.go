package voice

import (
	"time"
)

// VoiceState 语音对话状态枚举
type VoiceState int

const (
	StateIdle        VoiceState = 0 // 初始空闲状态
	StateListening   VoiceState = 1 // 监听中
	StateRecognizing VoiceState = 2 // 识别中
	StateThinking    VoiceState = 3 // 思考中
	StateResponding  VoiceState = 4 // 回复生成中
	StatePlaying     VoiceState = 5 // 播放中
	StateError       VoiceState = 6 // 错误状态
)

// String 返回状态的字符串表示
func (s VoiceState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateListening:
		return "listening"
	case StateRecognizing:
		return "recognizing"
	case StateThinking:
		return "thinking"
	case StateResponding:
		return "responding"
	case StatePlaying:
		return "playing"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// IsActive 判断状态是否为活跃状态
func (s VoiceState) IsActive() bool {
	return s != StateIdle && s != StateError
}

// MessageType 消息类型枚举
type MessageType string

const (
	// ASR 消息类型
	MsgASRResult   MessageType = "asr_result"   // ASR 实时识别结果
	MsgASRComplete MessageType = "asr_complete" // ASR 识别完成

	// LLM 消息类型
	MsgLLMText     MessageType = "llm_text"     // LLM 实时文本输出
	MsgLLMComplete MessageType = "llm_complete" // LLM 回复完成

	// TTS 消息类型
	MsgTTSStarted  MessageType = "tts_started"  // TTS 开始
	MsgTTSAudio    MessageType = "tts_audio"    // TTS 音频数据
	MsgTTSComplete MessageType = "tts_complete" // TTS 完成

	// 控制消息类型
	MsgStateUpdate MessageType = "state_update" // 状态更新
	MsgError       MessageType = "error"        // 错误
	MsgInterrupt   MessageType = "interrupt"    // 打断命令
	MsgPing        MessageType = "ping"         // 心跳
	MsgPong        MessageType = "pong"         // 心跳响应
	MsgStart       MessageType = "start"        // 开始对话
	MsgStop        MessageType = "stop"         // 停止对话
)

// InterruptSource 打断来源枚举
type InterruptSource int

const (
	InterruptUserSpeech InterruptSource = 0 // 用户说话打断
	InterruptUserClick  InterruptSource = 1 // 用户点击打断
	InterruptServerCmd  InterruptSource = 2 // 服务器命令打断
	InterruptTimeout    InterruptSource = 3 // 超时打断
)

// String 返回打断来源的字符串表示
func (s InterruptSource) String() string {
	switch s {
	case InterruptUserSpeech:
		return "user_speech"
	case InterruptUserClick:
		return "user_click"
	case InterruptServerCmd:
		return "server_cmd"
	case InterruptTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

// Session 会话实体
type Session struct {
	ID             string     // 会话ID
	UserID         string     // 用户ID
	State          VoiceState // 当前状态
	RecognizedText string     // 已识别文本
	ResponseText   string     // 回复文本
	Context        []string   // 上下文对话历史
	CreatedAt      time.Time  // 创建时间
	LastActiveAt   time.Time  // 最后活跃时间
	IsInterrupted  bool       // 是否被打断
}

// NewSession 创建新会话
func NewSession(id, userID string) *Session {
	now := time.Now()
	return &Session{
		ID:             id,
		UserID:         userID,
		State:          StateIdle,
		RecognizedText: "",
		ResponseText:   "",
		Context:        make([]string, 0),
		CreatedAt:      now,
		LastActiveAt:   now,
		IsInterrupted:  false,
	}
}

// AddContext 添加对话上下文
func (s *Session) AddContext(text string) {
	s.Context = append(s.Context, text)
	s.LastActiveAt = time.Now()
}

// ClearContext 清除对话上下文
func (s *Session) ClearContext() {
	s.Context = make([]string, 0)
	s.LastActiveAt = time.Now()
}

// UpdateState 更新状态
func (s *Session) UpdateState(state VoiceState) {
	s.State = state
	s.LastActiveAt = time.Now()
}

// SetInterrupted 设置打断标记
func (s *Session) SetInterrupted(interrupted bool) {
	s.IsInterrupted = interrupted
	s.LastActiveAt = time.Now()
}

// UpdateActivity 更新最后活跃时间
func (s *Session) UpdateActivity() {
	s.LastActiveAt = time.Now()
}

// WSMessage WebSocket 消息结构
type WSMessage struct {
	Type      MessageType `json:"type"`                 // 消息类型
	SessionID string      `json:"session_id,omitempty"` // 会话ID
	Data      interface{} `json:"data,omitempty"`       // 消息数据
	Timestamp int64       `json:"timestamp"`            // 时间戳
}

// NewWSMessage 创建新的 WebSocket 消息
func NewWSMessage(msgType MessageType, sessionID string, data interface{}) *WSMessage {
	return &WSMessage{
		Type:      msgType,
		SessionID: sessionID,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ASRResult ASR 识别结果
type ASRResult struct {
	Text       string  `json:"text"`       // 识别文本
	IsFinal    bool    `json:"is_final"`   // 是否为最终结果
	Confidence float64 `json:"confidence"` // 置信度 0-1
}

// NewASRResult 创建 ASR 识别结果
func NewASRResult(text string, isFinal bool, confidence float64) *ASRResult {
	return &ASRResult{
		Text:       text,
		IsFinal:    isFinal,
		Confidence: confidence,
	}
}

// LLMResponse LLM 回复结构
type LLMResponse struct {
	Text       string `json:"text"`        // 回复文本片段
	IsChunk    bool   `json:"is_chunk"`    // 是否为文本片段
	IsComplete bool   `json:"is_complete"` // 是否完成
	FullText   string `json:"full_text"`   // 完整回复文本
}

// NewLLMResponse 创建 LLM 回复
func NewLLMResponse(text string, isChunk, isComplete bool, fullText string) *LLMResponse {
	return &LLMResponse{
		Text:       text,
		IsChunk:    isChunk,
		IsComplete: isComplete,
		FullText:   fullText,
	}
}

// TTSAudio TTS 音频结构
type TTSAudio struct {
	Data      []byte `json:"data,omitempty"` // 音频数据 (PCM)
	Timestamp int64  `json:"timestamp"`      // 时间戳
	IsLast    bool   `json:"is_last"`        // 是否为最后一帧
}

// NewTTSAudio 创建 TTS 音频
func NewTTSAudio(data []byte, timestamp int64, isLast bool) *TTSAudio {
	return &TTSAudio{
		Data:      data,
		Timestamp: timestamp,
		IsLast:    isLast,
	}
}

// ErrorResponse 错误响应结构
type ErrorResponse struct {
	Code    int    `json:"code"`    // 错误码
	Message string `json:"message"` // 错误信息
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(code int, message string) *ErrorResponse {
	return &ErrorResponse{
		Code:    code,
		Message: message,
	}
}

// StateUpdateData 状态更新数据
type StateUpdateData struct {
	SessionID string     `json:"session_id"` // 会话ID
	State     VoiceState `json:"state"`      // 当前状态
	StateStr  string     `json:"state_str"`  // 状态字符串
}

// NewStateUpdateData 创建状态更新数据
func NewStateUpdateData(sessionID string, state VoiceState) *StateUpdateData {
	return &StateUpdateData{
		SessionID: sessionID,
		State:     state,
		StateStr:  state.String(),
	}
}

// InterruptData 打断数据
type InterruptData struct {
	SessionID string          `json:"session_id"` // 会话ID
	Source    InterruptSource `json:"source"`     // 打断来源
	SourceStr string          `json:"source_str"` // 打断来源字符串
}

// NewInterruptData 创建打断数据
func NewInterruptData(sessionID string, source InterruptSource) *InterruptData {
	return &InterruptData{
		SessionID: sessionID,
		Source:    source,
		SourceStr: source.String(),
	}
}

// ErrorCode 错误码定义
const (
	ErrCodeSessionNotFound = 1001 // 会话不存在
	ErrCodeSessionExpired  = 1002 // 会话已过期
	ErrCodeInvalidState    = 1003 // 无效状态转换
	ErrCodeASRFailure      = 1004 // ASR 服务失败
	ErrCodeLLMFailure      = 1005 // LLM 服务失败
	ErrCodeTTSFailure      = 1006 // TTS 服务失败
	ErrCodeWebRTCFailure   = 1007 // WebRTC 连接失败
	ErrCodeInternalError   = 1008 // 内部错误
	ErrCodeTimeout         = 1009 // 超时错误
	ErrCodeDegradedService = 1010 // 服务降级
)
