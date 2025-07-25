package mcp

import (
	"context"

	"github.com/PerceptivePenguin/MCPRAG-Go/pkg/types"
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
	types.BaseConfig
	
	// MCP 特定配置
	ServerName  string   `json:"server_name" yaml:"server_name"`
	Transport   string   `json:"transport" yaml:"transport"`     // stdio, http, sse
	Command     string   `json:"command,omitempty" yaml:"command,omitempty"`
	Args        []string `json:"args,omitempty" yaml:"args,omitempty"`
	BaseURL     string   `json:"base_url,omitempty" yaml:"base_url,omitempty"`
}

// DefaultClientConfig 返回默认的客户端配置
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseConfig: types.DefaultBaseConfig(),
		Transport:  "stdio",
	}
}

// NewSequentialThinkingConfig 创建Sequential Thinking服务器配置
func NewSequentialThinkingConfig() ClientConfig {
	return ClientConfig{
		BaseConfig: types.DefaultBaseConfig(),
		ServerName: "sequential-thinking",
		Transport:  "stdio",
		Command:    "npx",
		Args:       []string{"@modelcontextprotocol/server-sequential-thinking"},
	}
}

// NewDeepWikiConfig 创建DeepWiki服务器配置
func NewDeepWikiConfig() ClientConfig {
	return ClientConfig{
		BaseConfig: types.DefaultBaseConfig(),
		ServerName: "deepwiki",
		Transport:  "stdio",
		Command:    "npx",
		Args:       []string{"mcp-deepwiki@latest"},
	}
}

// NewContext7Config 创建Context7服务器配置
func NewContext7Config() ClientConfig {
	return ClientConfig{
		BaseConfig: types.DefaultBaseConfig(),
		ServerName: "context7",
		Transport:  "stdio",
		Command:    "npx",
		Args:       []string{"@upstash/context7-mcp@latest"},
	}
}

// 注意: 删除了具体的客户端类型(DeepWikiClient, Context7Client)和参数类型
// 因为统一客户端使用interface{}接收参数，由MCP协议动态验证
// 这样更灵活，符合原TypeScript版本的设计理念