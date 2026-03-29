package main

import (
	"log"

	"voice-assistant/backend/component/db"
	"voice-assistant/backend/config"
	"voice-assistant/backend/router"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	db.Init(db.Config{
		Source: cfg.Database.MySQL.Source,
	})

	// TODO: 初始化 Redis
	// redis.Init(&cfg.Redis)

	// 初始化路由
	r := router.Setup(cfg.Server.Mode)

	// 启动服务
	log.Printf("服务启动在 :%s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}