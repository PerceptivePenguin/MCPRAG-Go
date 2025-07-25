package chat

import (
	"context"
	"time"

	"github.com/sashabaranov/go-openai"
)

// ChatClient 定义了聊天客户端的接口
type ChatClient interface {
	// Chat 发送聊天请求并返回响应
	Chat(ctx context.Context, messages []Message) (*Response, error)
	
	// ChatStream 发送聊天请求并返回流式响应
	ChatStream(ctx context.Context, messages []Message) (<-chan StreamResponse, error)
	
	// AppendToolResult 添加工具调用结果到消息历史
	AppendToolResult(toolCallID, result string)
	
	// GetMessages 获取当前消息历史
	GetMessages() []Message
	
	// ClearMessages 清空消息历史
	ClearMessages()
	
	// SetSystemPrompt 设置系统提示
	SetSystemPrompt(prompt string)
}

// Message 表示聊天消息
type Message struct {
	Role         string      `json:"role"`
	Content      string      `json:"content"`
	Name         string      `json:"name,omitempty"`
	ToolCalls    []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID   string      `json:"tool_call_id,omitempty"`
}

// ToolCall 表示工具调用
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function FunctionCall     `json:"function"`
}

// FunctionCall 表示函数调用
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// Response 表示聊天响应
type Response struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Finish    bool       `json:"finish"`
	Usage     Usage      `json:"usage,omitempty"`
}

// StreamResponse 表示流式响应
type StreamResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Finished  bool       `json:"finished"`
	Error     error      `json:"error,omitempty"`
}

// Usage 表示 token 使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ClientConfig 客户端配置
type ClientConfig struct {
	APIKey           string        `json:"apiKey"`
	BaseURL          string        `json:"baseUrl,omitempty"`
	Model            string        `json:"model"`
	Temperature      float32       `json:"temperature"`
	MaxTokens        int           `json:"maxTokens"`
	TopP             float32       `json:"topP"`
	FrequencyPenalty float32       `json:"frequencyPenalty"`
	PresencePenalty  float32       `json:"presencePenalty"`
	Timeout          time.Duration `json:"timeout"`
	MaxRetries       int           `json:"maxRetries"`
	RetryDelay       time.Duration `json:"retryDelay"`
	EnableTools      bool          `json:"enableTools"`
}

// DefaultClientConfig 返回默认的客户端配置
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Model:            openai.GPT4o,
		Temperature:      0.7,
		MaxTokens:        4096,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		Timeout:          60 * time.Second,
		MaxRetries:       3,
		RetryDelay:       time.Second,
		EnableTools:      true,
	}
}

// Client OpenAI 聊天客户端实现
type Client struct {
	config     ClientConfig
	client     *openai.Client
	messages   []Message
	systemPrompt string
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type     string            `json:"type"`
	Function FunctionDefinition `json:"function"`
}

// FunctionDefinition 函数定义
type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
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

// convertToOpenAIMessages 转换消息格式为 OpenAI 格式
func convertToOpenAIMessages(messages []Message) []openai.ChatCompletionMessage {
	openaiMessages := make([]openai.ChatCompletionMessage, len(messages))
	
	for i, msg := range messages {
		openaiMsg := openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
			Name:    msg.Name,
		}
		
		// 转换工具调用
		if len(msg.ToolCalls) > 0 {
			openaiToolCalls := make([]openai.ToolCall, len(msg.ToolCalls))
			for j, toolCall := range msg.ToolCalls {
				openaiToolCalls[j] = openai.ToolCall{
					ID:   toolCall.ID,
					Type: openai.ToolType(toolCall.Type),
					Function: openai.FunctionCall{
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					},
				}
			}
			openaiMsg.ToolCalls = openaiToolCalls
		}
		
		// 设置工具调用 ID
		if msg.ToolCallID != "" {
			openaiMsg.ToolCallID = msg.ToolCallID
		}
		
		openaiMessages[i] = openaiMsg
	}
	
	return openaiMessages
}

// convertFromOpenAIResponse 转换 OpenAI 响应格式
func convertFromOpenAIResponse(resp openai.ChatCompletionResponse) *Response {
	if len(resp.Choices) == 0 {
		return &Response{Finish: true}
	}
	
	choice := resp.Choices[0]
	response := &Response{
		Content: choice.Message.Content,
		Finish:  choice.FinishReason == "stop" || choice.FinishReason == "length",
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	
	// 转换工具调用
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]ToolCall, len(choice.Message.ToolCalls))
		for i, toolCall := range choice.Message.ToolCalls {
			toolCalls[i] = ToolCall{
				ID:   toolCall.ID,
				Type: string(toolCall.Type),
				Function: FunctionCall{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			}
		}
		response.ToolCalls = toolCalls
	}
	
	return response
}

// convertFromOpenAIStreamResponse 转换 OpenAI 流式响应格式
func convertFromOpenAIStreamResponse(resp openai.ChatCompletionStreamResponse) StreamResponse {
	if len(resp.Choices) == 0 {
		return StreamResponse{Finished: true}
	}
	
	choice := resp.Choices[0]
	streamResp := StreamResponse{
		Content:  choice.Delta.Content,
		Finished: choice.FinishReason != "",
	}
	
	// 转换工具调用
	if len(choice.Delta.ToolCalls) > 0 {
		toolCalls := make([]ToolCall, len(choice.Delta.ToolCalls))
		for i, toolCall := range choice.Delta.ToolCalls {
			toolCalls[i] = ToolCall{
				ID:   toolCall.ID,
				Type: string(toolCall.Type),
				Function: FunctionCall{
					Name:      toolCall.Function.Name,
					Arguments: toolCall.Function.Arguments,
				},
			}
		}
		streamResp.ToolCalls = toolCalls
	}
	
	return streamResp
}

// convertToOpenAITools 转换工具定义为 OpenAI 格式
func convertToOpenAITools(tools []ToolDefinition) []openai.Tool {
	openaiTools := make([]openai.Tool, len(tools))
	
	for i, tool := range tools {
		openaiTools[i] = openai.Tool{
			Type: openai.ToolType(tool.Type),
			Function: &openai.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}
	
	return openaiTools
}