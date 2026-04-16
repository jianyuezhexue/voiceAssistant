package llm

import (
	"context"
	"fmt"
	"voice-assistant/backend/component/tool"

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

// GLM5
func (a *LLM) NewQwenChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	apiKey := "sk-e692504205e74522b45710e1c25065ad"
	modelName := "glm-5"
	chatModel, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:      apiKey,
		Timeout:     0,
		Model:       modelName,
		MaxTokens:   tool.Of(2048),
		Temperature: tool.Of(float32(0.7)),
		TopP:        tool.Of(float32(0.7)),
	})
	if err != nil {
		return nil, err
	}

	return chatModel, nil
}

// qwen3.5-flash
func (a *LLM) NewQwen35flashModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	apiKey := "sk-e692504205e74522b45710e1c25065ad"
	modelName := "qwen3.5-flash"
	chatModel, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:      apiKey,
		Timeout:     0,
		Model:       modelName,
		MaxTokens:   tool.Of(2048),
		Temperature: tool.Of(float32(0.7)),
		TopP:        tool.Of(float32(0.7)),
	})
	if err != nil {
		return nil, err
	}

	return chatModel, nil
}

// Ark模型(qwen-plus)
func (a *LLM) NewArkChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
		APIKey:  "sk-e692504205e74522b45710e1c25065ad",
		Model:   "qwen-plus",
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
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
