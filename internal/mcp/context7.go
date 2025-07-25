package mcp

import (
	"context"
	"errors"
	"time"

	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// NewContext7Client 创建新的 Context7 客户端
func NewContext7Client(config ClientConfig) *Context7Client {
	if config.ServerName == "" {
		config.ServerName = "context7"
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
	
	return &Context7Client{
		config:    config,
		connected: false,
	}
}

// Initialize 初始化 Context7 客户端
func (c *Context7Client) Initialize(ctx context.Context) error {
	if c.connected {
		return nil
	}
	
	// 在实际实现中，这里会启动 MCP 服务器进程并建立连接
	// 目前先模拟连接成功
	c.connected = true
	return nil
}

// ListTools 获取 Context7 可用工具列表
func (c *Context7Client) ListTools(ctx context.Context) ([]Tool, error) {
	if !c.connected {
		return nil, WrapError("listTools", c.config.ServerName, ErrClientNotConnected)
	}
	
	tools := []Tool{
		{
			Name:        "resolve-library-id",
			Description: "Resolve library name to Context7-compatible library ID",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"libraryName": map[string]interface{}{
						"type":        "string",
						"description": "Library name to search for and retrieve a Context7-compatible library ID",
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
						"description": "Exact Context7-compatible library ID from resolve-library-id",
					},
					"tokens": map[string]interface{}{
						"type":        "number",
						"description": "Maximum number of tokens to retrieve (default: 10000)",
					},
					"topic": map[string]interface{}{
						"type":        "string",
						"description": "Topic to focus documentation on",
					},
				},
				"required": []string{"context7CompatibleLibraryID"},
			},
		},
	}
	
	return tools, nil
}

// CallTool 调用 Context7 工具
func (c *Context7Client) CallTool(ctx context.Context, name string, args interface{}) (*ToolResult, error) {
	if !c.connected {
		return nil, WrapToolError("callTool", c.config.ServerName, name, ErrClientNotConnected)
	}
	
	switch name {
	case "resolve-library-id":
		return c.callResolveLibraryID(ctx, args)
	case "get-library-docs":
		return c.callGetLibraryDocs(ctx, args)
	default:
		return nil, WrapToolError("callTool", c.config.ServerName, name, ErrToolNotFound)
	}
}

// callResolveLibraryID 调用库 ID 解析工具
func (c *Context7Client) callResolveLibraryID(ctx context.Context, args interface{}) (*ToolResult, error) {
	// 验证参数类型
	var resolveArgs Context7ResolveArgs
	switch v := args.(type) {
	case Context7ResolveArgs:
		resolveArgs = v
	case map[string]interface{}:
		libraryName, ok := v["libraryName"].(string)
		if !ok {
			return nil, WrapToolError("callTool", c.config.ServerName, "resolve-library-id", 
				errors.New("missing required parameter: libraryName"))
		}
		resolveArgs = Context7ResolveArgs{LibraryName: libraryName}
	default:
		return nil, WrapToolError("callTool", c.config.ServerName, "resolve-library-id", ErrInvalidArgs)
	}
	
	// 执行重试逻辑
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result, err := c.executeResolveLibraryID(ctx, resolveArgs)
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
				return nil, WrapToolError("callTool", c.config.ServerName, "resolve-library-id", ctx.Err())
			case <-time.After(c.config.RetryDelay * time.Duration(attempt+1)):
				// 指数退避
			}
		}
	}
	
	return nil, WrapToolError("callTool", c.config.ServerName, "resolve-library-id", 
		ErrMaxRetriesExceeded)
}

// callGetLibraryDocs 调用获取库文档工具
func (c *Context7Client) callGetLibraryDocs(ctx context.Context, args interface{}) (*ToolResult, error) {
	// 验证参数类型
	var docsArgs Context7DocsArgs
	switch v := args.(type) {
	case Context7DocsArgs:
		docsArgs = v
	case map[string]interface{}:
		libraryID, ok := v["context7CompatibleLibraryID"].(string)
		if !ok {
			return nil, WrapToolError("callTool", c.config.ServerName, "get-library-docs", 
				errors.New("missing required parameter: context7CompatibleLibraryID"))
		}
		docsArgs = Context7DocsArgs{Context7CompatibleLibraryID: libraryID}
		
		if tokens, ok := v["tokens"].(int); ok {
			docsArgs.Tokens = tokens
		} else {
			docsArgs.Tokens = 10000 // 默认值
		}
		
		if topic, ok := v["topic"].(string); ok {
			docsArgs.Topic = topic
		}
	default:
		return nil, WrapToolError("callTool", c.config.ServerName, "get-library-docs", ErrInvalidArgs)
	}
	
	// 执行重试逻辑
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result, err := c.executeGetLibraryDocs(ctx, docsArgs)
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
				return nil, WrapToolError("callTool", c.config.ServerName, "get-library-docs", ctx.Err())
			case <-time.After(c.config.RetryDelay * time.Duration(attempt+1)):
				// 指数退避
			}
		}
	}
	
	return nil, WrapToolError("callTool", c.config.ServerName, "get-library-docs", 
		ErrMaxRetriesExceeded)
}

// executeResolveLibraryID 执行库 ID 解析
func (c *Context7Client) executeResolveLibraryID(ctx context.Context, args Context7ResolveArgs) (*ToolResult, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	
	// 在实际实现中，这里会调用真实的 MCP 客户端
	// 目前返回模拟结果
	select {
	case <-timeoutCtx.Done():
		return nil, NewMCPError("executeResolveLibraryID", c.config.ServerName, "resolve-library-id", 
			ErrCallTimeout, true)
	default:
		// 模拟处理时间
		time.Sleep(100 * time.Millisecond)
		
		// 根据库名称返回模拟的库 ID
		var libraryID string
		switch args.LibraryName {
		case "golang", "go":
			libraryID = "/golang/go"
		case "gin", "gin-gonic":
			libraryID = "/gin-gonic/gin"
		case "gorilla/mux":
			libraryID = "/gorilla/mux"
		default:
			libraryID = "/unknown/library"
		}
		
		result := &ToolResult{
			Content: []Content{
				{
					Type: "text",
					Text: `Selected Library: ` + libraryID + `

This library was chosen based on the search query "` + args.LibraryName + `".

Trust Score: 8.5
Documentation Coverage: High
Code Snippets Available: 500+`,
				},
			},
			IsError: false,
		}
		
		return result, nil
	}
}

// executeGetLibraryDocs 执行获取库文档
func (c *Context7Client) executeGetLibraryDocs(ctx context.Context, args Context7DocsArgs) (*ToolResult, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	
	// 在实际实现中，这里会调用真实的 MCP 客户端
	// 目前返回模拟结果
	select {
	case <-timeoutCtx.Done():
		return nil, NewMCPError("executeGetLibraryDocs", c.config.ServerName, "get-library-docs", 
			ErrCallTimeout, true)
	default:
		// 模拟处理时间
		time.Sleep(200 * time.Millisecond)
		
		topicText := ""
		if args.Topic != "" {
			topicText = " (focused on " + args.Topic + ")"
		}
		
		result := &ToolResult{
			Content: []Content{
				{
					Type: "text",
					Text: `# Library Documentation` + topicText + `

## Library: ` + args.Context7CompatibleLibraryID + `

## Overview
This is comprehensive documentation for the library, retrieved from Context7.

## Key Features
- Feature 1: Core functionality
- Feature 2: Advanced usage patterns
- Feature 3: Best practices

## Code Examples
` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello from library!")
}
` + "```" + `

## API Reference
Detailed API documentation with parameters, return values, and usage examples.

*Documentation retrieved with ` + "10000" + ` token limit*`,
				},
			},
			IsError: false,
		}
		
		return result, nil
	}
}

// Close 关闭 Context7 客户端连接
func (c *Context7Client) Close() error {
	if !c.connected {
		return nil
	}
	
	c.connected = false
	return nil
}

// IsConnected 检查 Context7 客户端连接状态
func (c *Context7Client) IsConnected() bool {
	return c.connected
}

// createStdioTransport 创建 stdio 传输层（内部使用）
func (c *Context7Client) createStdioTransport() (*stdio.StdioServerTransport, error) {
	// 在实际实现中，这里会启动外部进程并创建 stdio 传输
	return nil, errors.New("not implemented yet")
}

// createMCPClient 创建 MCP 客户端（内部使用）
func (c *Context7Client) createMCPClient(transport *stdio.StdioServerTransport) *mcp_golang.Client {
	// 在实际实现中，这里会创建真正的 MCP 客户端
	return mcp_golang.NewClient(transport)
}