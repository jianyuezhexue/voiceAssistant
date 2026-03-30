package asr

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"voice-assistant/backend/api"
	asrComponent "voice-assistant/backend/component/asr"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
	// 获取配置
	token := ctx.GetHeader("X-Token")
	appKey := ctx.GetHeader("X-Appkey")

	// 默认值
	if token == "" {
		token = "392572cfc26a44ef94a0cccce18e6691"
	}
	if appKey == "" {
		appKey = "auRliXRagRX2txBf"
	}

	// 升级为 WebSocket
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("[ASR] WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Printf("[ASR] New client connected")

	// 创建 ASR 客户端
	client := asrComponent.NewClient(token, appKey)

	// 启动识别
	err = client.Start(func(text string) {
		// 发送识别结果给前端
		result := map[string]interface{}{
			"type": "text",
			"text": text,
		}
		data, _ := json.Marshal(result)
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[ASR] Send result error: %v", err)
		}
	})

	if err != nil {
		log.Printf("[ASR] Start error: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"Failed to start ASR: `+err.Error()+`"}`))
		return
	}
	defer client.Close()

	log.Printf("[ASR] Started successfully")

	// 处理前端发送的音频数据
	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[ASR] Read message error: %v", err)
			break
		}

		switch msgType {
		case websocket.BinaryMessage:
			// 发送音频数据到阿里云 ASR
			if err := client.SendAudio(data); err != nil {
				log.Printf("[ASR] Send audio error: %v", err)
			}
		case websocket.TextMessage:
			// 处理控制命令
			var cmd map[string]interface{}
			if err := json.Unmarshal(data, &cmd); err == nil {
				if cmdType, ok := cmd["type"].(string); ok {
					switch cmdType {
					case "stop":
						client.Stop()
					}
				}
			}
		}
	}

	log.Printf("[ASR] Client disconnected")
}
