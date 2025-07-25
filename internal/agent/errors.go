package agent

import (
	"errors"
	"fmt"
)

var (
	// ErrAgentNotStarted Agent 未启动错误
	ErrAgentNotStarted = errors.New("agent not started")
	
	// ErrAgentAlreadyStarted Agent 已启动错误
	ErrAgentAlreadyStarted = errors.New("agent already started")
	
	// ErrInvalidOptions 配置选项无效错误
	ErrInvalidOptions = errors.New("invalid options")
	
	// ErrToolCallFailed 工具调用失败错误
	ErrToolCallFailed = errors.New("tool call failed")
	
	// ErrMaxToolCallsExceeded 超过最大工具调用次数错误
	ErrMaxToolCallsExceeded = errors.New("max tool calls exceeded")
	
	// ErrToolCallTimeout 工具调用超时错误
	ErrToolCallTimeout = errors.New("tool call timeout")
	
	// ErrNoToolsAvailable 无可用工具错误
	ErrNoToolsAvailable = errors.New("no tools available")
	
	// ErrRAGRetrievalFailed RAG 检索失败错误
	ErrRAGRetrievalFailed = errors.New("rag retrieval failed")
	
	// ErrContextTooLong 上下文过长错误
	ErrContextTooLong = errors.New("context too long")
	
	// ErrChatClientError Chat 客户端错误
	ErrChatClientError = errors.New("chat client error")
	
	// ErrMCPClientError MCP 客户端错误
	ErrMCPClientError = errors.New("mcp client error")
)

// AgentError Agent 错误类型
type AgentError struct {
	Op        string // 操作名称
	Msg       string // 错误消息
	Err       error  // 原始错误
	Retryable bool   // 是否可重试
}

// Error 实现 error 接口
func (e *AgentError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("agent %s: %s: %v", e.Op, e.Msg, e.Err)
	}
	return fmt.Sprintf("agent %s: %s", e.Op, e.Msg)
}

// Unwrap 支持 errors.Is 和 errors.As
func (e *AgentError) Unwrap() error {
	return e.Err
}

// IsRetryable 检查错误是否可重试
func (e *AgentError) IsRetryable() bool {
	return e.Retryable
}

// NewAgentError 创建新的 Agent 错误
func NewAgentError(op, msg string, retryable bool) *AgentError {
	return &AgentError{
		Op:        op,
		Msg:       msg,
		Retryable: retryable,
	}
}

// WrapAgentError 包装错误为 AgentError
func WrapAgentError(op, msg string, err error, retryable bool) *AgentError {
	return &AgentError{
		Op:        op,
		Msg:       msg,
		Err:       err,
		Retryable: retryable,
	}
}

// WrapChatError 包装 Chat 错误
func WrapChatError(op string, err error) *AgentError {
	return WrapAgentError(op, "chat client error", err, true)
}

// WrapMCPError 包装 MCP 错误
func WrapMCPError(op string, err error) *AgentError {
	return WrapAgentError(op, "mcp client error", err, true)
}

// WrapRAGError 包装 RAG 错误
func WrapRAGError(op string, err error) *AgentError {
	return WrapAgentError(op, "rag retrieval error", err, true)
}

// IsTemporaryError 检查是否为临时错误
func IsTemporaryError(err error) bool {
	var agentErr *AgentError
	if errors.As(err, &agentErr) {
		return agentErr.IsRetryable()
	}
	
	// 检查常见的临时错误
	return errors.Is(err, ErrToolCallTimeout) ||
		errors.Is(err, ErrToolCallFailed) ||
		errors.Is(err, ErrRAGRetrievalFailed) ||
		errors.Is(err, ErrChatClientError) ||
		errors.Is(err, ErrMCPClientError)
}

// ShouldRetry 检查是否应该重试
func ShouldRetry(err error, attempt, maxRetries int) bool {
	if attempt >= maxRetries {
		return false
	}
	
	return IsTemporaryError(err)
}

// GetRetryDelay 根据错误类型和尝试次数计算重试延迟
func GetRetryDelay(baseDelay int, attempt int, backoff float64) int {
	delay := float64(baseDelay)
	for i := 0; i < attempt; i++ {
		delay *= backoff
	}
	return int(delay)
}

// ErrorStats 错误统计信息
type ErrorStats struct {
	TotalErrors      int            `json:"totalErrors"`
	RetryableErrors  int            `json:"retryableErrors"`
	ErrorsByType     map[string]int `json:"errorsByType"`
	ErrorsByOp       map[string]int `json:"errorsByOp"`
}

// NewErrorStats 创建新的错误统计
func NewErrorStats() *ErrorStats {
	return &ErrorStats{
		ErrorsByType: make(map[string]int),
		ErrorsByOp:   make(map[string]int),
	}
}

// RecordError 记录错误统计
func (s *ErrorStats) RecordError(err error) {
	s.TotalErrors++
	
	if IsTemporaryError(err) {
		s.RetryableErrors++
	}
	
	var agentErr *AgentError
	if errors.As(err, &agentErr) {
		s.ErrorsByOp[agentErr.Op]++
		
		// 根据原始错误分类
		if agentErr.Err != nil {
			errorType := fmt.Sprintf("%T", agentErr.Err)
			s.ErrorsByType[errorType]++
		} else {
			s.ErrorsByType["AgentError"]++
		}
	} else {
		errorType := fmt.Sprintf("%T", err)
		s.ErrorsByType[errorType]++
	}
}

// Reset 重置错误统计
func (s *ErrorStats) Reset() {
	s.TotalErrors = 0
	s.RetryableErrors = 0
	s.ErrorsByType = make(map[string]int)
	s.ErrorsByOp = make(map[string]int)
}

// String 返回错误统计的字符串表示
func (s *ErrorStats) String() string {
	return fmt.Sprintf("ErrorStats: Total=%d, Retryable=%d, Types=%d, Ops=%d",
		s.TotalErrors, s.RetryableErrors, len(s.ErrorsByType), len(s.ErrorsByOp))
}