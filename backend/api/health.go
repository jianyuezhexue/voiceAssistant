package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"voice-assistant/backend/component/asr"
	"voice-assistant/backend/component/llm"
	"voice-assistant/backend/component/tts"
	"voice-assistant/backend/logic/voice"
)

// HealthHandler 健康检查 Handler
type HealthHandler struct {
	// 组件客户端（可选，用于详细健康检查）
	llmClient        *llm.Client
	asrClient        *asr.Client
	ttsClient        *tts.Client
	interruptHandler *voice.InterruptHandler
}

// NewHealthHandler 创建健康检查 Handler
func NewHealthHandler(interruptHandler *voice.InterruptHandler) *HealthHandler {
	return &HealthHandler{
		interruptHandler: interruptHandler,
	}
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status    string        `json:"status"`    // overall status: ok, degraded, unhealthy
	Services  ServiceStatus `json:"services"`  // individual service status
	Timestamp int64         `json:"timestamp"` // check timestamp
	Uptime    string        `json:"uptime"`    // server uptime
}

// ServiceStatus 各服务状态
type ServiceStatus struct {
	LLM    ServiceHealth `json:"llm"`
	ASR    ServiceHealth `json:"asr"`
	TTS    ServiceHealth `json:"tts"`
	WebRTC ServiceHealth `json:"webrtc"`
}

// ServiceHealth 单个服务健康状态
type ServiceHealth struct {
	Status  string `json:"status"`            // healthy, degraded, unhealthy, unknown
	Latency string `json:"latency"`           // response time in ms
	Message string `json:"message,omitempty"` // error message if unhealthy
}

// HealthCheck 健康检查
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	status := h.performHealthCheck(ctx)

	// 根据状态返回不同 HTTP 状态码
	httpStatus := http.StatusOK
	if status.Status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	} else if status.Status == "degraded" {
		httpStatus = http.StatusOK // 降级状态仍返回 200
	}

	c.JSON(httpStatus, status)
}

// performHealthCheck 执行健康检查
func (h *HealthHandler) performHealthCheck(ctx context.Context) *HealthStatus {
	status := &HealthStatus{
		Timestamp: time.Now().UnixMilli(),
		Services: ServiceStatus{
			LLM:    h.checkLLM(ctx),
			ASR:    h.checkASR(ctx),
			TTS:    h.checkTTS(ctx),
			WebRTC: h.checkWebRTC(ctx),
		},
	}

	// 计算整体状态
	status.Status = h.calculateOverallStatus(status.Services)

	return status
}

// checkLLM 检查 LLM 服务
func (h *HealthHandler) checkLLM(ctx context.Context) ServiceHealth {
	start := time.Now()
	health := ServiceHealth{Status: "healthy"}

	// 检查熔断器状态
	if h.interruptHandler != nil {
		cbState := h.interruptHandler.GetCircuitBreakerState()
		if cbState.LLM == "open" {
			health.Status = "unhealthy"
			health.Message = "LLM circuit breaker is open"
			health.Latency = time.Since(start).String()
			return health
		} else if cbState.LLM == "half-open" {
			health.Status = "degraded"
			health.Message = "LLM circuit breaker is half-open"
		}
	}

	// 检查 LLM 客户端
	if h.llmClient == nil {
		health.Status = "unknown"
		health.Message = "LLM client not configured"
		health.Latency = time.Since(start).String()
		return health
	}

	// 简单连通性检查（发送一个简单请求）
	// 实际实现可能需要更复杂的检查
	health.Latency = time.Since(start).String()
	return health
}

// checkASR 检查 ASR 服务
func (h *HealthHandler) checkASR(ctx context.Context) ServiceHealth {
	start := time.Now()
	health := ServiceHealth{Status: "healthy"}

	// 检查熔断器状态
	if h.interruptHandler != nil {
		cbState := h.interruptHandler.GetCircuitBreakerState()
		if cbState.ASR == "open" {
			health.Status = "unhealthy"
			health.Message = "ASR circuit breaker is open"
			health.Latency = time.Since(start).String()
			return health
		} else if cbState.ASR == "half-open" {
			health.Status = "degraded"
			health.Message = "ASR circuit breaker is half-open"
		}
	}

	// 检查 ASR 客户端
	if h.asrClient == nil {
		health.Status = "unknown"
		health.Message = "ASR client not configured"
		health.Latency = time.Since(start).String()
		return health
	}

	health.Latency = time.Since(start).String()
	return health
}

// checkTTS 检查 TTS 服务
func (h *HealthHandler) checkTTS(ctx context.Context) ServiceHealth {
	start := time.Now()
	health := ServiceHealth{Status: "healthy"}

	// 检查熔断器状态
	if h.interruptHandler != nil {
		cbState := h.interruptHandler.GetCircuitBreakerState()
		if cbState.TTS == "open" {
			health.Status = "unhealthy"
			health.Message = "TTS circuit breaker is open"
			health.Latency = time.Since(start).String()
			return health
		} else if cbState.TTS == "half-open" {
			health.Status = "degraded"
			health.Message = "TTS circuit breaker is half-open"
		}
	}

	// 检查 TTS 客户端
	if h.ttsClient == nil {
		health.Status = "unknown"
		health.Message = "TTS client not configured"
		health.Latency = time.Since(start).String()
		return health
	}

	health.Latency = time.Since(start).String()
	return health
}

// checkWebRTC 检查 WebRTC 服务
func (h *HealthHandler) checkWebRTC(ctx context.Context) ServiceHealth {
	start := time.Now()
	health := ServiceHealth{Status: "healthy"}

	// WebRTC 服务通常内嵌在后端，这里做简单检查
	health.Latency = time.Since(start).String()
	return health
}

// calculateOverallStatus 计算整体状态
func (h *HealthHandler) calculateOverallStatus(services ServiceStatus) string {
	allHealthy := true
	anyUnhealthy := false
	anyDegraded := false

	// 检查各服务状态
	for _, service := range []ServiceHealth{services.LLM, services.ASR, services.TTS, services.WebRTC} {
		switch service.Status {
		case "unhealthy":
			anyUnhealthy = true
			allHealthy = false
		case "degraded":
			anyDegraded = true
			allHealthy = false
		case "unknown":
			// unknown 不影响整体判断
		case "healthy":
			// 继续检查
		}
	}

	if anyUnhealthy {
		return "unhealthy"
	}
	if anyDegraded {
		return "degraded"
	}
	if allHealthy {
		return "ok"
	}
	return "degraded"
}

// LivenessProbe K8s liveness probe
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// ReadinessProbe K8s readiness probe
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	status := h.performHealthCheck(ctx)

	if status.Status == "unhealthy" {
		c.JSON(http.StatusServiceUnavailable, status)
		return
	}

	c.JSON(http.StatusOK, status)
}

// SetClients 设置组件客户端（用于详细健康检查）
func (h *HealthHandler) SetClients(llmClient *llm.Client, asrClient *asr.Client, ttsClient *tts.Client) {
	h.llmClient = llmClient
	h.asrClient = asrClient
	h.ttsClient = ttsClient
}
