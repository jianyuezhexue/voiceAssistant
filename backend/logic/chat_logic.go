package logic

import (
	"context"
	"sync"
	"time"

	"voice-assistant/backend/component/llm"
)

// ChatMessage 聊天消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatSession 聊天会话
type ChatSession struct {
	ID        string
	Messages  []ChatMessage
	CreatedAt time.Time
}

// ChatLogic 聊天逻辑
type ChatLogic struct {
	llmClient *llm.Client
	sessions  sync.Map // map[string]*ChatSession
}

// NewChatLogic 创建聊天逻辑
func NewChatLogic(llmClient *llm.Client) *ChatLogic {
	return &ChatLogic{
		llmClient: llmClient,
	}
}

// ProcessMessage 处理用户消息，返回AI回复
func (l *ChatLogic) ProcessMessage(ctx context.Context, sessionID, userMessage string) (string, error) {
	// 获取或创建会话
	sessionInterface, _ := l.sessions.LoadOrStore(sessionID, &ChatSession{
		ID:        sessionID,
		Messages:  make([]ChatMessage, 0),
		CreatedAt: time.Now(),
	})
	session := sessionInterface.(*ChatSession)

	// 添加用户消息
	session.Messages = append(session.Messages, ChatMessage{
		Role:    "user",
		Content: userMessage,
	})

	// 调用 LLM
	messages := make([]llm.Message, len(session.Messages))
	for i, m := range session.Messages {
		messages[i] = llm.Message{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	response, err := l.llmClient.Chat(ctx, messages)
	if err != nil {
		return "", err
	}

	// 添加助手消息到历史
	session.Messages = append(session.Messages, ChatMessage{
		Role:    "assistant",
		Content: response.Text,
	})

	return response.Text, nil
}
