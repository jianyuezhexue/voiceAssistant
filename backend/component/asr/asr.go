package asr

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"voice-assistant/backend/config"

	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
)

// ResultType enumerates the kinds of events emitted on ResultChan.
const (
	ResultInterim     = "Interim"     // intermediate recognition result
	ResultSentenceEnd = "SentenceEnd" // a complete sentence has been recognised
	ResultError       = "Error"       // service error
)

// ASRResult carries a single recognition event from the NLS service.
type ASRResult struct {
	Type string // ResultInterim | ResultSentenceEnd | ResultError
	Text string // recognised text (or error payload for ResultError)
}

// RealTimeASR wraps the Alibaba Cloud NLS SpeechTranscription SDK.
type RealTimeASR struct {
	config     *nls.ConnectionConfig
	logger     *nls.NlsLogger
	ResultChan chan ASRResult

	mu        sync.Mutex
	st        *nls.SpeechTranscription
	isRunning bool

	chanClosed atomic.Bool
}

// NewRealTimeASR creates an ASR instance using the global config.
// Set debug=true to enable verbose NLS SDK logs.
func NewRealTimeASR(debug bool) (*RealTimeASR, error) {
	cfg := config.Config.Asr
	if cfg.AppKey == "" {
		return nil, errors.New("ASR AppKey is required")
	}

	connCfg := nls.NewConnectionConfigWithToken(nls.DEFAULT_URL, cfg.AppKey, cfg.Token)

	logger := nls.NewNlsLogger(os.Stderr, "[ASR]", 0)
	logger.SetDebug(debug)
	logger.SetLogSil(!debug)

	return &RealTimeASR{
		config:     connCfg,
		logger:     logger,
		ResultChan: make(chan ASRResult, 20),
	}, nil
}

// Start connects to the NLS service and blocks until the session is ready.
// Returns the result channel that emits recognition events.
func (r *RealTimeASR) Start(format string, sampleRate int) (<-chan ASRResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRunning {
		return nil, errors.New("ASR already running")
	}

	cfg := config.Config.Asr

	silence := cfg.MaxSentenceSilence
	if silence <= 0 {
		silence = 1500
	}

	param := nls.DefaultSpeechTranscriptionParam()
	param.Format = format
	param.SampleRate = sampleRate
	param.EnableIntermediateResult = true
	param.EnablePunctuationPrediction = true
	param.EnableInverseTextNormalization = true
	param.MaxSentenceSilence = silence

	extra := map[string]any{}
	if cfg.VocabularyId != "" {
		extra["vocabulary_id"] = cfg.VocabularyId
	}
	if cfg.CustomizationId != "" {
		extra["customization_id"] = cfg.CustomizationId
	}
	var extraParam map[string]any
	if len(extra) > 0 {
		extraParam = extra
	}

	st, err := nls.NewSpeechTranscription(
		r.config, r.logger,
		r.onTaskFailed, r.onStarted,
		r.onSentenceBegin, r.onSentenceEnd, r.onResultChanged,
		r.onCompleted, r.onClosed,
		nil,
	)
	if err != nil {
		return nil, err
	}
	r.st = st

	ready, err := st.Start(param, extraParam)
	if err != nil {
		return nil, err
	}

	select {
	case ok := <-ready:
		if !ok {
			return nil, errors.New("ASR start rejected by server")
		}
	case <-time.After(10 * time.Second):
		return nil, errors.New("ASR start timeout")
	}

	r.isRunning = true
	return r.ResultChan, nil
}

// SendAudio sends raw PCM audio bytes to the NLS service.
func (r *RealTimeASR) SendAudio(data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRunning || r.st == nil {
		return errors.New("ASR not running")
	}
	return r.st.SendAudioData(data)
}

// SendAudioStream reads from reader and forwards audio in chunks of bufSize bytes.
func (r *RealTimeASR) SendAudioStream(reader io.Reader, bufSize int) error {
	buf := make([]byte, bufSize)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if sendErr := r.SendAudio(buf[:n]); sendErr != nil {
				return sendErr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// Stop signals the NLS service that audio input is complete, waits for the
// final TranscriptionCompleted event, then closes ResultChan.
func (r *RealTimeASR) Stop() error {
	r.mu.Lock()
	if !r.isRunning || r.st == nil {
		r.mu.Unlock()
		return nil
	}
	st := r.st
	r.isRunning = false
	r.mu.Unlock()

	ready, err := st.Stop()
	if err != nil {
		r.closeResultChan()
		return err
	}

	select {
	case <-ready:
	case <-time.After(10 * time.Second):
		r.logger.Println("Stop timeout, forcing shutdown")
		st.Shutdown()
	}

	r.closeResultChan()
	return nil
}

// closeResultChan closes ResultChan exactly once.
func (r *RealTimeASR) closeResultChan() {
	if r.chanClosed.CompareAndSwap(false, true) {
		close(r.ResultChan)
	}
}

// safeSend writes to ResultChan without panicking if the channel is already closed.
// Drops the event if the channel buffer is full.
func (r *RealTimeASR) safeSend(result ASRResult) {
	defer func() { recover() }()
	if r.chanClosed.Load() {
		return
	}
	select {
	case r.ResultChan <- result:
	default:
		r.logger.Println("ResultChan full, dropping event:", result.Type)
	}
}

// nlsPayload is the minimal NLS JSON response structure for text extraction.
type nlsPayload struct {
	Payload struct {
		Result string `json:"result"`
	} `json:"payload"`
}

// extractText parses the NLS raw JSON and returns the recognised text.
func extractText(raw string) string {
	var p nlsPayload
	if json.Unmarshal([]byte(raw), &p) == nil {
		return p.Payload.Result
	}
	return ""
}

// ─── NLS SDK callbacks ────────────────────────────────────────────────────────

func (r *RealTimeASR) onTaskFailed(text string, _ any) {
	r.logger.Println("TaskFailed:", text)
	r.mu.Lock()
	r.isRunning = false
	r.st = nil
	r.mu.Unlock()
	r.safeSend(ASRResult{Type: ResultError, Text: text})
	r.closeResultChan()
}

func (r *RealTimeASR) onStarted(_ string, _ any) {
	r.logger.Println("TranscriptionStarted")
}

func (r *RealTimeASR) onSentenceBegin(_ string, _ any) {}

func (r *RealTimeASR) onSentenceEnd(text string, _ any) {
	r.safeSend(ASRResult{Type: ResultSentenceEnd, Text: extractText(text)})
}

func (r *RealTimeASR) onResultChanged(text string, _ any) {
	r.safeSend(ASRResult{Type: ResultInterim, Text: extractText(text)})
}

func (r *RealTimeASR) onCompleted(_ string, _ any) {
	r.logger.Println("TranscriptionCompleted")
}

func (r *RealTimeASR) onClosed(_ any) {
	r.logger.Println("Connection closed")
}
