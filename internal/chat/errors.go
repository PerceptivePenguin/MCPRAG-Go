package chat

import (
	"time"

	pkgerrors "github.com/PerceptivePenguin/MCPRAG-Go/pkg/errors"
)

var (
	// Chat 特定错误，基于pkg/errors的通用错误类型
	ErrAPIKeyRequired    = pkgerrors.NewError(pkgerrors.ErrorTypeAuth, "api key is required")
	ErrInvalidModel      = pkgerrors.NewError(pkgerrors.ErrorTypeValidation, "invalid model specified")
	ErrEmptyMessages     = pkgerrors.NewError(pkgerrors.ErrorTypeValidation, "messages cannot be empty")
	ErrInvalidMessage    = pkgerrors.NewError(pkgerrors.ErrorTypeValidation, "invalid message format")
	ErrStreamClosed      = pkgerrors.NewError(pkgerrors.ErrorTypeInternal, "stream is closed")
	ErrRateLimited       = pkgerrors.NewError(pkgerrors.ErrorTypeRateLimit, "api rate limited")
	ErrQuotaExceeded     = pkgerrors.NewError(pkgerrors.ErrorTypeRateLimit, "api quota exceeded")
	ErrNetworkError      = pkgerrors.NewError(pkgerrors.ErrorTypeNetwork, "network error")
	ErrInvalidResponse   = pkgerrors.NewError(pkgerrors.ErrorTypeExternal, "invalid response format")
	ErrContextCanceled   = pkgerrors.NewError(pkgerrors.ErrorTypeTimeout, "context canceled")
	ErrTimeout           = pkgerrors.NewError(pkgerrors.ErrorTypeTimeout, "request timeout")
)

// 使用pkg/errors中的统一错误类型
type ChatError = pkgerrors.BaseError

// NewChatError 创建新的聊天错误
func NewChatError(op, model string, err error, errorType pkgerrors.ErrorType) *ChatError {
	details := map[string]string{"model": model}
	if err != nil {
		return pkgerrors.NewErrorWithCause(errorType, "chat operation failed", err).WithOperation(op).WithComponent("chat").WithDetails(details)
	}
	return pkgerrors.NewError(errorType, "chat operation failed").WithOperation(op).WithComponent("chat").WithDetails(details)
}

// WrapNetworkError 包装网络错误为 ChatError
func WrapNetworkError(op, model string, err error) *ChatError {
	return NewChatError(op, model, err, pkgerrors.ErrorTypeNetwork)
}

// WrapRateLimitError 包装速率限制错误为 ChatError
func WrapRateLimitError(op, model string, err error) *ChatError {
	return NewChatError(op, model, err, pkgerrors.ErrorTypeRateLimit)
}

// WrapTimeoutError 包装超时错误为 ChatError
func WrapTimeoutError(op, model string, err error) *ChatError {
	return NewChatError(op, model, err, pkgerrors.ErrorTypeTimeout)
}

// IsTemporaryError 检查是否为临时错误
func IsTemporaryError(err error) bool {
	return pkgerrors.IsTemporaryError(err)
}

// GetRetryDelay 根据错误类型获取重试延迟
func GetRetryDelay(err error, attempt int) time.Duration {
	return pkgerrors.GetRetryDelay(err, attempt)
}

// ShouldRetry 检查是否应该重试
func ShouldRetry(err error, attempt, maxRetries int) bool {
	return pkgerrors.ShouldRetry(err, attempt, maxRetries)
}

// WrapError 包装通用错误为 ChatError (向后兼容)
func WrapError(op, model string, err error) *ChatError {
	return NewChatError(op, model, err, pkgerrors.ErrorTypeInternal)
}

// WrapQuotaError 包装配额错误为 ChatError
func WrapQuotaError(op, model string, err error) *ChatError {
	return NewChatError(op, model, err, pkgerrors.ErrorTypeRateLimit)
}