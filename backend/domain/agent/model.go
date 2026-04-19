package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"voice-assistant/backend/domain/aiInfra/mcp"
	"voice-assistant/backend/domain/llm"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/gin-gonic/gin"
)

type Agent struct {
	Ctx *gin.Context
}

func NewAgent(ctx *gin.Context) *Agent {
	return &Agent{
		Ctx: ctx,
	}
}

// toolErrorMiddleware 拦截工具调用错误，将错误转换为友好的结果返回，
// 让 LLM 能够感知到工具失败并自主决定如何回答，而不是直接终止 ReAct 循环。
func toolErrorMiddleware() compose.ToolMiddleware {
	return compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				output, err := next(ctx, input)
				if err != nil {
					// 工具调用失败，返回一个错误信息字符串，LLM 会将其作为工具结果继续推理
					log.Printf("Tool [%s] call failed: %v, returning fallback result", input.Name, err)
					return &compose.ToolOutput{
						Result: fmt.Sprintf("Search tool is currently unavailable: %v. Please answer the user's question based on your own knowledge.", err),
					}, nil // 注意：这里返回 nil error，让 ReAct 循环继续
				}
				return output, nil
			}
		},
	}
}

// ChatModelAgent 通用聊天Agent
func (a *Agent) ChatModelAgent() *adk.ChatModelAgent {

	// 实例化大模型
	// model, err := llm.NewLLM().NewQwenChatModel(a.Ctx)
	model, err := llm.NewLLM().NewQwen35flashModel(a.Ctx)
	if err != nil {
		panic(err)
	}

	// 换mcp - 使用 background context 而非 gin.Context，避免请求超时导致 MCP 客户端初始化失败
	tools, err := mcp.NewBingSearchTools(context.Background())
	if err != nil {
		log.Printf("NewBingSearchTools failed, err=%v, search tool will be disabled", err)
	}

	log.Printf("[Agent] searchTool initialized: %v, tools count: %d", tools != nil, len(tools))

	// 实例化Agent
	chatAgent, err := adk.NewChatModelAgent(a.Ctx, &adk.ChatModelAgentConfig{
		Name:        "intelligent_assistant",
		Description: "An intelligent assistant capable of using multiple tools to solve complex problems",
		Instruction: "You are a professional assistant. When the user asks questions that require up-to-date information or web search, you can use the 'bing_search' tool to search for relevant information. Only answer directly from your knowledge if the tool is unavailable.",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
				ToolCallMiddlewares: []compose.ToolMiddleware{
					toolErrorMiddleware(),
				},
			},
		},
	})

	return chatAgent
}

// CommonChat 通用对话，带 60s 超时保护
func (a *Agent) CommonChat(query string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// // 意图识别 & Query 改写
	// model, err := llm.NewLLM().NewQwen35flashModel(a.Ctx)
	// if err != nil {
	// 	log.Printf("[Agent] Query rewrite model init failed: %v, using raw query", err)
	// }

	// rewrittenQuery := query
	// if model != nil {
	// 	queryChangePrompt := &schema.Message{
	// 		Role:    schema.User,
	// 		Content: fmt.Sprintf("用户输入如下：%s\n综合评估用户输入是否完整，必要时优化提示词使目标更精准，结合用户意图，有必要的话加上今天的日期: %s\n只输出改写后的提示词，不要解释。", query, time.Now().Format("2006-01-02")),
	// 	}
	// 	log.Printf("[Agent] Rewriting query: %s", query)
	// 	queryChange, err := model.Generate(ctx, []*schema.Message{queryChangePrompt})
	// 	if err != nil {
	// 		log.Printf("[Agent] Query rewrite failed: %v, using raw query", err)
	// 	} else {
	// 		rewrittenQuery = queryChange.Content
	// 		log.Printf("[Agent] Rewritten query: %s", rewrittenQuery)
	// 	}
	// }

	// 初始化 Runner
	runner := adk.NewRunner(a.Ctx, adk.RunnerConfig{
		Agent:           a.ChatModelAgent(),
		EnableStreaming: false,
	})

	// 每次对话使用唯一 checkpointID，避免多会话共享历史
	checkpointID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	log.Printf("[Agent] Starting runner with checkpointID=%s", checkpointID)

	iter := runner.Query(ctx, query, adk.WithCheckPointID(checkpointID))
	var result string
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if strings.Contains(event.Err.Error(), "invoke tool") {
				log.Printf("[Agent] Tool invoke error: %v", event.Err)
				continue
			}
			return "", event.Err
		}
		if event.Action != nil {
			log.Printf("[Agent Event] Action: exit=%v, transfer=%v, break=%v", event.Action.Exit, event.Action.TransferToAgent != nil, event.Action.BreakLoop != nil)
		}
		prints.Event(event)
		if event.Output != nil {
			result = event.Output.MessageOutput.Message.Content
		}
	}
	log.Printf("[Agent] CommonChat done, result length=%d", len(result))
	return result, nil
}
