package chat

// TextTalk
type TextTalkRep struct {
	SessionId string `json:"sessionId" binding:"required"`
	Text      string `json:"text" binding:"required"`
}

// SpeechTalkReq
type SpeechTalkReq struct {
	SessionId string `json:"sessionId" binding:"required"`
	Text      string `json:"text" binding:"required"`
}

// MsgType
type WsMsgType struct {
	SessionId string    `json:"sessionId" binding:"required"`
	Type      string    `json:"type"`
	Data      WsMsgData `json:"data"`
	Timestamp int64     `json:"timestamp"`
	Id        string    `json:"id"`
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
	SessionId string `json:"sessionId"`
	Text      string `json:"text,omitempty"`
	Audio     []byte `json:"audio,omitempty"`  // TTS 音频分片（json 自动 base64 编码）
	Format    string `json:"format,omitempty"` // TTS 音频格式, e.g. mp3
}
