package chat

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"voice-assistant/backend/api"
	"voice-assistant/backend/component/wspool"
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

	// 从查询参数获取 sessionId
	sessionId := ctx.Query("sessionId")
	if sessionId == "" {
		sessionId = ctx.ClientIP()
	}

	//  连接数预检（升级前检查，避免无效升级）
	pool := wspool.GetPool()
	if pool.Count() >= pool.MaxConnections() {
		a.Error(fmt.Errorf("[Ws] 连接数已满 (%d)，拒绝新连接", pool.Count()))
		return
	}

	// 升级 HTTP → WebSocket
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("[Ws] 升级失败: %v", err)
		a.Error(fmt.Errorf("[Ws] 升级失败: %v", err))
		return
	}
	// 升级成功后 conn 的关闭由 WSClient 内部管理，不再 defer conn.Close()

	// 注册到连接池
	client, err := pool.Register(sessionId, conn)
	if err != nil {
		if errors.Is(err, wspool.ErrSessionExists) {
			// session 重复，返回 409
			conn.WriteJSON(gin.H{"code": 409, "error": "session 已存在"})
		} else {
			// 极端并发下二次拦截（池满）
			conn.WriteJSON(gin.H{"code": 429, "error": err.Error()})
		}
		conn.Close()
		return
	}

	log.Printf("[Ws] 客户端连接成功, session=%s, 当前连接数=%d", sessionId, pool.Count())

	// 启动读写协程 + 业务层消费
	client.Start()

	chatLogic := logic.NewChatLogic(ctx)
	chatLogic.Talk(client)

	// 阻塞直到连接关闭
	<-client.Done()
	log.Printf("[Ws] 连接关闭, session=%s, 剩余连接数=%d", sessionId, pool.Count())
}
