package chat

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// ToolHandler 工具处理器接口
type ToolHandler interface {
	// GetName 获取工具名称
	GetName() string
	
	// GetDescription 获取工具描述
	GetDescription() string
	
	// GetParameters 获取工具参数 schema
	GetParameters() map[string]interface{}
	
	// Execute 执行工具调用
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	handlers map[string]ToolHandler
}

// NewToolRegistry 创建新的工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		handlers: make(map[string]ToolHandler),
	}
}

// RegisterTool 注册工具
func (r *ToolRegistry) RegisterTool(handler ToolHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}
	
	name := handler.GetName()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("tool %s already registered", name)
	}
	
	r.handlers[name] = handler
	return nil
}

// UnregisterTool 注销工具
func (r *ToolRegistry) UnregisterTool(name string) {
	delete(r.handlers, name)
}

// GetTool 获取工具处理器
func (r *ToolRegistry) GetTool(name string) (ToolHandler, error) {
	handler, exists := r.handlers[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}
	return handler, nil
}

// ListTools 列出所有工具
func (r *ToolRegistry) ListTools() []string {
	names := make([]string, 0, len(r.handlers))
	for name := range r.handlers {
		names = append(names, name)
	}
	return names
}

// GetToolDefinitions 获取工具定义
func (r *ToolRegistry) GetToolDefinitions() []ToolDefinition {
	definitions := make([]ToolDefinition, 0, len(r.handlers))
	
	for _, handler := range r.handlers {
		definition := ToolDefinition{
			Type: ToolTypeFunction,
			Function: FunctionDefinition{
				Name:        handler.GetName(),
				Description: handler.GetDescription(),
				Parameters:  handler.GetParameters(),
			},
		}
		definitions = append(definitions, definition)
	}
	
	return definitions
}

// ExecuteTool 执行工具调用
func (r *ToolRegistry) ExecuteTool(ctx context.Context, toolCall ToolCall) (string, error) {
	handler, err := r.GetTool(toolCall.Function.Name)
	if err != nil {
		return "", fmt.Errorf("failed to get tool handler: %w", err)
	}
	
	// 解析参数
	var args map[string]interface{}
	if toolCall.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			return "", fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	}
	
	// 执行工具
	result, err := handler.Execute(ctx, args)
	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}
	
	return result, nil
}

// ClientWithTools 支持工具调用的客户端
type ClientWithTools struct {
	*Client
	registry *ToolRegistry
}

// NewClientWithTools 创建支持工具调用的客户端
func NewClientWithTools(config ClientConfig, registry *ToolRegistry) (*ClientWithTools, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	
	if registry == nil {
		registry = NewToolRegistry()
	}
	
	return &ClientWithTools{
		Client:   client,
		registry: registry,
	}, nil
}

// RegisterTool 注册工具
func (c *ClientWithTools) RegisterTool(handler ToolHandler) error {
	return c.registry.RegisterTool(handler)
}

// ChatWithTools 支持工具调用的聊天
func (c *ClientWithTools) ChatWithTools(ctx context.Context, messages []Message) (*Response, error) {
	// 启用工具调用
	originalEnableTools := c.config.EnableTools
	c.config.EnableTools = true
	defer func() {
		c.config.EnableTools = originalEnableTools
	}()
	
	// 准备消息
	allMessages := c.prepareMessages(messages)
	
	// 执行聊天循环，处理工具调用
	for {
		// 发送请求
		response, err := c.executeChatWithTools(ctx, allMessages)
		if err != nil {
			return nil, err
		}
		
		// 如果没有工具调用，返回响应
		if len(response.ToolCalls) == 0 {
			return response, nil
		}
		
		// 添加助手消息到历史
		assistantMsg := Message{
			Role:      RoleAssistant,
			Content:   response.Content,
			ToolCalls: response.ToolCalls,
		}
		allMessages = append(allMessages, assistantMsg)
		
		// 执行工具调用
		for _, toolCall := range response.ToolCalls {
			result, err := c.registry.ExecuteTool(ctx, toolCall)
			if err != nil {
				result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Function.Name, err)
			}
			
			// 添加工具结果消息
			toolMsg := Message{
				Role:       RoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			}
			allMessages = append(allMessages, toolMsg)
		}
		
		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			return nil, WrapError("chatWithTools", c.config.Model, ctx.Err())
		default:
			// 继续下一轮对话
		}
	}
}

// executeChatWithTools 执行支持工具的聊天请求
func (c *ClientWithTools) executeChatWithTools(ctx context.Context, messages []Message) (*Response, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	
	// 转换消息格式
	openaiMessages := convertToOpenAIMessages(messages)
	
	// 构建请求
	request := c.buildChatCompletionRequest(openaiMessages)
	
	// 添加工具定义
	if c.config.EnableTools && len(c.registry.handlers) > 0 {
		toolDefinitions := c.registry.GetToolDefinitions()
		request.Tools = convertToOpenAITools(toolDefinitions)
	}
	
	// 发送请求
	resp, err := c.Client.client.CreateChatCompletion(timeoutCtx, request)
	if err != nil {
		return nil, c.Client.handleAPIError("chatWithTools", err)
	}
	
	// 转换响应
	response := convertFromOpenAIResponse(resp)
	return response, nil
}

// buildChatCompletionRequest 构建聊天完成请求
func (c *ClientWithTools) buildChatCompletionRequest(messages []openai.ChatCompletionMessage) openai.ChatCompletionRequest {
	return openai.ChatCompletionRequest{
		Model:            c.config.Model,
		Messages:         messages,
		Temperature:      c.config.Temperature,
		MaxTokens:        c.config.MaxTokens,
		TopP:             c.config.TopP,
		FrequencyPenalty: c.config.FrequencyPenalty,
		PresencePenalty:  c.config.PresencePenalty,
	}
}

// ChatStreamWithTools 支持工具调用的流式聊天
func (c *ClientWithTools) ChatStreamWithTools(ctx context.Context, messages []Message) (<-chan StreamResponse, error) {
	// 启用工具调用
	originalEnableTools := c.config.EnableTools
	c.config.EnableTools = true
	defer func() {
		c.config.EnableTools = originalEnableTools
	}()
	
	// 创建响应通道
	respChan := make(chan StreamResponse, 100)
	
	// 启动流式处理协程
	go c.handleStreamChatWithTools(ctx, messages, respChan)
	
	return respChan, nil
}

// handleStreamChatWithTools 处理支持工具的流式聊天
func (c *ClientWithTools) handleStreamChatWithTools(ctx context.Context, 
	messages []Message, respChan chan<- StreamResponse) {
	defer close(respChan)
	
	// 准备消息
	allMessages := c.prepareMessages(messages)
	
	for {
		// 执行流式聊天
		streamRespChan, err := c.executeStreamChatWithTools(ctx, allMessages)
		if err != nil {
			respChan <- StreamResponse{Error: err, Finished: true}
			return
		}
		
		// 收集响应
		var finalContent string
		var finalToolCalls []ToolCall
		var finished bool
		
		for streamResp := range streamRespChan {
			if streamResp.Error != nil {
				respChan <- streamResp
				return
			}
			
			finalContent += streamResp.Content
			finalToolCalls = append(finalToolCalls, streamResp.ToolCalls...)
			finished = streamResp.Finished
			
			// 转发响应
			respChan <- streamResp
			
			if finished {
				break
			}
		}
		
		// 如果没有工具调用，结束
		if len(finalToolCalls) == 0 {
			return
		}
		
		// 添加助手消息
		assistantMsg := Message{
			Role:      RoleAssistant,
			Content:   finalContent,
			ToolCalls: finalToolCalls,
		}
		allMessages = append(allMessages, assistantMsg)
		
		// 执行工具调用
		for _, toolCall := range finalToolCalls {
			result, err := c.registry.ExecuteTool(ctx, toolCall)
			if err != nil {
				result = fmt.Sprintf("Error executing tool %s: %v", toolCall.Function.Name, err)
			}
			
			// 添加工具结果消息
			toolMsg := Message{
				Role:       RoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			}
			allMessages = append(allMessages, toolMsg)
		}
		
		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			respChan <- StreamResponse{Error: WrapError("chatStreamWithTools", c.config.Model, ctx.Err()), Finished: true}
			return
		default:
			// 继续下一轮对话
		}
	}
}

// executeStreamChatWithTools 执行支持工具的流式聊天请求
func (c *ClientWithTools) executeStreamChatWithTools(ctx context.Context, 
	messages []Message) (<-chan StreamResponse, error) {
	
	// 转换消息格式
	openaiMessages := convertToOpenAIMessages(messages)
	
	// 构建流式请求
	request := c.buildChatCompletionRequest(openaiMessages)
	request.Stream = true
	
	// 添加工具定义
	if c.config.EnableTools && len(c.registry.handlers) > 0 {
		toolDefinitions := c.registry.GetToolDefinitions()
		request.Tools = convertToOpenAITools(toolDefinitions)
	}
	
	// 创建流
	stream, err := c.Client.client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return nil, c.Client.handleAPIError("chatStreamWithTools", err)
	}
	
	// 创建响应通道
	respChan := make(chan StreamResponse, 100)
	
	// 启动流处理协程
	go c.processToolStream(ctx, stream, respChan)
	
	return respChan, nil
}

// processToolStream 处理工具流
func (c *ClientWithTools) processToolStream(ctx context.Context, 
	stream *openai.ChatCompletionStream, respChan chan<- StreamResponse) {
	defer close(respChan)
	defer stream.Close()
	
	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				respChan <- StreamResponse{Finished: true}
				return
			}
			respChan <- StreamResponse{Error: c.Client.handleAPIError("processToolStream", err), Finished: true}
			return
		}
		
		// 转换响应
		streamResp := convertFromOpenAIStreamResponse(response)
		
		// 发送响应
		select {
		case <-ctx.Done():
			respChan <- StreamResponse{Error: WrapError("processToolStream", c.config.Model, ctx.Err()), Finished: true}
			return
		case respChan <- streamResp:
			if streamResp.Finished {
				return
			}
		}
	}
}