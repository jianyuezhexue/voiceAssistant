package tts

import (
	"context"
	"errors"
	"log"
	"sync"
)

// TTSConfig TTS配置
type TTSConfig struct {
	ModelPath    string
	LexiconPath  string
	SpeakersPath string
	SampleRate   int
	Speed        float32
}

// DefaultTTSConfig 默认配置
var DefaultTTSConfig = TTSConfig{
	SampleRate: 24000,
	Speed:      1.0,
}

// TTSAudio TTS音频结构
type TTSAudio struct {
	Data      []byte
	Timestamp int64
	IsLast    bool
	Duration  int
}

// Client TTS客户端
type Client struct {
	config        *TTSConfig
	mu            sync.RWMutex
	isRunning     bool
	isInterrupted bool
}

// NewClient 创建TTS客户端
func NewClient(config *TTSConfig) (*Client, error) {
	if config == nil {
		config = &DefaultTTSConfig
	}
	if config.ModelPath == "" {
		return nil, errors.New("model path is required")
	}
	log.Printf("[TTS] Client created (stub mode, model: %s)", config.ModelPath)
	return &Client{config: config, isRunning: false, isInterrupted: false}, nil
}

func (c *Client) StreamSynthesize(ctx context.Context, text string) (<-chan *TTSAudio, error) {
	return nil, errors.New("not implemented")
}

func (c *Client) StreamSynthesizeWithCallback(ctx context.Context, text string, callback func(*TTSAudio)) error {
	return errors.New("not implemented")
}

func (c *Client) Synthesize(text string) (*TTSAudio, error) {
	return nil, errors.New("not implemented")
}

func (c *Client) Interrupt() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isRunning {
		c.isInterrupted = true
	}
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.isRunning = false
	log.Printf("[TTS] Client closed")
	return nil
}

func SynthesizerFromFile(modelPath string) (*Client, error) {
	return NewClient(&TTSConfig{ModelPath: modelPath})
}

type MultiSpeakerClient struct {
	client         *Client
	defaultSpeaker string
}

func NewMultiSpeakerClient(config *TTSConfig) (*MultiSpeakerClient, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &MultiSpeakerClient{client: client}, nil
}

func (msc *MultiSpeakerClient) SetDefaultSpeaker(speaker string) {
	msc.defaultSpeaker = speaker
}

func (msc *MultiSpeakerClient) AddSpeaker(speaker string) {}

func (msc *MultiSpeakerClient) StreamSynthesizeWithSpeaker(ctx context.Context, text, speaker string) (<-chan *TTSAudio, error) {
	return msc.client.StreamSynthesize(ctx, text)
}

func (msc *MultiSpeakerClient) Close() error {
	return msc.client.Close()
}

type StreamingTTSClient struct {
	client       *Client
	inputBuffer  []string
	outputBuffer chan *TTSAudio
	mu           sync.Mutex
	isRunning    bool
}

func NewStreamingTTSClient(config *TTSConfig) (*StreamingTTSClient, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	return &StreamingTTSClient{
		client:       client,
		inputBuffer:  make([]string, 0),
		outputBuffer: make(chan *TTSAudio, 100),
		isRunning:    false,
	}, nil
}

func (stc *StreamingTTSClient) AddText(text string) {
	stc.mu.Lock()
	defer stc.mu.Unlock()
	stc.inputBuffer = append(stc.inputBuffer, text)
}

func (stc *StreamingTTSClient) Start(ctx context.Context) {}

func (stc *StreamingTTSClient) GetOutput() <-chan *TTSAudio {
	return stc.outputBuffer
}

func (stc *StreamingTTSClient) Interrupt() {
	stc.client.Interrupt()
}

func (stc *StreamingTTSClient) Close() error {
	stc.mu.Lock()
	stc.isRunning = false
	stc.mu.Unlock()
	return stc.client.Close()
}
