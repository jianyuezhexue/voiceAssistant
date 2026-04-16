package agent

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"
	"voice-assistant/backend/domain/llm"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
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

// ChatModelAgent 通用聊天Agent
func (a *Agent) ChatModelAgent() *adk.ChatModelAgent {

	// 实例化大模型
	// todo 这里未来使用工厂模式制定不同的大模型
	model, err := llm.NewLLM().NewQwenChatModel(a.Ctx)
	if err != nil {
		panic(err)
	}

	// 实例化搜索Tool
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	ipv4HTTPClient := &http.Client{
		Timeout: 15 * time.Second, // 降低超时，避免长时间等待
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 强制将所有连接转换为 IPv4
				return dialer.DialContext(ctx, "tcp4", addr)
			},
		},
	}
	cfg := &duckduckgo.Config{
		Region: duckduckgo.RegionWT,
		// Timeout:    time.Duration(600 * time.Second),
		MaxResults: 5,
		HTTPClient: ipv4HTTPClient, // 使用强制 IPv4 的 Client
	}
	searchTool, err := duckduckgo.NewTextSearchTool(context.Background(), cfg)
	if err != nil {
		// 使用 log.Printf 代替 log.Fatalf，避免进程退出
		log.Printf("NewTextSearchTool of duckduckgo failed, err=%v, search tool will be disabled", err)
		// searchTool 保持为 nil，在 Tools 配置时跳过
	}

	// 使用httprequest 作为搜索工具

	// 构建 Tools 列表，仅在 searchTool 初始化成功时添加
	var tools []tool.BaseTool
	if searchTool != nil {
		tools = append(tools, searchTool)
	}

	chatAgent, err := adk.NewChatModelAgent(a.Ctx, &adk.ChatModelAgentConfig{
		Name:        "intelligent_assistant",
		Description: "An intelligent assistant capable of using multiple tools to solve complex problems",
		Instruction: "You are a professional assistant who can use the provided tools to help users solve problems",
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})

	return chatAgent
}

// 通用对话
func (a *Agent) CommonChat(query string) (string, error) {

	// 及时恢复服务
	defer func() {
		if r := recover(); r != nil {
			log.Println("panic:", r)
		}
	}()

	// Init Agent runner
	runner := adk.NewRunner(a.Ctx, adk.RunnerConfig{
		Agent: a.ChatModelAgent(),
		// enable stream output
		EnableStreaming: false,
		// enable checkpoint for interrupt & resume
		// CheckPointStore: newInMemoryStore(),
	})

	// Start runner with a new checkpoint id
	checkpointID := "1"
	iter := runner.Query(a.Ctx, query, adk.WithCheckPointID(checkpointID))
	var result string
	var err error
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			err = event.Err
			log.Printf("Agent run error: %v", event.Err)
			break // 不要 exit，改为 break 退出循环
		}
		prints.Event(event)
		if event.Output != nil {
			result = event.Output.MessageOutput.Message.Content
		}
	}
	return result, err
}
