package mcp

import (
	pkgerrors "github.com/PerceptivePenguin/MCPRAG-Go/pkg/errors"
)

var (
	// MCP 特定错误，基于pkg/errors的通用错误类型
	ErrClientNotInitialized = pkgerrors.NewError(pkgerrors.ErrorTypeInternal, "mcp client not initialized")
	ErrClientNotConnected   = pkgerrors.NewError(pkgerrors.ErrorTypeNetwork, "mcp client not connected")
	ErrToolNotFound         = pkgerrors.NewError(pkgerrors.ErrorTypeNotFound, "tool not found")
	ErrInvalidArgs          = pkgerrors.NewError(pkgerrors.ErrorTypeValidation, "invalid arguments")
	ErrCallTimeout          = pkgerrors.NewError(pkgerrors.ErrorTypeTimeout, "tool call timeout")
	ErrConnectionFailed     = pkgerrors.NewError(pkgerrors.ErrorTypeNetwork, "connection failed")
	ErrMaxRetriesExceeded   = pkgerrors.NewError(pkgerrors.ErrorTypeCapacity, "max retries exceeded")
	ErrInvalidConfig        = pkgerrors.NewError(pkgerrors.ErrorTypeConfiguration, "invalid configuration")
)

// 使用pkg/errors中的统一错误类型
type MCPError = pkgerrors.BaseError

// NewMCPError 创建新的 MCP 错误
func NewMCPError(op, server, tool string, err error, errorType pkgerrors.ErrorType) *MCPError {
	details := map[string]string{"server": server}
	if tool != "" {
		details["tool"] = tool
	}
	
	if err != nil {
		return pkgerrors.NewErrorWithCause(errorType, "mcp operation failed", err).WithOperation(op).WithComponent("mcp").WithDetails(details)
	}
	return pkgerrors.NewError(errorType, "mcp operation failed").WithOperation(op).WithComponent("mcp").WithDetails(details)
}

// WrapConnectionError 包装连接错误为 MCPError
func WrapConnectionError(op, server string, err error) *MCPError {
	return NewMCPError(op, server, "", err, pkgerrors.ErrorTypeNetwork)
}

// WrapToolError 包装工具错误为 MCPError
func WrapToolError(op, server, tool string, err error) *MCPError {
	return NewMCPError(op, server, tool, err, pkgerrors.ErrorTypeExternal)
}

// WrapTimeoutError 包装超时错误为 MCPError
func WrapTimeoutError(op, server string, err error) *MCPError {
	return NewMCPError(op, server, "", err, pkgerrors.ErrorTypeTimeout)
}

// WrapError 包装通用错误为 MCPError (向后兼容)
func WrapError(op, server string, err error) *MCPError {
	return NewMCPError(op, server, "", err, pkgerrors.ErrorTypeInternal)
}