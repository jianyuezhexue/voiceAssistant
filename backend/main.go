package main

import (
	"fmt"
	"log"

	"voice-assistant/backend/config"
	"voice-assistant/backend/router"
)

func main() {

	// 初始化路由
	r := router.Setup()

	// 启动服务
	config := config.Config
	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	log.Printf("服务启动在 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}
