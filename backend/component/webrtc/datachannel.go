package webrtc

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

// DataChannelSubType DataChannel子流类型
type DataChannelSubType string

const (
	SubTypeASRAudio DataChannelSubType = "asr_audio" // ASR音频子流
	SubTypeTTSAudio DataChannelSubType = "tts_audio" // TTS音频子流
)

// DataChannelMessageType DataChannel消息类型
type DataChannelMessageType string

const (
	MsgTypeAudio   DataChannelMessageType = "audio"   // 音频消息
	MsgTypeText    DataChannelMessageType = "text"    // 文本消息
	MsgTypeControl DataChannelMessageType = "control" // 控制命令
)

// ControlCommand 控制命令
type ControlCommand struct {
	Command string                 `json:"command"` // 命令类型: start, stop, interrupt
	Params  map[string]interface{} `json:"params,omitempty"`
}

// DataChannelConfig DataChannel配置
type DataChannelConfig struct {
	Ordered        bool // 是否有序传输 (UDP模式为false)
	MaxRetransmits int  // 最大重传次数 (UDP模式为0)
}

// DefaultDataChannelConfig 默认配置 (UDP模式)
var DefaultDataChannelConfig = DataChannelConfig{
	Ordered:        false,
	MaxRetransmits: 0,
}

// OnMessageCallback 消息回调函数
type OnMessageCallback func(msgType DataChannelMessageType, data interface{})

// OnAudioCallback 音频数据回调
type OnAudioCallback func(subType DataChannelSubType, audioData []byte)

// OnTextCallback 文本消息回调
type OnTextCallback func(text string)

// OnControlCallback 控制命令回调
type OnControlCallback func(cmd *ControlCommand)

// IDataChannel DataChannel接口
type IDataChannel interface {
	// SendAudio 发送音频数据
	SendAudio(subType DataChannelSubType, audioData []byte) error
	// SendText 发送文本消息
	SendText(text string) error
	// SendControl 发送控制命令
	SendControl(cmd *ControlCommand) error
	// Close 关闭DataChannel
	Close() error
}

// DataChannelHandler DataChannel事件处理器
type DataChannelHandler struct {
	pc      *webrtc.PeerConnection
	channel *webrtc.DataChannel
	config  DataChannelConfig
	mu      sync.RWMutex

	onMessageCallback OnMessageCallback
	onAudioCallback   OnAudioCallback
	onTextCallback    OnTextCallback
	onControlCallback OnControlCallback

	// 子流追踪
	activeSubStreams map[DataChannelSubType]bool
	subStreamLock    sync.RWMutex
}

// NewDataChannelHandler 创建DataChannel处理器
func NewDataChannelHandler(config DataChannelConfig) *DataChannelHandler {
	return &DataChannelHandler{
		config:           config,
		activeSubStreams: make(map[DataChannelSubType]bool),
	}
}

// NewPeerConnection 创建PeerConnection
// ICEServers: 使用 Google STUN 服务器
func NewPeerConnection(iceServers ...string) (*webrtc.PeerConnection, error) {
	if len(iceServers) == 0 {
		iceServers = []string{"stun:stun.l.google.com:19302"}
	}

	var cfg webrtc.Configuration
	if len(iceServers) == 1 {
		cfg = webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{URLs: []string{iceServers[0]}},
			},
		}
	} else {
		ices := make([]webrtc.ICEServer, len(iceServers))
		for i, s := range iceServers {
			ices[i] = webrtc.ICEServer{URLs: []string{s}}
		}
		cfg = webrtc.Configuration{ICEServers: ices}
	}

	pc, err := webrtc.NewPeerConnection(cfg)
	if err != nil {
		log.Printf("[WebRTC] NewPeerConnection error: %v", err)
		return nil, err
	}

	log.Printf("[WebRTC] PeerConnection created with ICE servers: %v", iceServers)
	return pc, nil
}

// CreateDataChannel 创建DataChannel
func (h *DataChannelHandler) CreateDataChannel(pc *webrtc.PeerConnection, label string) (*webrtc.DataChannel, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.pc = pc

	// 配置DataChannel参数
	options := &webrtc.DataChannelInit{
		Ordered:        &h.config.Ordered,
		MaxRetransmits: func() *uint16 { v := uint16(h.config.MaxRetransmits); return &v }(),
	}

	channel, err := pc.CreateDataChannel(label, options)
	if err != nil {
		log.Printf("[WebRTC] CreateDataChannel error: %v", err)
		return nil, err
	}

	h.channel = channel
	h.setupChannelHandlers()

	log.Printf("[WebRTC] DataChannel '%s' created (Ordered=%v, MaxRetransmits=%d)",
		label, h.config.Ordered, h.config.MaxRetransmits)
	return channel, nil
}

// setupChannelHandlers 设置DataChannel事件处理器
func (h *DataChannelHandler) setupChannelHandlers() {
	h.channel.OnOpen(func() {
		log.Printf("[WebRTC] DataChannel opened")
		h.subStreamLock.Lock()
		for sub := range h.activeSubStreams {
			log.Printf("[WebRTC] Active sub-stream: %s", sub)
		}
		h.subStreamLock.Unlock()
	})

	h.channel.OnClose(func() {
		log.Printf("[WebRTC] DataChannel closed")
	})

	h.channel.OnMessage(func(msg webrtc.DataChannelMessage) {
		h.handleMessage(msg)
	})
}

// handleMessage 处理DataChannel消息
func (h *DataChannelHandler) handleMessage(msg webrtc.DataChannelMessage) {
	// 判断消息类型
	if msg.IsString {
		// 字符串消息 - 控制命令或文本
		h.handleStringMessage(msg.Data)
	} else {
		// 二进制消息 - 音频数据
		h.handleBinaryMessage(msg.Data)
	}
}

// handleStringMessage 处理字符串消息
func (h *DataChannelHandler) handleStringMessage(data []byte) {
	var msg struct {
		Type    string                 `json:"type"`
		SubType string                 `json:"sub_type,omitempty"`
		Text    string                 `json:"text,omitempty"`
		Command string                 `json:"command,omitempty"`
		Params  map[string]interface{} `json:"params,omitempty"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("[WebRTC] UnMarshal message error: %v, data: %s", err, string(data))
		return
	}

	switch DataChannelMessageType(msg.Type) {
	case MsgTypeText:
		if h.onTextCallback != nil {
			h.onTextCallback(msg.Text)
		}
		if h.onMessageCallback != nil {
			h.onMessageCallback(MsgTypeText, msg.Text)
		}

	case MsgTypeControl:
		cmd := &ControlCommand{
			Command: msg.Command,
			Params:  msg.Params,
		}
		log.Printf("[WebRTC] Control command: %s, params: %v", cmd.Command, cmd.Params)
		if h.onControlCallback != nil {
			h.onControlCallback(cmd)
		}
		if h.onMessageCallback != nil {
			h.onMessageCallback(MsgTypeControl, cmd)
		}

	default:
		log.Printf("[WebRTC] Unknown message type: %s", msg.Type)
	}
}

// handleBinaryMessage 处理二进制消息 (音频数据)
func (h *DataChannelHandler) handleBinaryMessage(data []byte) {
	// 检查是否包含子流标识头
	if len(data) < 4 {
		log.Printf("[WebRTC] Binary message too short: %d bytes", len(data))
		return
	}

	// 解析子流类型 (前4字节作为子流标识)
	// 格式: [1字节子流类型][3字节长度][音频数据]
	subTypeByte := data[0]
	var subType DataChannelSubType
	switch subTypeByte {
	case 0x01:
		subType = SubTypeASRAudio
	case 0x02:
		subType = SubTypeTTSAudio
	default:
		subType = SubTypeASRAudio // 默认作为ASR音频
	}

	audioData := data[1:]

	h.subStreamLock.RLock()
	active := h.activeSubStreams[subType]
	h.subStreamLock.RUnlock()

	if !active {
		log.Printf("[WebRTC] Sub-stream %s not active, ignoring audio", subType)
		return
	}

	if h.onAudioCallback != nil {
		h.onAudioCallback(subType, audioData)
	}
	if h.onMessageCallback != nil {
		h.onMessageCallback(MsgTypeAudio, audioData)
	}
}

// SendAudio 发送音频数据
// subType: 子流类型 (asr_audio / tts_audio)
// audioData: 音频数据 (PCM格式)
func (h *DataChannelHandler) SendAudio(subType DataChannelSubType, audioData []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.channel == nil || h.channel.ReadyState() != webrtc.DataChannelStateOpen {
		return errors.New("data channel not open")
	}

	// 添加子流标识头
	// 格式: [1字节子流类型][音频数据]
	var subTypeByte byte
	switch subType {
	case SubTypeASRAudio:
		subTypeByte = 0x01
	case SubTypeTTSAudio:
		subTypeByte = 0x02
	default:
		subTypeByte = 0x01
	}

	// 构建消息: [子流类型][音频数据]
	message := make([]byte, 1+len(audioData))
	message[0] = subTypeByte
	copy(message[1:], audioData)

	if err := h.channel.Send(message); err != nil {
		log.Printf("[WebRTC] SendAudio error: %v", err)
		return err
	}

	return nil
}

// SendAudioToStream 发送音频到指定子流
func (h *DataChannelHandler) SendAudioToStream(streamID string, audioData []byte) error {
	// streamID 用于区分不同的音频流，这里简单处理为子流类型
	subType := DataChannelSubType(streamID)
	return h.SendAudio(subType, audioData)
}

// SendText 发送文本消息
func (h *DataChannelHandler) SendText(text string) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.channel == nil || h.channel.ReadyState() != webrtc.DataChannelStateOpen {
		return errors.New("data channel not open")
	}

	msg := map[string]interface{}{
		"type": MsgTypeText,
		"text": text,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err := h.channel.SendText(string(data)); err != nil {
		log.Printf("[WebRTC] SendText error: %v", err)
		return err
	}

	return nil
}

// SendControl 发送控制命令
func (h *DataChannelHandler) SendControl(cmd *ControlCommand) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.channel == nil || h.channel.ReadyState() != webrtc.DataChannelStateOpen {
		return errors.New("data channel not open")
	}

	msg := map[string]interface{}{
		"type":    MsgTypeControl,
		"command": cmd.Command,
		"params":  cmd.Params,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if err := h.channel.SendText(string(data)); err != nil {
		log.Printf("[WebRTC] SendControl error: %v", err)
		return err
	}

	log.Printf("[WebRTC] Sent control command: %s", cmd.Command)
	return nil
}

// SendTTSAudio 发送TTS音频数据 (便捷方法)
func (h *DataChannelHandler) SendTTSAudio(audioData []byte) error {
	return h.SendAudio(SubTypeTTSAudio, audioData)
}

// Close 关闭DataChannel
func (h *DataChannelHandler) Close() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.channel != nil {
		if err := h.channel.Close(); err != nil {
			log.Printf("[WebRTC] DataChannel close error: %v", err)
			return err
		}
		h.channel = nil
	}

	if h.pc != nil {
		if err := h.pc.Close(); err != nil {
			log.Printf("[WebRTC] PeerConnection close error: %v", err)
			return err
		}
		h.pc = nil
	}

	log.Printf("[WebRTC] DataChannelHandler closed")
	return nil
}

// SetOnMessage 设置消息回调
func (h *DataChannelHandler) SetOnMessage(callback OnMessageCallback) {
	h.onMessageCallback = callback
}

// SetOnAudio 设置音频数据回调
func (h *DataChannelHandler) SetOnAudio(callback OnAudioCallback) {
	h.onAudioCallback = callback
}

// SetOnText 设置文本消息回调
func (h *DataChannelHandler) SetOnText(callback OnTextCallback) {
	h.onTextCallback = callback
}

// SetOnControl 设置控制命令回调
func (h *DataChannelHandler) SetOnControl(callback OnControlCallback) {
	h.onControlCallback = callback
}

// ActivateSubStream 激活子流
func (h *DataChannelHandler) ActivateSubStream(subType DataChannelSubType) {
	h.subStreamLock.Lock()
	defer h.subStreamLock.Unlock()
	h.activeSubStreams[subType] = true
	log.Printf("[WebRTC] Activated sub-stream: %s", subType)
}

// DeactivateSubStream 停用子流
func (h *DataChannelHandler) DeactivateSubStream(subType DataChannelSubType) {
	h.subStreamLock.Lock()
	defer h.subStreamLock.Unlock()
	h.activeSubStreams[subType] = false
	log.Printf("[WebRTC] Deactivated sub-stream: %s", subType)
}

// IsSubStreamActive 检查子流是否激活
func (h *DataChannelHandler) IsSubStreamActive(subType DataChannelSubType) bool {
	h.subStreamLock.RLock()
	defer h.subStreamLock.RUnlock()
	return h.activeSubStreams[subType]
}

// GetDataChannel 获取底层DataChannel
func (h *DataChannelHandler) GetDataChannel() *webrtc.DataChannel {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.channel
}

// GetPeerConnection 获取底层PeerConnection
func (h *DataChannelHandler) GetPeerConnection() *webrtc.PeerConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.pc
}

// CreateOffer 创建Offer (用于作为发起方)
func (h *DataChannelHandler) CreateOffer() (*webrtc.SessionDescription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.pc == nil {
		return nil, errors.New("peer connection not initialized")
	}

	offer, err := h.pc.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	if err := h.pc.SetLocalDescription(offer); err != nil {
		return nil, err
	}

	return &offer, nil
}

// SetRemoteDescription 设置远端描述 (用于作为接收方)
func (h *DataChannelHandler) SetRemoteDescription(sdp *webrtc.SessionDescription) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.pc == nil {
		return errors.New("peer connection not initialized")
	}

	return h.pc.SetRemoteDescription(*sdp)
}

// CreateAnswer 创建Answer (用于作为接收方)
func (h *DataChannelHandler) CreateAnswer() (*webrtc.SessionDescription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.pc == nil {
		return nil, errors.New("peer connection not initialized")
	}

	answer, err := h.pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	if err := h.pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

// AddIceCandidate 添加ICE候选
func (h *DataChannelHandler) AddIceCandidate(candidate *webrtc.ICECandidate) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.pc == nil {
		return errors.New("peer connection not initialized")
	}

	return h.pc.AddICECandidate(candidate.ToJSON())
}

// ReadFromDataChannel 从DataChannel读取数据直到关闭或错误
func (h *DataChannelHandler) ReadFromDataChannel(reader io.Reader) error {
	// 注意: pion的DataChannel不直接支持io.Reader
	// 这个方法用于演示如何处理流式数据
	// 实际使用中建议使用OnMessage回调
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if n > 0 {
			if err := h.channel.Send(buf[:n]); err != nil {
				return err
			}
		}
	}
}
