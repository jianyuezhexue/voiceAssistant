package router

import (
	"voice-assistant/backend/api"
	"voice-assistant/backend/api/knowledge"
	"voice-assistant/backend/api/todo"
	"voice-assistant/backend/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 初始化路由
func Setup() *gin.Engine {
	r := gin.Default()

	// 设置默认用户ID（解决 currUserId 缺失问题）
	r.Use(middleware.AuthMiddleware())

	// 健康检查
	r.GET("/health", api.NewBase().Health)

	// API v1
	v1 := r.Group("/api/v1")
	{

		// websocket链接

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

	return r
}
