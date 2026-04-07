package router

import (
	"log"

	"github.com/gin-gonic/gin"
	"voice-assistant/backend/api"
	apiasr "voice-assistant/backend/api/asr"
	apichat "voice-assistant/backend/api/chat"
	"voice-assistant/backend/api/knowledge"
	"voice-assistant/backend/api/todo"
	voicehandler "voice-assistant/backend/api/voice"
	"voice-assistant/backend/component/llm"
	componenttts "voice-assistant/backend/component/tts"
	"voice-assistant/backend/component/webrtc"
	"voice-assistant/backend/config"
	domainvoice "voice-assistant/backend/domain/voice"
	logicchat "voice-assistant/backend/logic"
	logicvoice "voice-assistant/backend/logic/voice"
)

// Setup 初始化路由
func Setup(mode string, cfg *config.Config) *gin.Engine {
	gin.SetMode(mode)

	r := gin.Default()

	// 设置默认用户ID（解决 currUserId 缺失问题）
	r.Use(func(c *gin.Context) {
		c.Set("currUserId", "1")
		c.Set("currUserName", "系统")
		c.Next()
	})

	// 初始化语音组件
	voiceHandler := initVoiceHandler(cfg)

	// 初始化聊天组件
	chatHandler := initChatHandler(cfg)

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
		v1.POST("/chat", chatHandler.Chat)
	}

	// WebSocket 路由
	r.GET("/ws/asr", apiasr.NewASR().Handle)

	// 语音对话 WebSocket 路由
	r.GET("/ws/voice", voiceHandler.HandleWS)

	return r
}

// initVoiceHandler 初始化语音处理器
func initVoiceHandler(cfg *config.Config) *voicehandler.VoiceHandler {
	// 创建会话管理器
	sessionManager := domainvoice.NewSessionManager(cfg.Session.Timeout)

	// 创建 LLM 客户端
	var llmClient *llm.Client
	if cfg.LLM.APIKey != "" {
		llmClient = llm.NewClient(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
	}

	// 创建 TTS 客户端
	var ttsClient *componenttts.Client
	if cfg.TTS.ModelPath != "" {
		client, err := componenttts.NewClient(&componenttts.TTSConfig{
			ModelPath:    cfg.TTS.ModelPath,
			LexiconPath:  cfg.TTS.LexiconPath,
			SpeakersPath: cfg.TTS.SpeakersPath,
			SampleRate:   cfg.TTS.SampleRate,
			Speed:        cfg.TTS.Speed,
		})
		if err != nil {
			log.Printf("failed to create TTS client: %v", err)
		} else {
			ttsClient = client
		}
	}

	// 创建 WebRTC DataChannel Handler
	dcHandler := webrtc.NewDataChannelHandler(webrtc.DefaultDataChannelConfig)

	// 创建语音对话逻辑
	dialogueLogicConfig := &logicvoice.VoiceDialogueLogicConfig{
		LLMClient:      llmClient,
		TTSClient:      ttsClient,
		DCServer:       dcHandler,
		SessionManager: sessionManager,
	}
	dialogueLogic := logicvoice.NewVoiceDialogueLogic(dialogueLogicConfig)

	// 创建打断处理器
	interruptConfig := &logicvoice.InterruptHandlerConfig{
		TTSClient:      ttsClient,
		SessionManager: sessionManager,
	}
	interruptHandler := logicvoice.NewInterruptHandler(interruptConfig)

	// 创建语音 Handler
	return voicehandler.NewVoiceHandler(dialogueLogic, interruptHandler)
}

// initChatHandler 初始化聊天处理器
func initChatHandler(cfg *config.Config) *apichat.Handler {
	// 创建 LLM 客户端
	var llmClient *llm.Client
	if cfg.LLM.APIKey != "" {
		llmClient = llm.NewClient(cfg.LLM.APIKey, cfg.LLM.BaseURL, cfg.LLM.Model)
	}

	// 创建聊天逻辑
	chatLogic := logicchat.NewChatLogic(llmClient)

	// 创建聊天 Handler
	return apichat.NewHandler(chatLogic)
}
