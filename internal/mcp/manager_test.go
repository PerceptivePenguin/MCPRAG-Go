package mcp

import (
	"context"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	config := DefaultManagerConfig()
	manager := NewManager(config)
	
	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
	
	if manager.IsStarted() {
		t.Error("Expected manager to be initially stopped")
	}
	
	if len(manager.ListClients()) != 0 {
		t.Error("Expected empty client list for new manager")
	}
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()
	
	if !config.AutoReconnect {
		t.Error("Expected AutoReconnect to be true by default")
	}
	
	if config.ReconnectInterval != 30*time.Second {
		t.Errorf("Expected ReconnectInterval 30s, got %v", config.ReconnectInterval)
	}
	
	if config.HealthCheckInterval != 60*time.Second {
		t.Errorf("Expected HealthCheckInterval 60s, got %v", config.HealthCheckInterval)
	}
	
	if !config.EnableHealthCheck {
		t.Error("Expected EnableHealthCheck to be true by default")
	}
}

func TestManagerRegisterClients(t *testing.T) {
	config := DefaultManagerConfig()
	manager := NewManager(config)
	
	// Register DeepWiki client
	clientConfig := DefaultClientConfig()
	err := manager.RegisterDeepWikiClient(clientConfig)
	if err != nil {
		t.Errorf("Expected no error registering DeepWiki client, got %v", err)
	}
	
	// Register Context7 client
	err = manager.RegisterContext7Client(clientConfig)
	if err != nil {
		t.Errorf("Expected no error registering Context7 client, got %v", err)
	}
	
	clients := manager.ListClients()
	expectedClients := 2
	if len(clients) != expectedClients {
		t.Errorf("Expected %d clients, got %d", len(clients), expectedClients)
	}
}

func TestManagerStartStop(t *testing.T) {
	config := DefaultManagerConfig()
	// Disable health check for this test to avoid background goroutines
	config.EnableHealthCheck = false
	manager := NewManager(config)
	
	// Register clients
	clientConfig := DefaultClientConfig()
	manager.RegisterDeepWikiClient(clientConfig)
	manager.RegisterContext7Client(clientConfig)
	
	ctx := context.Background()
	
	// Test start
	err := manager.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error starting manager, got %v", err)
	}
	
	if !manager.IsStarted() {
		t.Error("Expected manager to be started")
	}
	
	// Test double start (should be no-op)
	err = manager.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error on double start, got %v", err)
	}
	
	// Test stop
	err = manager.Stop()
	if err != nil {
		t.Errorf("Expected no error stopping manager, got %v", err)
	}
	
	if manager.IsStarted() {
		t.Error("Expected manager to be stopped")
	}
	
	// Test double stop (should be no-op)
	err = manager.Stop()
	if err != nil {
		t.Errorf("Expected no error on double stop, got %v", err)
	}
}

func TestManagerListAllTools(t *testing.T) {
	config := DefaultManagerConfig()
	config.EnableHealthCheck = false
	manager := NewManager(config)
	
	ctx := context.Background()
	
	// Test before starting
	_, err := manager.ListAllTools(ctx)
	if err == nil {
		t.Error("Expected error when listing tools before starting")
	}
	
	// Register clients and start
	clientConfig := DefaultClientConfig()
	manager.RegisterDeepWikiClient(clientConfig)
	manager.RegisterContext7Client(clientConfig)
	
	err = manager.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error starting manager, got %v", err)
	}
	
	tools, err := manager.ListAllTools(ctx)
	if err != nil {
		t.Errorf("Expected no error listing tools, got %v", err)
	}
	
	// Should have 3 tools: deepwiki_fetch, resolve-library-id, get-library-docs
	expectedToolCount := 3
	if len(tools) != expectedToolCount {
		t.Errorf("Expected %d tools, got %d", expectedToolCount, len(tools))
	}
	
	// Check specific tools
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	
	expectedTools := []string{"deepwiki_fetch", "resolve-library-id", "get-library-docs"}
	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool '%s' not found", expectedTool)
		}
	}
}

func TestManagerCallTool(t *testing.T) {
	config := DefaultManagerConfig()
	config.EnableHealthCheck = false
	manager := NewManager(config)
	
	ctx := context.Background()
	
	// Test before starting
	_, err := manager.CallTool(ctx, "deepwiki_fetch", nil)
	if err == nil {
		t.Error("Expected error when calling tool before starting")
	}
	
	// Register clients and start
	clientConfig := DefaultClientConfig()
	manager.RegisterDeepWikiClient(clientConfig)
	manager.RegisterContext7Client(clientConfig)
	
	err = manager.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error starting manager, got %v", err)
	}
	
	// Test successful tool call
	args := DeepWikiArgs{URL: "test"}
	result, err := manager.CallTool(ctx, "deepwiki_fetch", args)
	if err != nil {
		t.Errorf("Expected no error calling tool, got %v", err)
	}
	
	if result == nil {
		t.Error("Expected non-nil result")
	}
	
	// Test calling non-existent tool
	_, err = manager.CallTool(ctx, "nonexistent", nil)
	if err == nil {
		t.Error("Expected error when calling non-existent tool")
	}
}

func TestManagerGetClientStatus(t *testing.T) {
	config := DefaultManagerConfig()
	config.EnableHealthCheck = false
	manager := NewManager(config)
	
	// Register clients
	clientConfig := DefaultClientConfig()
	manager.RegisterDeepWikiClient(clientConfig)
	manager.RegisterContext7Client(clientConfig)
	
	// Get status before starting
	status := manager.GetClientStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 clients in status, got %d", len(status))
	}
	
	if status["deepwiki"] {
		t.Error("Expected deepwiki to be disconnected before starting")
	}
	
	if status["context7"] {
		t.Error("Expected context7 to be disconnected before starting")
	}
	
	// Start manager and check status
	ctx := context.Background()
	err := manager.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error starting manager, got %v", err)
	}
	
	status = manager.GetClientStatus()
	if !status["deepwiki"] {
		t.Error("Expected deepwiki to be connected after starting")
	}
	
	if !status["context7"] {
		t.Error("Expected context7 to be connected after starting")
	}
}

func TestManagerRefreshTools(t *testing.T) {
	config := DefaultManagerConfig()
	config.EnableHealthCheck = false
	manager := NewManager(config)
	
	ctx := context.Background()
	
	// Test before starting
	err := manager.RefreshTools(ctx)
	if err == nil {
		t.Error("Expected error when refreshing tools before starting")
	}
	
	// Register clients and start
	clientConfig := DefaultClientConfig()
	manager.RegisterDeepWikiClient(clientConfig)
	manager.RegisterContext7Client(clientConfig)
	
	err = manager.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error starting manager, got %v", err)
	}
	
	// Refresh tools
	err = manager.RefreshTools(ctx)
	if err != nil {
		t.Errorf("Expected no error refreshing tools, got %v", err)
	}
}

func TestManagerGetToolsForClient(t *testing.T) {
	config := DefaultManagerConfig()
	config.EnableHealthCheck = false
	manager := NewManager(config)
	
	ctx := context.Background()
	
	// Register clients and start
	clientConfig := DefaultClientConfig()
	manager.RegisterDeepWikiClient(clientConfig)
	manager.RegisterContext7Client(clientConfig)
	
	err := manager.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error starting manager, got %v", err)
	}
	
	// Test getting tools for deepwiki client
	tools, err := manager.GetToolsForClient(ctx, "deepwiki")
	if err != nil {
		t.Errorf("Expected no error getting tools for deepwiki, got %v", err)
	}
	
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool for deepwiki, got %d", len(tools))
	}
	
	if tools[0].Name != "deepwiki_fetch" {
		t.Errorf("Expected tool name 'deepwiki_fetch', got '%s'", tools[0].Name)
	}
	
	// Test getting tools for context7 client
	tools, err = manager.GetToolsForClient(ctx, "context7")
	if err != nil {
		t.Errorf("Expected no error getting tools for context7, got %v", err)
	}
	
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools for context7, got %d", len(tools))
	}
	
	// Test getting tools for non-existent client
	_, err = manager.GetToolsForClient(ctx, "nonexistent")
	if err == nil {
		t.Error("Expected error getting tools for non-existent client")
	}
}

func TestManagerConfigMethods(t *testing.T) {
	config := DefaultManagerConfig()
	manager := NewManager(config)
	
	// Test getting config
	retrievedConfig := manager.GetConfig()
	if retrievedConfig.AutoReconnect != config.AutoReconnect {
		t.Error("Expected same AutoReconnect setting")
	}
	
	// Test setting auto reconnect
	manager.SetAutoReconnect(false)
	if manager.GetConfig().AutoReconnect {
		t.Error("Expected AutoReconnect to be false after setting")
	}
	
	// Test setting health check
	manager.SetHealthCheck(false)
	if manager.GetConfig().EnableHealthCheck {
		t.Error("Expected EnableHealthCheck to be false after setting")
	}
	
	// Test updating config
	newConfig := DefaultManagerConfig()
	newConfig.ReconnectInterval = 60 * time.Second
	manager.UpdateConfig(newConfig)
	
	if manager.GetConfig().ReconnectInterval != 60*time.Second {
		t.Error("Expected ReconnectInterval to be updated")
	}
}

func TestManagerUnregisterClient(t *testing.T) {
	config := DefaultManagerConfig()
	config.EnableHealthCheck = false
	manager := NewManager(config)
	
	// Register client
	clientConfig := DefaultClientConfig()
	err := manager.RegisterDeepWikiClient(clientConfig)
	if err != nil {
		t.Errorf("Expected no error registering client, got %v", err)
	}
	
	if len(manager.ListClients()) != 1 {
		t.Error("Expected 1 client after registration")
	}
	
	// Unregister client
	err = manager.UnregisterClient("deepwiki")
	if err != nil {
		t.Errorf("Expected no error unregistering client, got %v", err)
	}
	
	if len(manager.ListClients()) != 0 {
		t.Error("Expected 0 clients after unregistration")
	}
	
	// Test unregistering non-existent client
	err = manager.UnregisterClient("nonexistent")
	if err == nil {
		t.Error("Expected error unregistering non-existent client")
	}
}