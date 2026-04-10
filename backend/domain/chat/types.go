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

// MsgType
type WsMsgType struct {
	SessionID string    `json:"session_id" binding:"required"`
	Type      string    `json:"type"`
	Data      WsMsgData `json:"data"`
	Timestamp int64     `json:"timestamp"`
	Id        string    `json:"id"`
	SessionId string    `json:"sessionId"`
}

type WsMsgData struct {
	Audio  []byte `json:"audio"`
	Format string `json:"format"`
	IsLast bool   `json:"isLast"`
	Text   string `json:"text"`
}

// TalkResp
type TalkResp struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
}
