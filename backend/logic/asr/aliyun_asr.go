package asr

// import (
// 	"encoding/json"
// 	"log"
// 	"sync"

// 	"github.com/aliyun/alibabacloud-nls-go-sdk"
// )

// // AliyunRealTimeASR 阿里云实时语音识别
// type AliyunRealTimeASR struct {
// 	token   string
// 	appKey  string
// 	conn    *nls.Connection
// 	handler func(string)
// 	mu      sync.Mutex
// }

// // NewAliyunRealTimeASR 创建阿里云实时语音识别实例
// func NewAliyunRealTimeASR(token, appKey string) *AliyunRealTimeASR {
// 	return &AliyunRealTimeASR{
// 		token:  token,
// 		appKey: appKey,
// 	}
// }

// // Start 开始识别
// func (a *AliyunRealTimeASR) Start(handler func(string)) error {
// 	a.handler = handler

// 	// 创建 WebSocket 认证 URL
// 	url := "wss://nls-gateway-cn-shanghai.aliyuncs.com/ws/v1"

// 	// 创建连接配置
// 	config := nls.ConnectionConfig{
// 		Url:      url,
// 		Token:    a.token,
// 		AppKey:   a.appKey,
// 		Debug:    true,
// 		OnMessage: a.onMessage,
// 	}

// 	// 建立连接
// 	conn, err := nls.CreateConnection(config)
// 	if err != nil {
// 		log.Printf("[AliyunASR] CreateConnection error: %v", err)
// 		return err
// 	}
// 	a.conn = conn

// 	// 启动连接
// 	if err := conn.Start(); err != nil {
// 		log.Printf("[AliyunASR] Start error: %v", err)
// 		return err
// 	}

// 	log.Printf("[AliyunASR] Started successfully")
// 	return nil
// }

// // SendAudio 发送音频数据
// func (a *AliyunRealTimeASR) SendAudio(data []byte) error {
// 	a.mu.Lock()
// 	defer a.mu.Unlock()

// 	if a.conn == nil {
// 		return nil
// 	}

// 	// 发送音频数据
// 	return a.conn.SendAudio(data)
// }

// // Stop 停止识别
// func (a *AliyunRealTimeASR) Stop() error {
// 	if a.conn == nil {
// 		return nil
// 	}
// 	return a.conn.Stop()
// }

// // Close 关闭连接
// func (a *AliyunRealTimeASR) Close() error {
// 	if a.conn == nil {
// 		return nil
// 	}
// 	return a.conn.Close()
// }

// // onMessage 处理服务端消息
// func (a *AliyunRealTimeASR) onMessage(message []byte) {
// 	log.Printf("[AliyunASR] Received: %s", string(message))

// 	var resp NLSResponse
// 	if err := json.Unmarshal(message, &resp); err != nil {
// 		log.Printf("[AliyunASR] Parse error: %v", err)
// 		return
// 	}

// 	// 处理识别结果
// 	if resp.Payload.Result != "" && a.handler != nil {
// 		a.handler(resp.Payload.Result)
// 	}
// }

// // NLSResponse 阿里云 NLS 响应
// type NLSResponse struct {
// 	Type     string `json:"type"`
// 	TaskID   string `json:"task_id"`
// 	Payload  NLSPayload `json:"payload,omitempty"`
// }

// // NLSPayload NLS 负载
// type NLSPayload struct {
// 	Result     string `json:"result,omitempty"`
// }
