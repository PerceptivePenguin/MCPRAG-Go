package chat

import (
	"errors"
	"fmt"
)

var (
	// ErrAPIKeyRequired API 密钥未设置错误
	ErrAPIKeyRequired = errors.New("api key is required")
	
	// ErrInvalidModel 模型无效错误
	ErrInvalidModel = errors.New("invalid model specified")
	
	// ErrEmptyMessages 消息为空错误
	ErrEmptyMessages = errors.New("messages cannot be empty")
	
	// ErrInvalidMessage 消息格式无效错误
	ErrInvalidMessage = errors.New("invalid message format")
	
	// ErrStreamClosed 流已关闭错误
	ErrStreamClosed = errors.New("stream is closed")
	
	// ErrRateLimited API 速率限制错误
	ErrRateLimited = errors.New("api rate limited")
	
	// ErrQuotaExceeded API 配额超限错误
	ErrQuotaExceeded = errors.New("api quota exceeded")
	
	// ErrNetworkError 网络错误
	ErrNetworkError = errors.New("network error")
	
	// ErrInvalidResponse 响应格式无效错误
	ErrInvalidResponse = errors.New("invalid response format")
	
	// ErrContextCanceled 上下文取消错误
	ErrContextCanceled = errors.New("context canceled")
	
	// ErrTimeout 请求超时错误
	ErrTimeout = errors.New("request timeout")
)

// ChatError 聊天错误类型
type ChatError struct {
	Op        string // 操作名称
	Model     string // 模型名称
	Err       error  // 原始错误
	Retryable bool   // 是否可重试
	Code      int    // 错误代码
}

// Error 实现 error 接口
func (e *ChatError) Error() string {
	if e.Model != "" {
		return fmt.Sprintf("chat %s [%s]: %v", e.Op, e.Model, e.Err)
	}
	return fmt.Sprintf("chat %s: %v", e.Op, e.Err)
}

// Unwrap 支持 errors.Is 和 errors.As
func (e *ChatError) Unwrap() error {
	return e.Err
}

// IsRetryable 检查错误是否可重试
func (e *ChatError) IsRetryable() bool {
	return e.Retryable
}

// GetCode 获取错误代码
func (e *ChatError) GetCode() int {
	return e.Code
}

// NewChatError 创建新的聊天错误
func NewChatError(op, model string, err error, retryable bool, code int) *ChatError {
	return &ChatError{
		Op:        op,
		Model:     model,
		Err:       err,
		Retryable: retryable,
		Code:      code,
	}
}

// WrapError 包装错误为 ChatError
func WrapError(op, model string, err error) *ChatError {
	return NewChatError(op, model, err, false, 0)
}

// WrapRetryableError 包装可重试错误为 ChatError
func WrapRetryableError(op, model string, err error, code int) *ChatError {
	return NewChatError(op, model, err, true, code)
}

// WrapNetworkError 包装网络错误为 ChatError
func WrapNetworkError(op, model string, err error) *ChatError {
	return NewChatError(op, model, errors.Join(ErrNetworkError, err), true, 0)
}

// WrapRateLimitError 包装速率限制错误为 ChatError
func WrapRateLimitError(op, model string, err error) *ChatError {
	return NewChatError(op, model, errors.Join(ErrRateLimited, err), true, 429)
}

// WrapQuotaError 包装配额错误为 ChatError
func WrapQuotaError(op, model string, err error) *ChatError {
	return NewChatError(op, model, errors.Join(ErrQuotaExceeded, err), false, 429)
}

// WrapTimeoutError 包装超时错误为 ChatError
func WrapTimeoutError(op, model string, err error) *ChatError {
	return NewChatError(op, model, errors.Join(ErrTimeout, err), true, 408)
}

// IsTemporaryError 检查是否为临时错误
func IsTemporaryError(err error) bool {
	var chatErr *ChatError
	if errors.As(err, &chatErr) {
		return chatErr.IsRetryable()
	}
	
	// 检查常见的临时错误
	return errors.Is(err, ErrRateLimited) ||
		errors.Is(err, ErrNetworkError) ||
		errors.Is(err, ErrTimeout)
}

// GetRetryDelay 根据错误类型获取重试延迟
func GetRetryDelay(err error, attempt int) int {
	var chatErr *ChatError
	if errors.As(err, &chatErr) {
		switch {
		case errors.Is(err, ErrRateLimited):
			// 速率限制：指数退避，从 1 秒开始
			return 1 << uint(attempt) // 1, 2, 4, 8, 16 秒
		case errors.Is(err, ErrNetworkError):
			// 网络错误：线性增长
			return (attempt + 1) * 2 // 2, 4, 6, 8, 10 秒
		case errors.Is(err, ErrTimeout):
			// 超时错误：固定延迟
			return 5 // 5 秒
		default:
			// 其他错误：默认延迟
			return attempt + 1 // 1, 2, 3, 4, 5 秒
		}
	}
	
	return attempt + 1
}

// ShouldRetry 检查是否应该重试
func ShouldRetry(err error, attempt, maxRetries int) bool {
	if attempt >= maxRetries {
		return false
	}
	
	return IsTemporaryError(err)
}