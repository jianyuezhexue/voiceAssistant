package chat

import (
	"voice-assistant/backend/api"
	"voice-assistant/backend/logic"

	"github.com/gin-gonic/gin"
)

// ChatRequest Chat请求结构
type ChatRequest struct {
	SessionID string `json:"session_id" binding:"required"`
	Message   string `json:"message" binding:"required"`
}

// ChatResponse Chat响应结构
type ChatResponse struct {
	SessionID string `json:"session_id"`
	Text      string `json:"text"`
	CreatedAt string `json:"created_at"`
}

// Handler Chat API处理器
type Handler struct {
	api.Base
}

// NewHandler 创建Chat处理器
func NewHandler() *Handler {
	return &Handler{}
}

// Chat 处理文字对话请求
func (a Handler) Chat(c *gin.Context) {

	// 参数校验
	req := &ChatRequest{}
	err := a.Bind(c, req)
	if err != nil {
		a.Error(err)
		return
	}

	// 生成 sessionID
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// 调用逻辑层
	chatLogic := logic.NewChatLogic()
	res, err := chatLogic.ProcessMessage(c.Request.Context(), sessionID, req.Message)
	if err != nil {
		a.Error(err)
		return
	}

	// 接口返回
	a.Success(res, "对话成功")
}
