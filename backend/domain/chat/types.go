package chat

// TextTalk
type TextTalkRep struct {
	SessionID string `json:"session_id" binding:"required"`
	Text      string `json:"text" binding:"required"`
}

// SpeechTalkReq
type SpeechTalkReq struct {
	SessionID string `json:"session_id" binding:"required"`
	Text      string `json:"text" binding:"required"`
}
