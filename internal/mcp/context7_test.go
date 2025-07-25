package mcp

import (
	"context"
	"errors"
	"testing"
)

func TestNewContext7Client(t *testing.T) {
	config := DefaultClientConfig()
	client := NewContext7Client(config)
	
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	
	if client.config.ServerName != "context7" {
		t.Errorf("Expected server name 'context7', got '%s'", client.config.ServerName)
	}
	
	if client.connected {
		t.Error("Expected client to be initially disconnected")
	}
}

func TestContext7ClientListTools(t *testing.T) {
	config := DefaultClientConfig()
	client := NewContext7Client(config)
	ctx := context.Background()
	
	// Initialize client
	client.Initialize(ctx)
	tools, err := client.ListTools(ctx)
	
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	expectedTools := []string{"resolve-library-id", "get-library-docs"}
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}
	
	for i, expectedName := range expectedTools {
		if tools[i].Name != expectedName {
			t.Errorf("Expected tool name '%s', got '%s'", expectedName, tools[i].Name)
		}
	}
}

func TestContext7ClientCallResolveLibraryID(t *testing.T) {
	config := DefaultClientConfig()
	client := NewContext7Client(config)
	ctx := context.Background()
	client.Initialize(ctx)
	
	tests := []struct {
		name        string
		args        interface{}
		expectError bool
	}{
		{
			name: "Valid struct args",
			args: Context7ResolveArgs{
				LibraryName: "golang",
			},
			expectError: false,
		},
		{
			name: "Valid map args",
			args: map[string]interface{}{
				"libraryName": "gin",
			},
			expectError: false,
		},
		{
			name: "Missing required parameter",
			args: map[string]interface{}{
				"wrongParam": "value",
			},
			expectError: true,
		},
		{
			name:        "Invalid args type",
			args:        "invalid",
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.CallTool(ctx, "resolve-library-id", tt.args)
			
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
		})
	}
}

func TestContext7ClientCallGetLibraryDocs(t *testing.T) {
	config := DefaultClientConfig()
	client := NewContext7Client(config)
	ctx := context.Background()
	client.Initialize(ctx)
	
	tests := []struct {
		name        string
		args        interface{}
		expectError bool
	}{
		{
			name: "Valid struct args",
			args: Context7DocsArgs{
				Context7CompatibleLibraryID: "/golang/go",
				Tokens:                      5000,
				Topic:                       "interfaces",
			},
			expectError: false,
		},
		{
			name: "Valid map args with defaults",
			args: map[string]interface{}{
				"context7CompatibleLibraryID": "/gin-gonic/gin",
			},
			expectError: false,
		},
		{
			name: "Valid map args with all parameters",
			args: map[string]interface{}{
				"context7CompatibleLibraryID": "/gorilla/mux",
				"tokens":                      8000,
				"topic":                       "routing",
			},
			expectError: false,
		},
		{
			name: "Missing required parameter",
			args: map[string]interface{}{
				"tokens": 1000,
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.CallTool(ctx, "get-library-docs", tt.args)
			
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
		})
	}
}

func TestContext7ClientCallInvalidTool(t *testing.T) {
	config := DefaultClientConfig()
	client := NewContext7Client(config)
	ctx := context.Background()
	client.Initialize(ctx)
	
	_, err := client.CallTool(ctx, "invalid-tool", map[string]interface{}{})
	
	if err == nil {
		t.Error("Expected error for invalid tool")
	}
	
	var mcpErr *MCPError
	if !errors.As(err, &mcpErr) {
		t.Error("Expected MCPError")
	}
	
	if !errors.Is(err, ErrToolNotFound) {
		t.Error("Expected ErrToolNotFound")
	}
}

func TestContext7ClientWithoutConnection(t *testing.T) {
	config := DefaultClientConfig()
	client := NewContext7Client(config)
	ctx := context.Background()
	
	// Test ListTools without connection
	_, err := client.ListTools(ctx)
	if err == nil {
		t.Error("Expected error when listing tools without connection")
	}
	
	// Test CallTool without connection
	args := Context7ResolveArgs{LibraryName: "test"}
	_, err = client.CallTool(ctx, "resolve-library-id", args)
	if err == nil {
		t.Error("Expected error when calling tool without connection")
	}
}

func TestContext7ClientLibraryIDSimulation(t *testing.T) {
	config := DefaultClientConfig()
	client := NewContext7Client(config)
	ctx := context.Background()
	client.Initialize(ctx)
	
	testCases := []struct {
		input    string
		expected string
	}{
		{"golang", "/golang/go"},
		{"go", "/golang/go"},
		{"gin", "/gin-gonic/gin"},
		{"gin-gonic", "/gin-gonic/gin"},
		{"gorilla/mux", "/gorilla/mux"},
		{"unknown", "/unknown/library"},
	}
	
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			args := Context7ResolveArgs{LibraryName: tc.input}
			result, err := client.CallTool(ctx, "resolve-library-id", args)
			
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}
			
			if len(result.Content) == 0 {
				t.Error("Expected content in result")
				return
			}
			
			content := result.Content[0].Text
			if !contains(content, tc.expected) {
				t.Errorf("Expected content to contain '%s', got '%s'", tc.expected, content)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}