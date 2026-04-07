package audio

import (
	"sync"
	"testing"
	// "time"
)

// TestAudioBuffer_Add 测试添加包
func TestAudioBuffer_Add(t *testing.T) {
	buf := NewAudioBuffer()

	pkt := NewAudioPacket(1, 1000, []byte{1, 2, 3}, 16000, false)

	if !buf.Add(pkt) {
		t.Error("Add should succeed")
	}

	if buf.GetPacketCount() != 1 {
		t.Errorf("Packet count: got %d, want 1", buf.GetPacketCount())
	}
}

// TestAudioBuffer_AddOutOfOrder 测试乱序包
func TestAudioBuffer_AddOutOfOrder(t *testing.T) {
	buf := NewAudioBuffer()

	// 添加乱序包
	pkt3 := NewAudioPacket(3, 3000, []byte{3}, 16000, false)
	pkt1 := NewAudioPacket(1, 1000, []byte{1}, 16000, false)
	pkt2 := NewAudioPacket(2, 2000, []byte{2}, 16000, false)

	// 按乱序添加
	buf.Add(pkt3)
	buf.Add(pkt1)
	buf.Add(pkt2)

	if buf.GetPacketCount() != 3 {
		t.Errorf("Packet count after out-of-order add: got %d, want 3", buf.GetPacketCount())
	}
}

// TestAudioBuffer_Flush 测试按序刷新
func TestAudioBuffer_Flush(t *testing.T) {
	buf := NewAudioBuffer()

	// 添加顺序包
	pkt1 := NewAudioPacket(1, 1000, []byte{1}, 16000, false)
	pkt2 := NewAudioPacket(2, 2000, []byte{2}, 16000, false)
	pkt3 := NewAudioPacket(3, 3000, []byte{3}, 16000, false)

	buf.Add(pkt1)
	buf.Add(pkt2)
	buf.Add(pkt3)

	var flushedSeqs []uint32
	buf.Flush(func(pkt *AudioPacket) {
		flushedSeqs = append(flushedSeqs, pkt.Sequence)
	})

	if len(flushedSeqs) != 3 {
		t.Errorf("Flushed count: got %d, want 3", len(flushedSeqs))
	}

	// 验证顺序
	for i, seq := range flushedSeqs {
		if seq != uint32(i+1) {
			t.Errorf("Sequence order mismatch at index %d: got %d, want %d", i, seq, i+1)
		}
	}
}

// TestAudioBuffer_ExpiredPacket 测试过期包丢弃
func TestAudioBuffer_ExpiredPacket(t *testing.T) {
	buf := NewAudioBuffer()

	// 添加初始包设置基础序列号
	pkt1 := NewAudioPacket(1, 1000, []byte{1}, 16000, false)
	buf.Add(pkt1)

	// 更新基础序列号
	buf.Flush(func(pkt *AudioPacket) {})

	// 尝试添加过期包 (序列号小于基础序列号)
	oldPkt := NewAudioPacket(0, 500, []byte{0}, 16000, false)
	if buf.Add(oldPkt) {
		t.Error("Expired packet should be rejected")
	}
}

// TestAudioBuffer_NextSequence 测试序列号递增
func TestAudioBuffer_NextSequence(t *testing.T) {
	buf := NewAudioBuffer()

	nextSeq := buf.NextSequence()
	if nextSeq != 0 {
		t.Errorf("Initial NextSequence: got %d, want 0", nextSeq)
	}

	buf.Add(NewAudioPacket(0, 1000, []byte{1}, 16000, false))
	nextSeq = buf.NextSequence()
	if nextSeq != 1 {
		t.Errorf("NextSequence after add: got %d, want 1", nextSeq)
	}
}

// TestAudioBuffer_GetBaseSequence 测试获取基础序列号
func TestAudioBuffer_GetBaseSequence(t *testing.T) {
	buf := NewAudioBuffer()

	if buf.GetBaseSequence() != 0 {
		t.Errorf("Initial GetBaseSequence: got %d, want 0", buf.GetBaseSequence())
	}

	buf.Add(NewAudioPacket(5, 1000, []byte{1}, 16000, false))
	buf.Flush(func(pkt *AudioPacket) {})

	if buf.GetBaseSequence() != 6 {
		t.Errorf("GetBaseSequence after flush: got %d, want 6", buf.GetBaseSequence())
	}
}

// TestAudioBuffer_Clear 测试清空缓冲
func TestAudioBuffer_Clear(t *testing.T) {
	buf := NewAudioBuffer()

	buf.Add(NewAudioPacket(1, 1000, []byte{1}, 16000, false))
	buf.Add(NewAudioPacket(2, 2000, []byte{2}, 16000, false))

	buf.Clear()

	if buf.GetPacketCount() != 0 {
		t.Errorf("Packet count after clear: got %d, want 0", buf.GetPacketCount())
	}
}

// TestAudioBuffer_Concurrent 测试并发安全
func TestAudioBuffer_Concurrent(t *testing.T) {
	buf := NewAudioBuffer()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(seq uint32) {
			defer wg.Done()
			buf.Add(NewAudioPacket(seq, int64(seq)*1000, []byte{byte(seq)}, 16000, false))
		}(uint32(i))
	}

	wg.Wait()

	// 验证包数量
	count := buf.GetPacketCount()
	if count != 100 {
		t.Errorf("Packet count after concurrent add: got %d, want 100", count)
	}

	// 验证可以正常刷新
	var flushedCount int
	buf.Flush(func(pkt *AudioPacket) {
		flushedCount++
	})

	if flushedCount != 100 {
		t.Errorf("Flushed count: got %d, want 100", flushedCount)
	}
}

// TestTTSBuffer_AddWithTimeout 测试 TTS 缓冲超时
func TestTTSBuffer_AddWithTimeout(t *testing.T) {
	ttsBuf := NewTTSBuffer(50) // 50ms 窗口

	// 添加正常包
	pkt1 := NewAudioPacket(1, 1000, []byte{1}, 16000, false)
	if !ttsBuf.AddWithTimeout(pkt1, 1000) {
		t.Error("Normal packet should be added")
	}

	// 添加过期但还在容忍窗口内的包
	oldPkt := NewAudioPacket(0, 900, []byte{0}, 16000, false)
	// 当前时间 1000，包时间 900，延迟 100ms > 50ms 窗口，应该被拒绝
	if ttsBuf.AddWithTimeout(oldPkt, 1000) {
		t.Error("Expired packet outside window should be rejected")
	}
}

// TestTTSBuffer_LatePacketWithinWindow 测试容忍窗口内的延迟包
func TestTTSBuffer_LatePacketWithinWindow(t *testing.T) {
	ttsBuf := NewTTSBuffer(100) // 100ms 窗口

	// 添加初始包
	pkt1 := NewAudioPacket(1, 1000, []byte{1}, 16000, false)
	ttsBuf.Add(pkt1)
	ttsBuf.Flush(func(pkt *AudioPacket) {})

	// 添加窗口内的延迟包
	latePkt := NewAudioPacket(0, 950, []byte{0}, 16000, false)
	// 当前时间 1000，包时间 950，延迟 50ms < 100ms 窗口，应该被接受
	if !ttsBuf.AddWithTimeout(latePkt, 1000) {
		t.Error("Late packet within window should be accepted")
	}
}

// TestAudioBufferManager 测试多流缓冲管理
func TestAudioBufferManager(t *testing.T) {
	mgr := NewAudioBufferManager()

	// 获取或创建流缓冲
	buf1 := mgr.GetOrCreate("stream1")
	buf2 := mgr.GetOrCreate("stream2")

	if buf1 == buf2 {
		t.Error("Different streams should have different buffers")
	}

	// 获取已存在的缓冲
	buf1Again, exists := mgr.Get("stream1")
	if !exists {
		t.Error("Stream1 should exist")
	}
	if buf1Again != buf1 {
		t.Error("Get should return same buffer")
	}

	// 流数量
	if mgr.GetStreamCount() != 2 {
		t.Errorf("Stream count: got %d, want 2", mgr.GetStreamCount())
	}

	// 删除流
	mgr.Delete("stream1")
	if mgr.GetStreamCount() != 1 {
		t.Errorf("Stream count after delete: got %d, want 1", mgr.GetStreamCount())
	}

	// 清空所有
	mgr.ClearAll()
	if mgr.GetStreamCount() != 0 {
		t.Errorf("Stream count after clear: got %d, want 0", mgr.GetStreamCount())
	}
}

// TestNewAudioPacket 测试创建音频包
func TestNewAudioPacket(t *testing.T) {
	data := []byte{1, 2, 3, 4}
	pkt := NewAudioPacket(100, 1234567890, data, 24000, true)

	if pkt.Sequence != 100 {
		t.Errorf("Sequence: got %d, want 100", pkt.Sequence)
	}
	if pkt.Timestamp != 1234567890 {
		t.Errorf("Timestamp: got %d, want 1234567890", pkt.Timestamp)
	}
	if len(pkt.Data) != 4 {
		t.Errorf("Data length: got %d, want 4", len(pkt.Data))
	}
	if pkt.SampleRate != 24000 {
		t.Errorf("SampleRate: got %d, want 24000", pkt.SampleRate)
	}
	if !pkt.IsLast {
		t.Error("IsLast should be true")
	}
}
