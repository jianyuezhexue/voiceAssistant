package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"voice-assistant/backend/component/asr"
	"voice-assistant/backend/component/wspool"
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

// handleText calls the LLM and sends the reply back to the client.
// Runs in a goroutine so it does not block the dispatch loop.
func (l *ChatLogic) handleText(client *wspool.WSClient, msg chat.WsMsgType) {
	resp, err := l.TextTalk(msg)
	if err != nil {
		l.sendError(client, msg.SessionId, err.Error())
		return
	}
	l.sendMsg(client, resp)
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
			// if result.Text != "" {
			// 	go l.handleText(client, chat.WsMsgType{
			// 		SessionId: sessionId,
			// 		Data:      chat.WsMsgData{Text: result.Text},
			// 	})
			// }
			fmt.Printf("[一句话最终文字]:%s", result.Text)

		case asr.ResultError:
			l.asrMu.Lock()
			l.asrClient = nil
			l.asrMu.Unlock()
			l.sendError(client, sessionId, "语音识别失败，请重试")
		}
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

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
