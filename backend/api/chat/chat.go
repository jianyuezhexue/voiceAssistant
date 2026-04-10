package chat

import (
	"log"
	"net/http"
	"voice-assistant/backend/api"
	"voice-assistant/backend/domain/chat"
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
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("升级失败:", err)
		return
	}
	defer conn.Close()

	log.Println("客户端连接成功")

	for {
		// 读取消息
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取失败:", err)
			break
		}

		log.Println("收到:", string(msg))

		// 回写（echo）
		err = conn.WriteMessage(msgType, msg)
		if err != nil {
			log.Println("发送失败:", err)
			break
		}
	}

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
