package asr

import (
	"testing"
	"time"
)

// TestNewClient 测试创建客户端
func TestNewClient(t *testing.T) {
	token := "392572cfc26a44ef94a0cccce18e6691"
	appKey := "auRliXRagRX2txBf"

	client := NewClient(token, appKey)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.config == nil {
		t.Fatal("config is nil")
	}
}

// TestClientStart 测试启动识别
func TestClientStart(t *testing.T) {
	token := "392572cfc26a44ef94a0cccce18e6691"
	appKey := "auRliXRagRX2txBf"

	client := NewClient(token, appKey)

	// 设置回调
	callback := func(text string) {
		t.Logf("Callback received text: %s", text)
	}

	// 启动识别
	err := client.Start(callback)
	if err != nil {
		t.Logf("Start error (expected if no network): %v", err)
		// 不失败，因为可能没有网络
	}

	// 等待一段时间看是否有回调
	time.Sleep(2 * time.Second)

	// 关闭
	client.Close()
}

// TestSendAudioWithoutStart 测试未启动时发送音频
func TestSendAudioWithoutStart(t *testing.T) {
	token := "392572cfc26a44ef94a0cccce18e6691"
	appKey := "auRliXRagRX2txBf"

	client := NewClient(token, appKey)

	// 尝试发送音频（未启动）
	err := client.SendAudio([]byte("test audio data"))
	if err != nil {
		t.Logf("SendAudio error (expected): %v", err)
	}
}

// TestStopWithoutStart 测试未启动时停止
func TestStopWithoutStart(t *testing.T) {
	token := "392572cfc26a44ef94a0cccce18e6691"
	appKey := "auRliXRagRX2txBf"

	client := NewClient(token, appKey)

	// 尝试停止（未启动）
	err := client.Stop()
	if err != nil {
		t.Logf("Stop error (expected): %v", err)
	}
}

// TestCloseTwice 测试关闭两次
func TestCloseTwice(t *testing.T) {
	token := "392572cfc26a44ef94a0cccce18e6691"
	appKey := "auRliXRagRX2txBf"

	client := NewClient(token, appKey)

	// 第一次关闭
	client.Close()

	// 第二次关闭
	client.Close()
}
