package mcp

import (
	"errors"
	"testing"
)

func TestMCPError(t *testing.T) {
	originalErr := errors.New("connection refused")
	mcpErr := NewMCPError("connect", "deepwiki", "fetch", originalErr, true)
	
	expectedMsg := "mcp connect deepwiki.fetch: connection refused"
	if mcpErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, mcpErr.Error())
	}
	
	if !mcpErr.IsRetryable() {
		t.Error("Expected error to be retryable")
	}
	
	if !errors.Is(mcpErr, originalErr) {
		t.Error("Expected error to wrap original error")
	}
}

func TestMCPErrorWithoutTool(t *testing.T) {
	originalErr := errors.New("initialization failed")
	mcpErr := NewMCPError("initialize", "context7", "", originalErr, false)
	
	expectedMsg := "mcp initialize context7: initialization failed"
	if mcpErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, mcpErr.Error())
	}
	
	if mcpErr.IsRetryable() {
		t.Error("Expected error to not be retryable")
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("network error")
	mcpErr := WrapError("callTool", "deepwiki", originalErr)
	
	if mcpErr.Op != "callTool" {
		t.Errorf("Expected Op 'callTool', got '%s'", mcpErr.Op)
	}
	
	if mcpErr.Server != "deepwiki" {
		t.Errorf("Expected Server 'deepwiki', got '%s'", mcpErr.Server)
	}
	
	if mcpErr.Tool != "" {
		t.Errorf("Expected empty Tool, got '%s'", mcpErr.Tool)
	}
	
	if mcpErr.IsRetryable() {
		t.Error("Expected error to not be retryable by default")
	}
}

func TestWrapToolError(t *testing.T) {
	originalErr := errors.New("tool not found")
	mcpErr := WrapToolError("callTool", "context7", "resolve-library-id", originalErr)
	
	if mcpErr.Tool != "resolve-library-id" {
		t.Errorf("Expected Tool 'resolve-library-id', got '%s'", mcpErr.Tool)
	}
	
	expectedMsg := "mcp callTool context7.resolve-library-id: tool not found"
	if mcpErr.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, mcpErr.Error())
	}
}

func TestWrapRetryableError(t *testing.T) {
	originalErr := errors.New("temporary failure")
	mcpErr := WrapRetryableError("callTool", "deepwiki", originalErr)
	
	if !mcpErr.IsRetryable() {
		t.Error("Expected error to be retryable")
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrClientNotInitialized", ErrClientNotInitialized, "mcp client not initialized"},
		{"ErrClientNotConnected", ErrClientNotConnected, "mcp client not connected"},
		{"ErrToolNotFound", ErrToolNotFound, "tool not found"},
		{"ErrInvalidArgs", ErrInvalidArgs, "invalid arguments"},
		{"ErrCallTimeout", ErrCallTimeout, "tool call timeout"},
		{"ErrConnectionFailed", ErrConnectionFailed, "connection failed"},
		{"ErrMaxRetriesExceeded", ErrMaxRetriesExceeded, "max retries exceeded"},
		{"ErrInvalidConfig", ErrInvalidConfig, "invalid configuration"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("Expected error message '%s', got '%s'", tt.msg, tt.err.Error())
			}
		})
	}
}