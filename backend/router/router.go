package router

import (
	"github.com/gin-gonic/gin"
	"voice-assistant/backend/api"
	apiasr "voice-assistant/backend/api/asr"
	apichat "voice-assistant/backend/api/chat"
	"voice-assistant/backend/api/knowledge"
	"voice-assistant/backend/api/todo"
	voicehandler "voice-assistant/backend/api/voice"
	"voice-assistant/backend/component/llm"
	"voice-assistant/backend/config"
)

// Setup 初始化路由
func Setup(mode string, cfg *config.Config) *gin.Engine {
	gin.SetMode(mode)

	r := gin.Default()

	// 初始化 LLM 单例（惰性加载，不影响路由初始化性能）
	if cfg.LLM.APIKey != "" {
		llm.GetClient(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
	}

	// 设置默认用户ID（解决 currUserId 缺失问题）
	r.Use(func(c *gin.Context) {
		c.Set("currUserId", "1")
		c.Set("currUserName", "系统")
		c.Next()
	})

	// 初始化语音组件
	voiceHandler := initVoiceHandler(cfg)

	// 健康检查
	healthHandler := api.NewHealthHandler(nil)
	r.GET("/health", healthHandler.HealthCheck)
	r.GET("/healthz", healthHandler.LivenessProbe)
	r.GET("/readyz", healthHandler.ReadinessProbe)

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

		// Chat 路由
		chatApi := apichat.NewHandler()
		chatGroup := v1.Group("/chat")
		{
			chatGroup.POST("/send", chatApi.Chat)
		}
	}

	// WebSocket 路由
	r.GET("/ws/asr", apiasr.NewASR().Handle)

	// 语音对话 WebSocket 路由
	r.GET("/ws/voice", voiceHandler.HandleWS)

	return r
}

// initVoiceHandler 初始化语音处理器
func initVoiceHandler(cfg *config.Config) *voicehandler.VoiceHandler {
	return voicehandler.NewVoiceHandler(cfg)
}
