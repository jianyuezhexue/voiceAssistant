package wspool

import (
	"sync"
	"time"
)

type Config struct {
	// 全站最大连接数
	MaxConnections int
	// 写队列容量
	WriteChanCap int
	// 读队列容量
	ReadChanCap int
	// 写队列阻塞最大等待，超时丢弃
	WriteWaitTimeout time.Duration
	// 空闲超时：超过此时间无任何消息自动关闭
	IdleTimeout time.Duration
	// 主动 Ping 间隔
	PingPeriod time.Duration
	// 单次写超时
	WriteDeadline time.Duration
	// 最大消息体大小（bytes）
	MaxMessageSize int64
}

var WspoolConfig *Config
var ConfigOnce sync.Once

// 单例初始化wspoll配置
func init() {
	ConfigOnce.Do(func() {
		WspoolConfig = &Config{
			MaxConnections:   1000,
			WriteChanCap:     20,
			ReadChanCap:      50,
			WriteWaitTimeout: 1 * time.Second,
			// IdleTimeout:      5 * time.Minute,
			IdleTimeout:    1 * time.Minute, // 临时测试
			PingPeriod:     30 * time.Second,
			WriteDeadline:  10 * time.Second,
			MaxMessageSize: 1024 * 1024, // 1MB
		}
	})
}
