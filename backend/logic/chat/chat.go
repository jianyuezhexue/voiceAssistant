package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"voice-assistant/backend/component/asr"
	"voice-assistant/backend/component/wspool"
	"voice-assistant/backend/domain/agent"
	"voice-assistant/backend/domain/chat"
	"voice-assistant/backend/logic"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
)

type ChatLogic struct {
	logic.BaseLogic
	asrClient *asr.RealTimeASR
}

func NewChatLogic(ctx *gin.Context) *ChatLogic {
	return &ChatLogic{BaseLogic: logic.BaseLogic{Ctx: ctx}}
}

// Talk 通过 wspool 的读通道驱动对话
func (l *ChatLogic) Talk(client *wspool.WSClient) {
	for {
		// 从读通道获取消息（阻塞直到有消息或连接关闭）
		wsMsg := client.ReadMessage()
		if wsMsg == nil {
			log.Printf("[ChatLogic] session=%s 读通道关闭，退出对话", client.SessionId)
			return
		}

		// 解析消息
		var msgData chat.WsMsgType
		if err := json.Unmarshal(wsMsg.Data, &msgData); err != nil {
			log.Printf("[ChatLogic] session=%s 消息解析失败: %v", client.SessionId, err)
			continue
		}

		// 对话类型分流
		// 文字对话
		res := chat.TalkResp{}
		if msgData.Type == chat.MsgTypeUserText.String() {
			res, _ = l.TextTalk(msgData)
		}

		// 语音对话
		if msgData.Type == chat.MsgTypeUserAudio.String() {
			// 初始化实时语音识别
			if l.asrClient == nil {
				asrClient, err := asr.NewRealTimeASR(true)
				if err != nil {
					log.Printf("[ChatLogic] session=%s 创建ASR实例失败: %v", client.SessionId, err)
					continue
				}
				l.asrClient = asrClient
			}

			res, _ = l.SpeechTalk(msgData)
		}

		// 通过写通道回写
		resJSON, _ := json.Marshal(res)
		if !client.Send(resJSON) {
			log.Printf("[ChatLogic] session=%s 写队列满或已关闭，退出对话", client.SessionId)
			return
		}
	}
}

// 处理大模型对话
func (l *ChatLogic) genMessage(req chat.WsMsgType) ([]*schema.Message, error) {
	// todo 根据 sessionId 存储和读取所有历史对话

	// todo 超过10句，或者token超过 1000,则将历史对话总结压缩成1一句话

	// 大模型文档
	// 创建模板，使用 FString 格式
	template := prompt.FromMessages(schema.FString,
		// 系统消息模板
		schema.SystemMessage("你是一个{role}。你需要用{style}的语气回答问题。"),
		// todo 这里实现历史对话内容和历史对话压缩
		// schema.MessagesPlaceholder("chat_history", true),
		// 用户消息模板
		schema.UserMessage("问题: {question}"),
	)

	// 使用模板生成消息
	messages, err := template.Format(context.Background(), map[string]any{
		"role":     "专业的个人助手",
		"style":    "积极、温暖且专业",
		"question": req.Data.Text,
	})

	return messages, err
}

// TextTalkRep 文字对话
func (l *ChatLogic) TextTalk(req chat.WsMsgType) (chat.TalkResp, error) {

	// // 实例化大模型
	// llm, err := llm.NewLLM().NewQwenChatModel(l.Ctx)
	// if err != nil {
	// 	return chat.TalkResp{Text: "大模型初始化失败,请稍后再试"}, err
	// }

	// messages, _ := l.genMessage(req)
	// aiAnswer, err := llm.Generate(l.Ctx, messages)
	// if err != nil {
	// 	return chat.TalkResp{Text: "大模型对话失败,请稍后再试"}, err
	// }

	// 实例化通用Agent
	commoAgent := agent.NewAgent(l.Ctx)

	// 执行带工具的对话
	agentAnswer, err := commoAgent.CommonChat(req.Data.Text)
	if err != nil {
		return chat.TalkResp{Text: "大模型对话失败,请稍后再试"}, err
	}

	res := chat.TalkResp{
		Type:      chat.MsgTypeLLMComplete.String(),
		SessionId: req.SessionId,
		// Text:      aiAnswer.Content,
		Text: agentAnswer,
	}
	return res, nil
}

// SpeechTalk 语音对话
func (l *ChatLogic) SpeechTalk(req chat.WsMsgType) (chat.TalkResp, error) {

	// 校验初始化完成
	if l.asrClient == nil {
		fmt.Println("asrClient 初始化失败")
		return chat.TalkResp{Text: "系统繁忙请稍后再试"}, nil
	}

	// todo 和前端ws连接同步关闭来避免内存溢出
	resultChan, err := l.asrClient.Start("PCM", 16000)
	if err != nil {
		fmt.Println(err)
		return chat.TalkResp{Text: "语音识别启动,请稍后再试"}, err
	}

	// 发送实时语音
	err = l.asrClient.SendAudio(req.Data.Audio)
	if err != nil {
		fmt.Println(err)
		return chat.TalkResp{Text: "语音发送失败,请稍后再试"}, err
	}

	// 开启协程处理识别结果
	resp := chat.TalkResp{}
	go func() {
		for res := range resultChan {
			switch res.Type {
			case "Temp":
				fmt.Printf("\r[临时] %s", res.RawMessage) // 覆盖式打印中间结果
				resp = chat.TalkResp{
					Type:      chat.MsgTypeASRResult.String(),
					SessionId: req.SessionId,
					Text:      res.Text,
				}
			case "Final":
				fmt.Printf("\n[最终] %s\n", res.RawMessage)
				resp = chat.TalkResp{
					Type:      chat.MsgTypeLLMComplete.String(),
					SessionId: req.SessionId,
					Text:      res.Text,
				}
			case "Error":
				fmt.Printf("\n[错误] %s\n", res.RawMessage)
			}
		}
		fmt.Println("识别通道已关闭")
	}()
	return resp, nil
}
