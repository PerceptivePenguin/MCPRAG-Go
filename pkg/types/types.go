// Package types 提供了项目中使用的通用类型定义
//
// 这个包包含了在多个模块之间共享的数据结构，包括：
// - 消息和工具调用结构
// - 响应和流式响应结构
// - 配置相关结构
// - 统计信息结构
//
// 通过将这些通用类型定义放在 pkg/types 包中，我们可以：
// - 减少代码重复
// - 确保类型定义的一致性
// - 简化模块间的依赖关系
// - 提高代码的可维护性
package types

import "time"

// 导出所有主要类型，方便导入和使用

// 消息和工具相关类型
type (
	// MessageInterface 消息接口
	MessageInterface interface {
		GetRole() string
		GetContent() string
		GetToolCalls() []ToolCall
	}
	
	// ResponseInterface 响应接口
	ResponseInterface interface {
		GetContent() string
		GetToolCalls() []ToolCall
		IsFinished() bool
		GetError() error
	}
	
	// StatsInterface 统计接口
	StatsInterface interface {
		RecordRequest(requestType string, duration time.Duration, success bool)
		RecordError(errorType string)
		GetStats() Stats
		Reset()
	}
)

// GetRole 实现 MessageInterface
func (m *Message) GetRole() string {
	return m.Role
}

// GetContent 实现 MessageInterface
func (m *Message) GetContent() string {
	return m.Content
}

// GetToolCalls 实现 MessageInterface
func (m *Message) GetToolCalls() []ToolCall {
	return m.ToolCalls
}

// GetContent 实现 ResponseInterface for Response
func (r *Response) GetContent() string {
	return r.Content
}

// GetToolCalls 实现 ResponseInterface for Response
func (r *Response) GetToolCalls() []ToolCall {
	return r.ToolCalls
}

// IsFinished 实现 ResponseInterface for Response
func (r *Response) IsFinished() bool {
	return r.Finish
}

// GetError 实现 ResponseInterface for Response
func (r *Response) GetError() error {
	if r.Error != "" {
		return NewStringError(r.Error)
	}
	return nil
}

// GetContent 实现 ResponseInterface for StreamResponse
func (sr *StreamResponse) GetContent() string {
	return sr.Content
}

// GetToolCalls 实现 ResponseInterface for StreamResponse
func (sr *StreamResponse) GetToolCalls() []ToolCall {
	return sr.ToolCalls
}

// IsFinished 实现 ResponseInterface for StreamResponse
func (sr *StreamResponse) IsFinished() bool {
	return sr.Finished
}

// GetError 实现 ResponseInterface for StreamResponse
func (sr *StreamResponse) GetError() error {
	return sr.Error
}

// StringError 简单的字符串错误类型
type StringError string

func (e StringError) Error() string {
	return string(e)
}

// NewStringError 创建新的字符串错误
func NewStringError(msg string) error {
	return StringError(msg)
}