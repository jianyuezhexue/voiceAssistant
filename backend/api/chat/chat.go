package chat

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"voice-assistant/backend/logic"
)

// ChatRequest Chat请求结构
type ChatRequest struct {
	SessionID string `json:"session_id"`
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
	chatLogic *logic.ChatLogic
}

// NewHandler 创建Chat处理器
func NewHandler(chatLogic *logic.ChatLogic) *Handler {
	return &Handler{
		chatLogic: chatLogic,
	}
}

// Chat 处理文字对话请求
func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 生成 sessionID
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "default"
	}

	// 调用 logic 处理
	text, err := h.chatLogic.ProcessMessage(c.Request.Context(), sessionID, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ChatResponse{
		SessionID: sessionID,
		Text:      text,
		CreatedAt: time.Now().Format(time.RFC3339),
	})
}
