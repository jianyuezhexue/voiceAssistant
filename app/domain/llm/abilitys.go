package llm

import (
	"context"
	"fmt"
	"voice-assistant/app/component/tool"
	"voice-assistant/app/config"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type LLMInterface interface {
	NewQwenChatModel(ctx context.Context) (model.ToolCallingChatModel, error)
	NewOllamaChatModel(ctx context.Context) (model.ToolCallingChatModel, error)
	NewQwen35flashModel(ctx context.Context) (model.ToolCallingChatModel, error)
}

// NewQwenChatModel 百炼平台聊天模型 (qwen3.6-plus)
func (a *LLM) NewQwenChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	chatModel, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL:     config.Config.Dashscope.BaseURL,
		APIKey:      config.Config.Dashscope.APIKey,
		Timeout:     0,
		Model:       "qwen3.6-plus",
		MaxTokens:   tool.Of(2048),
		Temperature: tool.Of(float32(0.7)),
		TopP:        tool.Of(float32(0.7)),
	})
	if err != nil {
		return nil, err
	}

	return chatModel, nil
}

// NewQwen35flashModel 百炼平台快速模型 (deepseek-v4-flash)
func (a *LLM) NewQwen35flashModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	chatModel, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL:     config.Config.Dashscope.BaseURL,
		APIKey:      config.Config.Dashscope.APIKey,
		Timeout:     0,
		Model:       "deepseek-v4-flash",
		MaxTokens:   tool.Of(2048),
		Temperature: tool.Of(float32(0.7)),
		TopP:        tool.Of(float32(0.7)),
	})
	if err != nil {
		return nil, err
	}

	return chatModel, nil
}

// NewArkChatModel 百炼平台 Ark 模型 (qwen-plus)
func (a *LLM) NewArkChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		APIKey:  config.Config.Dashscope.APIKey,
		Model:   "qwen-plus",
		BaseURL: config.Config.Dashscope.BaseURL,
		Thinking: &arkModel.Thinking{
			Type: arkModel.ThinkingTypeDisabled,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("ark.NewChatModel failed: %v", err))
	}
	return cm, nil
}

// ollama gpt模型
func (a *LLM) NewOllamaChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: "http://localhost:11434", // Ollama 服务地址
		Model:   "gpt-oss:20b",            // 模型名称
	})
	return chatModel, err
}
