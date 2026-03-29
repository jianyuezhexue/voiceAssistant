package router

import (
	"github.com/gin-gonic/gin"
	"voice-assistant/backend/api/asr"
	"voice-assistant/backend/api/knowledge"
	"voice-assistant/backend/api/todo"
)

// Setup 初始化路由
func Setup(mode string) *gin.Engine {
	gin.SetMode(mode)

	r := gin.Default()

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API v1
	v1 := r.Group("/api/v1")
	{
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

	// WebSocket 路由
	r.GET("/ws/asr", asr.NewASR().Handle)

	return r
}