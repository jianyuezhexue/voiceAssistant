package voice

import (
	"context"
	"testing"
	"time"
)

// MockSessionRepository 模拟会话仓储
type MockSessionRepository struct {
	sessions map[string]*Session
}

func NewMockSessionRepository() *MockSessionRepository {
	return &MockSessionRepository{
		sessions: make(map[string]*Session),
	}
}

func (r *MockSessionRepository) Save(ctx context.Context, session *Session) error {
	r.sessions[session.ID] = session
	return nil
}

func (r *MockSessionRepository) Get(ctx context.Context, id string) (*Session, error) {
	if session, exists := r.sessions[id]; exists {
		return session, nil
	}
	return nil, ErrSessionNotFound
}

func (r *MockSessionRepository) Update(ctx context.Context, session *Session) error {
	if _, exists := r.sessions[session.ID]; !exists {
		return ErrSessionNotFound
	}
	r.sessions[session.ID] = session
	return nil
}

func (r *MockSessionRepository) Delete(ctx context.Context, id string) error {
	if _, exists := r.sessions[id]; !exists {
		return ErrSessionNotFound
	}
	delete(r.sessions, id)
	return nil
}

func (r *MockSessionRepository) GetByUserID(ctx context.Context, userID string) (*Session, error) {
	for _, session := range r.sessions {
		if session.UserID == userID {
			return session, nil
		}
	}
	return nil, ErrSessionNotFound
}

// TestSessionManager_Create 测试创建会话
func TestSessionManager_Create(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	session, err := sm.Create("user123")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if session == nil {
		t.Fatal("Session is nil")
	}

	if session.ID == "" {
		t.Error("Session ID is empty")
	}

	if session.UserID != "user123" {
		t.Errorf("UserID mismatch: got %s, want user123", session.UserID)
	}

	if session.State != StateListening {
		t.Errorf("Initial state mismatch: got %s, want %s", session.State, StateListening)
	}
}

// TestSessionManager_Get 测试获取会话
func TestSessionManager_Get(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	// 创建会话
	created, _ := sm.Create("user123")

	// 获取存在的会话
	session, err := sm.Get(created.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.ID != created.ID {
		t.Errorf("Session ID mismatch: got %s, want %s", session.ID, created.ID)
	}

	// 获取不存在的会话
	_, err = sm.Get("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Get nonexistent session: got %v, want ErrSessionNotFound", err)
	}
}

// TestSessionManager_Update 测试更新会话
func TestSessionManager_Update(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	// 创建会话
	session, _ := sm.Create("user123")

	// 更新状态
	updated, err := sm.Update(session.ID, StateThinking)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.State != StateThinking {
		t.Errorf("State mismatch: got %s, want %s", updated.State, StateThinking)
	}

	// 更新不存在的会话
	_, err = sm.Update("nonexistent", StateThinking)
	if err != ErrSessionNotFound {
		t.Errorf("Update nonexistent: got %v, want ErrSessionNotFound", err)
	}
}

// TestSessionManager_Delete 测试删除会话
func TestSessionManager_Delete(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	// 创建会话
	session, _ := sm.Create("user123")

	// 删除会话
	err := sm.Delete(session.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 验证删除
	_, err = sm.Get(session.ID)
	if err != ErrSessionNotFound {
		t.Errorf("Get after delete: got %v, want ErrSessionNotFound", err)
	}

	// 删除不存在的会话
	err = sm.Delete("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("Delete nonexistent: got %v, want ErrSessionNotFound", err)
	}
}

// TestSessionManager_GetByUserID 测试根据用户ID获取会话
func TestSessionManager_GetByUserID(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	// 创建会话
	created, _ := sm.Create("user123")

	// 获取用户会话
	session, err := sm.GetByUserID("user123")
	if err != nil {
		t.Fatalf("GetByUserID failed: %v", err)
	}

	if session.ID != created.ID {
		t.Errorf("Session ID mismatch: got %s, want %s", session.ID, created.ID)
	}

	// 获取不存在的用户会话
	_, err = sm.GetByUserID("nonexistent")
	if err != ErrSessionNotFound {
		t.Errorf("GetByUserID nonexistent: got %v, want ErrSessionNotFound", err)
	}
}

// TestSessionManager_GetAll 测试获取所有会话
func TestSessionManager_GetAll(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	// 创建多个会话
	sm.Create("user1")
	sm.Create("user2")
	sm.Create("user3")

	sessions := sm.GetAll()
	if len(sessions) != 3 {
		t.Errorf("GetAll count mismatch: got %d, want 3", len(sessions))
	}
}

// TestSessionManager_Concurrent 测试并发安全
func TestSessionManager_Concurrent(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	done := make(chan bool)

	// 并发创建会话
	for i := 0; i < 100; i++ {
		go func(userID string) {
			sm.Create(userID)
			done <- true
		}("user" + string(rune(i)))
	}

	// 等待所有协程完成
	for i := 0; i < 100; i++ {
		<-done
	}

	count := sm.GetSessionCount()
	if count != 100 {
		t.Errorf("Session count mismatch after concurrent create: got %d, want 100", count)
	}
}

// TestSessionManager_Interrupt 测试打断
func TestSessionManager_Interrupt(t *testing.T) {
	sm := NewSessionManager(30 * time.Minute)

	// 创建会话
	session, _ := sm.Create("user123")
	sm.Update(session.ID, StatePlaying)

	// 打断会话
	interrupted, err := sm.Interrupt(session.ID, InterruptUserSpeech)
	if err != nil {
		t.Fatalf("Interrupt failed: %v", err)
	}

	if !interrupted.IsInterrupted {
		t.Error("IsInterrupted should be true")
	}

	if interrupted.State != StateListening {
		t.Errorf("State after interrupt: got %s, want %s", interrupted.State, StateListening)
	}
}

// TestSession_StateTransitions 测试状态转换
func TestSession_StateTransitions(t *testing.T) {
	session := NewSession("test-id", "user123")

	tests := []struct {
		from    VoiceState
		to      VoiceState
		wantErr bool
	}{
		{StateIdle, StateListening, false},
		{StateListening, StateRecognizing, false},
		{StateRecognizing, StateThinking, false},
		{StateThinking, StateResponding, false},
		{StateResponding, StatePlaying, false},
		{StatePlaying, StateListening, false},
	}

	for _, tc := range tests {
		session.State = tc.from
		session.UpdateState(tc.to)

		if session.State != tc.to {
			t.Errorf("State transition %s -> %s failed: got %s", tc.from, tc.to, session.State)
		}
	}
}

// TestVoiceState_String 测试状态字符串表示
func TestVoiceState_String(t *testing.T) {
	tests := []struct {
		state VoiceState
		want  string
	}{
		{StateIdle, "idle"},
		{StateListening, "listening"},
		{StateRecognizing, "recognizing"},
		{StateThinking, "thinking"},
		{StateResponding, "responding"},
		{StatePlaying, "playing"},
		{StateError, "error"},
	}

	for _, tc := range tests {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("VoiceState.String() = %s, want %s", got, tc.want)
		}
	}
}

// TestVoiceState_IsActive 测试状态活跃判断
func TestVoiceState_IsActive(t *testing.T) {
	tests := []struct {
		state VoiceState
		want  bool
	}{
		{StateIdle, false},
		{StateListening, true},
		{StateRecognizing, true},
		{StateThinking, true},
		{StateResponding, true},
		{StatePlaying, true},
		{StateError, false},
	}

	for _, tc := range tests {
		if got := tc.state.IsActive(); got != tc.want {
			t.Errorf("VoiceState.IsActive() = %v, want %v for state %s", got, tc.want, tc.state)
		}
	}
}

// TestInterruptSource_String 测试打断来源字符串
func TestInterruptSource_String(t *testing.T) {
	tests := []struct {
		source InterruptSource
		want   string
	}{
		{InterruptUserSpeech, "user_speech"},
		{InterruptUserClick, "user_click"},
		{InterruptServerCmd, "server_cmd"},
		{InterruptTimeout, "timeout"},
	}

	for _, tc := range tests {
		if got := tc.source.String(); got != tc.want {
			t.Errorf("InterruptSource.String() = %s, want %s", got, tc.want)
		}
	}
}
