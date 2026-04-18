package asr

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"
	"voice-assistant/backend/config"

	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
)

// ASRConfig 封装了初始化实时语音识别所需的基本配置
type ASRConfig struct {
	AppKey string // 项目Appkey
	Token  string // 访问Token (推荐缓存使用，避免频繁请求)
	// 如果使用 AK 获取 Token，则填写以下两项，Token留空
	AccessKeyId     string
	AccessKeySecret string
}

// ASRResult 定义了识别过程中返回的结构化结果
type ASRResult struct {
	Type       string // 结果类型: "Temp"(中间结果), "Final"(最终结果), "Error", "SentenceStart", "SentenceEnd"
	Text       string // 识别文本
	RawMessage string // 服务端返回的原始JSON字符串
}

// RealTimeASR 实时语音识别客户端封装
type RealTimeASR struct {
	config     *nls.ConnectionConfig
	logger     *nls.NlsLogger
	resultChan chan ASRResult
	st         *nls.SpeechTranscription
	cancelOnce sync.Once
	isRunning  bool
	mutex      sync.Mutex
}

// NewRealTimeASR 创建一个新的实时语音识别实例
// 参数:
//   - cfg: ASR配置
//   - debug: 是否开启SDK调试日志
func NewRealTimeASR(debug bool) (*RealTimeASR, error) {

	// 读取配置
	cfg := config.Config.Asr
	if cfg.AppKey == "" {
		return nil, errors.New("AppKey is required")
	}

	// 使用token初始化链接配置
	// todo 生产这里要改成动态获取token
	var connCfg *nls.ConnectionConfig
	connCfg = nls.NewConnectionConfigWithToken(nls.DEFAULT_URL, cfg.AppKey, cfg.Token)

	// 初始化日志，默认输出到标准错误，可在此修改为文件
	logger := nls.NewNlsLogger(os.Stderr, "[NLS-ASR]", 0)
	logger.SetDebug(debug)
	logger.SetLogSil(!debug) // 如果非Debug模式，静默普通日志

	return &RealTimeASR{
		config:     connCfg,
		logger:     logger,
		resultChan: make(chan ASRResult, 10), // 缓冲通道，防止阻塞回调
	}, nil
}

// Start 启动识别服务，返回用于接收识别结果的通道
// 参数:
//   - audioFormat: 音频格式，如 "PCM", "OPU"
//   - sampleRate: 采样率，如 16000
func (r *RealTimeASR) Start(audioFormat string, sampleRate int) (<-chan ASRResult, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.isRunning {
		return nil, errors.New("ASR is already running")
	}

	// 1. 配置识别参数
	param := nls.DefaultSpeechTranscriptionParam()
	param.Format = audioFormat
	param.SampleRate = sampleRate
	param.EnableIntermediateResult = true // 开启中间结果，实时性更好
	param.EnablePunctuationPrediction = true
	param.EnableInverseTextNormalization = true

	// 2. 创建识别对象并注册回调
	// 注意：文档示例中回调函数是全局的，这里通过闭包捕获 resultChan
	st, err := nls.NewSpeechTranscription(
		r.config,
		r.logger,
		r.onTaskFailed, r.onStarted,
		r.onSentenceBegin, r.onSentenceEnd, r.onResultChanged,
		r.onCompleted, r.onClosed,
		nil, // param 留空，使用默认的 logger 或其他上下文
	)

	if err != nil {
		return nil, err
	}
	r.st = st

	// 3. 启动识别
	ready, err := st.Start(param, nil)
	if err != nil {
		return nil, err
	}

	// 4. 等待服务端确认启动
	select {
	case done := <-ready:
		if !done {
			r.isRunning = false
			return nil, errors.New("ASR start failed: server returned false")
		}
	case <-time.After(10 * time.Second):
		r.isRunning = false
		return nil, errors.New("ASR start timeout")
	}

	r.isRunning = true
	return r.resultChan, nil
}

// SendAudio 发送音频数据
// 注意：调用方需根据采样率和格式控制发送频率（例如每20ms发送320字节PCM数据）
func (r *RealTimeASR) SendAudio(data []byte) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isRunning || r.st == nil {
		return errors.New("ASR not started or already stopped")
	}
	return r.st.SendAudioData(data)
}

// SendAudioStream 辅助函数：从 io.Reader 流式读取并发送音频
func (r *RealTimeASR) SendAudioStream(reader io.Reader, bufferSize int) error {
	buffer := make([]byte, bufferSize)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			if sendErr := r.SendAudio(buffer[:n]); sendErr != nil {
				return sendErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// Stop 停止识别，并等待最终结果返回
func (r *RealTimeASR) Stop() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if !r.isRunning || r.st == nil {
		return nil // 已经停止或从未启动
	}

	ready, err := r.st.Stop()
	if err != nil {
		return err
	}

	// 等待服务端确认停止
	select {
	case <-ready:
		// 成功停止
	case <-time.After(10 * time.Second):
		r.logger.Println("Stop timeout, forcing shutdown")
		r.st.Shutdown()
	}

	r.isRunning = false
	close(r.resultChan) // 关闭通道，通知接收方结束
	return nil
}

// --- 私有回调函数实现 ---

func (r *RealTimeASR) onTaskFailed(text string, _ interface{}) {
	r.resultChan <- ASRResult{Type: "Error", RawMessage: text}
	r.logger.Println("ASR TaskFailed:", text)
}

func (r *RealTimeASR) onStarted(text string, _ interface{}) {
	r.logger.Println("ASR Started:", text)
}

func (r *RealTimeASR) onSentenceBegin(text string, _ interface{}) {
	r.resultChan <- ASRResult{Type: "SentenceStart", RawMessage: text}
}

func (r *RealTimeASR) onSentenceEnd(text string, _ interface{}) {
	r.resultChan <- ASRResult{Type: "SentenceEnd", RawMessage: text}
}

func (r *RealTimeASR) onResultChanged(text string, _ interface{}) {
	r.resultChan <- ASRResult{Type: "Temp", RawMessage: text}
}

func (r *RealTimeASR) onCompleted(text string, _ interface{}) {
	r.resultChan <- ASRResult{Type: "Final", RawMessage: text}
}

func (r *RealTimeASR) onClosed(_ interface{}) {
	r.logger.Println("ASR Connection Closed")
}
