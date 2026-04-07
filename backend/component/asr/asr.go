package asr

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync"
)

// ASRConfig ASR配置
type ASRConfig struct {
	ModelPath  string  // 模型路径
	TokensPath string  // 词表路径
	SampleRate int     // 采样率
	Threshold  float32 // VAD阈值
}

// DefaultASRConfig 默认配置
var DefaultASRConfig = ASRConfig{
	SampleRate: 16000,
	Threshold:  0.5,
}

// ASRResult ASR识别结果
type ASRResult struct {
	Text       string
	IsFinal    bool
	Confidence float64
	Timestamp  int64
}

// Client ASR客户端
type Client struct {
	config    *ASRConfig
	token     string
	appKey    string
	mu        sync.Mutex
	isRunning bool
}

// NewClient 创建ASR客户端 (兼容API层调用方式)
func NewClient(token, appKey string) (*Client, error) {
	log.Printf("[ASR] Client created (token: %s..., appKey: %s...)", token[:8], appKey[:8])
	return &Client{
		token:     token,
		appKey:    appKey,
		isRunning: false,
	}, nil
}

// Start 开始识别
func (c *Client) Start(callback func(text string)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isRunning {
		return errors.New("ASR client already running")
	}
	c.isRunning = true
	log.Printf("[ASR] Client started")
	go func() {
		// 模拟识别回调
	}()
	return nil
}

// SendAudio 发送音频数据
func (c *Client) SendAudio(audioData []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isRunning {
		return errors.New("ASR client not running")
	}
	log.Printf("[ASR] Received audio data: %d bytes", len(audioData))
	return nil
}

// Stop 停止识别
func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isRunning {
		return errors.New("ASR client not running")
	}
	c.isRunning = false
	log.Printf("[ASR] Client stopped")
	return nil
}

// StreamRecognize 流式识别
func (c *Client) StreamRecognize(ctx context.Context, audioStream io.Reader) (<-chan *ASRResult, error) {
	return nil, errors.New("not implemented")
}

// StreamRecognizeWithCallback 流式识别带回调
func (c *Client) StreamRecognizeWithCallback(ctx context.Context, audioStream io.Reader, callback func(*ASRResult)) error {
	return errors.New("not implemented")
}

// Close 关闭客户端
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isRunning = false
	log.Printf("[ASR] Client closed")
	return nil
}

// RecognizerFromFile 从文件创建识别器
func RecognizerFromFile(modelPath, tokensPath string) (*Client, error) {
	return &Client{
		config:    &ASRConfig{ModelPath: modelPath, TokensPath: tokensPath},
		isRunning: false,
	}, nil
}

// StreamingClient 流式识别客户端
type StreamingClient struct {
	client *Client
}

// NewStreamingClient 创建流式识别客户端
func NewStreamingClient(config *ASRConfig) (*StreamingClient, error) {
	client, err := NewClient("streaming-token", "streaming-appkey")
	if err != nil {
		return nil, err
	}
	return &StreamingClient{client: client}, nil
}

// FeedAudio 喂入音频数据
func (sc *StreamingClient) FeedAudio(audioData []byte) error {
	return nil
}

// Recognize 识别
func (sc *StreamingClient) Recognize(ctx context.Context) (*ASRResult, error) {
	return nil, nil
}

// Close 关闭
func (sc *StreamingClient) Close() error {
	return sc.client.Close()
}

// Helper function for JSON marshaling
func marshalResult(result *ASRResult) ([]byte, error) {
	return json.Marshal(result)
}
