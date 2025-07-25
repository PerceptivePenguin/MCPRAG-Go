package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// Client 通用的MCP客户端实现
// 基于原始TypeScript版本的设计，支持配置驱动的多服务器连接
type Client struct {
	config      ClientConfig
	tools       []Tool
	connected   bool
	mu          sync.RWMutex
	mcpClient   *mcp_golang.Client
	serverCmd   *exec.Cmd
	transport   *stdio.StdioServerTransport
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
func (c *Client) initializeStdio(ctx context.Context) error {
	// 1. 启动外部MCP服务器进程
	if c.config.Command == "" {
		return WrapError("initializeStdio", c.config.ServerName, 
			fmt.Errorf("server command not specified"))
	}

	// 创建服务器进程
	c.serverCmd = exec.CommandContext(ctx, c.config.Command, c.config.Args...)
	
	// 设置标准输入/输出管道
	stdout, err := c.serverCmd.StdoutPipe()
	if err != nil {
		return WrapError("initializeStdio", c.config.ServerName, 
			fmt.Errorf("failed to create stdout pipe: %w", err))
	}
	
	stdin, err := c.serverCmd.StdinPipe()
	if err != nil {
		return WrapError("initializeStdio", c.config.ServerName, 
			fmt.Errorf("failed to create stdin pipe: %w", err))
	}
	
	// 启动服务器进程
	if err := c.serverCmd.Start(); err != nil {
		return WrapError("initializeStdio", c.config.ServerName, 
			fmt.Errorf("failed to start server: %w", err))
	}
	
	// 创建stdio传输层
	transport := stdio.NewStdioServerTransportWithIO(stdout, stdin)
	c.transport = transport

	// 3. 建立MCP协议连接
	c.mcpClient = mcp_golang.NewClient(transport)
	
	// 初始化客户端连接
	_, err = c.mcpClient.Initialize(ctx)
	if err != nil {
		c.cleanup()
		return WrapError("initializeStdio", c.config.ServerName, 
			fmt.Errorf("failed to initialize MCP client: %w", err))
	}

	// 4. 动态发现工具列表
	if err := c.discoverTools(ctx); err != nil {
		c.cleanup()
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

// discoverTools 通过MCP协议动态发现工具列表
// 对应TypeScript版本中的工具发现逻辑
func (c *Client) discoverTools(ctx context.Context) error {
	// 通过MCP协议列出可用工具
	toolsResult, err := c.mcpClient.ListTools(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	// 将MCP工具转换为内部Tool结构
	tools := make([]Tool, len(toolsResult.Tools))
	for i, mcpTool := range toolsResult.Tools {
		description := ""
		if mcpTool.Description != nil {
			description = *mcpTool.Description
		}
		
		// 类型断言将interface{}转换为map[string]interface{}
		inputSchema := make(map[string]interface{})
		if mcpTool.InputSchema != nil {
			if schema, ok := mcpTool.InputSchema.(map[string]interface{}); ok {
				inputSchema = schema
			}
		}
		
		tools[i] = Tool{
			Name:        mcpTool.Name,
			Description: description,
			InputSchema: inputSchema,
		}
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
// 对应TypeScript版本的callTool()方法
func (c *Client) executeToolCall(ctx context.Context, name string, args interface{}) (*ToolResult, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// 通过MCP协议调用真实工具
	mcpResult, err := c.mcpClient.CallTool(timeoutCtx, name, args)
	if err != nil {
		return nil, WrapToolError("executeToolCall", c.config.ServerName, name, 
			fmt.Errorf("MCP tool call failed: %w", err))
	}

	// 将MCP结果转换为内部ToolResult结构
	content := make([]Content, len(mcpResult.Content))
	for i, mcpContent := range mcpResult.Content {
		text := ""
		if mcpContent.TextContent != nil {
			text = mcpContent.TextContent.Text
		}
		
		content[i] = Content{
			Type: string(mcpContent.Type),
			Text: text,
		}
	}

	// MCP响应默认不是错误，除非有特殊标记
	isError := false

	return &ToolResult{
		Content: content,
		IsError: isError,
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

	c.cleanup()
	c.connected = false
	return nil
}

// cleanup 清理所有资源
func (c *Client) cleanup() {
	// 1. 关闭MCP客户端连接
	if c.mcpClient != nil {
		// MCP客户端会自动关闭传输层
		c.mcpClient = nil
	}

	// 2. 终止服务器进程
	if c.serverCmd != nil && c.serverCmd.Process != nil {
		c.serverCmd.Process.Kill()
		c.serverCmd.Wait() // 等待进程完全退出
		c.serverCmd = nil
	}

	// 3. 清理传输资源
	c.transport = nil
	c.tools = nil
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

	return c.discoverTools(ctx)
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