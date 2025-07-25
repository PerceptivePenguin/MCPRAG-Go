package errors

import (
	"errors"
	"fmt"
	"time"
)

// ErrorType 定义错误类型
type ErrorType string

const (
	// 基础错误类型
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeAuth           ErrorType = "authentication"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeExternal       ErrorType = "external"
	ErrorTypeInternal       ErrorType = "internal"
	ErrorTypeCapacity       ErrorType = "capacity"
	ErrorTypeNotImplemented ErrorType = "not_implemented"
	ErrorTypeNetwork        ErrorType = "network"
	ErrorTypeConfiguration  ErrorType = "configuration"
)

// BaseError 基础错误结构
type BaseError struct {
	Type       ErrorType         `json:"type"`
	Code       string            `json:"code,omitempty"`
	Message    string            `json:"message"`
	Operation  string            `json:"operation,omitempty"`
	Component  string            `json:"component,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
	Cause      error             `json:"-"`
	Retryable  bool              `json:"retryable"`
	Timestamp  time.Time         `json:"timestamp"`
}

// Error 实现 error 接口
func (e *BaseError) Error() string {
	if e.Component != "" && e.Operation != "" {
		return fmt.Sprintf("%s %s.%s: %s", e.Component, e.Component, e.Operation, e.Message)
	} else if e.Operation != "" {
		return fmt.Sprintf("%s %s: %s", e.Component, e.Operation, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Component, e.Message)
}

// Unwrap 返回原始错误
func (e *BaseError) Unwrap() error {
	return e.Cause
}

// Is 检查错误类型匹配
func (e *BaseError) Is(target error) bool {
	if baseErr, ok := target.(*BaseError); ok {
		return e.Type == baseErr.Type && e.Code == baseErr.Code
	}
	return false
}

// IsRetryable 检查错误是否可重试
func (e *BaseError) IsRetryable() bool {
	return e.Retryable
}

// GetType 获取错误类型
func (e *BaseError) GetType() ErrorType {
	return e.Type
}

// GetCode 获取错误代码
func (e *BaseError) GetCode() string {
	return e.Code
}

// GetDetails 获取错误详情
func (e *BaseError) GetDetails() map[string]string {
	return e.Details
}

// GetHTTPStatusCode 根据错误类型返回对应的HTTP状态码
func (e *BaseError) GetHTTPStatusCode() int {
	switch e.Type {
	case ErrorTypeValidation:
		return 400 // Bad Request
	case ErrorTypeAuth:
		return 401 // Unauthorized
	case ErrorTypeNotFound:
		return 404 // Not Found
	case ErrorTypeConflict:
		return 409 // Conflict
	case ErrorTypeCapacity:
		return 413 // Payload Too Large
	case ErrorTypeRateLimit:
		return 429 // Too Many Requests
	case ErrorTypeNotImplemented:
		return 501 // Not Implemented
	case ErrorTypeExternal:
		return 502 // Bad Gateway
	case ErrorTypeTimeout:
		return 504 // Gateway Timeout
	default:
		return 500 // Internal Server Error
	}
}

// WithOperation 添加操作上下文
func (e *BaseError) WithOperation(operation string) *BaseError {
	return &BaseError{
		Type:      e.Type,
		Code:      e.Code,
		Message:   e.Message,
		Operation: operation,
		Component: e.Component,
		Details:   e.Details,
		Cause:     e.Cause,
		Retryable: e.Retryable,
		Timestamp: e.Timestamp,
	}
}

// WithComponent 添加组件上下文
func (e *BaseError) WithComponent(component string) *BaseError {
	return &BaseError{
		Type:      e.Type,
		Code:      e.Code,
		Message:   e.Message,
		Operation: e.Operation,
		Component: component,
		Details:   e.Details,
		Cause:     e.Cause,
		Retryable: e.Retryable,
		Timestamp: e.Timestamp,
	}
}

// WithDetails 添加错误详情
func (e *BaseError) WithDetails(details map[string]string) *BaseError {
	newDetails := make(map[string]string)
	for k, v := range e.Details {
		newDetails[k] = v
	}
	for k, v := range details {
		newDetails[k] = v
	}
	
	return &BaseError{
		Type:      e.Type,
		Code:      e.Code,
		Message:   e.Message,
		Operation: e.Operation,
		Component: e.Component,
		Details:   newDetails,
		Cause:     e.Cause,
		Retryable: e.Retryable,
		Timestamp: e.Timestamp,
	}
}

// WithCause 添加原始错误
func (e *BaseError) WithCause(cause error) *BaseError {
	return &BaseError{
		Type:      e.Type,
		Code:      e.Code,
		Message:   e.Message,
		Operation: e.Operation,
		Component: e.Component,
		Details:   e.Details,
		Cause:     cause,
		Retryable: e.Retryable,
		Timestamp: e.Timestamp,
	}
}

// NewError 创建新的基础错误
func NewError(errorType ErrorType, message string) *BaseError {
	return &BaseError{
		Type:      errorType,
		Message:   message,
		Retryable: isDefaultRetryable(errorType),
		Timestamp: time.Now(),
	}
}

// NewErrorWithCode 创建带错误代码的错误
func NewErrorWithCode(errorType ErrorType, code, message string) *BaseError {
	return &BaseError{
		Type:      errorType,
		Code:      code,
		Message:   message,
		Retryable: isDefaultRetryable(errorType),
		Timestamp: time.Now(),
	}
}

// NewErrorWithCause 创建包装其他错误的错误
func NewErrorWithCause(errorType ErrorType, message string, cause error) *BaseError {
	return &BaseError{
		Type:      errorType,
		Message:   message,
		Cause:     cause,
		Retryable: isDefaultRetryable(errorType),
		Timestamp: time.Now(),
	}
}

// WrapError 包装现有错误
func WrapError(err error, errorType ErrorType, message string) *BaseError {
	return NewErrorWithCause(errorType, message, err)
}

// isDefaultRetryable 根据错误类型返回默认的重试设置
func isDefaultRetryable(errorType ErrorType) bool {
	switch errorType {
	case ErrorTypeTimeout, ErrorTypeExternal, ErrorTypeRateLimit, ErrorTypeNetwork:
		return true
	case ErrorTypeInternal:
		return true // 一些内部错误可能是临时的
	default:
		return false
	}
}

// IsTemporaryError 检查是否为临时错误
func IsTemporaryError(err error) bool {
	var baseErr *BaseError
	if errors.As(err, &baseErr) {
		return baseErr.IsRetryable()
	}
	return false
}

// GetRetryDelay 根据错误类型和尝试次数计算重试延迟
func GetRetryDelay(err error, attempt int) time.Duration {
	var baseErr *BaseError
	if errors.As(err, &baseErr) {
		switch baseErr.Type {
		case ErrorTypeRateLimit:
			// 速率限制：指数退避，从 1 秒开始
			return time.Duration(1<<uint(attempt)) * time.Second // 1, 2, 4, 8, 16 秒
		case ErrorTypeNetwork, ErrorTypeExternal:
			// 网络/外部错误：线性增长
			return time.Duration((attempt+1)*2) * time.Second // 2, 4, 6, 8, 10 秒
		case ErrorTypeTimeout:
			// 超时错误：固定延迟
			return 5 * time.Second
		default:
			// 其他错误：默认延迟
			return time.Duration(attempt+1) * time.Second // 1, 2, 3, 4, 5 秒
		}
	}
	
	return time.Duration(attempt+1) * time.Second
}

// ShouldRetry 检查是否应该重试
func ShouldRetry(err error, attempt, maxRetries int) bool {
	if attempt >= maxRetries {
		return false
	}
	
	return IsTemporaryError(err)
}