package mcp

import (
	"context"
	"fmt"
	"sync"
)

// ClientRegistry MCP 客户端注册表
type ClientRegistry struct {
	mu      sync.RWMutex
	clients map[string]MCPClient
	tools   map[string]string // tool name -> client name
}

// NewClientRegistry 创建新的客户端注册表
func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients: make(map[string]MCPClient),
		tools:   make(map[string]string),
	}
}

// RegisterClient 注册 MCP 客户端
func (r *ClientRegistry) RegisterClient(name string, client MCPClient) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.clients[name]; exists {
		return fmt.Errorf("client %s already registered", name)
	}
	
	r.clients[name] = client
	return nil
}

// UnregisterClient 注销 MCP 客户端
func (r *ClientRegistry) UnregisterClient(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	client, exists := r.clients[name]
	if !exists {
		return fmt.Errorf("client %s not found", name)
	}
	
	// 关闭客户端连接
	if err := client.Close(); err != nil {
		return fmt.Errorf("failed to close client %s: %w", name, err)
	}
	
	// 移除客户端
	delete(r.clients, name)
	
	// 移除相关工具
	for toolName, clientName := range r.tools {
		if clientName == name {
			delete(r.tools, toolName)
		}
	}
	
	return nil
}

// GetClient 获取指定名称的客户端
func (r *ClientRegistry) GetClient(name string) (MCPClient, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	client, exists := r.clients[name]
	return client, exists
}

// GetClientByTool 根据工具名称获取对应的客户端
func (r *ClientRegistry) GetClientByTool(toolName string) (MCPClient, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	clientName, exists := r.tools[toolName]
	if !exists {
		return nil, "", false
	}
	
	client, exists := r.clients[clientName]
	if !exists {
		return nil, "", false
	}
	
	return client, clientName, true
}

// ListClients 列出所有注册的客户端
func (r *ClientRegistry) ListClients() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	names := make([]string, 0, len(r.clients))
	for name := range r.clients {
		names = append(names, name)
	}
	return names
}

// RefreshTools 刷新所有客户端的工具列表
func (r *ClientRegistry) RefreshTools(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 清空现有工具映射
	r.tools = make(map[string]string)
	
	// 遍历所有客户端，获取工具列表
	for clientName, client := range r.clients {
		if !client.IsConnected() {
			continue
		}
		
		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools for client %s: %w", clientName, err)
		}
		
		// 注册工具到映射表
		for _, tool := range tools {
			if existingClient, exists := r.tools[tool.Name]; exists {
				return fmt.Errorf("tool %s conflicts between clients %s and %s", 
					tool.Name, existingClient, clientName)
			}
			r.tools[tool.Name] = clientName
		}
	}
	
	return nil
}

// ListAllTools 列出所有可用工具
func (r *ClientRegistry) ListAllTools(ctx context.Context) ([]Tool, error) {
	r.mu.RLock()
	clients := make(map[string]MCPClient, len(r.clients))
	for name, client := range r.clients {
		clients[name] = client
	}
	r.mu.RUnlock()
	
	var allTools []Tool
	
	// 遍历所有客户端，收集工具
	for clientName, client := range clients {
		if !client.IsConnected() {
			continue
		}
		
		tools, err := client.ListTools(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list tools for client %s: %w", clientName, err)
		}
		
		allTools = append(allTools, tools...)
	}
	
	return allTools, nil
}

// CallTool 调用指定工具
func (r *ClientRegistry) CallTool(ctx context.Context, toolName string, args interface{}) (*ToolResult, error) {
	client, clientName, exists := r.GetClientByTool(toolName)
	if !exists {
		return nil, NewMCPError("callTool", "registry", toolName, ErrToolNotFound, false)
	}
	
	if !client.IsConnected() {
		return nil, NewMCPError("callTool", clientName, toolName, ErrClientNotConnected, false)
	}
	
	return client.CallTool(ctx, toolName, args)
}

// InitializeAll 初始化所有客户端
func (r *ClientRegistry) InitializeAll(ctx context.Context) error {
	r.mu.RLock()
	clients := make(map[string]MCPClient, len(r.clients))
	for name, client := range r.clients {
		clients[name] = client
	}
	r.mu.RUnlock()
	
	for name, client := range clients {
		if err := client.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize client %s: %w", name, err)
		}
	}
	
	// 初始化后刷新工具列表
	return r.RefreshTools(ctx)
}

// CloseAll 关闭所有客户端
func (r *ClientRegistry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var errors []error
	
	for name, client := range r.clients {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close client %s: %w", name, err))
		}
	}
	
	// 清空注册表
	r.clients = make(map[string]MCPClient)
	r.tools = make(map[string]string)
	
	if len(errors) > 0 {
		return fmt.Errorf("errors occurred while closing clients: %v", errors)
	}
	
	return nil
}

// GetStatus 获取所有客户端的状态
func (r *ClientRegistry) GetStatus() map[string]bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	status := make(map[string]bool, len(r.clients))
	for name, client := range r.clients {
		status[name] = client.IsConnected()
	}
	
	return status
}

// GetToolsForClient 获取指定客户端的工具列表
func (r *ClientRegistry) GetToolsForClient(ctx context.Context, clientName string) ([]Tool, error) {
	client, exists := r.GetClient(clientName)
	if !exists {
		return nil, fmt.Errorf("client %s not found", clientName)
	}
	
	if !client.IsConnected() {
		return nil, fmt.Errorf("client %s not connected", clientName)
	}
	
	return client.ListTools(ctx)
}