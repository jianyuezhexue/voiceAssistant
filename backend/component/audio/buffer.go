package audio

import (
	"sync"
	"time"
)

// AudioPacket 音频包结构
type AudioPacket struct {
	Sequence   uint32 // 序列号
	Timestamp  int64  // 时间戳 (ms)
	Data       []byte // 音频数据 (PCM)
	SampleRate int    // 采样率
	IsLast     bool   // 是否为最后一帧
}

// NewAudioPacket 创建音频包
func NewAudioPacket(seq uint32, timestamp int64, data []byte, sampleRate int, isLast bool) *AudioPacket {
	return &AudioPacket{
		Sequence:   seq,
		Timestamp:  timestamp,
		Data:       data,
		SampleRate: sampleRate,
		IsLast:     isLast,
	}
}

// AudioBuffer 音频缓冲管理器
type AudioBuffer struct {
	packets map[uint32]*AudioPacket
	baseSeq uint32
	lock    sync.Mutex
	maxSize int
}

// NewAudioBuffer 创建音频缓冲
func NewAudioBuffer() *AudioBuffer {
	return &AudioBuffer{
		packets: make(map[uint32]*AudioPacket),
		baseSeq: 0,
		maxSize: 1000, // 最大缓冲 1000 个包
	}
}

// Add 添加包到缓冲
// 返回值: 是否添加成功 (false 表示包已过期或缓冲已满)
func (b *AudioBuffer) Add(pkt *AudioPacket) bool {
	b.lock.Lock()
	defer b.lock.Unlock()

	// 检查是否过期 (序列号小于基础序列号)
	if pkt.Sequence < b.baseSeq {
		return false
	}

	// 检查是否已存在
	if _, exists := b.packets[pkt.Sequence]; exists {
		return false
	}

	// 缓冲已满，移除最老的包
	if len(b.packets) >= b.maxSize {
		b.removeOldestPacket()
	}

	b.packets[pkt.Sequence] = pkt
	return true
}

// removeOldestPacket 移除最老的包
func (b *AudioBuffer) removeOldestPacket() {
	if len(b.packets) == 0 {
		return
	}

	var oldestSeq uint32
	var oldestTime int64 = -1

	for seq, pkt := range b.packets {
		if oldestTime == -1 || pkt.Timestamp < oldestTime {
			oldestSeq = seq
			oldestTime = pkt.Timestamp
		}
	}

	delete(b.packets, oldestSeq)
}

// Flush 按序列号顺序刷新缓冲
// fn: 回调函数，接收按顺序排列的音频包
func (b *AudioBuffer) Flush(fn func(*AudioPacket)) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if len(b.packets) == 0 {
		return
	}

	// 按序列号排序
	seqs := make([]uint32, 0, len(b.packets))
	for seq := range b.packets {
		seqs = append(seqs, seq)
	}

	// 排序
	for i := 0; i < len(seqs)-1; i++ {
		for j := i + 1; j < len(seqs); j++ {
			if seqs[i] > seqs[j] {
				seqs[i], seqs[j] = seqs[j], seqs[i]
			}
		}
	}

	// 更新基础序列号并调用回调
	newBaseSeq := b.baseSeq
	for _, seq := range seqs {
		pkt := b.packets[seq]

		// 跳过过期包
		if seq < b.baseSeq {
			delete(b.packets, seq)
			continue
		}

		fn(pkt)
		newBaseSeq = seq + 1
		delete(b.packets, seq)
	}

	b.baseSeq = newBaseSeq
}

// NextSequence 获取下一个序列号
func (b *AudioBuffer) NextSequence() uint32 {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.baseSeq + uint32(len(b.packets))
}

// GetBaseSequence 获取基础序列号
func (b *AudioBuffer) GetBaseSequence() uint32 {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.baseSeq
}

// GetPacketCount 获取当前缓冲的包数量
func (b *AudioBuffer) GetPacketCount() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return len(b.packets)
}

// Clear 清空缓冲
func (b *AudioBuffer) Clear() {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.packets = make(map[uint32]*AudioPacket)
}

// TTSBuffer TTS 低延迟缓冲
type TTSBuffer struct {
	*AudioBuffer
	latePacketWindow time.Duration // 延迟包容忍窗口 (ms)
}

// NewTTSBuffer 创建 TTS 低延迟缓冲
func NewTTSBuffer(latePacketWindowMs int) *TTSBuffer {
	return &TTSBuffer{
		AudioBuffer:      NewAudioBuffer(),
		latePacketWindow: time.Duration(latePacketWindowMs) * time.Millisecond,
	}
}

// AddWithTimeout 添加包，带超时检查
// now: 当前时间戳 (ms)
// 返回值: 是否添加成功
func (b *TTSBuffer) AddWithTimeout(pkt *AudioPacket, now int64) bool {
	// 检查是否在容忍窗口内的过期包
	if pkt.Sequence < b.GetBaseSequence() {
		// 包已过期，检查是否在容忍窗口内
		latency := now - pkt.Timestamp
		if latency > b.latePacketWindow.Milliseconds() {
			return false // 超时丢弃
		}
	}

	return b.Add(pkt)
}

// SetLatePacketWindow 设置延迟包容忍窗口
func (b *TTSBuffer) SetLatePacketWindow(windowMs int) {
	b.latePacketWindow = time.Duration(windowMs) * time.Millisecond
}

// GetLatePacketWindow 获取延迟包容忍窗口
func (b *TTSBuffer) GetLatePacketWindow() time.Duration {
	return b.latePacketWindow
}

// AudioBufferManager 多流缓冲管理器
type AudioBufferManager struct {
	buffers map[string]*AudioBuffer
	lock    sync.RWMutex
}

// NewAudioBufferManager 创建多流缓冲管理器
func NewAudioBufferManager() *AudioBufferManager {
	return &AudioBufferManager{
		buffers: make(map[string]*AudioBuffer),
	}
}

// GetOrCreate 获取或创建流缓冲
func (m *AudioBufferManager) GetOrCreate(streamID string) *AudioBuffer {
	m.lock.Lock()
	defer m.lock.Unlock()

	if buf, exists := m.buffers[streamID]; exists {
		return buf
	}

	buf := NewAudioBuffer()
	m.buffers[streamID] = buf
	return buf
}

// Get 获取流缓冲
func (m *AudioBufferManager) Get(streamID string) (*AudioBuffer, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	buf, exists := m.buffers[streamID]
	return buf, exists
}

// Delete 删除流缓冲
func (m *AudioBufferManager) Delete(streamID string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.buffers, streamID)
}

// ClearAll 清空所有缓冲
func (m *AudioBufferManager) ClearAll() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, buf := range m.buffers {
		buf.Clear()
	}
}

// GetStreamCount 获取流数量
func (m *AudioBufferManager) GetStreamCount() int {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return len(m.buffers)
}
