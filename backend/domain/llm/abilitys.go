package llm

import (
	"context"
	"fmt"
	"voice-assistant/backend/component/tool"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

type AgentsInterface interface {
	NewQwenChatModel(ctx context.Context) (*qwen.ChatModel, error)
	NewOllamaChatModel(ctx context.Context) (*ollama.ChatModel, error)
}

// 千问模型
func (a *Agents) NewQwenChatModel(ctx context.Context) (*qwen.ChatModel, error) {
	apiKey := "sk-e692504205e74522b45710e1c25065ad"
	modelName := "qwen-plus"
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
func (a *Agents) NewArkChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
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
func (a *Agents) NewOllamaChatModel(ctx context.Context) (*ollama.ChatModel, error) {
	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: "http://localhost:11434", // Ollama 服务地址
		Model:   "gpt-oss:20b",            // 模型名称
	})
	return chatModel, err
}

func (a *Agents) NewPlanAgent() adk.Agent {
	model, err := a.NewArkChatModel(context.Background())
	agent, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "生成业务模型",
		Description: "生成业务模型数据",
		Instruction: `根据用户输入的内容，生成业务模型`,
		Model:       model,
	})
	if err != nil {
		panic(fmt.Sprintf("NewChatModelAgent failed: %v", err))
	}
	return agent
}
