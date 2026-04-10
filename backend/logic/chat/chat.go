package logic

import (
	"encoding/json"
	"log"
	"voice-assistant/backend/domain/chat"
	"voice-assistant/backend/logic"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type ChatLogic struct {
	logic.BaseLogic
}

func NewChatLogic(ctx *gin.Context) *ChatLogic {
	return &ChatLogic{BaseLogic: logic.BaseLogic{Ctx: ctx}}
}

// 综合对话
// todo 这里改成 ws池
func (l *ChatLogic) Talk(conn *websocket.Conn) {

	for {
		// 读取消息
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			log.Println("读取失败:", err)
			break
		}

		// 解析WsMsgType
		var msgData chat.WsMsgType
		err = json.Unmarshal(msg, &msgData)

		// 对话类型分流
		res := chat.TalkResp{}
		if msgData.Type == chat.MsgTypeUserText.String() {
			res, _ = l.TextTalk(msgData)
		} else if msgData.Type == chat.MsgTypeUserAudio.String() {
			res, _ = l.SpeechTalk(msgData)
		}

		// 回写（echo）
		resJSON, _ := json.Marshal(res)
		err = conn.WriteMessage(msgType, resJSON)
		if err != nil {
			log.Println("发送失败:", err)
			break
		}
	}
}

// TextTalkRep 文字对话
func (l *ChatLogic) TextTalk(req chat.WsMsgType) (chat.TalkResp, error) {
	res := chat.TalkResp{
		Type:      chat.MsgTypeLLMComplete.String(),
		SessionID: req.SessionID,
		Text:      "测试文字对话",
	}
	return res, nil
}

// SpeechTalk 语音对话
func (l *ChatLogic) SpeechTalk(req chat.WsMsgType) (chat.TalkResp, error) {
	res := chat.TalkResp{
		Type:      chat.MsgTypeLLMComplete.String(),
		SessionID: req.SessionID,
		Text:      "测试语音对话",
	}
	return res, nil
}
