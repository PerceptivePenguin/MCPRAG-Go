package agent

import (
	"time"

	pkgerrors "github.com/PerceptivePenguin/MCPRAG-Go/pkg/errors"
)

var (
	// Agent 特定错误，基于pkg/errors的通用错误类型
	ErrAgentNotStarted         = pkgerrors.NewError(pkgerrors.ErrorTypeInternal, "agent not started")
	ErrAgentAlreadyStarted     = pkgerrors.NewError(pkgerrors.ErrorTypeConflict, "agent already started")
	ErrInvalidOptions          = pkgerrors.NewError(pkgerrors.ErrorTypeValidation, "invalid options")
	ErrToolCallFailed          = pkgerrors.NewError(pkgerrors.ErrorTypeExternal, "tool call failed")
	ErrMaxToolCallsExceeded    = pkgerrors.NewError(pkgerrors.ErrorTypeCapacity, "max tool calls exceeded")
	ErrToolCallTimeout         = pkgerrors.NewError(pkgerrors.ErrorTypeTimeout, "tool call timeout")
	ErrNoToolsAvailable        = pkgerrors.NewError(pkgerrors.ErrorTypeNotFound, "no tools available")
	ErrRAGRetrievalFailed      = pkgerrors.NewError(pkgerrors.ErrorTypeExternal, "rag retrieval failed")
	ErrContextTooLong          = pkgerrors.NewError(pkgerrors.ErrorTypeValidation, "context too long")
	ErrChatClientError         = pkgerrors.NewError(pkgerrors.ErrorTypeExternal, "chat client error")
	ErrMCPClientError          = pkgerrors.NewError(pkgerrors.ErrorTypeExternal, "mcp client error")
)

// 使用pkg/errors中的统一错误类型
type AgentError = pkgerrors.BaseError

// NewAgentError 创建新的 Agent 错误
func NewAgentError(op, msg string, retryable bool) *AgentError {
	err := pkgerrors.NewError(pkgerrors.ErrorTypeInternal, msg).WithOperation(op).WithComponent("agent")
	if retryable {
		return err.WithDetails(map[string]string{"retryable": "true"})
	}
	return err
}

// WrapAgentError 包装错误为 AgentError
func WrapAgentError(op, msg string, err error, retryable bool) *AgentError {
	agentErr := pkgerrors.NewErrorWithCause(pkgerrors.ErrorTypeInternal, msg, err).WithOperation(op).WithComponent("agent")
	if retryable {
		return agentErr.WithDetails(map[string]string{"retryable": "true"})
	}
	return agentErr
}

// WrapChatError 包装 Chat 错误
func WrapChatError(op string, err error) *AgentError {
	return pkgerrors.NewErrorWithCause(pkgerrors.ErrorTypeExternal, "chat client error", err).WithOperation(op).WithComponent("agent")
}

// WrapMCPError 包装 MCP 错误
func WrapMCPError(op string, err error) *AgentError {
	return pkgerrors.NewErrorWithCause(pkgerrors.ErrorTypeExternal, "mcp client error", err).WithOperation(op).WithComponent("agent")
}

// WrapRAGError 包装 RAG 错误
func WrapRAGError(op string, err error) *AgentError {
	return pkgerrors.NewErrorWithCause(pkgerrors.ErrorTypeExternal, "rag retrieval error", err).WithOperation(op).WithComponent("agent")
}

// IsTemporaryError 检查是否为临时错误
func IsTemporaryError(err error) bool {
	return pkgerrors.IsTemporaryError(err)
}

// ShouldRetry 检查是否应该重试
func ShouldRetry(err error, attempt, maxRetries int) bool {
	return pkgerrors.ShouldRetry(err, attempt, maxRetries)
}

// GetRetryDelay 根据错误类型和尝试次数计算重试延迟
func GetRetryDelay(err error, attempt int) time.Duration {
	return pkgerrors.GetRetryDelay(err, attempt)
}

// 使用pkg/errors中的统一错误统计
type ErrorStats = pkgerrors.ErrorStats

// NewErrorStats 创建新的错误统计
func NewErrorStats() *ErrorStats {
	return pkgerrors.NewErrorStats()
}