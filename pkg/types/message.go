package types

import "time"

// Message 表示聊天消息的通用结构
type Message struct {
	Role         string      `json:"role"`
	Content      string      `json:"content"`
	Name         string      `json:"name,omitempty"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID   string      `json:"tool_call_id,omitempty"`
}

// ToolCall 表示工具调用的通用结构
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function FunctionCall     `json:"function"`
}

// FunctionCall 表示函数调用的通用结构
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Role 常量定义
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// ToolType 常量定义
const (
	ToolTypeFunction = "function"
)

// ToolDefinition 工具定义的通用结构
type ToolDefinition struct {
	Type     string            `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 函数定义的通用结构
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolResult 工具调用结果的通用结构
type ToolResult struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Args     interface{}   `json:"args"`
	Result   string        `json:"result"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}