package redis

import (
	"context"
	"log"

	"voice-assistant/backend/config"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

// Init 初始化 Redis 连接
func Init(cfg *config.RedisConfig) {
	Client = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx := context.Background()
	if err := Client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis 连接失败: %v", err)
	}

	log.Println("Redis 连接成功")
}

// GetClient 获取 Redis 客户端
func GetClient() *redis.Client {
	return Client
}
