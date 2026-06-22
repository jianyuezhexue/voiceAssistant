package router

import (
	"os"
	"path/filepath"
	"strings"
	"voice-assistant/backend/api"
	"voice-assistant/backend/api/chat"
	"voice-assistant/backend/api/knowledge"
	"voice-assistant/backend/api/todo"
	"voice-assistant/backend/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 初始化路由
func Setup() *gin.Engine {
	r := gin.Default()

	// 检查检查和中间件
	r.GET("/health", api.NewBase().Health)
	r.Use(middleware.AuthMiddleware())

	// API v1
	v1 := r.Group("/api/v1")
	{
		// 聊天
		chatAPi := chat.NewChat()
		chatGroup := v1.Group("/chat")
		{
			// ws链接
			chatGroup.GET("ws", chatAPi.WsConn)
		}

		// Todo 路由
		todoApi := todo.NewTodo()
		todoGroup := v1.Group("/todos")
		{
			todoGroup.POST("", todoApi.Create)
			todoGroup.PUT("/:id", todoApi.Update)
			todoGroup.GET("/:id", todoApi.Get)
			todoGroup.GET("/list", todoApi.List)
			todoGroup.DELETE("", todoApi.Del)
		}

		// Knowledge 路由
		knowledgeApi := knowledge.NewKnowledge()
		knowledgeGroup := v1.Group("/knowledge")
		{
			knowledgeGroup.POST("", knowledgeApi.Create)
			knowledgeGroup.PUT("/:id", knowledgeApi.Update)
			knowledgeGroup.GET("/:id", knowledgeApi.Get)
			knowledgeGroup.GET("/list", knowledgeApi.List)
			knowledgeGroup.POST("/search", knowledgeApi.Search)
			knowledgeGroup.DELETE("", knowledgeApi.Del)
		}
	}

	// 前端静态文件托管（生产：Go 单进程托管 Vite 打包产物）
	distDir := os.Getenv("WEB_DIST_DIR")
	if distDir == "" {
		distDir = "./web/dist"
	}
	indexFile := filepath.Join(distDir, "index.html")
	// /assets/* 走快速静态路径
	r.Static("/assets", filepath.Join(distDir, "assets"))
	// 其余根目录静态文件（favicon.svg / icons.svg / audio-processor.js 等）+ SPA 回退
	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") {
			c.JSON(404, gin.H{"code": -1, "message": "not found"})
			return
		}
		// 尝试从 dist 中提供对应文件
		fp := filepath.Join(distDir, filepath.Clean(p))
		if info, err := os.Stat(fp); err == nil && !info.IsDir() {
			c.File(fp)
			return
		}
		// SPA 回退：未匹配路由交由前端路由处理
		c.File(indexFile)
	})

	return r
}
