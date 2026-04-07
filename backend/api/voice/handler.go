package voice

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"voice-assistant/backend/domain/voice"
	voicelogic "voice-assistant/backend/logic/voice"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有来源，生产环境应限制
	},
}

// VoiceHandler 语音对话 WebSocket Handler
type VoiceHandler struct {
	dialogueLogic    *voicelogic.VoiceDialogueLogic
	interruptHandler *voicelogic.InterruptHandler

	// 连接管理
	connections map[string]*websocket.Conn
	connMu      sync.RWMutex

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewVoiceHandler 创建语音 Handler
func NewVoiceHandler(dialogueLogic *voicelogic.VoiceDialogueLogic, interruptHandler *voicelogic.InterruptHandler) *VoiceHandler {
	ctx, cancel := context.WithCancel(context.Background())

	handler := &VoiceHandler{
		dialogueLogic:    dialogueLogic,
		interruptHandler: interruptHandler,
		connections:      make(map[string]*websocket.Conn),
		ctx:              ctx,
		cancel:           cancel,
	}

	// 设置回调
	dialogueLogic.SetCallbacks(&voicelogic.VoiceDialogueCallbacks{
		OnStateChange: handler.onStateChange,
		OnASRResult:   handler.onASRResult,
		OnLLMResponse: handler.onLLMResponse,
		OnTTSAudio:    handler.onTTSAudio,
		OnError:       handler.onError,
	})

	return handler
}

// HandleWS 处理 WebSocket 连接
func (h *VoiceHandler) HandleWS(c *gin.Context) {
	// WebSocket 握手升级
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[VoiceHandler] WebSocket upgrade error: %v", err)
		return
	}

	// 获取用户ID
	userID := c.GetString("currUserId")
	if userID == "" {
		userID = "anonymous"
	}

	// 创建会话
	session, err := h.dialogueLogic.CreateSession(userID)
	if err != nil {
		log.Printf("[VoiceHandler] Create session error: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","data":{"code":1001,"message":"创建会话失败"}}`))
		conn.Close()
		return
	}

	sessionID := session.ID

	// 注册连接
	h.registerConn(sessionID, conn)

	// 启动连接处理
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.handleConnection(sessionID, conn)
	}()

	// 发送连接成功消息
	h.sendMessage(conn, sessionID, voice.MsgStateUpdate, voice.NewStateUpdateData(sessionID, session.State))

	log.Printf("[VoiceHandler] WebSocket connected: session=%s, user=%s", sessionID, userID)
}

// handleConnection 处理连接
func (h *VoiceHandler) handleConnection(sessionID string, conn *websocket.Conn) {
	defer func() {
		h.unregisterConn(sessionID)
		h.dialogueLogic.DeleteSession(sessionID)
		conn.Close()
		log.Printf("[VoiceHandler] WebSocket disconnected: session=%s", sessionID)
	}()

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动心跳
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.heartbeat(sessionID, conn)
	}()

	// 消息处理循环
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[VoiceHandler] Read error: %v", err)
			}
			break
		}

		switch messageType {
		case websocket.TextMessage:
			h.handleTextMessage(sessionID, conn, message)
		case websocket.BinaryMessage:
			h.handleBinaryMessage(sessionID, conn, message)
		}
	}
}

// handleTextMessage 处理文本消息
func (h *VoiceHandler) handleTextMessage(sessionID string, conn *websocket.Conn, message []byte) {
	var msg voice.WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("[VoiceHandler] Unmarshal error: %v", err)
		h.sendError(conn, sessionID, voice.ErrCodeInternalError, "消息格式错误")
		return
	}

	msg.SessionID = sessionID

	log.Printf("[VoiceHandler] Received message: type=%s, session=%s", msg.Type, sessionID)

	switch msg.Type {
	case voice.MsgStart:
		h.onStart(sessionID, conn, msg)

	case voice.MsgStop:
		h.onStop(sessionID, conn, msg)

	case voice.MsgInterrupt:
		h.onInterrupt(sessionID, conn, msg)

	case voice.MsgPing:
		h.onPing(sessionID, conn, msg)

	case voice.MsgASRResult:
		// 前端ASR结果（某些实现可能在前端做ASR）
		h.onASRResultMessage(sessionID, conn, msg)

	default:
		log.Printf("[VoiceHandler] Unknown message type: %s", msg.Type)
	}
}

// handleBinaryMessage 处理二进制消息（备用）
func (h *VoiceHandler) handleBinaryMessage(sessionID string, conn *websocket.Conn, message []byte) {
	log.Printf("[VoiceHandler] Received binary message: %d bytes", len(message))
}

// onStart 处理开始消息
func (h *VoiceHandler) onStart(sessionID string, conn *websocket.Conn, msg voice.WSMessage) {
	session, err := h.dialogueLogic.GetSession(sessionID)
	if err != nil {
		h.sendError(conn, sessionID, voice.ErrCodeSessionNotFound, "会话不存在")
		return
	}

	// 状态切换到 LISTENING
	session.UpdateState(voice.StateListening)
	h.sendStateUpdate(conn, sessionID, voice.StateListening)

	log.Printf("[VoiceHandler] Session started: %s", sessionID)
}

// onStop 处理停止消息
func (h *VoiceHandler) onStop(sessionID string, conn *websocket.Conn, msg voice.WSMessage) {
	session, err := h.dialogueLogic.GetSession(sessionID)
	if err != nil {
		return
	}

	// 处理打断
	h.interruptHandler.Handle(h.ctx, sessionID, voice.InterruptUserClick)

	// 状态切换到 IDLE
	session.UpdateState(voice.StateIdle)
	h.sendStateUpdate(conn, sessionID, voice.StateIdle)

	log.Printf("[VoiceHandler] Session stopped: %s", sessionID)
}

// onInterrupt 处理打断消息
func (h *VoiceHandler) onInterrupt(sessionID string, conn *websocket.Conn, msg voice.WSMessage) {
	// 解析打断数据
	var interruptData voice.InterruptData
	if msg.Data != nil {
		data, ok := msg.Data.(map[string]interface{})
		if ok {
			sourceStr, ok := data["source"].(string)
			if ok {
				switch sourceStr {
				case "user_speech":
					interruptData.Source = voice.InterruptUserSpeech
				case "user_click":
					interruptData.Source = voice.InterruptUserClick
				case "server_cmd":
					interruptData.Source = voice.InterruptServerCmd
				default:
					interruptData.Source = voice.InterruptUserSpeech
				}
			}
		}
	}

	if msg.Data == nil {
		interruptData.Source = voice.InterruptUserSpeech
	}

	interruptData.SessionID = sessionID
	interruptData.SourceStr = interruptData.Source.String()

	// 处理打断
	if err := h.interruptHandler.Handle(h.ctx, sessionID, interruptData.Source); err != nil {
		log.Printf("[VoiceHandler] Interrupt error: %v", err)
		h.sendError(conn, sessionID, voice.ErrCodeInternalError, "打断处理失败")
		return
	}

	// 发送打断确认
	h.sendMessage(conn, sessionID, voice.MsgInterrupt, &interruptData)
}

// onPing 处理心跳
func (h *VoiceHandler) onPing(sessionID string, conn *websocket.Conn, msg voice.WSMessage) {
	h.sendMessage(conn, sessionID, voice.MsgPong, nil)
}

// onASRResultMessage 处理ASR结果消息
func (h *VoiceHandler) onASRResultMessage(sessionID string, conn *websocket.Conn, msg voice.WSMessage) {
	// 解析 ASR 结果
	if msg.Data == nil {
		return
	}

	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		return
	}

	text, ok := data["text"].(string)
	if !ok {
		return
	}

	isFinal, _ := data["is_final"].(bool)

	_ = voice.NewASRResult(text, isFinal, 1.0)

	// 通知 ASR 结果
	if h.dialogueLogic != nil {
		session, err := h.dialogueLogic.GetSession(sessionID)
		if err == nil {
			session.RecognizedText = text
		}
	}
}

// heartbeat 心跳处理
func (h *VoiceHandler) heartbeat(sessionID string, conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			return
		case <-time.After(30 * time.Second):
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[VoiceHandler] Ping error: %v", err)
				return
			}
		}
	}
}

// registerConn 注册连接
func (h *VoiceHandler) registerConn(sessionID string, conn *websocket.Conn) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	h.connections[sessionID] = conn
}

// unregisterConn 注销连接
func (h *VoiceHandler) unregisterConn(sessionID string) {
	h.connMu.Lock()
	defer h.connMu.Unlock()
	delete(h.connections, sessionID)
}

// getConn 获取连接
func (h *VoiceHandler) getConn(sessionID string) (*websocket.Conn, bool) {
	h.connMu.RLock()
	defer h.connMu.RUnlock()
	conn, ok := h.connections[sessionID]
	return conn, ok
}

// sendMessage 发送消息
func (h *VoiceHandler) sendMessage(conn *websocket.Conn, sessionID string, msgType voice.MessageType, data interface{}) {
	msg := voice.NewWSMessage(msgType, sessionID, data)
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[VoiceHandler] Marshal error: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		log.Printf("[VoiceHandler] Write error: %v", err)
	}
}

// sendStateUpdate 发送状态更新
func (h *VoiceHandler) sendStateUpdate(conn *websocket.Conn, sessionID string, state voice.VoiceState) {
	h.sendMessage(conn, sessionID, voice.MsgStateUpdate, voice.NewStateUpdateData(sessionID, state))
}

// sendError 发送错误
func (h *VoiceHandler) sendError(conn *websocket.Conn, sessionID string, code int, message string) {
	h.sendMessage(conn, sessionID, voice.MsgError, voice.NewErrorResponse(code, message))
}

// ==================== 回调函数实现 ====================

// onStateChange 状态变化回调
func (h *VoiceHandler) onStateChange(sessionID string, state voice.VoiceState) {
	conn, ok := h.getConn(sessionID)
	if !ok {
		return
	}

	session, err := h.dialogueLogic.GetSession(sessionID)
	if err != nil {
		return
	}

	h.sendStateUpdate(conn, sessionID, state)
	session.UpdateState(state)

	log.Printf("[VoiceHandler] State changed: session=%s, state=%s", sessionID, state.String())
}

// onASRResult ASR结果回调
func (h *VoiceHandler) onASRResult(sessionID string, result *voice.ASRResult) {
	conn, ok := h.getConn(sessionID)
	if !ok {
		return
	}

	session, err := h.dialogueLogic.GetSession(sessionID)
	if err != nil {
		return
	}

	msgType := voice.MsgASRResult
	if result.IsFinal {
		msgType = voice.MsgASRComplete
		session.RecognizedText = result.Text
	}

	h.sendMessage(conn, sessionID, msgType, result)

	log.Printf("[VoiceHandler] ASR result: session=%s, text=%s, isFinal=%v", sessionID, result.Text, result.IsFinal)
}

// onLLMResponse LLM响应回调
func (h *VoiceHandler) onLLMResponse(sessionID string, resp *voice.LLMResponse) {
	conn, ok := h.getConn(sessionID)
	if !ok {
		return
	}

	msgType := voice.MsgLLMText
	if resp.IsComplete {
		msgType = voice.MsgLLMComplete
	}

	h.sendMessage(conn, sessionID, msgType, resp)

	log.Printf("[VoiceHandler] LLM response: session=%s, isChunk=%v, isComplete=%v", sessionID, resp.IsChunk, resp.IsComplete)
}

// onTTSAudio TTS音频回调
func (h *VoiceHandler) onTTSAudio(sessionID string, audio *voice.TTSAudio) {
	conn, ok := h.getConn(sessionID)
	if !ok {
		return
	}

	// TTS音频通过DataChannel发送，这里只发送完成通知
	if audio.IsLast {
		h.sendMessage(conn, sessionID, voice.MsgTTSComplete, nil)
		log.Printf("[VoiceHandler] TTS complete: session=%s", sessionID)
	}
}

// onError 错误回调
func (h *VoiceHandler) onError(sessionID string, err error) {
	conn, ok := h.getConn(sessionID)
	if !ok {
		return
	}

	code := voice.ErrCodeInternalError
	if errors.Is(err, context.Canceled) {
		return // 用户取消不算错误
	}

	h.sendError(conn, sessionID, code, err.Error())

	log.Printf("[VoiceHandler] Error: session=%s, error=%v", sessionID, err)
}

// Close 关闭 Handler
func (h *VoiceHandler) Close() error {
	h.cancel()

	h.connMu.Lock()
	defer h.connMu.Unlock()

	for sessionID, conn := range h.connections {
		conn.Close()
		delete(h.connections, sessionID)
	}

	h.wg.Wait()
	return nil
}
