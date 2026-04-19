package asr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
	"voice-assistant/backend/config"

	nls "github.com/aliyun/alibabacloud-nls-go-sdk"
)

// TTS wraps the Alibaba Cloud NLS SpeechSynthesis SDK with a minimal,
// thread-safe API. Each Synthesize call opens a short-lived NLS connection,
// so one TTS instance may be shared across goroutines/sessions.
type TTS struct {
	connCfg  *nls.ConnectionConfig
	defParam nls.SpeechSynthesisStartParam
	logger   *nls.NlsLogger
	longText bool
}

// ChunkHandler receives audio chunks as they stream in from the NLS service.
// Returning a non-nil error cancels the current synthesis.
type ChunkHandler func(chunk []byte) error

// NewTTS builds a TTS client from the global config.
//   - debug  : enable verbose NLS SDK logs.
//   - longText: use the long-text synthesizer endpoint (supports up to ~10k
//     chars). Use false for short replies (< 300 chars) for lower latency.
func NewTTS(debug, longText bool) (*TTS, error) {
	cfg := config.Config.Tts
	if cfg.AppKey == "" {
		return nil, errors.New("TTS AppKey is required")
	}

	logger := nls.NewNlsLogger(os.Stderr, "[TTS]", 0)
	logger.SetDebug(debug)
	logger.SetLogSil(!debug)

	return &TTS{
		connCfg:  nls.NewConnectionConfigWithToken(nls.DEFAULT_URL, cfg.AppKey, cfg.Token),
		defParam: buildParam(cfg),
		logger:   logger,
		longText: longText,
	}, nil
}

// Synthesize converts a single text into audio, delivering chunks to onChunk
// as they arrive. Blocks until synthesis completes, onChunk returns an error,
// or ctx is cancelled.
func (t *TTS) Synthesize(ctx context.Context, text string, onChunk ChunkHandler) error {
	if text == "" {
		return nil
	}

	done := make(chan error, 1)
	var handlerErr error

	synth, err := nls.NewSpeechSynthesis(
		t.connCfg, t.logger, t.longText,
		func(msg string, _ any) { // onTaskFailed
			trySend(done, fmt.Errorf("tts task failed: %s", msg))
		},
		func(data []byte, _ any) { // onSynthesisResult
			if handlerErr != nil {
				return
			}
			if err := onChunk(data); err != nil {
				handlerErr = err
			}
		},
		nil, // onMetaInfo (subtitles, unused)
		func(_ string, _ any) { // onCompleted
			trySend(done, handlerErr)
		},
		nil, // onClose (ignored: Completed/TaskFailed drive termination)
		nil,
	)
	if err != nil {
		return fmt.Errorf("create synthesizer: %w", err)
	}

	ready, err := synth.Start(text, t.defParam, nil)
	if err != nil {
		return fmt.Errorf("start synthesis: %w", err)
	}

	select {
	case ok := <-ready:
		if !ok {
			synth.Shutdown()
			return errors.New("tts start rejected by server")
		}
	case <-ctx.Done():
		synth.Shutdown()
		return ctx.Err()
	case <-time.After(10 * time.Second):
		synth.Shutdown()
		return errors.New("tts start timeout")
	}

	select {
	case err := <-done:
		synth.Shutdown()
		return err
	case <-ctx.Done():
		synth.Shutdown()
		return ctx.Err()
	}
}

// SynthesizeStream pulls text segments from textCh and synthesizes each in
// order, streaming audio chunks through onChunk. Designed to bridge a
// sentence-segmented LLM stream into continuous WebSocket audio output.
// Returns when textCh is closed, onChunk fails, or ctx is cancelled.
func (t *TTS) SynthesizeStream(ctx context.Context, textCh <-chan string, onChunk ChunkHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case text, ok := <-textCh:
			if !ok {
				return nil
			}
			if text == "" {
				continue
			}
			if err := t.Synthesize(ctx, text, onChunk); err != nil {
				return err
			}
		}
	}
}

func buildParam(cfg config.Tts) nls.SpeechSynthesisStartParam {
	p := nls.DefaultSpeechSynthesisParam()
	if cfg.Voice != "" {
		p.Voice = cfg.Voice
	}
	if cfg.Format != "" {
		p.Format = cfg.Format
	}
	if cfg.SampleRate != 0 {
		p.SampleRate = cfg.SampleRate
	}
	if cfg.Volume != 0 {
		p.Volume = cfg.Volume
	}
	if cfg.SpeechRate != 0 {
		p.SpeechRate = cfg.SpeechRate
	}
	if cfg.PitchRate != 0 {
		p.PitchRate = cfg.PitchRate
	}
	return p
}

func trySend(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}
