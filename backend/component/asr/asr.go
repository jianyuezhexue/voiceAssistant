package asr

import (
	"log"

	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
)

// Client 阿里云实时语音识别客户端
type Client struct {
	config   *nls.ConnectionConfig
	sr       *nls.SpeechRecognition
	callback func(string)
}

// NewClient 创建 ASR 客户端
func NewClient(token, appKey string) *Client {
	// WebSocket URL
	url := "wss://nls-gateway-cn-shanghai.aliyuncs.com/ws/v1"

	// 创建连接配置
	config := nls.NewConnectionConfigWithToken(url, appKey, token)

	return &Client{
		config: config,
	}
}

// Start 开始实时语音识别
func (c *Client) Start(callback func(string)) error {
	c.callback = callback

	// 创建语音识别实例
	sr, err := nls.NewSpeechRecognition(
		c.config,
		nil, // logger
		c.onTaskFailed,
		c.onStarted,
		c.onResultChanged,
		c.onCompleted,
		c.onClosed,
		nil, // user param
	)
	if err != nil {
		log.Printf("[ASR] NewSpeechRecognition error: %v", err)
		return err
	}
	c.sr = sr

	// 设置识别参数
	param := nls.DefaultSpeechRecognitionParam()
	param.Format = "opus" // 阿里云支持 opus 格式

	// 启动识别
	startCh, err := c.sr.Start(param, nil)
	if err != nil {
		log.Printf("[ASR] Start error: %v", err)
		return err
	}

	// 等待启动完成
	result := <-startCh
	if !result {
		log.Printf("[ASR] Start failed")
		return err
	}

	log.Printf("[ASR] Started successfully")
	return nil
}

// SendAudio 发送音频数据
func (c *Client) SendAudio(data []byte) error {
	if c.sr == nil {
		return nil
	}
	return c.sr.SendAudioData(data)
}

// Stop 停止识别
func (c *Client) Stop() error {
	if c.sr == nil {
		return nil
	}
	stopCh, err := c.sr.Stop()
	if err != nil {
		return err
	}
	<-stopCh
	return nil
}

// Close 关闭连接
func (c *Client) Close() {
	if c.sr != nil {
		c.sr.Shutdown()
		c.sr = nil
	}
}

// 回调函数
func (c *Client) onTaskFailed(text string, param interface{}) {
	log.Printf("[ASR] Task failed: %s", text)
}

func (c *Client) onStarted(text string, param interface{}) {
	log.Printf("[ASR] Started: %s", text)
}

func (c *Client) onResultChanged(text string, param interface{}) {
	log.Printf("[ASR] Result changed: %s", text)
	if c.callback != nil {
		c.callback(text)
	}
}

func (c *Client) onCompleted(text string, param interface{}) {
	log.Printf("[ASR] Completed: %s", text)
}

func (c *Client) onClosed(param interface{}) {
	log.Printf("[ASR] Closed")
}
