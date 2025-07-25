package mcp

import (
	"testing"
	"time"
)

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()
	
	if config.Transport != "stdio" {
		t.Errorf("Expected transport 'stdio', got %s", config.Transport)
	}
	
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", config.Timeout)
	}
	
	if config.MaxRetries != 3 {
		t.Errorf("Expected max retries 3, got %d", config.MaxRetries)
	}
	
	if config.RetryDelay != time.Second {
		t.Errorf("Expected retry delay 1s, got %v", config.RetryDelay)
	}
}

func TestDeepWikiArgs(t *testing.T) {
	args := DeepWikiArgs{
		URL:      "https://github.com/test/repo",
		MaxDepth: 1,
		Mode:     "aggregate",
		Verbose:  true,
	}
	
	if args.URL != "https://github.com/test/repo" {
		t.Errorf("Expected URL 'https://github.com/test/repo', got %s", args.URL)
	}
	
	if args.MaxDepth != 1 {
		t.Errorf("Expected MaxDepth 1, got %d", args.MaxDepth)
	}
	
	if args.Mode != "aggregate" {
		t.Errorf("Expected Mode 'aggregate', got %s", args.Mode)
	}
	
	if !args.Verbose {
		t.Error("Expected Verbose true, got false")
	}
}

func TestContext7Args(t *testing.T) {
	resolveArgs := Context7ResolveArgs{
		LibraryName: "golang",
	}
	
	if resolveArgs.LibraryName != "golang" {
		t.Errorf("Expected LibraryName 'golang', got %s", resolveArgs.LibraryName)
	}
	
	docsArgs := Context7DocsArgs{
		Context7CompatibleLibraryID: "/golang/go",
		Tokens:                      5000,
		Topic:                       "interfaces",
	}
	
	if docsArgs.Context7CompatibleLibraryID != "/golang/go" {
		t.Errorf("Expected LibraryID '/golang/go', got %s", docsArgs.Context7CompatibleLibraryID)
	}
	
	if docsArgs.Tokens != 5000 {
		t.Errorf("Expected Tokens 5000, got %d", docsArgs.Tokens)
	}
	
	if docsArgs.Topic != "interfaces" {
		t.Errorf("Expected Topic 'interfaces', got %s", docsArgs.Topic)
	}
}

func TestToolResult(t *testing.T) {
	result := &ToolResult{
		Content: []Content{
			{Type: "text", Text: "Hello, World!"},
		},
		IsError: false,
	}
	
	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content item, got %d", len(result.Content))
	}
	
	if result.Content[0].Type != "text" {
		t.Errorf("Expected content type 'text', got %s", result.Content[0].Type)
	}
	
	if result.Content[0].Text != "Hello, World!" {
		t.Errorf("Expected content text 'Hello, World!', got %s", result.Content[0].Text)
	}
	
	if result.IsError {
		t.Error("Expected IsError false, got true")
	}
}