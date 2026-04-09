package main

import (
	"fmt"
	"log"

	"voice-assistant/backend/component/db"
	"voice-assistant/backend/config"
	"voice-assistant/backend/router"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()

	// 初始化数据库
	db.Init(db.Config{
		Source: cfg.Database.MySQL.Source,
	})

	// 初始化路由
	r := router.Setup(cfg.Server.Mode, cfg)

	// 启动服务
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("服务启动在 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
