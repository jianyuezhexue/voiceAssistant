package voice

import (
	"context"
	"log"
	"sync"

	"voice-assistant/backend/component/asr"
	"voice-assistant/backend/component/llm"
	"voice-assistant/backend/component/tts"
)

// StreamRouter 流式数据路由器
// 实现 ASR → LLM → TTS Pipeline 并行处理
type StreamRouter struct {
	// ASR 输入通道
	asrInput chan *asr.ASRResult

	// LLM 输入/输出通道
	llmInput  chan *llm.LLMResponse
	llmOutput chan *llm.LLMResponse

	// TTS 输入通道
	ttsInput chan *tts.TTSAudio

	// 回调函数
	onASRResult func(result *asr.ASRResult)
	onLLMResponse func(resp *llm.LLMResponse)
	onTTSAudio   func(audio *tts.TTSAudio)

	// 状态管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	// 流水线状态
	asrActive bool
	llmActive bool
	ttsActive bool
}

// NewStreamRouter 创建流式路由器
func NewStreamRouter() *StreamRouter {
	ctx, cancel := context.WithCancel(context.Background())

	router := &StreamRouter{
		asrInput:  make(chan *asr.ASRResult, 100),
		llmInput:  make(chan *llm.LLMResponse, 100),
		llmOutput: make(chan *llm.LLMResponse, 100),
		ttsInput:  make(chan *tts.TTSAudio, 100),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 启动流水线处理
	router.startPipeline()

	return router
}

// startPipeline 启动流水线处理
func (r *StreamRouter) startPipeline() {
	// ASR 结果处理
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.processASR()
	}()

	// LLM 响应处理
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.processLLM()
	}()

	// TTS 音频处理
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.processTTS()
	}()
}

// RouteASR 路由 ASR 结果
func (r *StreamRouter) RouteASR(result *asr.ASRResult) {
	r.mu.Lock()
	if !r.asrActive {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	select {
	case r.asrInput <- result:
	case <-r.ctx.Done():
	}
}

// RouteLLM 路由 LLM 响应
func (r *StreamRouter) RouteLLM(resp *llm.LLMResponse) {
	r.mu.Lock()
	if !r.llmActive {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	select {
	case r.llmInput <- resp:
	case <-r.ctx.Done():
	}
}

// RouteTTS 路由 TTS 音频
func (r *StreamRouter) RouteTTS(audio *tts.TTSAudio) {
	r.mu.Lock()
	if !r.ttsActive {
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()

	select {
	case r.ttsInput <- audio:
	case <-r.ctx.Done():
	}
}

// processASR 处理 ASR 结果
func (r *StreamRouter) processASR() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case result, ok := <-r.asrInput:
			if !ok {
				return
			}

			r.mu.RLock()
			callback := r.onASRResult
			r.mu.RUnlock()

			if callback != nil {
				callback(result)
			}

			// 如果是最终结果，激活 LLM 处理
			if result.IsFinal {
				log.Printf("[StreamRouter] ASR final result: %s", result.Text)
			}
		}
	}
}

// processLLM 处理 LLM 响应
func (r *StreamRouter) processLLM() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case resp, ok := <-r.llmInput:
			if !ok {
				return
			}

			r.mu.RLock()
			callback := r.onLLMResponse
			r.mu.RUnlock()

			if callback != nil {
				callback(resp)
			}

			// 如果是首字符，激活 TTS 处理 (Pipeline 并行)
			if resp.IsChunk && resp.Text != "" {
				log.Printf("[StreamRouter] LLM first chunk received")
			}

			// 如果完成，清理状态
			if resp.IsComplete {
				log.Printf("[StreamRouter] LLM complete")
			}
		}
	}
}

// processTTS 处理 TTS 音频
func (r *StreamRouter) processTTS() {
	for {
		select {
		case <-r.ctx.Done():
			return
		case audio, ok := <-r.ttsInput:
			if !ok {
				return
			}

			r.mu.RLock()
			callback := r.onTTSAudio
			r.mu.RUnlock()

			if callback != nil {
				callback(audio)
			}

			// 如果是最后一片，清理状态
			if audio.IsLast {
				log.Printf("[StreamRouter] TTS complete")
			}
		}
	}
}

// SetOnASRResult 设置 ASR 结果回调
func (r *StreamRouter) SetOnASRResult(callback func(result *asr.ASRResult)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onASRResult = callback
}

// SetOnLLMResponse 设置 LLM 响应回调
func (r *StreamRouter) SetOnLLMResponse(callback func(resp *llm.LLMResponse)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onLLMResponse = callback
}

// SetOnTTSAudio 设置 TTS 音频回调
func (r *StreamRouter) SetOnTTSAudio(callback func(audio *tts.TTSAudio)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onTTSAudio = callback
}

// StartASR 启动 ASR 处理
func (r *StreamRouter) StartASR() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asrActive = true
}

// StopASR 停止 ASR 处理
func (r *StreamRouter) StopASR() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asrActive = false
}

// StartLLM 启动 LLM 处理
func (r *StreamRouter) StartLLM() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llmActive = true
}

// StopLLM 停止 LLM 处理
func (r *StreamRouter) StopLLM() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.llmActive = false
}

// StartTTS 启动 TTS 处理
func (r *StreamRouter) StartTTS() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ttsActive = true
}

// StopTTS 停止 TTS 处理
func (r *StreamRouter) StopTTS() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ttsActive = false
}

// Reset 重置路由器
func (r *StreamRouter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.asrActive = false
	r.llmActive = false
	r.ttsActive = false

	// 清空通道
	for {
		select {
		case <-r.asrInput:
		default:
			goto clearLLM
		}
	}
clearLLM:
	for {
		select {
		case <-r.llmInput:
		default:
			goto clearTTS
		}
	}
clearTTS:
	for {
		select {
		case <-r.ttsInput:
		default:
			return
		}
	}
}

// Close 关闭路由器
func (r *StreamRouter) Close() {
	r.cancel()

	// 关闭所有通道
	close(r.asrInput)
	close(r.llmInput)
	close(r.llmOutput)
	close(r.ttsInput)

	// 等待所有 goroutine 结束
	r.wg.Wait()

	log.Printf("[StreamRouter] Closed")
}

// PipelineState 流水线状态
type PipelineState int

const (
	PipelineIdle PipelineState = iota
	PipelineASR
	PipelineLLM
	PipelineTTS
	PipelineComplete
)

// String 返回状态字符串
func (s PipelineState) String() string {
	switch s {
	case PipelineIdle:
		return "idle"
	case PipelineASR:
		return "asr"
	case PipelineLLM:
		return "llm"
	case PipelineTTS:
		return "tts"
	case PipelineComplete:
		return "complete"
	default:
		return "unknown"
	}
}
