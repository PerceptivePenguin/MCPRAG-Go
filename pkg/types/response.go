package types

import "time"

// Response 表示聊天响应的通用结构
type Response struct {
	ID           string        `json:"id"`
	Content      string        `json:"content"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	Finish       bool          `json:"finish"`
	Usage        TokenUsage    `json:"usage,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	Timestamp    time.Time     `json:"timestamp"`
	Error        string        `json:"error,omitempty"`
}

// StreamResponse 表示流式响应的通用结构
type StreamResponse struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Finished  bool       `json:"finished"`
	Error     error      `json:"error,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// TokenUsage Token 使用统计的通用结构
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// RequestResult 请求处理结果的通用结构
type RequestResult struct {
	ID           string        `json:"id"`
	Success      bool          `json:"success"`
	Response     *Response     `json:"response,omitempty"`
	Error        error         `json:"error,omitempty"`
	Duration     time.Duration `json:"duration"`
	TokenUsage   TokenUsage    `json:"token_usage"`
	Timestamp    time.Time     `json:"timestamp"`
}