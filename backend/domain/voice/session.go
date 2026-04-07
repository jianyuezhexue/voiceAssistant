package voice

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	// ErrSessionNotFound 会话不存在
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired 会话已过期
	ErrSessionExpired = errors.New("session expired")
)

// ISessionRepository 会话仓储接口
type ISessionRepository interface {
	Save(ctx context.Context, session *Session) error
	Get(ctx context.Context, id string) (*Session, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, id string) error
	GetByUserID(ctx context.Context, userID string) (*Session, error)
}

// SessionManager 会话管理器
type SessionManager struct {
	sessions     map[string]*Session
	userSessions map[string]string // userID -> sessionID
	lock         sync.RWMutex
	timeout      time.Duration
}

// NewSessionManager 创建会话管理器
func NewSessionManager(timeout time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions:     make(map[string]*Session),
		userSessions: make(map[string]string),
		timeout:      timeout,
	}

	// 启动超时清理协程
	go sm.cleanupLoop()

	return sm
}

// cleanupLoop 定期清理过期会话
func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanupExpired()
	}
}

// cleanupExpired 清理过期会话
func (sm *SessionManager) cleanupExpired() {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		if now.Sub(session.LastActiveAt) > sm.timeout {
			delete(sm.sessions, id)
			delete(sm.userSessions, session.UserID)
		}
	}
}

// Create 创建新会话
func (sm *SessionManager) Create(userID string) (*Session, error) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	// 检查用户是否已有活跃会话
	if sessionID, exists := sm.userSessions[userID]; exists {
		if session, ok := sm.sessions[sessionID]; ok {
			// 更新现有会话
			session.UpdateActivity()
			session.State = StateListening
			return session, nil
		}
	}

	// 创建新会话
	session := NewSession(generateSessionID(), userID)
	session.State = StateListening

	sm.sessions[session.ID] = session
	sm.userSessions[userID] = session.ID

	return session, nil
}

// Get 获取会话
func (sm *SessionManager) Get(sessionID string) (*Session, error) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	// 检查是否过期
	if time.Since(session.LastActiveAt) > sm.timeout {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// Update 更新会话
func (sm *SessionManager) Update(sessionID string, state VoiceState) (*Session, error) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	session.UpdateState(state)
	return session, nil
}

// Delete 删除会话
func (sm *SessionManager) Delete(sessionID string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	delete(sm.userSessions, session.UserID)
	delete(sm.sessions, sessionID)

	return nil
}

// GetByUserID 根据用户ID获取活跃会话
func (sm *SessionManager) GetByUserID(userID string) (*Session, error) {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	sessionID, exists := sm.userSessions[userID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}

	// 检查是否过期
	if time.Since(session.LastActiveAt) > sm.timeout {
		return nil, ErrSessionExpired
	}

	return session, nil
}

// GetAll 获取所有会话
func (sm *SessionManager) GetAll() []*Session {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetSessionCount 获取会话数量
func (sm *SessionManager) GetSessionCount() int {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	return len(sm.sessions)
}

// UpdateActivity 更新会话活跃时间
func (sm *SessionManager) UpdateActivity(sessionID string) error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	session.UpdateActivity()
	return nil
}

// Interrupt 中断会话
func (sm *SessionManager) Interrupt(sessionID string, source InterruptSource) (*Session, error) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	session.SetInterrupted(true)
	session.UpdateState(StateListening)
	session.UpdateActivity()

	return session, nil
}

// generateSessionID 生成会话ID
func generateSessionID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}
