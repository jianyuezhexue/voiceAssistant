package chat

// SpeechTalkReq
type SpeechTalkReq struct {
	SessionID string `json:"session_id" binding:"required"`
	Text      string `json:"text" binding:"required"`
}
