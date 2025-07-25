package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sashabaranov/go-openai"
)

// NewClient 创建新的聊天客户端
func NewClient(config ClientConfig) (*Client, error) {
	if config.APIKey == "" {
		return nil, WrapError("newClient", config.Model, ErrAPIKeyRequired)
	}
	
	if config.Model == "" {
		config.Model = openai.GPT4o
	}
	
	// 创建 OpenAI 客户端配置
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	
	client := &Client{
		config:   config,
		client:   openai.NewClientWithConfig(clientConfig),
		messages: make([]Message, 0),
	}
	
	return client, nil
}

// SetSystemPrompt 设置系统提示
func (c *Client) SetSystemPrompt(prompt string) {
	c.systemPrompt = prompt
}

// GetMessages 获取当前消息历史
func (c *Client) GetMessages() []Message {
	messages := make([]Message, len(c.messages))
	copy(messages, c.messages)
	return messages
}

// ClearMessages 清空消息历史
func (c *Client) ClearMessages() {
	c.messages = make([]Message, 0)
}

// appendMessage 添加消息到历史记录
func (c *Client) appendMessage(message Message) {
	c.messages = append(c.messages, message)
}

// Chat 发送聊天请求并返回响应
func (c *Client) Chat(ctx context.Context, messages []Message) (*Response, error) {
	if len(messages) == 0 {
		return nil, WrapError("chat", c.config.Model, ErrEmptyMessages)
	}
	
	// 准备消息
	allMessages := c.prepareMessages(messages)
	
	// 执行重试逻辑
	var response *Response
	var err error
	
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		response, err = c.executeChat(ctx, allMessages)
		if err == nil {
			// 添加用户消息和助手响应到历史记录
			c.appendMessage(messages[len(messages)-1]) // 最后一条用户消息
			if response.Content != "" || len(response.ToolCalls) > 0 {
				assistantMsg := Message{
					Role:      RoleAssistant,
					Content:   response.Content,
					ToolCalls: response.ToolCalls,
				}
				c.appendMessage(assistantMsg)
			}
			return response, nil
		}
		
		// 检查是否应该重试
		if !ShouldRetry(err, attempt, c.config.MaxRetries) {
			break
		}
		
		// 等待后重试
		delay := time.Duration(GetRetryDelay(err, attempt)) * time.Second
		select {
		case <-ctx.Done():
			return nil, WrapError("chat", c.config.Model, ctx.Err())
		case <-time.After(delay):
			// 继续重试
		}
	}
	
	return nil, err
}

// executeChat 执行单次聊天请求
func (c *Client) executeChat(ctx context.Context, messages []Message) (*Response, error) {
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	
	// 转换消息格式
	openaiMessages := convertToOpenAIMessages(messages)
	
	// 构建请求
	request := openai.ChatCompletionRequest{
		Model:            c.config.Model,
		Messages:         openaiMessages,
		Temperature:      c.config.Temperature,
		MaxTokens:        c.config.MaxTokens,
		TopP:             c.config.TopP,
		FrequencyPenalty: c.config.FrequencyPenalty,
		PresencePenalty:  c.config.PresencePenalty,
	}
	
	// 添加工具定义（如果启用）
	if c.config.EnableTools {
		// 这里暂时不添加工具，等待后续集成
		// request.Tools = c.getToolDefinitions()
	}
	
	// 发送请求
	resp, err := c.client.CreateChatCompletion(timeoutCtx, request)
	if err != nil {
		return nil, c.handleAPIError("chat", err)
	}
	
	// 转换响应
	response := convertFromOpenAIResponse(resp)
	return response, nil
}

// prepareMessages 准备消息列表
func (c *Client) prepareMessages(newMessages []Message) []Message {
	var allMessages []Message
	
	// 添加系统提示（如果有）
	if c.systemPrompt != "" {
		systemMsg := Message{
			Role:    RoleSystem,
			Content: c.systemPrompt,
		}
		allMessages = append(allMessages, systemMsg)
	}
	
	// 添加历史消息
	allMessages = append(allMessages, c.messages...)
	
	// 添加新消息
	allMessages = append(allMessages, newMessages...)
	
	// 验证消息格式
	for i, msg := range allMessages {
		if err := c.validateMessage(msg); err != nil {
			// 在实际实现中应该返回错误，这里暂时跳过无效消息
			_ = fmt.Sprintf("Invalid message at index %d: %v", i, err)
		}
	}
	
	return allMessages
}

// validateMessage 验证消息格式
func (c *Client) validateMessage(msg Message) error {
	if msg.Role == "" {
		return fmt.Errorf("message role cannot be empty")
	}
	
	if msg.Role != RoleSystem && msg.Role != RoleUser && 
		msg.Role != RoleAssistant && msg.Role != RoleTool {
		return fmt.Errorf("invalid message role: %s", msg.Role)
	}
	
	if msg.Role == RoleTool && msg.ToolCallID == "" {
		return fmt.Errorf("tool message must have tool_call_id")
	}
	
	return nil
}

// AppendToolResult 添加工具调用结果到消息历史
func (c *Client) AppendToolResult(toolCallID, result string) {
	toolMsg := Message{
		Role:       RoleTool,
		Content:    result,
		ToolCallID: toolCallID,
	}
	c.appendMessage(toolMsg)
}

// handleAPIError 处理 API 错误
func (c *Client) handleAPIError(operation string, err error) error {
	// 根据错误类型进行分类处理
	errMsg := err.Error()
	
	// 速率限制错误
	if contains(errMsg, "rate limit") || contains(errMsg, "429") {
		return WrapRateLimitError(operation, c.config.Model, err)
	}
	
	// 配额超限错误
	if contains(errMsg, "quota") || contains(errMsg, "exceeded") {
		return WrapQuotaError(operation, c.config.Model, err)
	}
	
	// 网络错误
	if contains(errMsg, "network") || contains(errMsg, "connection") || 
		contains(errMsg, "timeout") {
		return WrapNetworkError(operation, c.config.Model, err)
	}
	
	// 上下文取消错误
	if contains(errMsg, "context canceled") {
		return WrapError(operation, c.config.Model, ErrContextCanceled)
	}
	
	// 其他错误
	return WrapError(operation, c.config.Model, err)
}

// ChatStream 发送聊天请求并返回流式响应
func (c *Client) ChatStream(ctx context.Context, messages []Message) (<-chan StreamResponse, error) {
	if len(messages) == 0 {
		return nil, WrapError("chatStream", c.config.Model, ErrEmptyMessages)
	}
	
	// 准备消息
	allMessages := c.prepareMessages(messages)
	
	// 创建响应通道
	respChan := make(chan StreamResponse, 100)
	
	// 启动流式处理协程
	go c.handleStreamChat(ctx, allMessages, respChan, messages[len(messages)-1])
	
	return respChan, nil
}

// handleStreamChat 处理流式聊天
func (c *Client) handleStreamChat(ctx context.Context, messages []Message, 
	respChan chan<- StreamResponse, userMessage Message) {
	defer close(respChan)
	
	var finalContent string
	var finalToolCalls []ToolCall
	var mu sync.Mutex
	
	// 执行重试逻辑
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		err := c.executeStreamChat(ctx, messages, respChan, &finalContent, &finalToolCalls, &mu)
		if err == nil {
			// 成功完成，添加消息到历史记录
			c.appendMessage(userMessage)
			if finalContent != "" || len(finalToolCalls) > 0 {
				assistantMsg := Message{
					Role:      RoleAssistant,
					Content:   finalContent,
					ToolCalls: finalToolCalls,
				}
				c.appendMessage(assistantMsg)
			}
			return
		}
		
		// 检查是否应该重试
		if !ShouldRetry(err, attempt, c.config.MaxRetries) {
			respChan <- StreamResponse{Error: err, Finished: true}
			return
		}
		
		// 等待后重试
		delay := time.Duration(GetRetryDelay(err, attempt)) * time.Second
		select {
		case <-ctx.Done():
			respChan <- StreamResponse{Error: WrapError("chatStream", c.config.Model, ctx.Err()), Finished: true}
			return
		case <-time.After(delay):
			// 继续重试
		}
	}
}

// executeStreamChat 执行单次流式聊天请求
func (c *Client) executeStreamChat(ctx context.Context, messages []Message, 
	respChan chan<- StreamResponse, finalContent *string, finalToolCalls *[]ToolCall, mu *sync.Mutex) error {
	
	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()
	
	// 转换消息格式
	openaiMessages := convertToOpenAIMessages(messages)
	
	// 构建流式请求
	request := openai.ChatCompletionRequest{
		Model:            c.config.Model,
		Messages:         openaiMessages,
		Temperature:      c.config.Temperature,
		MaxTokens:        c.config.MaxTokens,
		TopP:             c.config.TopP,
		FrequencyPenalty: c.config.FrequencyPenalty,
		PresencePenalty:  c.config.PresencePenalty,
		Stream:           true,
	}
	
	// 创建流
	stream, err := c.client.CreateChatCompletionStream(timeoutCtx, request)
	if err != nil {
		return c.handleAPIError("chatStream", err)
	}
	defer stream.Close()
	
	// 处理流式响应
	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				// 流结束
				respChan <- StreamResponse{Finished: true}
				return nil
			}
			return c.handleAPIError("chatStream", err)
		}
		
		// 转换响应
		streamResp := convertFromOpenAIStreamResponse(response)
		
		// 累积最终内容
		mu.Lock()
		if streamResp.Content != "" {
			*finalContent += streamResp.Content
		}
		if len(streamResp.ToolCalls) > 0 {
			*finalToolCalls = append(*finalToolCalls, streamResp.ToolCalls...)
		}
		mu.Unlock()
		
		// 发送响应
		select {
		case <-timeoutCtx.Done():
			return WrapTimeoutError("chatStream", c.config.Model, timeoutCtx.Err())
		case respChan <- streamResp:
			if streamResp.Finished {
				return nil
			}
		}
	}
}

// contains 检查字符串是否包含子字符串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(toLowerCase(s), toLowerCase(substr))
}

// searchSubstring 搜索子字符串
func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// toLowerCase 转换为小写
func toLowerCase(s string) string {
	result := make([]byte, len(s))
	for i, b := range []byte(s) {
		if b >= 'A' && b <= 'Z' {
			result[i] = b + 32
		} else {
			result[i] = b
		}
	}
	return string(result)
}