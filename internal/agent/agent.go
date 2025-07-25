package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/chat"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/mcp"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/rag"
)

// NewAgent 创建新的 Agent
func NewAgent(opts ...Option) (*Agent, error) {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	
	if err := options.Validate(); err != nil {
		return nil, WrapAgentError("newAgent", "invalid options", err, false)
	}
	
	// 创建组件
	chatClient, err := chat.NewClientWithTools(options.ChatConfig, nil)
	if err != nil {
		return nil, WrapChatError("newAgent", err)
	}
	
	mcpManager := mcp.NewManager(options.MCPConfig)
	
	ragRetriever, err := rag.NewRetriever(&options.RAGConfig)
	if err != nil {
		return nil, WrapRAGError("newAgent", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	agent := &Agent{
		options:      options,
		chatClient:   chatClient,
		mcpManager:   mcpManager,
		ragRetriever: ragRetriever,
		stats:        NewAgentStats(),
		errorStats:   NewErrorStats(),
		ctx:          ctx,
		cancel:       cancel,
	}
	
	// 设置系统提示
	if options.SystemPrompt != "" {
		chatClient.SetSystemPrompt(options.SystemPrompt)
	}
	
	return agent, nil
}

// Start 启动 Agent
func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.started {
		return WrapAgentError("start", "agent already started", ErrAgentAlreadyStarted, false)
	}
	
	// 启动 MCP 管理器
	if err := a.mcpManager.Start(ctx); err != nil {
		return WrapMCPError("start", err)
	}
	
	// 注册 MCP 工具到 Chat 客户端
	if err := a.registerMCPTools(); err != nil {
		return WrapAgentError("start", "failed to register MCP tools", err, false)
	}
	
	a.started = true
	
	// 启动指标收集（如果启用）
	if a.options.EnableMetrics {
		go a.metricsCollector()
	}
	
	return nil
}

// Stop 停止 Agent
func (a *Agent) Stop() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if !a.started {
		return nil
	}
	
	// 取消上下文
	a.cancel()
	
	// 停止 MCP 管理器
	if err := a.mcpManager.Stop(); err != nil {
		return WrapMCPError("stop", err)
	}
	
	a.started = false
	return nil
}

// IsStarted 检查 Agent 是否已启动
func (a *Agent) IsStarted() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.started
}

// Process 处理请求
func (a *Agent) Process(ctx context.Context, req Request) (*Response, error) {
	if !a.IsStarted() {
		return nil, WrapAgentError("process", "agent not started", ErrAgentNotStarted, false)
	}
	
	start := time.Now()
	defer func() {
		a.stats.RecordRequest(time.Since(start))
	}()
	
	// 增加并发计数
	a.stats.IncrementConcurrentRequests()
	defer a.stats.DecrementConcurrentRequests()
	
	response := &Response{
		ID:        req.ID,
		Timestamp: time.Now(),
	}
	
	// 执行重试逻辑
	var err error
	for attempt := 0; attempt <= a.options.MaxRetries; attempt++ {
		response, err = a.executeProcess(ctx, req)
		if err == nil {
			response.ResponseTime = time.Since(start)
			return response, nil
		}
		
		// 记录错误
		a.errorStats.RecordError(err)
		
		// 检查是否应该重试
		if !ShouldRetry(err, attempt, a.options.MaxRetries) {
			break
		}
		
		// 等待后重试
		delay := GetRetryDelay(err, attempt)
		
		select {
		case <-ctx.Done():
			return nil, WrapAgentError("process", "context canceled", ctx.Err(), false)
		case <-time.After(delay):
			// 继续重试
		}
	}
	
	response.Error = err.Error()
	response.ResponseTime = time.Since(start)
	return response, err
}

// ProcessStream 处理流式请求
func (a *Agent) ProcessStream(ctx context.Context, req Request) (<-chan StreamResponse, error) {
	if !a.IsStarted() {
		return nil, WrapAgentError("processStream", "agent not started", ErrAgentNotStarted, false)
	}
	
	respChan := make(chan StreamResponse, 100)
	
	go func() {
		defer close(respChan)
		
		// 增加并发计数
		a.stats.IncrementConcurrentRequests()
		defer a.stats.DecrementConcurrentRequests()
		
		start := time.Now()
		defer func() {
			a.stats.RecordRequest(time.Since(start))
		}()
		
		err := a.executeProcessStream(ctx, req, respChan)
		if err != nil {
			a.errorStats.RecordError(err)
			respChan <- StreamResponse{
				ID:        req.ID,
				Error:     err,
				Finished:  true,
				Timestamp: time.Now(),
			}
		}
	}()
	
	return respChan, nil
}

// executeProcess 执行处理请求
func (a *Agent) executeProcess(ctx context.Context, req Request) (*Response, error) {
	// 准备消息
	messages, err := a.prepareMessages(ctx, req)
	if err != nil {
		return nil, WrapAgentError("executeProcess", "failed to prepare messages", err, true)
	}
	
	// 重置工具调用计数
	a.mu.Lock()
	a.toolCalls = 0
	a.mu.Unlock()
	
	// 执行聊天循环
	if req.EnableTools {
		return a.executeWithTools(ctx, req, messages)
	}
	
	return a.executeWithoutTools(ctx, req, messages)
}

// executeProcessStream 执行流式处理请求
func (a *Agent) executeProcessStream(ctx context.Context, req Request, respChan chan<- StreamResponse) error {
	// 准备消息
	messages, err := a.prepareMessages(ctx, req)
	if err != nil {
		return WrapAgentError("executeProcessStream", "failed to prepare messages", err, true)
	}
	
	// 重置工具调用计数
	a.mu.Lock()
	a.toolCalls = 0
	a.mu.Unlock()
	
	// 执行流式聊天
	if req.EnableTools {
		return a.executeStreamWithTools(ctx, req, messages, respChan)
	}
	
	return a.executeStreamWithoutTools(ctx, req, messages, respChan)
}

// prepareMessages 准备消息
func (a *Agent) prepareMessages(ctx context.Context, req Request) ([]chat.Message, error) {
	var messages []chat.Message
	
	// 添加用户查询
	userMsg := chat.Message{
		Role:    chat.RoleUser,
		Content: req.Query,
	}
	
	// 如果启用 RAG，检索相关上下文
	if req.EnableRAG && a.options.EnableRAGContext {
		ragStart := time.Now()
		
		ragQuery := rag.Query{
			Text:      req.Query,
			TopK:      5,
			Threshold: 0.5,
		}
		
		result, err := a.ragRetriever.Retrieve(ctx, ragQuery)
		if err != nil {
			// RAG 失败不应该中断整个流程，记录错误但继续
			a.errorStats.RecordError(WrapRAGError("prepareMessages", err))
		} else {
			ragLatency := time.Since(ragStart)
			hitRate := float64(len(result.Documents)) / float64(ragQuery.TopK)
			a.stats.RecordRAGQuery(ragLatency, hitRate)
			
			// 构建 RAG 上下文
			if len(result.Documents) > 0 {
				ragContext := a.buildRAGContext(result.Documents)
				userMsg.Content = ragContext + "\n\n" + req.Query
			}
		}
	}
	
	// 添加用户提供的上下文
	if len(req.Context) > 0 {
		contextStr := ""
		for i, ctx := range req.Context {
			contextStr += fmt.Sprintf("Context %d: %s\n", i+1, ctx)
		}
		userMsg.Content = contextStr + "\n" + userMsg.Content
	}
	
	messages = append(messages, userMsg)
	
	// 检查上下文长度
	if err := a.checkContextLength(messages); err != nil {
		return nil, err
	}
	
	return messages, nil
}

// buildRAGContext 构建 RAG 上下文
func (a *Agent) buildRAGContext(docs []rag.Document) string {
	context := "Relevant information from knowledge base:\n\n"
	
	for i, doc := range docs {
		if len(context) > a.options.RAGContextLength {
			break
		}
		
		context += fmt.Sprintf("%d. %s\n", i+1, doc.Content)
		if doc.Metadata != nil {
			if source, ok := doc.Metadata["source"]; ok {
				context += fmt.Sprintf("   Source: %v\n", source)
			}
		}
		context += "\n"
	}
	
	return context
}

// checkContextLength 检查上下文长度
func (a *Agent) checkContextLength(messages []chat.Message) error {
	totalLength := 0
	for _, msg := range messages {
		totalLength += len(msg.Content)
	}
	
	if totalLength > a.options.MaxContextLength {
		return WrapAgentError("checkContextLength", "context too long", ErrContextTooLong, false)
	}
	
	return nil
}

// executeWithTools 执行带工具的处理
func (a *Agent) executeWithTools(ctx context.Context, req Request, messages []chat.Message) (*Response, error) {
	response, err := a.chatClient.ChatWithTools(ctx, messages)
	if err != nil {
		return nil, WrapChatError("executeWithTools", err)
	}
	
	// 转换响应
	agentResp := &Response{
		ID:      req.ID,
		Content: response.Content,
	}
	
	// 处理工具调用
	for _, toolCall := range response.ToolCalls {
		agentToolCall := ToolCall{
			ID:   toolCall.ID,
			Name: toolCall.Function.Name,
			Args: toolCall.Function.Arguments,
		}
		
		start := time.Now()
		
		// 这里工具调用已经在 ChatWithTools 中执行了
		// 我们只需要记录统计信息
		duration := time.Since(start)
		a.stats.RecordToolCall(toolCall.Function.Name, duration)
		
		agentToolCall.Duration = duration
		agentResp.ToolCalls = append(agentResp.ToolCalls, agentToolCall)
	}
	
	// 记录 token 使用情况
	agentResp.TokenUsage = TokenUsage{
		PromptTokens:     response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:      response.Usage.TotalTokens,
	}
	
	return agentResp, nil
}

// executeWithoutTools 执行不带工具的处理
func (a *Agent) executeWithoutTools(ctx context.Context, req Request, messages []chat.Message) (*Response, error) {
	response, err := a.chatClient.Chat(ctx, messages)
	if err != nil {
		return nil, WrapChatError("executeWithoutTools", err)
	}
	
	return &Response{
		ID:      req.ID,
		Content: response.Content,
		TokenUsage: TokenUsage{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}, nil
}

// executeStreamWithTools 执行带工具的流式处理
func (a *Agent) executeStreamWithTools(ctx context.Context, req Request, messages []chat.Message, respChan chan<- StreamResponse) error {
	streamChan, err := a.chatClient.ChatStreamWithTools(ctx, messages)
	if err != nil {
		return WrapChatError("executeStreamWithTools", err)
	}
	
	for streamResp := range streamChan {
		if streamResp.Error != nil {
			return WrapChatError("executeStreamWithTools", streamResp.Error)
		}
		
		agentStreamResp := StreamResponse{
			ID:        req.ID,
			Content:   streamResp.Content,
			Finished:  streamResp.Finished,
			Timestamp: time.Now(),
		}
		
		// 处理工具调用
		for _, toolCall := range streamResp.ToolCalls {
			agentToolCall := ToolCall{
				ID:   toolCall.ID,
				Name: toolCall.Function.Name,
				Args: toolCall.Function.Arguments,
			}
			agentStreamResp.ToolCalls = append(agentStreamResp.ToolCalls, agentToolCall)
		}
		
		select {
		case <-ctx.Done():
			return WrapAgentError("executeStreamWithTools", "context canceled", ctx.Err(), false)
		case respChan <- agentStreamResp:
			if streamResp.Finished {
				return nil
			}
		}
	}
	
	return nil
}

// executeStreamWithoutTools 执行不带工具的流式处理
func (a *Agent) executeStreamWithoutTools(ctx context.Context, req Request, messages []chat.Message, respChan chan<- StreamResponse) error {
	streamChan, err := a.chatClient.ChatStream(ctx, messages)
	if err != nil {
		return WrapChatError("executeStreamWithoutTools", err)
	}
	
	for streamResp := range streamChan {
		if streamResp.Error != nil {
			return WrapChatError("executeStreamWithoutTools", streamResp.Error)
		}
		
		agentStreamResp := StreamResponse{
			ID:        req.ID,
			Content:   streamResp.Content,
			Finished:  streamResp.Finished,
			Timestamp: time.Now(),
		}
		
		select {
		case <-ctx.Done():
			return WrapAgentError("executeStreamWithoutTools", "context canceled", ctx.Err(), false)
		case respChan <- agentStreamResp:
			if streamResp.Finished {
				return nil
			}
		}
	}
	
	return nil
}

// registerMCPTools 注册 MCP 工具
func (a *Agent) registerMCPTools() error {
	// 获取所有可用工具
	tools, err := a.mcpManager.ListAllTools(a.ctx)
	if err != nil {
		return WrapMCPError("registerMCPTools", err)
	}
	
	// 为每个工具创建处理器
	for _, tool := range tools {
		handler := &MCPToolHandler{
			manager: a.mcpManager,
			tool:    tool,
		}
		
		if err := a.chatClient.RegisterTool(handler); err != nil {
			return WrapAgentError("registerMCPTools", 
				fmt.Sprintf("failed to register tool %s", tool.Name), err, false)
		}
	}
	
	return nil
}

// GetStats 获取统计信息
func (a *Agent) GetStats() AgentStats {
	return a.stats.GetAgentStats()
}

// GetErrorStats 获取错误统计
func (a *Agent) GetErrorStats() *ErrorStats {
	return a.errorStats
}

// ResetStats 重置统计信息
func (a *Agent) ResetStats() {
	a.stats.Reset()
	a.errorStats.Reset()
}

// metricsCollector 指标收集器
func (a *Agent) metricsCollector() {
	ticker := time.NewTicker(a.options.MetricsInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.collectMetrics()
		}
	}
}

// collectMetrics 收集指标
func (a *Agent) collectMetrics() {
	// 在实际实现中，这里会收集 Prometheus 指标
	// 目前只是一个占位符
	stats := a.GetStats()
	errorStats := a.GetErrorStats()
	
	// 记录指标（示例）
	_ = stats
	_ = errorStats
}

// MCPToolHandler MCP 工具处理器
type MCPToolHandler struct {
	manager *mcp.Manager
	tool    mcp.Tool
}

// GetName 获取工具名称
func (h *MCPToolHandler) GetName() string {
	return h.tool.Name
}

// GetDescription 获取工具描述
func (h *MCPToolHandler) GetDescription() string {
	return h.tool.Description
}

// GetParameters 获取工具参数
func (h *MCPToolHandler) GetParameters() map[string]interface{} {
	return h.tool.InputSchema
}

// Execute 执行工具
func (h *MCPToolHandler) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	result, err := h.manager.CallTool(ctx, h.tool.Name, args)
	if err != nil {
		return "", err
	}
	
	if len(result.Content) > 0 {
		return result.Content[0].Text, nil
	}
	
	return "", nil
}

// UpdateOptions 更新配置选项
func (a *Agent) UpdateOptions(opts ...Option) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	newOptions := a.options.Clone()
	for _, opt := range opts {
		opt(&newOptions)
	}
	
	if err := newOptions.Validate(); err != nil {
		return WrapAgentError("updateOptions", "invalid options", err, false)
	}
	
	a.options = newOptions
	return nil
}

// GetOptions 获取当前配置选项
func (a *Agent) GetOptions() Options {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.options.Clone()
}

// SetSystemPrompt 更新系统提示
func (a *Agent) SetSystemPrompt(prompt string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	a.options.SystemPrompt = prompt
	a.chatClient.SetSystemPrompt(prompt)
}

// GetSystemPrompt 获取当前系统提示
func (a *Agent) GetSystemPrompt() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.options.SystemPrompt
}

// Health 健康检查
func (a *Agent) Health() map[string]interface{} {
	health := map[string]interface{}{
		"started": a.IsStarted(),
		"uptime":  time.Since(a.stats.StartTime).String(),
	}
	
	if a.IsStarted() {
		health["mcpManager"] = a.mcpManager.GetClientStatus()
		health["stats"] = a.GetStats()
		health["errors"] = a.GetErrorStats()
	}
	
	return health
}