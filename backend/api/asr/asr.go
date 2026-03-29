package asr

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"voice-assistant/backend/api"
	asrLogic "voice-assistant/backend/logic/asr"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// ASR 语音识别 API
type ASR struct {
	api.Base
}

// NewASR 创建 ASR API 实例
func NewASR() *ASR {
	return &ASR{}
}

// Handle 处理 WebSocket 连接
func (a *ASR) Handle(ctx *gin.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	logic := asrLogic.NewASRLogic(ctx)

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// 处理音频数据
		// TODO: 发送到阿里云 ASR 服务
		_ = messageType
		_ = data

		// 处理识别结果
		result, err := logic.Process(string(data))
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"`+err.Error()+`"}`))
			continue
		}

		// 返回解析结果
		conn.WriteMessage(websocket.TextMessage, []byte(`{"text":"`+result.Text+`"}`))
	}
}