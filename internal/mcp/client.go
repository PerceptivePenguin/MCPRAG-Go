package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Client 通用的MCP客户端实现
// 基于原始TypeScript版本的设计，支持配置驱动的多服务器连接
type Client struct {
	config    ClientConfig
	tools     []Tool
	connected bool
	mu        sync.RWMutex
}

// NewClient 创建新的MCP客户端
// 参数配置决定连接到哪个MCP服务器
func NewClient(config ClientConfig) *Client {
	// 设置默认值
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
	if config.Transport == "" {
		config.Transport = "stdio"
	}

	return &Client{
		config:    config,
		connected: false,
		tools:     make([]Tool, 0),
	}
}

// Initialize 初始化客户端连接
// 对应TypeScript版本的init()方法
func (c *Client) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	switch c.config.Transport {
	case "stdio":
		return c.initializeStdio(timeoutCtx)
	case "http":
		return c.initializeHTTP(timeoutCtx)
	case "sse":
		return c.initializeSSE(timeoutCtx)
	default:
		return WrapError("initialize", c.config.ServerName, 
			fmt.Errorf("unsupported transport: %s", c.config.Transport))
	}
}

// initializeStdio 初始化stdio传输连接
// 对应TypeScript版本的connectToServer()方法
// 注意：当前为架构重构版本，使用模拟实现
func (c *Client) initializeStdio(ctx context.Context) error {
	// TODO: 在实际实现中，这里会：
	// 1. 启动外部MCP服务器进程 (c.config.Command, c.config.Args)
	// 2. 创建stdio传输层
	// 3. 建立MCP协议连接
	// 4. 动态发现工具列表
	
	// 当前提供模拟实现以验证架构正确性
	select {
	case <-ctx.Done():
		return WrapError("initializeStdio", c.config.ServerName, ctx.Err())
	case <-time.After(100 * time.Millisecond):
		// 模拟连接延迟
	}
	
	// 模拟动态工具发现
	if err := c.mockDiscoverTools(ctx); err != nil {
		return WrapError("initializeStdio", c.config.ServerName, 
			fmt.Errorf("failed to discover tools: %w", err))
	}

	c.connected = true
	return nil
}

// initializeHTTP 初始化HTTP传输连接
func (c *Client) initializeHTTP(ctx context.Context) error {
	// TODO: 实现HTTP传输支持
	return WrapError("initializeHTTP", c.config.ServerName, 
		fmt.Errorf("HTTP transport not implemented yet"))
}

// initializeSSE 初始化SSE传输连接
func (c *Client) initializeSSE(ctx context.Context) error {
	// TODO: 实现SSE传输支持
	return WrapError("initializeSSE", c.config.ServerName, 
		fmt.Errorf("SSE transport not implemented yet"))
}

// mockDiscoverTools 模拟动态工具发现
// 基于服务器名称返回相应的工具列表
// 实际实现中会通过MCP协议动态获取
func (c *Client) mockDiscoverTools(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 根据服务器类型模拟相应的工具
	var tools []Tool
	switch c.config.ServerName {
	case "deepwiki":
		tools = []Tool{
			{
				Name:        "deepwiki_fetch",
				Description: "Fetch technical documentation from repositories",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"url": map[string]interface{}{
							"type":        "string",
							"description": "URL, owner/repo name, or keyword to fetch",
						},
					},
					"required": []string{"url"},
				},
			},
		}
	case "context7":
		tools = []Tool{
			{
				Name:        "resolve-library-id",
				Description: "Resolve library name to Context7-compatible library ID",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"libraryName": map[string]interface{}{
							"type":        "string",
							"description": "Library name to search for",
						},
					},
					"required": []string{"libraryName"},
				},
			},
			{
				Name:        "get-library-docs",
				Description: "Fetch up-to-date documentation for a library",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"context7CompatibleLibraryID": map[string]interface{}{
							"type":        "string",
							"description": "Context7-compatible library ID",
						},
					},
					"required": []string{"context7CompatibleLibraryID"},
				},
			},
		}
	case "sequential-thinking":
		tools = []Tool{
			{
				Name:        "sequentialthinking",
				Description: "Structured step-by-step reasoning",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"thought": map[string]interface{}{
							"type":        "string",
							"description": "Current thinking step",
						},
					},
					"required": []string{"thought"},
				},
			},
		}
	default:
		// 未知服务器，返回空工具列表
		tools = []Tool{}
	}

	c.tools = tools
	return nil
}

// ListTools 获取可用工具列表
// 对应TypeScript版本的getTools()方法
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, WrapError("listTools", c.config.ServerName, ErrClientNotConnected)
	}

	// 返回工具列表的副本
	tools := make([]Tool, len(c.tools))
	copy(tools, c.tools)
	return tools, nil
}

// CallTool 调用指定工具
// 对应TypeScript版本的callTool()方法，支持重试机制
func (c *Client) CallTool(ctx context.Context, name string, args interface{}) (*ToolResult, error) {
	c.mu.RLock()
	connected := c.connected
	c.mu.RUnlock()

	if !connected {
		return nil, WrapToolError("callTool", c.config.ServerName, name, ErrClientNotConnected)
	}

	// 验证工具是否存在
	if !c.hasToolNamed(name) {
		return nil, WrapToolError("callTool", c.config.ServerName, name, ErrToolNotFound)
	}

	// 执行重试逻辑
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result, err := c.executeToolCall(ctx, name, args)
		if err == nil {
			return result, nil
		}

		// 检查是否为可重试错误
		if mcpErr, ok := err.(*MCPError); ok && !mcpErr.IsRetryable() {
			break
		}

		// 如果不是最后一次尝试，等待后重试
		if attempt < c.config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, WrapToolError("callTool", c.config.ServerName, name, ctx.Err())
			case <-time.After(c.config.RetryDelay * time.Duration(attempt+1)):
				// 指数退避
			}
		}
	}

	return nil, WrapToolError("callTool", c.config.ServerName, name, ErrMaxRetriesExceeded)
}

// executeToolCall 执行单次工具调用
// 当前为模拟实现，验证架构正确性
func (c *Client) executeToolCall(ctx context.Context, name string, args interface{}) (*ToolResult, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		return nil, NewMCPError("executeToolCall", c.config.ServerName, name, 
			ErrCallTimeout, true)
	case <-time.After(50 * time.Millisecond):
		// 模拟工具执行时间
	}

	// TODO: 实际实现中会通过MCP协议调用真实工具
	// 当前提供基于工具名称的模拟响应
	content := []Content{
		{
			Type: "text",
			Text: fmt.Sprintf("Mock result from tool '%s' on server '%s'", 
				name, c.config.ServerName),
		},
	}

	return &ToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// hasToolNamed 检查是否有指定名称的工具
func (c *Client) hasToolNamed(name string) bool {
	for _, tool := range c.tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

// Close 关闭客户端连接
// 对应TypeScript版本的close()方法
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	// TODO: 实际实现中会：
	// 1. 关闭MCP客户端连接
	// 2. 终止服务器进程
	// 3. 清理传输资源
	
	// 模拟清理过程
	time.Sleep(10 * time.Millisecond)

	c.connected = false
	c.tools = nil
	return nil
}

// IsConnected 检查连接状态
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// RefreshTools 刷新工具列表
// 新增功能：支持运行时刷新工具列表
func (c *Client) RefreshTools(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return WrapError("refreshTools", c.config.ServerName, ErrClientNotConnected)
	}

	return c.mockDiscoverTools(ctx)
}

// GetConfig 获取客户端配置
func (c *Client) GetConfig() ClientConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// GetServerName 获取服务器名称
func (c *Client) GetServerName() string {
	return c.config.ServerName
}