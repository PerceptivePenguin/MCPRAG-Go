package mcp

import (
	"context"
	"testing"
)

func TestNewClientRegistry(t *testing.T) {
	registry := NewClientRegistry()
	
	if registry == nil {
		t.Fatal("Expected non-nil registry")
	}
	
	if len(registry.ListClients()) != 0 {
		t.Error("Expected empty client list for new registry")
	}
}

func TestClientRegistryRegisterClient(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	
	// Test successful registration
	err := registry.RegisterClient("deepwiki", client)
	if err != nil {
		t.Errorf("Expected no error registering client, got %v", err)
	}
	
	clients := registry.ListClients()
	if len(clients) != 1 {
		t.Errorf("Expected 1 client, got %d", len(clients))
	}
	
	if clients[0] != "deepwiki" {
		t.Errorf("Expected client name 'deepwiki', got '%s'", clients[0])
	}
	
	// Test duplicate registration
	err = registry.RegisterClient("deepwiki", client)
	if err == nil {
		t.Error("Expected error when registering duplicate client")
	}
}

func TestClientRegistryGetClient(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	
	// Test getting non-existent client
	_, exists := registry.GetClient("nonexistent")
	if exists {
		t.Error("Expected false for non-existent client")
	}
	
	// Register and get client
	registry.RegisterClient("deepwiki", client)
	retrievedClient, exists := registry.GetClient("deepwiki")
	
	if !exists {
		t.Error("Expected true for existing client")
	}
	
	if retrievedClient != client {
		t.Error("Expected same client instance")
	}
}

func TestClientRegistryUnregisterClient(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	
	// Test unregistering non-existent client
	err := registry.UnregisterClient("nonexistent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent client")
	}
	
	// Register and unregister client
	registry.RegisterClient("deepwiki", client)
	err = registry.UnregisterClient("deepwiki")
	if err != nil {
		t.Errorf("Expected no error unregistering client, got %v", err)
	}
	
	if len(registry.ListClients()) != 0 {
		t.Error("Expected empty client list after unregistering")
	}
}

func TestClientRegistryRefreshTools(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	
	// Register and initialize clients
	deepwikiClient := NewDeepWikiClient(config)
	context7Client := NewContext7Client(config)
	
	ctx := context.Background()
	deepwikiClient.Initialize(ctx)
	context7Client.Initialize(ctx)
	
	registry.RegisterClient("deepwiki", deepwikiClient)
	registry.RegisterClient("context7", context7Client)
	
	// Refresh tools
	err := registry.RefreshTools(ctx)
	if err != nil {
		t.Errorf("Expected no error refreshing tools, got %v", err)
	}
	
	// Test tool lookup
	client, clientName, exists := registry.GetClientByTool("deepwiki_fetch")
	if !exists {
		t.Error("Expected to find deepwiki_fetch tool")
	}
	if clientName != "deepwiki" {
		t.Errorf("Expected client name 'deepwiki', got '%s'", clientName)
	}
	if client != deepwikiClient {
		t.Error("Expected correct client instance")
	}
	
	// Test Context7 tools
	_, clientName, exists = registry.GetClientByTool("resolve-library-id")
	if !exists {
		t.Error("Expected to find resolve-library-id tool")
	}
	if clientName != "context7" {
		t.Errorf("Expected client name 'context7', got '%s'", clientName)
	}
}

func TestClientRegistryListAllTools(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	ctx := context.Background()
	
	// Test with no clients
	tools, err := registry.ListAllTools(ctx)
	if err != nil {
		t.Errorf("Expected no error with empty registry, got %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("Expected 0 tools, got %d", len(tools))
	}
	
	// Add clients
	deepwikiClient := NewDeepWikiClient(config)
	context7Client := NewContext7Client(config)
	
	deepwikiClient.Initialize(ctx)
	context7Client.Initialize(ctx)
	
	registry.RegisterClient("deepwiki", deepwikiClient)
	registry.RegisterClient("context7", context7Client)
	
	tools, err = registry.ListAllTools(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Should have 3 tools: deepwiki_fetch, resolve-library-id, get-library-docs
	expectedToolCount := 3
	if len(tools) != expectedToolCount {
		t.Errorf("Expected %d tools, got %d", expectedToolCount, len(tools))
	}
}

func TestClientRegistryCallTool(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	ctx := context.Background()
	
	// Test calling non-existent tool
	_, err := registry.CallTool(ctx, "nonexistent", nil)
	if err == nil {
		t.Error("Expected error when calling non-existent tool")
	}
	
	// Add and initialize client
	deepwikiClient := NewDeepWikiClient(config)
	deepwikiClient.Initialize(ctx)
	registry.RegisterClient("deepwiki", deepwikiClient)
	registry.RefreshTools(ctx)
	
	// Test successful tool call
	args := DeepWikiArgs{URL: "test"}
	result, err := registry.CallTool(ctx, "deepwiki_fetch", args)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if result == nil {
		t.Error("Expected non-nil result")
	}
}

func TestClientRegistryInitializeAll(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	ctx := context.Background()
	
	// Add clients
	deepwikiClient := NewDeepWikiClient(config)
	context7Client := NewContext7Client(config)
	
	registry.RegisterClient("deepwiki", deepwikiClient)
	registry.RegisterClient("context7", context7Client)
	
	// Initialize all
	err := registry.InitializeAll(ctx)
	if err != nil {
		t.Errorf("Expected no error initializing all, got %v", err)
	}
	
	// Check that clients are connected
	if !deepwikiClient.IsConnected() {
		t.Error("Expected deepwiki client to be connected")
	}
	
	if !context7Client.IsConnected() {
		t.Error("Expected context7 client to be connected")
	}
}

func TestClientRegistryCloseAll(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	ctx := context.Background()
	
	// Add and initialize clients
	deepwikiClient := NewDeepWikiClient(config)
	context7Client := NewContext7Client(config)
	
	deepwikiClient.Initialize(ctx)
	context7Client.Initialize(ctx)
	
	registry.RegisterClient("deepwiki", deepwikiClient)
	registry.RegisterClient("context7", context7Client)
	registry.RefreshTools(ctx)
	
	// Close all
	err := registry.CloseAll()
	if err != nil {
		t.Errorf("Expected no error closing all, got %v", err)
	}
	
	// Check that registry is empty
	if len(registry.ListClients()) != 0 {
		t.Error("Expected empty client list after closing all")
	}
	
	// Check that clients are disconnected
	if deepwikiClient.IsConnected() {
		t.Error("Expected deepwiki client to be disconnected")
	}
	
	if context7Client.IsConnected() {
		t.Error("Expected context7 client to be disconnected")
	}
}

func TestClientRegistryGetStatus(t *testing.T) {
	registry := NewClientRegistry()
	config := DefaultClientConfig()
	ctx := context.Background()
	
	// Add clients
	deepwikiClient := NewDeepWikiClient(config)
	context7Client := NewContext7Client(config)
	
	registry.RegisterClient("deepwiki", deepwikiClient)
	registry.RegisterClient("context7", context7Client)
	
	// Get status before initialization
	status := registry.GetStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 clients in status, got %d", len(status))
	}
	
	if status["deepwiki"] {
		t.Error("Expected deepwiki to be disconnected")
	}
	
	if status["context7"] {
		t.Error("Expected context7 to be disconnected")
	}
	
	// Initialize and check status
	deepwikiClient.Initialize(ctx)
	status = registry.GetStatus()
	
	if !status["deepwiki"] {
		t.Error("Expected deepwiki to be connected")
	}
	
	if status["context7"] {
		t.Error("Expected context7 to still be disconnected")
	}
}