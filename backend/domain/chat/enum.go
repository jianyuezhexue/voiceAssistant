package chat

// MessageType 消息类型
type MessageType string

const (
	MsgTypeUserText    MessageType = "user_text"    // 用户发送文本
	MsgTypeUserAudio   MessageType = "user_audio"   // 用户发送音频 (base64)
	MsgTypeASRResult   MessageType = "asr_result"   // ASR 实时识别结果
	MsgTypeASRComplete MessageType = "asr_complete" // ASR 识别完成
	MsgTypeLLMText     MessageType = "llm_text"     // LLM 流式文本
	MsgTypeLLMComplete MessageType = "llm_complete" // LLM 回复完成
	MsgTypeTTSAudio    MessageType = "tts_audio"    // TTS 音频数据 (base64)
	MsgTypeTTSComplete MessageType = "tts_complete" // TTS 播放完成
	MsgTypeStateUpdate MessageType = "state_update" // 状态更新
	MsgTypeError       MessageType = "error"        // 错误消息
	MsgTypeInterrupt   MessageType = "interrupt"    // 打断通知
	MsgTypePing        MessageType = "ping"         // 心跳
	MsgTypePong        MessageType = "pong"         // 心跳
)

// String 返回字符串表示
func (m MessageType) String() string {
	return string(m)
}
