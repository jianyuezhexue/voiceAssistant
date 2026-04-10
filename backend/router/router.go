package router

import (
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
			// 文本对话
			chatGroup.POST("", chatAPi.TextTalk)
			// 语音对话
			chatGroup.POST("/speech", chatAPi.SpeechTalk)
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

	return r
}
