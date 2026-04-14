package wspool

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WSMessage 读取到的 WebSocket 消息
type WSMessage struct {
	MsgType int
	Data    []byte
}

// WSClient 单个 WebSocket 连接封装
type WSClient struct {
	SessionId string
	conn      *websocket.Conn
	pool      *WSPool
	writeChan chan []byte
	readChan  chan *WSMessage

	ctx       context.Context
	cancel    context.CancelFunc
	once      sync.Once
	idleTimer *time.Timer
}

func newWSClient(sessionId string, conn *websocket.Conn, pool *WSPool) *WSClient {
	ctx, cancel := context.WithCancel(context.Background())
	c := &WSClient{
		SessionId: sessionId,
		conn:      conn,
		pool:      pool,
		writeChan: make(chan []byte, pool.cfg.WriteChanCap),
		readChan:  make(chan *WSMessage, pool.cfg.ReadChanCap),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 空闲超时，到期自动关闭
	c.idleTimer = time.AfterFunc(pool.cfg.IdleTimeout, func() {
		log.Printf("[WSClient] session=%s 空闲超时(%.0f分钟)，自动关闭", sessionId, pool.cfg.IdleTimeout.Minutes())
		c.Close()
	})
	return c
}

// Start 启动读写协程
func (c *WSClient) Start() {
	go c.writePump()
	go c.readPump()
}

// Done 供外部阻塞等待连接关闭
func (c *WSClient) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Close 安全关闭（幂等）
func (c *WSClient) Close() {
	c.once.Do(func() {
		log.Printf("[WSClient] session=%s 连接关闭", c.SessionId)
		c.idleTimer.Stop()
		c.cancel()
		close(c.writeChan)
		c.conn.Close()
		c.pool.remove(c.SessionId)
	})
}

// Send 投递消息到写队列
// 队列满时等待 WriteWaitTimeout，超时丢弃返回 false
func (c *WSClient) Send(data []byte) bool {
	// 快速路径
	select {
	case c.writeChan <- data:
		return true
	case <-c.ctx.Done():
		return false
	default:
	}

	// 慢速路径：等待最多 WriteWaitTimeout
	timer := time.NewTimer(c.pool.cfg.WriteWaitTimeout)
	defer timer.Stop()
	select {
	case c.writeChan <- data:
		return true
	case <-timer.C:
		log.Printf("[WSClient] session=%s 写队列满，消息丢弃", c.SessionId)
		return false
	case <-c.ctx.Done():
		return false
	}
}

// resetIdleTimer 收到任意消息后重置空闲计时器
func (c *WSClient) resetIdleTimer() {
	c.idleTimer.Reset(c.pool.cfg.IdleTimeout)
}

// ─────────────────────────────────────────
//  读协程
// ─────────────────────────────────────────

func (c *WSClient) readPump() {
	defer c.Close()

	c.conn.SetReadLimit(c.pool.cfg.MaxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.resetIdleTimer()
		return nil
	})

	for {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.Printf("[readPump] session=%s 异常断开: %v", c.SessionId, err)
			}
			return
		}

		// 收到消息，重置空闲超时
		c.resetIdleTimer()

		// 推入读通道，供业务层消费
		msg := &WSMessage{MsgType: msgType, Data: data}
		select {
		case c.readChan <- msg:
		case <-c.ctx.Done():
			return
		}
	}
}

// ReadMessage 从读通道获取一条消息
// 阻塞直到有消息或连接关闭，关闭时返回 nil
func (c *WSClient) ReadMessage() *WSMessage {
	select {
	case msg, ok := <-c.readChan:
		if !ok {
			return nil
		}
		return msg
	case <-c.ctx.Done():
		return nil
	}
}

// ─────────────────────────────────────────
//  写协程
// ─────────────────────────────────────────

func (c *WSClient) writePump() {
	ticker := time.NewTicker(c.pool.cfg.PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case data, ok := <-c.writeChan:
			c.conn.SetWriteDeadline(time.Now().Add(c.pool.cfg.WriteDeadline))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("[writePump] session=%s 写入失败: %v", c.SessionId, err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.pool.cfg.WriteDeadline))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[writePump] session=%s Ping失败: %v", c.SessionId, err)
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}
