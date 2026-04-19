package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"voice-assistant/backend/component/asr"
	"voice-assistant/backend/component/wspool"
	"voice-assistant/backend/config"
	"voice-assistant/backend/domain/agent"
	"voice-assistant/backend/domain/chat"
	"voice-assistant/backend/logic"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

// ChatLogic handles voice and text conversation for a single WebSocket session.
type ChatLogic struct {
	logic.BaseLogic
	asrMu     sync.Mutex
	asrClient *asr.RealTimeASR
}

func NewChatLogic(ctx *gin.Context) *ChatLogic {
	return &ChatLogic{BaseLogic: logic.BaseLogic{Ctx: ctx}}
}

// Talk is the main message-dispatch loop. It blocks until the client disconnects.
func (l *ChatLogic) Talk(client *wspool.WSClient) {
	defer l.cleanup(client.SessionId)

	for {
		wsMsg := client.ReadMessage()
		if wsMsg == nil {
			return
		}

		var msg chat.WsMsgType
		if err := json.Unmarshal(wsMsg.Data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case chat.MsgTypeUserText.String():
			go l.handleText(client, msg)
		case chat.MsgTypeUserAudio.String():
			l.handleAudio(client, msg)
		default:
			log.Printf("[ChatLogic] session=%s unknown message type: %s", client.SessionId, msg.Type)
		}
	}
}

// cleanup releases ASR resources when the session ends.
func (l *ChatLogic) cleanup(sessionId string) {
	l.asrMu.Lock()
	cur := l.asrClient
	l.asrClient = nil
	l.asrMu.Unlock()
	if cur != nil {
		log.Printf("[ChatLogic] session=%s cleaning up ASR", sessionId)
		cur.Stop()
	}
}

// ─── Text conversation ────────────────────────────────────────────────────────

// handleText calls the LLM and returns the text reply.
// If the request originated from voice input (msg.Type == user_audio), it
// also streams a TTS audio rendering back to the client; pure-text requests
// receive only the text reply.
// Runs in a goroutine so it does not block the dispatch loop.
func (l *ChatLogic) handleText(client *wspool.WSClient, msg chat.WsMsgType) {
	resp, err := l.TextTalk(msg)
	if err != nil {
		l.sendError(client, msg.SessionId, err.Error())
		return
	}
	l.sendMsg(client, resp)
	// if msg.Type == chat.MsgTypeUserAudio.String() {
	l.speak(client, resp.SessionId, resp.Text)
	// }

}

// TextTalk calls the LLM agent and returns a structured response.
func (l *ChatLogic) TextTalk(req chat.WsMsgType) (chat.TalkResp, error) {
	commonAgent := agent.NewAgent(l.Ctx)
	answer, err := commonAgent.CommonChat(req.Data.Text)
	if err != nil {
		return chat.TalkResp{Text: "大模型对话失败，请稍后再试"}, err
	}
	return chat.TalkResp{
		Type:      chat.MsgTypeLLMComplete.String(),
		SessionId: req.SessionId,
		Text:      answer,
	}, nil
}

// ─── Voice conversation ───────────────────────────────────────────────────────

// handleAudio manages the ASR lifecycle and forwards audio chunks.
// Called synchronously from the dispatch loop to preserve audio ordering.
func (l *ChatLogic) handleAudio(client *wspool.WSClient, msg chat.WsMsgType) {
	// First chunk: initialise and start the ASR session.
	l.asrMu.Lock()
	needStart := l.asrClient == nil
	l.asrMu.Unlock()

	if needStart {
		asrClient, err := asr.NewRealTimeASR(true)
		if err != nil {
			log.Printf("[ChatLogic] session=%s ASR init error: %v", client.SessionId, err)
			l.sendError(client, msg.SessionId, "语音识别服务初始化失败")
			return
		}

		resultChan, err := asrClient.Start("PCM", 16000)
		if err != nil {
			log.Printf("[ChatLogic] session=%s ASR start error: %v", client.SessionId, err)
			l.sendError(client, msg.SessionId, "语音识别服务启动失败")
			return
		}

		l.asrMu.Lock()
		l.asrClient = asrClient
		l.asrMu.Unlock()
		go l.processASRResults(client, msg.SessionId, resultChan)
	}

	// Forward audio data to the NLS service.
	if len(msg.Data.Audio) > 0 {
		l.asrMu.Lock()
		cur := l.asrClient
		l.asrMu.Unlock()
		if cur != nil {
			if err := cur.SendAudio(msg.Data.Audio); err != nil {
				log.Printf("[ChatLogic] session=%s SendAudio error: %v", client.SessionId, err)
			}
		}
	}

	// Last chunk: stop ASR and let processASRResults drain remaining events.
	if msg.Data.IsLast {
		log.Printf("[ChatLogic] session=%s last audio chunk received, stopping ASR", client.SessionId)
		l.asrMu.Lock()
		cur := l.asrClient
		l.asrClient = nil
		l.asrMu.Unlock()
		if cur != nil {
			if err := cur.Stop(); err != nil {
				log.Printf("[ChatLogic] session=%s ASR stop error: %v", client.SessionId, err)
			}
		}
	}
}

// processASRResults drains the ASR result channel and sends events to the client.
// For each complete sentence it also triggers an LLM call.
func (l *ChatLogic) processASRResults(client *wspool.WSClient, sessionId string, results <-chan asr.ASRResult) {
	for result := range results {
		switch result.Type {

		case asr.ResultInterim:
			l.sendMsg(client, chat.TalkResp{
				Type:      chat.MsgTypeASRResult.String(),
				SessionId: sessionId,
				Text:      result.Text,
			})
			fmt.Printf("[过程转文字]:%s", result.Text)

		case asr.ResultSentenceEnd:
			l.sendMsg(client, chat.TalkResp{
				Type:      chat.MsgTypeASRComplete.String(),
				SessionId: sessionId,
				Text:      result.Text,
			})
			if result.Text != "" {
				go l.handleText(client, chat.WsMsgType{
					SessionId: sessionId,
					Type:      chat.MsgTypeUserAudio.String(), // 标记为语音入口，触发 TTS 回复
					Data:      chat.WsMsgData{Text: result.Text},
				})
			}
			fmt.Printf("[一句话最终文字]:%s", result.Text)

		case asr.ResultError:
			l.asrMu.Lock()
			l.asrClient = nil
			l.asrMu.Unlock()
			l.sendError(client, sessionId, "语音识别失败，请重试")
		}
	}
}

// ─── TTS ─────────────────────────────────────────────────────────────────────

// speak synthesizes text into audio and streams chunks back to the client
// as tts_audio frames, followed by a tts_complete marker.
// Blocks until synthesis finishes, fails, or the 30s timeout expires.
func (l *ChatLogic) speak(client *wspool.WSClient, sessionId, text string) {
	if text == "" {
		return
	}

	tts, err := asr.NewTTS(false, false)
	if err != nil {
		log.Printf("[ChatLogic] session=%s TTS init error: %v", sessionId, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	format := config.Config.Tts.Format
	if format == "" {
		format = "mp3"
	}

	err = tts.Synthesize(ctx, text, func(chunk []byte) error {
		if !client.Send(mustMarshal(chat.TalkResp{
			Type:      chat.MsgTypeTTSAudio.String(),
			SessionId: sessionId,
			Audio:     chunk,
			Format:    format,
		})) {
			return fmt.Errorf("client send failed")
		}
		return nil
	})
	if err != nil {
		log.Printf("[ChatLogic] session=%s TTS synthesize error: %v", sessionId, err)
		l.sendError(client, sessionId, "语音合成失败")
		return
	}

	l.sendMsg(client, chat.TalkResp{
		Type:      chat.MsgTypeTTSComplete.String(),
		SessionId: sessionId,
	})
}

// SpeakStream renders a stream of text segments (typically sentence-level
// output from a streaming LLM) into continuous ws audio. Reserved for the
// future streaming-LLM path; see asr.TTS.SynthesizeStream.
func (l *ChatLogic) SpeakStream(client *wspool.WSClient, sessionId string, textCh <-chan string) {
	tts, err := asr.NewTTS(false, true)
	if err != nil {
		log.Printf("[ChatLogic] session=%s TTS init error: %v", sessionId, err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	format := config.Config.Tts.Format
	if format == "" {
		format = "mp3"
	}

	err = tts.SynthesizeStream(ctx, textCh, func(chunk []byte) error {
		if !client.Send(mustMarshal(chat.TalkResp{
			Type:      chat.MsgTypeTTSAudio.String(),
			SessionId: sessionId,
			Audio:     chunk,
			Format:    format,
		})) {
			return fmt.Errorf("client send failed")
		}
		return nil
	})
	if err != nil {
		log.Printf("[ChatLogic] session=%s TTS stream error: %v", sessionId, err)
		return
	}

	l.sendMsg(client, chat.TalkResp{
		Type:      chat.MsgTypeTTSComplete.String(),
		SessionId: sessionId,
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("[ChatLogic] marshal error: %v", err)
		return nil
	}
	return data
}

func (l *ChatLogic) sendMsg(client *wspool.WSClient, resp chat.TalkResp) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[ChatLogic] marshal error: %v", err)
		return
	}
	if !client.Send(data) {
		log.Printf("[ChatLogic] session=%s send failed (channel full or closed)", client.SessionId)
	}
}

func (l *ChatLogic) sendError(client *wspool.WSClient, sessionId, msg string) {
	l.sendMsg(client, chat.TalkResp{
		Type:      chat.MsgTypeError.String(),
		SessionId: sessionId,
		Text:      msg,
	})
}

// genMessage builds an LLM prompt from a user message.
// Kept for future direct-LLM usage; currently the agent pipeline is used instead.
func (l *ChatLogic) genMessage(req chat.WsMsgType) ([]*schema.Message, error) {
	template := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是一个{role}。你需要用{style}的语气回答问题。"),
		schema.UserMessage("问题: {question}"),
	)
	return template.Format(context.Background(), map[string]any{
		"role":     "专业的个人助手",
		"style":    "积极、温暖且专业",
		"question": req.Data.Text,
	})
}
