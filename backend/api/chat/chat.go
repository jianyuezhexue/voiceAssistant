package chat

import (
	"voice-assistant/backend/api"
	"voice-assistant/backend/domain/chat"
	logic "voice-assistant/backend/logic/chat"

	"github.com/gin-gonic/gin"
)

// Chat Chat API处理器
type Chat struct {
	api.Base
}

// NewChat 创建Chat处理器
func NewChat() *Chat {
	return &Chat{}
}

// 文本对话
func (a *Chat) TextTalk(ctx *gin.Context) {
	// 获取参数
	var req chat.TextTalkRep
	if err := a.Bind(ctx, &req); err != nil {
		a.Error(err)
		return
	}

	// 处理逻辑
	logic := logic.NewChatLogic(ctx)
	res, err := logic.TextTalk(&req)
	if err != nil {
		a.Error(err)
		return
	}

	// 返回成功
	a.Success(res, "")
}

// 语音对话
func (a *Chat) SpeechTalk(ctx *gin.Context) {
	// 获取参数
	var req chat.SpeechTalkReq
	if err := a.Bind(ctx, &req); err != nil {
		a.Error(err)
		return
	}

	// 处理逻辑
	logic := logic.NewChatLogic(ctx)
	res, err := logic.SpeechTalk(&req)
	if err != nil {
		a.Error(err)
		return
	}

	// 返回成功
	a.Success(res, "")
}
