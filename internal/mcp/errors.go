package mcp

import (
	"errors"
	"fmt"
)

var (
	// ErrClientNotInitialized 客户端未初始化错误
	ErrClientNotInitialized = errors.New("mcp client not initialized")
	
	// ErrClientNotConnected 客户端未连接错误
	ErrClientNotConnected = errors.New("mcp client not connected")
	
	// ErrToolNotFound 工具未找到错误
	ErrToolNotFound = errors.New("tool not found")
	
	// ErrInvalidArgs 参数无效错误
	ErrInvalidArgs = errors.New("invalid arguments")
	
	// ErrCallTimeout 调用超时错误
	ErrCallTimeout = errors.New("tool call timeout")
	
	// ErrConnectionFailed 连接失败错误
	ErrConnectionFailed = errors.New("connection failed")
	
	// ErrMaxRetriesExceeded 超过最大重试次数错误
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	
	// ErrInvalidConfig 配置无效错误
	ErrInvalidConfig = errors.New("invalid configuration")
)

// MCPError MCP 错误类型
type MCPError struct {
	Op       string // 操作名称
	Server   string // 服务器名称
	Tool     string // 工具名称
	Err      error  // 原始错误
	Retryable bool  // 是否可重试
}

// Error 实现 error 接口
func (e *MCPError) Error() string {
	if e.Tool != "" {
		return fmt.Sprintf("mcp %s %s.%s: %v", e.Op, e.Server, e.Tool, e.Err)
	}
	return fmt.Sprintf("mcp %s %s: %v", e.Op, e.Server, e.Err)
}

// Unwrap 支持 errors.Is 和 errors.As
func (e *MCPError) Unwrap() error {
	return e.Err
}

// IsRetryable 检查错误是否可重试
func (e *MCPError) IsRetryable() bool {
	return e.Retryable
}

// NewMCPError 创建新的 MCP 错误
func NewMCPError(op, server, tool string, err error, retryable bool) *MCPError {
	return &MCPError{
		Op:        op,
		Server:    server,
		Tool:      tool,
		Err:       err,
		Retryable: retryable,
	}
}

// WrapError 包装错误为 MCPError
func WrapError(op, server string, err error) *MCPError {
	return NewMCPError(op, server, "", err, false)
}

// WrapToolError 包装工具错误为 MCPError
func WrapToolError(op, server, tool string, err error) *MCPError {
	return NewMCPError(op, server, tool, err, false)
}

// WrapRetryableError 包装可重试错误为 MCPError
func WrapRetryableError(op, server string, err error) *MCPError {
	return NewMCPError(op, server, "", err, true)
}