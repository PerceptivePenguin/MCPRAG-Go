package mcp

import (
	"context"
	"time"
)

// MCPClient 定义了通用的 MCP 客户端接口
type MCPClient interface {
	// Initialize 初始化客户端连接
	Initialize(ctx context.Context) error
	
	// ListTools 获取可用工具列表
	ListTools(ctx context.Context) ([]Tool, error)
	
	// CallTool 调用指定工具
	CallTool(ctx context.Context, name string, args interface{}) (*ToolResult, error)
	
	// Close 关闭客户端连接
	Close() error
	
	// IsConnected 检查连接状态
	IsConnected() bool
}

// Tool 表示一个可用的工具
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolResult 表示工具调用的结果
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError"`
}

// Content 表示工具返回的内容
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClientConfig 客户端配置
type ClientConfig struct {
	ServerName  string        `json:"serverName"`
	Transport   string        `json:"transport"`
	Command     string        `json:"command,omitempty"`
	Args        []string      `json:"args,omitempty"`
	BaseURL     string        `json:"baseUrl,omitempty"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetries  int           `json:"maxRetries"`
	RetryDelay  time.Duration `json:"retryDelay"`
}

// DefaultClientConfig 返回默认的客户端配置
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Transport:  "stdio",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// DeepWikiClient DeepWiki 服务器客户端
type DeepWikiClient struct {
	config    ClientConfig
	connected bool
}

// Context7Client Context7 服务器客户端
type Context7Client struct {
	config    ClientConfig
	connected bool
}

// DeepWikiArgs DeepWiki 工具调用参数
type DeepWikiArgs struct {
	URL      string `json:"url" jsonschema:"required,description=URL, owner/repo name, or keyword to fetch"`
	MaxDepth int    `json:"maxDepth,omitempty" jsonschema:"description=Fetch depth (0-1), default is 1"`
	Mode     string `json:"mode,omitempty" jsonschema:"description=Output mode: aggregate or pages, default is aggregate"`
	Verbose  bool   `json:"verbose,omitempty" jsonschema:"description=Enable verbose output"`
}

// Context7ResolveArgs Context7 库 ID 解析参数
type Context7ResolveArgs struct {
	LibraryName string `json:"libraryName" jsonschema:"required,description=Library name to search for"`
}

// Context7DocsArgs Context7 文档获取参数
type Context7DocsArgs struct {
	Context7CompatibleLibraryID string  `json:"context7CompatibleLibraryID" jsonschema:"required,description=Context7-compatible library ID"`
	Tokens                      int     `json:"tokens,omitempty" jsonschema:"description=Max tokens to retrieve, default 10000"`
	Topic                       string  `json:"topic,omitempty" jsonschema:"description=Topic to focus on"`
}