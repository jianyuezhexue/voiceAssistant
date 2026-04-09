package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Message 用户/助手消息结构
type Message struct {
	Role    string `json:"role"`    // role: system, user, assistant
	Content string `json:"content"` // 消息内容
}

// LLMResponse LLM响应结构
type LLMResponse struct {
	Text       string `json:"text"`        // 回复文本片段
	IsChunk    bool   `json:"is_chunk"`    // 是否为文本片段
	IsComplete bool   `json:"is_complete"` // 是否完成
	FullText   string `json:"full_text"`   // 完整回复文本
	Error      error  `json:"-"`           // 错误信息
}

// ChatRequest Chat请求结构
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
	TopP        float64   `json:"top_p"`
	MaxTokens   int       `json:"max_tokens"`
}

// ChatResponse Chat响应结构 (非流式)
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择结构
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage 使用量结构
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk 流式响应块 (SSE格式)
type StreamChunk struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []SChoice `json:"choices"`
}

// SChoice 流式选择结构
type SChoice struct {
	Index        int     `json:"index"`
	Delta        Delta   `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

// Delta Delta结构
type Delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// IChatModel LLM模型接口
type IChatModel interface {
	// StreamChat 流式对话
	// messages: 对话历史
	// 返回响应通道
	StreamChat(ctx context.Context, messages []Message) (<-chan *LLMResponse, error)

	// Chat 同步对话
	Chat(ctx context.Context, messages []Message) (*LLMResponse, error)

	// Close 关闭客户端
	Close() error
}

// Client Qwen API客户端
type Client struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	topP        float64
	timeout     time.Duration
	httpClient  *http.Client
	mu          sync.RWMutex
}

// 单例客户端
var (
	defaultClient *Client
	defaultOnce   sync.Once
	defaultConfig *ClientConfig
)

// GetClient 获取单例 LLM 客户端（惰性加载）
func GetClient(apiKey, baseURL, model string) *Client {
	defaultOnce.Do(func() {
		defaultConfig = &ClientConfig{
			APIKey:      apiKey,
			BaseURL:     baseURL,
			Model:       model,
			MaxTokens:   2000,
			Temperature: 0.7,
			TopP:        0.8,
			Timeout:     30 * time.Second,
		}
		var err error
		defaultClient, err = NewClientWithConfig(defaultConfig)
		if err != nil {
			log.Printf("[LLM] Failed to create default client: %v", err)
			defaultClient = nil
		}
	})
	return defaultClient
}

// GetClientWithConfig 使用配置获取单例 LLM 客户端
func GetClientWithConfig(config *ClientConfig) *Client {
	defaultOnce.Do(func() {
		defaultConfig = config
		var err error
		defaultClient, err = NewClientWithConfig(config)
		if err != nil {
			log.Printf("[LLM] Failed to create default client: %v", err)
			defaultClient = nil
		}
	})
	return defaultClient
}

// NewClient 创建Qwen API客户端
func NewClient(apiKey, baseURL, model string) *Client {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if model == "" {
		model = "qwen-plus"
	}

	return &Client{
		apiKey:      apiKey,
		baseURL:     baseURL,
		model:       model,
		maxTokens:   2000,
		temperature: 0.7,
		topP:        0.8,
		timeout:     30 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ClientConfig 客户端配置
type ClientConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int
	Temperature float64
	TopP        float64
	Timeout     time.Duration
}

// NewClientWithConfig 使用配置创建客户端
func NewClientWithConfig(config *ClientConfig) (*Client, error) {
	if config.APIKey == "" {
		return nil, errors.New("API key is required")
	}

	client := &Client{
		apiKey:      config.APIKey,
		baseURL:     config.BaseURL,
		model:       config.Model,
		maxTokens:   config.MaxTokens,
		temperature: config.Temperature,
		topP:        config.TopP,
		timeout:     config.Timeout,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}

	if client.baseURL == "" {
		client.baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}
	if client.model == "" {
		client.model = "qwen-plus"
	}
	if client.maxTokens == 0 {
		client.maxTokens = 2000
	}
	if client.temperature == 0 {
		client.temperature = 0.7
	}
	if client.topP == 0 {
		client.topP = 0.8
	}
	if client.timeout == 0 {
		client.timeout = 30 * time.Second
	}

	return client, nil
}

// SetMaxTokens 设置最大token数
func (c *Client) SetMaxTokens(maxTokens int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.maxTokens = maxTokens
}

// SetTemperature 设置温度
func (c *Client) SetTemperature(temp float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.temperature = temp
}

// SetTimeout 设置超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = timeout
	c.httpClient.Timeout = timeout
}

// StreamChat 流式对话
func (c *Client) StreamChat(ctx context.Context, messages []Message) (<-chan *LLMResponse, error) {
	resultCh := make(chan *LLMResponse, 100)

	c.mu.RLock()
	url := c.baseURL + "/chat/completions"
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Stream:      true,
		Temperature: c.temperature,
		TopP:        c.topP,
		MaxTokens:   c.maxTokens,
	}
	c.mu.RUnlock()

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	log.Printf("[LLM] Starting stream chat with %d messages", len(messages))

	go func() {
		defer close(resultCh)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			log.Printf("[LLM] Request error: %v", err)
			resultCh <- &LLMResponse{
				IsComplete: true,
				Error:      err,
			}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
			log.Printf("[LLM] Response error: %v", err)
			resultCh <- &LLMResponse{
				IsComplete: true,
				Error:      err,
			}
			return
		}

		reader := resp.Body
		fullText := ""

		// 读取SSE流
		buf := make([]byte, 1024)
		lineBuf := ""

		for {
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					// 发送完成信号
					resultCh <- &LLMResponse{
						Text:       "",
						IsChunk:    false,
						IsComplete: true,
						FullText:   fullText,
					}
					return
				}
				resultCh <- &LLMResponse{
					IsComplete: true,
					Error:      err,
				}
				return
			}

			lineBuf += string(buf[:n])

			// 处理行
			for {
				idx := -1
				for i := 0; i < len(lineBuf); i++ {
					if lineBuf[i] == '\n' {
						idx = i
						break
					}
				}

				if idx < 0 {
					break
				}

				line := lineBuf[:idx]
				lineBuf = lineBuf[idx+1:]

				// 跳过空行和注释
				if line == "" || line[0] == ':' {
					continue
				}

				// 解析SSE数据
				if len(line) < 6 || line[:6] != "data: " {
					continue
				}

				data := line[6:]
				if data == "[DONE]" {
					resultCh <- &LLMResponse{
						Text:       "",
						IsChunk:    false,
						IsComplete: true,
						FullText:   fullText,
					}
					return
				}

				var chunk StreamChunk
				if err := json.Unmarshal([]byte(data), &chunk); err != nil {
					log.Printf("[LLM] Unmarshal chunk error: %v", err)
					continue
				}

				if len(chunk.Choices) > 0 {
					content := chunk.Choices[0].Delta.Content
					if content != "" {
						fullText += content
						resultCh <- &LLMResponse{
							Text:       content,
							IsChunk:    true,
							IsComplete: false,
							FullText:   fullText,
						}
					}
				}
			}
		}
	}()

	return resultCh, nil
}

// StreamChatWithCallback 流式对话带回调
func (c *Client) StreamChatWithCallback(ctx context.Context, messages []Message, callback func(*LLMResponse)) error {
	ch, err := c.StreamChat(ctx, messages)
	if err != nil {
		return err
	}

	for resp := range ch {
		if resp.Error != nil {
			return resp.Error
		}
		if callback != nil {
			callback(resp)
		}
		if resp.IsComplete {
			return nil
		}
	}

	return nil
}

// Chat 同步对话
func (c *Client) Chat(ctx context.Context, messages []Message) (*LLMResponse, error) {
	c.mu.RLock()
	url := c.baseURL + "/chat/completions"
	reqBody := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Stream:      false,
		Temperature: c.temperature,
		TopP:        c.topP,
		MaxTokens:   c.maxTokens,
	}
	c.mu.RUnlock()

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	log.Printf("[LLM] Sending chat request with %d messages", len(messages))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, err
	}

	if len(chatResp.Choices) == 0 {
		return nil, errors.New("no choices in response")
	}

	return &LLMResponse{
		Text:       chatResp.Choices[0].Message.Content,
		IsChunk:    false,
		IsComplete: true,
		FullText:   chatResp.Choices[0].Message.Content,
	}, nil
}

// ChatWithDegradation 带降级的对话 (5秒超时降级)
func (c *Client) ChatWithDegradation(ctx context.Context, messages []Message) (*LLMResponse, error) {
	// 创建5秒超时的context
	deadlineCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 尝试流式对话
	ch, err := c.StreamChat(deadlineCtx, messages)
	if err != nil {
		log.Printf("[LLM] StreamChat failed, trying sync chat: %v", err)
		// 流式失败，尝试同步对话
		return c.Chat(ctx, messages)
	}

	fullText := ""
	for resp := range ch {
		if resp.Error != nil {
			log.Printf("[LLM] Stream error: %v", resp.Error)
			// 流式出错，降级到同步
			return c.Chat(ctx, messages)
		}
		fullText = resp.FullText
	}

	return &LLMResponse{
		Text:       fullText,
		IsChunk:    false,
		IsComplete: true,
		FullText:   fullText,
	}, nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	// HTTP客户端不需要关闭
	return nil
}

// RedisClient 带Redis缓存的LLM客户端
type RedisClient struct {
	*Client
	redisClient *redis.Client
	keyPrefix   string
}

// NewRedisClient 创建带Redis缓存的LLM客户端
func NewRedisClient(redisAddr string, llmConfig *ClientConfig) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	client, err := NewClientWithConfig(llmConfig)
	if err != nil {
		return nil, err
	}

	return &RedisClient{
		Client:      client,
		redisClient: rdb,
		keyPrefix:   "llm:cache:",
	}, nil
}

// GetCachedResponse 获取缓存的响应
func (rc *RedisClient) GetCachedResponse(ctx context.Context, messages []Message) (string, error) {
	// 计算消息的hash作为缓存key
	key := rc.keyPrefix + fmt.Sprintf("%x", hashMessages(messages))

	result, err := rc.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return result, nil
}

// CacheResponse 缓存响应
func (rc *RedisClient) CacheResponse(ctx context.Context, messages []Message, response string) error {
	key := rc.keyPrefix + fmt.Sprintf("%x", hashMessages(messages))
	return rc.redisClient.Set(ctx, key, response, 1*time.Hour).Err()
}

// hashMessages 计算消息的hash
func hashMessages(messages []Message) []byte {
	// 简单hash实现
	data, _ := json.Marshal(messages)
	hash := 0
	for _, b := range data {
		hash = hash*31 + int(b)
	}
	return []byte(fmt.Sprintf("%d", hash))
}

// Close 关闭客户端
func (rc *RedisClient) Close() error {
	return rc.redisClient.Close()
}
