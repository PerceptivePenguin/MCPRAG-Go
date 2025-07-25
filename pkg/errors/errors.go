// Package errors 提供了统一的错误处理系统
//
// 这个包提供了项目中统一的错误定义、分类和处理机制，包括：
// - 基础错误类型和接口
// - 预定义的通用错误
// - 错误统计和监控
// - 重试逻辑支持
//
// 主要特性：
// - 结构化错误信息，支持错误分类、组件标识、操作上下文
// - 错误链支持，可以包装和追踪错误来源
// - 重试机制支持，自动判断错误是否可重试
// - 错误统计功能，提供详细的错误监控数据
// - HTTP状态码映射，方便Web API错误处理
package errors

import (
	"context"
	stderrors "errors"
	"time"
)

// 重新导出标准库的错误函数，方便使用
var (
	New    = stderrors.New
	Is     = stderrors.Is
	As     = stderrors.As
	Unwrap = stderrors.Unwrap
	Join   = stderrors.Join
)

// ErrorHandler 错误处理器接口
type ErrorHandler interface {
	// HandleError 处理错误
	HandleError(err error) error
	
	// ShouldRetry 检查是否应该重试
	ShouldRetry(err error, attempt int) bool
	
	// GetRetryDelay 获取重试延迟
	GetRetryDelay(err error, attempt int) time.Duration
}

// DefaultErrorHandler 默认错误处理器
type DefaultErrorHandler struct {
	MaxRetries int
	Stats      *ErrorStats
}

// NewDefaultErrorHandler 创建默认错误处理器
func NewDefaultErrorHandler(maxRetries int) *DefaultErrorHandler {
	return &DefaultErrorHandler{
		MaxRetries: maxRetries,
		Stats:      NewErrorStats(),
	}
}

// HandleError 处理错误
func (h *DefaultErrorHandler) HandleError(err error) error {
	if err == nil {
		return nil
	}
	
	h.Stats.RecordError(err)
	return err
}

// ShouldRetry 检查是否应该重试
func (h *DefaultErrorHandler) ShouldRetry(err error, attempt int) bool {
	if attempt >= h.MaxRetries {
		return false
	}
	
	shouldRetry := ShouldRetry(err, attempt, h.MaxRetries)
	if shouldRetry {
		h.Stats.RecordRetry(err)
	}
	return shouldRetry
}

// GetRetryDelay 获取重试延迟
func (h *DefaultErrorHandler) GetRetryDelay(err error, attempt int) time.Duration {
	return GetRetryDelay(err, attempt)
}

// RetryableOperation 可重试操作函数类型
type RetryableOperation func() error

// Retry 执行可重试操作
func Retry(operation RetryableOperation, handler ErrorHandler) error {
	var lastErr error
	
	for attempt := 0; ; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}
		
		lastErr = handler.HandleError(err)
		
		if !handler.ShouldRetry(err, attempt) {
			break
		}
		
		delay := handler.GetRetryDelay(err, attempt)
		if delay > 0 {
			time.Sleep(delay)
		}
	}
	
	return lastErr
}

// RetryWithContext 带上下文的重试操作
func RetryWithContext(ctx context.Context, operation func(context.Context) error, handler ErrorHandler) error {
	var lastErr error
	
	for attempt := 0; ; attempt++ {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			return ctx.Err()
		}
		
		err := operation(ctx)
		if err == nil {
			return nil
		}
		
		lastErr = handler.HandleError(err)
		
		if !handler.ShouldRetry(err, attempt) {
			break
		}
		
		delay := handler.GetRetryDelay(err, attempt)
		if delay > 0 {
			select {
			case <-time.After(delay):
				// 继续重试
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	
	return lastErr
}

// Chain 错误链，用于收集多个错误
type Chain struct {
	errors []error
}

// NewChain 创建新的错误链
func NewChain() *Chain {
	return &Chain{
		errors: make([]error, 0),
	}
}

// Add 添加错误到链中
func (c *Chain) Add(err error) {
	if err != nil {
		c.errors = append(c.errors, err)
	}
}

// HasErrors 检查是否有错误
func (c *Chain) HasErrors() bool {
	return len(c.errors) > 0
}

// Error 实现error接口
func (c *Chain) Error() string {
	if len(c.errors) == 0 {
		return ""
	}
	
	if len(c.errors) == 1 {
		return c.errors[0].Error()
	}
	
	return Join(c.errors...).Error()
}

// Errors 获取所有错误
func (c *Chain) Errors() []error {
	return c.errors
}

// First 获取第一个错误
func (c *Chain) First() error {
	if len(c.errors) == 0 {
		return nil
	}
	return c.errors[0]
}

// Last 获取最后一个错误
func (c *Chain) Last() error {
	if len(c.errors) == 0 {
		return nil
	}
	return c.errors[len(c.errors)-1]
}

// Count 获取错误数量
func (c *Chain) Count() int {
	return len(c.errors)
}

// Clear 清空错误链
func (c *Chain) Clear() {
	c.errors = c.errors[:0]
}