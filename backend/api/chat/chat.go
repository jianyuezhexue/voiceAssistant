package chat

import (
	"log"
	"net/http"
	"voice-assistant/backend/api"
	logic "voice-assistant/backend/logic/chat"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Chat Chat API处理器
type Chat struct {
	api.Base
}

// NewChat 创建Chat处理器
func NewChat() *Chat {
	return &Chat{}
}

// 升级 HTTP -> WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域（demo用）
	},
}

// Ws websockeDemo
func (a *Chat) WsDemo(ctx *gin.Context) {

	// 升级HTTP请求为WebSocket
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("升级失败:", err)
		return
	}
	defer conn.Close()

	log.Println("客户端连接成功")

	logic := logic.NewChatLogic(ctx)
	logic.Talk(conn)
}
