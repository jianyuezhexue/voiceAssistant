package logic

import (
	"voice-assistant/backend/domain/chat"
	"voice-assistant/backend/logic"

	"github.com/gin-gonic/gin"
)

type ChatLogic struct {
	logic.BaseLogic
}

func NewChatLogic(ctx *gin.Context) *ChatLogic {
	return &ChatLogic{BaseLogic: logic.BaseLogic{Ctx: ctx}}
}

// TextTalkRep 文字对话
func (l *ChatLogic) TextTalk(req *chat.TextTalkRep) (string, error) {
	return "", nil
}

// SpeechTalk 语音对话
func (l *ChatLogic) SpeechTalk(req *chat.SpeechTalkReq) (string, error) {
	return "", nil
}
