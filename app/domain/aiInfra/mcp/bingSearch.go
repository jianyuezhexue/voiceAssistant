package mcp

import (
	"context"
	"fmt"
	"log"

	mcpp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// BingSearchConfig holds the configuration for Bing Search MCP client.
type BingSearchConfig struct {
	Endpoint  string // MCP server endpoint URL, e.g., "http://localhost:9090/bing-search/mcp"
	AuthToken string // Authentication token
}

// NewBingSearchTools creates Bing search tools from MCP server.
// It connects to the MCP server, initializes, and returns all available tools.
func NewBingSearchTools(ctx context.Context) ([]tool.BaseTool, error) {
	config := &BingSearchConfig{
		Endpoint:  "http://localhost:9090/bing-search/mcp",
		AuthToken: "DefaultTokens",
	}

	log.Printf("[BingSearch] Connecting to MCP server at %s", config.Endpoint)

	// Build headers
	headers := make(map[string]string)
	if config.AuthToken != "" {
		headers["Authorization"] = "Bearer " + config.AuthToken
	}

	// Create HTTP client with auth headers (use streamable HTTP for MCP)
	cli, err := client.NewStreamableHttpClient(config.Endpoint,
		transport.WithHTTPHeaders(headers),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Start the client connection
	err = cli.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start MCP client: %w", err)
	}

	// Initialize MCP protocol handshake
	_, err = cli.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "bing-search-client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	// Use official eino-ext MCP adapter to get tools
	tools, err := mcpp.GetTools(ctx, &mcpp.Config{
		Cli: cli,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP tools: %w", err)
	}

	log.Printf("[BingSearch] Successfully loaded %d tools from MCP server", len(tools))
	return tools, nil
}
