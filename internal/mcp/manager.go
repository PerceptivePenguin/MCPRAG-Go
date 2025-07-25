package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager MCP 管理器，负责管理所有 MCP 客户端
type Manager struct {
	registry *ClientRegistry
	config   ManagerConfig
	mu       sync.RWMutex
	started  bool
}

// ManagerConfig 管理器配置
type ManagerConfig struct {
	AutoReconnect     bool          `json:"autoReconnect"`
	ReconnectInterval time.Duration `json:"reconnectInterval"`
	HealthCheckInterval time.Duration `json:"healthCheckInterval"`
	EnableHealthCheck bool          `json:"enableHealthCheck"`
}

// DefaultManagerConfig 返回默认的管理器配置
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		AutoReconnect:       true,
		ReconnectInterval:   30 * time.Second,
		HealthCheckInterval: 60 * time.Second,
		EnableHealthCheck:   true,
	}
}

// NewManager 创建新的 MCP 管理器
func NewManager(config ManagerConfig) *Manager {
	return &Manager{
		registry: NewClientRegistry(),
		config:   config,
		started:  false,
	}
}

// Start 启动 MCP 管理器
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.started {
		return nil
	}
	
	// 初始化所有客户端
	if err := m.registry.InitializeAll(ctx); err != nil {
		return fmt.Errorf("failed to initialize clients: %w", err)
	}
	
	m.started = true
	
	// 启动健康检查和自动重连（如果启用）
	if m.config.EnableHealthCheck {
		go m.healthCheckLoop(ctx)
	}
	
	return nil
}

// Stop 停止 MCP 管理器
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !m.started {
		return nil
	}
	
	// 关闭所有客户端
	if err := m.registry.CloseAll(); err != nil {
		return fmt.Errorf("failed to close all clients: %w", err)
	}
	
	m.started = false
	return nil
}

// RegisterDeepWikiClient 注册 DeepWiki 客户端
func (m *Manager) RegisterDeepWikiClient(config ClientConfig) error {
	client := NewDeepWikiClient(config)
	return m.registry.RegisterClient("deepwiki", client)
}

// RegisterContext7Client 注册 Context7 客户端
func (m *Manager) RegisterContext7Client(config ClientConfig) error {
	client := NewContext7Client(config)
	return m.registry.RegisterClient("context7", client)
}

// RegisterClient 注册自定义客户端
func (m *Manager) RegisterClient(name string, client MCPClient) error {
	return m.registry.RegisterClient(name, client)
}

// UnregisterClient 注销客户端
func (m *Manager) UnregisterClient(name string) error {
	return m.registry.UnregisterClient(name)
}

// ListClients 列出所有客户端
func (m *Manager) ListClients() []string {
	return m.registry.ListClients()
}

// GetClientStatus 获取客户端状态
func (m *Manager) GetClientStatus() map[string]bool {
	return m.registry.GetStatus()
}

// ListAllTools 列出所有可用工具
func (m *Manager) ListAllTools(ctx context.Context) ([]Tool, error) {
	m.mu.RLock()
	started := m.started
	m.mu.RUnlock()
	
	if !started {
		return nil, fmt.Errorf("manager not started")
	}
	
	return m.registry.ListAllTools(ctx)
}

// CallTool 调用指定工具
func (m *Manager) CallTool(ctx context.Context, toolName string, args interface{}) (*ToolResult, error) {
	m.mu.RLock()
	started := m.started
	m.mu.RUnlock()
	
	if !started {
		return nil, NewMCPError("callTool", "manager", toolName, 
			fmt.Errorf("manager not started"), false)
	}
	
	return m.registry.CallTool(ctx, toolName, args)
}

// RefreshTools 刷新工具列表
func (m *Manager) RefreshTools(ctx context.Context) error {
	m.mu.RLock()
	started := m.started
	m.mu.RUnlock()
	
	if !started {
		return fmt.Errorf("manager not started")
	}
	
	return m.registry.RefreshTools(ctx)
}

// GetToolsForClient 获取指定客户端的工具
func (m *Manager) GetToolsForClient(ctx context.Context, clientName string) ([]Tool, error) {
	m.mu.RLock()
	started := m.started
	m.mu.RUnlock()
	
	if !started {
		return nil, fmt.Errorf("manager not started")
	}
	
	return m.registry.GetToolsForClient(ctx, clientName)
}

// healthCheckLoop 健康检查循环
func (m *Manager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performHealthCheck(ctx)
		}
	}
}

// performHealthCheck 执行健康检查
func (m *Manager) performHealthCheck(ctx context.Context) {
	status := m.registry.GetStatus()
	
	for clientName, connected := range status {
		if !connected && m.config.AutoReconnect {
			go m.attemptReconnection(ctx, clientName)
		}
	}
}

// attemptReconnection 尝试重连客户端
func (m *Manager) attemptReconnection(ctx context.Context, clientName string) {
	client, exists := m.registry.GetClient(clientName)
	if !exists {
		return
	}
	
	// 尝试重新初始化
	if err := client.Initialize(ctx); err != nil {
		// 记录错误（在实际实现中应该使用日志系统）
		fmt.Printf("Failed to reconnect client %s: %v\n", clientName, err)
		return
	}
	
	// 刷新工具列表
	if err := m.registry.RefreshTools(ctx); err != nil {
		fmt.Printf("Failed to refresh tools after reconnecting %s: %v\n", clientName, err)
	}
}

// GetClient 获取指定客户端
func (m *Manager) GetClient(name string) (MCPClient, bool) {
	return m.registry.GetClient(name)
}

// GetClientByTool 根据工具名获取客户端
func (m *Manager) GetClientByTool(toolName string) (MCPClient, string, bool) {
	return m.registry.GetClientByTool(toolName)
}

// IsStarted 检查管理器是否已启动
func (m *Manager) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}

// SetAutoReconnect 设置自动重连
func (m *Manager) SetAutoReconnect(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.AutoReconnect = enabled
}

// SetHealthCheck 设置健康检查
func (m *Manager) SetHealthCheck(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config.EnableHealthCheck = enabled
}

// GetConfig 获取管理器配置
func (m *Manager) GetConfig() ManagerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// UpdateConfig 更新管理器配置
func (m *Manager) UpdateConfig(config ManagerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}