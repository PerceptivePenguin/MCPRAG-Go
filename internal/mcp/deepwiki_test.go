package mcp

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewDeepWikiClient(t *testing.T) {
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	
	if client.config.ServerName != "deepwiki" {
		t.Errorf("Expected server name 'deepwiki', got '%s'", client.config.ServerName)
	}
	
	if client.connected {
		t.Error("Expected client to be initially disconnected")
	}
}

func TestDeepWikiClientInitialize(t *testing.T) {
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	
	ctx := context.Background()
	err := client.Initialize(ctx)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if !client.IsConnected() {
		t.Error("Expected client to be connected after initialization")
	}
	
	// Test double initialization
	err = client.Initialize(ctx)
	if err != nil {
		t.Errorf("Expected no error on double initialization, got %v", err)
	}
}

func TestDeepWikiClientListTools(t *testing.T) {
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	
	ctx := context.Background()
	
	// Test without initialization
	_, err := client.ListTools(ctx)
	if err == nil {
		t.Error("Expected error when listing tools without connection")
	}
	
	// Test with initialization
	client.Initialize(ctx)
	tools, err := client.ListTools(ctx)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
	
	if tools[0].Name != "deepwiki_fetch" {
		t.Errorf("Expected tool name 'deepwiki_fetch', got '%s'", tools[0].Name)
	}
}

func TestDeepWikiClientCallTool(t *testing.T) {
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	ctx := context.Background()
	
	// Initialize client
	client.Initialize(ctx)
	
	tests := []struct {
		name        string
		toolName    string
		args        interface{}
		expectError bool
	}{
		{
			name:     "Valid call with struct args",
			toolName: "deepwiki_fetch",
			args: DeepWikiArgs{
				URL:      "https://github.com/test/repo",
				MaxDepth: 1,
				Mode:     "aggregate",
				Verbose:  false,
			},
			expectError: false,
		},
		{
			name:     "Valid call with map args",
			toolName: "deepwiki_fetch",
			args: map[string]interface{}{
				"url":      "vercel/ai",
				"maxDepth": 0,
				"mode":     "pages",
				"verbose":  true,
			},
			expectError: false,
		},
		{
			name:        "Invalid tool name",
			toolName:    "invalid_tool",
			args:        DeepWikiArgs{URL: "test"},
			expectError: true,
		},
		{
			name:        "Missing required URL",
			toolName:    "deepwiki_fetch",
			args:        map[string]interface{}{},
			expectError: true,
		},
		{
			name:        "Invalid args type",
			toolName:    "deepwiki_fetch",
			args:        "invalid",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.CallTool(ctx, tt.toolName, tt.args)
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}
			
			if result == nil {
				t.Error("Expected non-nil result")
				return
			}
			
			if len(result.Content) == 0 {
				t.Error("Expected non-empty content")
			}
			
			if result.IsError {
				t.Error("Expected result to not be an error")
			}
		})
	}
}

func TestDeepWikiClientCallToolWithoutConnection(t *testing.T) {
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	ctx := context.Background()
	
	args := DeepWikiArgs{URL: "test"}
	_, err := client.CallTool(ctx, "deepwiki_fetch", args)
	
	if err == nil {
		t.Error("Expected error when calling tool without connection")
	}
	
	var mcpErr *MCPError
	if !errors.As(err, &mcpErr) {
		t.Error("Expected MCPError")
	}
}

func TestDeepWikiClientClose(t *testing.T) {
	config := DefaultClientConfig()
	client := NewDeepWikiClient(config)
	ctx := context.Background()
	
	// Test close without connection
	err := client.Close()
	if err != nil {
		t.Errorf("Expected no error closing disconnected client, got %v", err)
	}
	
	// Test close with connection
	client.Initialize(ctx)
	if !client.IsConnected() {
		t.Error("Expected client to be connected")
	}
	
	err = client.Close()
	if err != nil {
		t.Errorf("Expected no error closing connected client, got %v", err)
	}
	
	if client.IsConnected() {
		t.Error("Expected client to be disconnected after close")
	}
}

func TestDeepWikiClientWithTimeout(t *testing.T) {
	config := DefaultClientConfig()
	config.Timeout = 50 * time.Millisecond
	client := NewDeepWikiClient(config)
	
	ctx := context.Background()
	client.Initialize(ctx)
	
	// 这个测试取决于模拟的延迟
	args := DeepWikiArgs{URL: "test"}
	result, err := client.CallTool(ctx, "deepwiki_fetch", args)
	
	// 由于我们的模拟延迟是 100ms，而超时是 50ms，应该不会超时
	// 但在实际实现中，这里可能会测试超时逻辑
	if err != nil {
		t.Logf("Call completed with potential timeout: %v", err)
	}
	
	if result != nil && result.IsError {
		t.Error("Expected result to not be an error")
	}
}