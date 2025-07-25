package chat

import (
	"context"

	"github.com/PerceptivePenguin/MCPRAG-Go/pkg/types"
	"github.com/sashabaranov/go-openai"
)

// ChatClient 定义了聊天客户端的接口
type ChatClient interface {
	// Chat 发送聊天请求并返回响应
	Chat(ctx context.Context, messages []types.Message) (*types.Response, error)
	
	// ChatStream 发送聊天请求并返回流式响应
	ChatStream(ctx context.Context, messages []types.Message) (<-chan types.StreamResponse, error)
	
	// AppendToolResult 添加工具调用结果到消息历史
	AppendToolResult(toolCallID, result string)
	
	// GetMessages 获取当前消息历史
	GetMessages() []types.Message
	
	// ClearMessages 清空消息历史
	ClearMessages()
	
	// SetSystemPrompt 设置系统提示
	SetSystemPrompt(prompt string)
}

// 使用pkg/types中的通用类型
// 为了向后兼容，可以创建类型别名
type (
	Message        = types.Message
	ToolCall       = types.ToolCall
	FunctionCall   = types.FunctionCall
	Response       = types.Response
	StreamResponse = types.StreamResponse
	Usage          = types.TokenUsage
)

// ClientConfig 客户端配置
type ClientConfig struct {
	types.BaseConfig
	types.ConnectionConfig
	
	// OpenAI 特定配置
	Model            string  `json:"model" yaml:"model"`
	Temperature      float32 `json:"temperature" yaml:"temperature"`
	MaxTokens        int     `json:"max_tokens" yaml:"max_tokens"`
	TopP             float32 `json:"top_p" yaml:"top_p"`
	FrequencyPenalty float32 `json:"frequency_penalty" yaml:"frequency_penalty"`
	PresencePenalty  float32 `json:"presence_penalty" yaml:"presence_penalty"`
	EnableTools      bool    `json:"enable_tools" yaml:"enable_tools"`
}

// DefaultClientConfig 返回默认的客户端配置
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseConfig:       types.DefaultBaseConfig(),
		ConnectionConfig: types.DefaultConnectionConfig(),
		Model:            openai.GPT4o,
		Temperature:      0.7,
		MaxTokens:        4096,
		TopP:             1.0,
		FrequencyPenalty: 0.0,
		PresencePenalty:  0.0,
		EnableTools:      true,
	}
}

// Client OpenAI 聊天客户端实现
type Client struct {
	config       ClientConfig
	client       *openai.Client
	messages     []types.Message
	systemPrompt string
}

// 使用pkg/types中的工具定义
type (
	ToolDefinition     = types.ToolDefinition
	FunctionDefinition = types.FunctionDefinition
)

// 使用pkg/types中的常量
const (
	RoleSystem       = types.RoleSystem
	RoleUser         = types.RoleUser
	RoleAssistant    = types.RoleAssistant
	RoleTool         = types.RoleTool
	ToolTypeFunction = types.ToolTypeFunction
)

// convertToOpenAIMessages 转换消息格式为 OpenAI 格式
func convertToOpenAIMessages(messages []types.Message) []openai.ChatCompletionMessage {
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
func convertFromOpenAIResponse(resp openai.ChatCompletionResponse) *types.Response {
	if len(resp.Choices) == 0 {
		return &types.Response{Finish: true}
	}
	
	choice := resp.Choices[0]
	response := &types.Response{
		Content: choice.Message.Content,
		Finish:  choice.FinishReason == "stop" || choice.FinishReason == "length",
		Usage: types.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	
	// 转换工具调用
	if len(choice.Message.ToolCalls) > 0 {
		toolCalls := make([]types.ToolCall, len(choice.Message.ToolCalls))
		for i, toolCall := range choice.Message.ToolCalls {
			toolCalls[i] = types.ToolCall{
				ID:   toolCall.ID,
				Type: string(toolCall.Type),
				Function: types.FunctionCall{
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
func convertFromOpenAIStreamResponse(resp openai.ChatCompletionStreamResponse) types.StreamResponse {
	if len(resp.Choices) == 0 {
		return types.StreamResponse{Finished: true}
	}
	
	choice := resp.Choices[0]
	streamResp := types.StreamResponse{
		Content:  choice.Delta.Content,
		Finished: choice.FinishReason != "",
	}
	
	// 转换工具调用
	if len(choice.Delta.ToolCalls) > 0 {
		toolCalls := make([]types.ToolCall, len(choice.Delta.ToolCalls))
		for i, toolCall := range choice.Delta.ToolCalls {
			toolCalls[i] = types.ToolCall{
				ID:   toolCall.ID,
				Type: string(toolCall.Type),
				Function: types.FunctionCall{
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
func convertToOpenAITools(tools []types.ToolDefinition) []openai.Tool {
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