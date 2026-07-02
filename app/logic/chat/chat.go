package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"voice-assistant/app/component/asr"
	"voice-assistant/app/component/wspool"
	"voice-assistant/app/config"
	"voice-assistant/app/domain/agent"
	"voice-assistant/app/domain/chat"
	"voice-assistant/app/logic"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

// ChatLogic 处理单个 WebSocket 会话下的语音和文本对话。
type ChatLogic struct {
	logic.BaseLogic
	asrMu     sync.Mutex
	asrClient *asr.RealTimeASR
}

func NewChatLogic(ctx *gin.Context) *ChatLogic {
	return &ChatLogic{BaseLogic: logic.BaseLogic{Ctx: ctx}}
}

// Talk 是消息分发主循环，会一直阻塞直到客户端断开连接。
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

// cleanup 在会话结束时释放 ASR 相关资源。
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

// handleText 调用大模型并返回文本回复。
// 如果请求来自语音输入（msg.Type == user_audio），还会向客户端流式返回 TTS 音频；
// 纯文本请求则只返回文本回复。
// 该方法在 goroutine 中运行，避免阻塞消息分发循环。
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

// TextTalk 调用大模型 Agent 并返回结构化响应。
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

// handleAudio 管理 ASR 的生命周期并转发音频分片。
// 由分发循环同步调用，以保证音频顺序。
func (l *ChatLogic) handleAudio(client *wspool.WSClient, msg chat.WsMsgType) {
	// 首个分片：初始化并启动 ASR 会话。
	l.asrMu.Lock()
	needStart := l.asrClient == nil
	l.asrMu.Unlock()

	if needStart {
		asrClient, err := asr.NewRealTimeASR(true)
		if err != nil {
			log.Printf("[ChatLogic] session=%s ASR init error: %v", client.SessionId, err)
			l.sendError(client, msg.SessionId, fmt.Sprintf("语音识别服务初始化失败: %v", err))
			return
		}

		resultChan, err := asrClient.Start("PCM", 16000)
		if err != nil {
			log.Printf("[ChatLogic] session=%s ASR start error: %v", client.SessionId, err)
			l.sendError(client, msg.SessionId, fmt.Sprintf("语音识别服务启动失败: %v", err))
			return
		}

		l.asrMu.Lock()
		l.asrClient = asrClient
		l.asrMu.Unlock()
		go l.processASRResults(client, msg.SessionId, resultChan)
	}

	// 将音频数据转发到 NLS 服务。
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

	// 最后一个分片：停止 ASR，并让 processASRResults 排空剩余事件。
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

// processASRResults 持续读取 ASR 结果通道，并将事件发送给客户端。
// 对每个完整句子，还会触发一次大模型调用。
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
			l.sendError(client, sessionId, fmt.Sprintf("语音识别失败: %s", result.Text))
		}
	}
}

// ─── TTS ─────────────────────────────────────────────────────────────────────

// speak 将文本合成为语音，并以 tts_audio 帧的形式将音频分片流式返回给客户端，
// 合成完成后发送 tts_complete 标记帧。
// 该方法会阻塞，直到合成结束、失败或超过 30 秒超时。
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

// SpeakStream 将一段连续的文本片段流（通常是流式大模型按句子产出的输出）
// 合成为连续的 WebSocket 音频帧。为未来流式 LLM 路径预留，
// 详见 asr.TTS.SynthesizeStream。
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

// genMessage 根据用户消息构造大模型 Prompt。
// 为未来直接调用大模型预留；当前统一使用 Agent 流水线。
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
