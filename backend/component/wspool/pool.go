package wspool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
)

var (
	ErrPoolFull      = errors.New("ws连接数已达上限")
	ErrSessionExists = errors.New("session已存在")
)

// WSPool 全局 WebSocket 连接池
type WSPool struct {
	cfg     *Config
	clients sync.Map     // map[sessionId]*WSClient
	count   atomic.Int32 // 当前连接数
}

var (
	instance *WSPool
	once     sync.Once
)

// 单例初始化
func init() {
	once.Do(func() {
		instance = &WSPool{
			cfg: WspoolConfig,
		}
	})
}

// GetPool 获取全局连接池实例
// 未初始化时 panic，确保启动顺序正确
func GetPool() *WSPool {
	if instance == nil {
		panic("wspool 未初始化，请先调用 wspool.Init()")
	}
	return instance
}

// Register 注册新的 WebSocket 连接
func (p *WSPool) Register(sessionId string, conn *websocket.Conn) (*WSClient, error) {
	// CAS 自增，防止并发超限
	for {
		cur := p.count.Load()
		if int(cur) >= p.cfg.MaxConnections {
			return nil, fmt.Errorf("%w (当前:%d 上限:%d)", ErrPoolFull, cur, p.cfg.MaxConnections)
		}
		if p.count.CompareAndSwap(cur, cur+1) {
			break
		}
	}

	client := newWSClient(sessionId, conn, p)

	// LoadOrStore 防止 sessionId 重复
	if _, loaded := p.clients.LoadOrStore(sessionId, client); loaded {
		p.count.Add(-1)
		client.idleTimer.Stop()
		return nil, ErrSessionExists
	}

	return client, nil
}

// Count 当前连接数
func (p *WSPool) Count() int {
	return int(p.count.Load())
}

// MaxConnections 上限
func (p *WSPool) MaxConnections() int {
	return p.cfg.MaxConnections
}

// SendTo 向指定 session 推送消息
func (p *WSPool) SendTo(sessionId string, data []byte) bool {
	v, ok := p.clients.Load(sessionId)
	if !ok {
		return false
	}
	return v.(*WSClient).Send(data)
}

// remove 由 WSClient.Close() 内部调用
func (p *WSPool) remove(sessionId string) {
	if _, loaded := p.clients.LoadAndDelete(sessionId); loaded {
		p.count.Add(-1)
	}
}
