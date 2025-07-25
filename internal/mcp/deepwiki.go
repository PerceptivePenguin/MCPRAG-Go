package mcp

import (
	"context"
	"errors"
	"time"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// NewDeepWikiClient 创建新的 DeepWiki 客户端
func NewDeepWikiClient(config ClientConfig) *DeepWikiClient {
	if config.ServerName == "" {
		config.ServerName = "deepwiki"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second
	}
	
	return &DeepWikiClient{
		config:    config,
		connected: false,
	}
}

// Initialize 初始化 DeepWiki 客户端
func (c *DeepWikiClient) Initialize(ctx context.Context) error {
	if c.connected {
		return nil
	}
	
	// 在实际实现中，这里会启动 MCP 服务器进程并建立连接
	// 目前先模拟连接成功
	c.connected = true
	return nil
}

// ListTools 获取 DeepWiki 可用工具列表
func (c *DeepWikiClient) ListTools(ctx context.Context) ([]Tool, error) {
	if !c.connected {
		return nil, WrapError("listTools", c.config.ServerName, ErrClientNotConnected)
	}
	
	tools := []Tool{
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
					"maxDepth": map[string]interface{}{
						"type":        "integer",
						"description": "Fetch depth (0-1), default is 1",
						"minimum":     0,
						"maximum":     1,
					},
					"mode": map[string]interface{}{
						"type":        "string",
						"description": "Output mode: aggregate or pages, default is aggregate",
						"enum":        []string{"aggregate", "pages"},
					},
					"verbose": map[string]interface{}{
						"type":        "boolean",
						"description": "Enable verbose output",
					},
				},
				"required": []string{"url"},
			},
		},
	}
	
	return tools, nil
}

// CallTool 调用 DeepWiki 工具
func (c *DeepWikiClient) CallTool(ctx context.Context, name string, args interface{}) (*ToolResult, error) {
	if !c.connected {
		return nil, WrapToolError("callTool", c.config.ServerName, name, ErrClientNotConnected)
	}
	
	if name != "deepwiki_fetch" {
		return nil, WrapToolError("callTool", c.config.ServerName, name, ErrToolNotFound)
	}
	
	// 验证参数类型
	deepwikiArgs, ok := args.(DeepWikiArgs)
	if !ok {
		// 尝试从 map 转换
		if argsMap, ok := args.(map[string]interface{}); ok {
			url, ok := argsMap["url"].(string)
			if !ok {
				return nil, WrapToolError("callTool", c.config.ServerName, name, 
					errors.New("missing required parameter: url"))
			}
			
			deepwikiArgs = DeepWikiArgs{URL: url}
			
			if maxDepth, ok := argsMap["maxDepth"].(int); ok {
				deepwikiArgs.MaxDepth = maxDepth
			} else {
				deepwikiArgs.MaxDepth = 1 // 默认值
			}
			
			if mode, ok := argsMap["mode"].(string); ok {
				deepwikiArgs.Mode = mode
			} else {
				deepwikiArgs.Mode = "aggregate" // 默认值
			}
			
			if verbose, ok := argsMap["verbose"].(bool); ok {
				deepwikiArgs.Verbose = verbose
			}
		} else {
			return nil, WrapToolError("callTool", c.config.ServerName, name, ErrInvalidArgs)
		}
	}
	
	// 执行重试逻辑
	var result *ToolResult
	var err error
	
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result, err = c.executeDeepWikiFetch(ctx, deepwikiArgs)
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
	
	return nil, WrapToolError("callTool", c.config.ServerName, name, 
		errors.Join(ErrMaxRetriesExceeded, err))
}

// executeDeepWikiFetch 执行 DeepWiki 抓取
func (c *DeepWikiClient) executeDeepWikiFetch(ctx context.Context, args DeepWikiArgs) (*ToolResult, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	
	// 在实际实现中，这里会调用真实的 MCP 客户端
	// 目前返回模拟结果
	select {
	case <-timeoutCtx.Done():
		return nil, NewMCPError("executeDeepWikiFetch", c.config.ServerName, "deepwiki_fetch", 
			ErrCallTimeout, true)
	default:
		// 模拟处理时间
		time.Sleep(100 * time.Millisecond)
		
		result := &ToolResult{
			Content: []Content{
				{
					Type: "text",
					Text: `# Repository Documentation

## Overview
This is a sample documentation fetched from ` + args.URL + `

## Key Features
- Feature 1: Documentation extraction
- Feature 2: Markdown formatting
- Feature 3: Structured output

## Usage
The repository provides various functionalities for documentation processing.`,
				},
			},
			IsError: false,
		}
		
		return result, nil
	}
}

// Close 关闭 DeepWiki 客户端连接
func (c *DeepWikiClient) Close() error {
	if !c.connected {
		return nil
	}
	
	c.connected = false
	return nil
}

// IsConnected 检查 DeepWiki 客户端连接状态
func (c *DeepWikiClient) IsConnected() bool {
	return c.connected
}

// createStdioTransport 创建 stdio 传输层（内部使用）
func (c *DeepWikiClient) createStdioTransport() (*stdio.StdioServerTransport, error) {
	// 在实际实现中，这里会启动外部进程并创建 stdio 传输
	return nil, errors.New("not implemented yet")
}

// createMCPClient 创建 MCP 客户端（内部使用）
func (c *DeepWikiClient) createMCPClient(transport *stdio.StdioServerTransport) *mcp_golang.Client {
	// 在实际实现中，这里会创建真正的 MCP 客户端
	return mcp_golang.NewClient(transport)
}