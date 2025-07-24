package rag

import (
	"context"
	"strings"
	"testing"
)

func TestNewOpenAIEmbedder(t *testing.T) {
	tests := []struct {
		name        string
		config      *EmbeddingConfig
		cache       Cache
		expectError bool
	}{
		{
			name:        "nil config uses default",
			config:      nil,
			cache:       nil,
			expectError: true, // Will fail due to missing API key
		},
		{
			name: "missing API key",
			config: &EmbeddingConfig{
				Model: "test-model",
			},
			cache:       nil,
			expectError: true,
		},
		{
			name: "valid config with API key",
			config: &EmbeddingConfig{
				Model:  "test-model",
				APIKey: "test-key",
			},
			cache:       nil,
			expectError: false,
		},
		{
			name: "config with custom base URL",
			config: &EmbeddingConfig{
				Model:   "test-model",
				APIKey:  "test-key",
				BaseURL: "https://custom-api.example.com",
			},
			cache:       nil,
			expectError: false,
		},
		{
			name: "config with headers",
			config: &EmbeddingConfig{
				Model:  "test-model",
				APIKey: "test-key",
				Headers: map[string]string{
					"Custom-Header": "value",
				},
			},
			cache:       nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedder, err := NewOpenAIEmbedder(tt.config, tt.cache)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if embedder == nil {
					t.Error("expected embedder to be created")
				}
				if embedder != nil {
					embedder.Close()
				}
			}
		})
	}
}

func TestOpenAIEmbedder_GetModel(t *testing.T) {
	config := &EmbeddingConfig{
		Model:  "test-model-name",
		APIKey: "test-key",
	}

	embedder, err := NewOpenAIEmbedder(config, nil)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}
	defer embedder.Close()

	model := embedder.GetModel()
	if model != "test-model-name" {
		t.Errorf("expected model 'test-model-name', got '%s'", model)
	}
}

func TestOpenAIEmbedder_GetDimension(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		{"text-embedding-3-small", 1536},
		{"text-embedding-3-large", 3072},
		{"text-embedding-ada-002", 1536},
		{"unknown-model", 1536}, // default
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			config := &EmbeddingConfig{
				Model:  tt.model,
				APIKey: "test-key",
			}

			embedder, err := NewOpenAIEmbedder(config, nil)
			if err != nil {
				t.Fatalf("failed to create embedder: %v", err)
			}
			defer embedder.Close()

			dimension := embedder.GetDimension()
			if dimension != tt.expected {
				t.Errorf("expected dimension %d, got %d", tt.expected, dimension)
			}
		})
	}
}

func TestOpenAIEmbedder_EmbedEmptyText(t *testing.T) {
	config := &EmbeddingConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}

	embedder, err := NewOpenAIEmbedder(config, nil)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}
	defer embedder.Close()

	ctx := context.Background()

	// Test empty text
	_, err = embedder.Embed(ctx, "")
	if err == nil {
		t.Error("expected error for empty text")
	}

	// Test whitespace-only text
	_, err = embedder.Embed(ctx, "   \n\t  ")
	if err == nil {
		t.Error("expected error for whitespace-only text")
	}
}

func TestOpenAIEmbedder_EmbedBatchEmpty(t *testing.T) {
	config := &EmbeddingConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}

	embedder, err := NewOpenAIEmbedder(config, nil)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}
	defer embedder.Close()

	ctx := context.Background()

	// Test empty slice
	_, err = embedder.EmbedBatch(ctx, []string{})
	if err == nil {
		t.Error("expected error for empty text slice")
	}

	// Test slice with only empty strings
	_, err = embedder.EmbedBatch(ctx, []string{"", "  ", "\n"})
	if err == nil {
		t.Error("expected error for slice with only empty strings")
	}
}

func TestOpenAIEmbedder_EmbedWithOptionsEmptyText(t *testing.T) {
	config := &EmbeddingConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}

	embedder, err := NewOpenAIEmbedder(config, nil)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}
	defer embedder.Close()

	ctx := context.Background()

	req := EmbeddingRequest{
		Text: "",
	}

	_, err = embedder.EmbedWithOptions(ctx, req)
	if err == nil {
		t.Error("expected error for empty text in request")
	}
}

func TestOpenAIEmbedder_Close(t *testing.T) {
	config := &EmbeddingConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}

	embedder, err := NewOpenAIEmbedder(config, nil)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	// Close should succeed
	err = embedder.Close()
	if err != nil {
		t.Errorf("unexpected error during close: %v", err)
	}

	// Second close should also succeed
	err = embedder.Close()
	if err != nil {
		t.Errorf("unexpected error during second close: %v", err)
	}
}

func TestOpenAIEmbedder_CloseWithCache(t *testing.T) {
	cache, err := NewLRUCache(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	config := &EmbeddingConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}

	embedder, err := NewOpenAIEmbedder(config, cache)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}

	// Close should also close the cache
	err = embedder.Close()
	if err != nil {
		t.Errorf("unexpected error during close: %v", err)
	}
}

func TestOpenAIEmbedder_IsRetryableError(t *testing.T) {
	config := &EmbeddingConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}

	embedder, err := NewOpenAIEmbedder(config, nil)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}
	defer embedder.Close()

	tests := []struct {
		name         string
		errorMsg     string
		shouldRetry  bool
	}{
		{
			name:        "rate limit error",
			errorMsg:    "rate limit exceeded",
			shouldRetry: true,
		},
		{
			name:        "timeout error",
			errorMsg:    "request timeout",
			shouldRetry: true,
		},
		{
			name:        "service unavailable",
			errorMsg:    "service unavailable",
			shouldRetry: true,
		},
		{
			name:        "internal server error",
			errorMsg:    "internal server error",
			shouldRetry: true,
		},
		{
			name:        "bad gateway",
			errorMsg:    "bad gateway",
			shouldRetry: true,
		},
		{
			name:        "gateway timeout",
			errorMsg:    "gateway timeout",
			shouldRetry: true,
		},
		{
			name:        "authentication error",
			errorMsg:    "unauthorized",
			shouldRetry: false,
		},
		{
			name:        "bad request",
			errorMsg:    "bad request",
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake error with the message
			fakeError := NewRAGError(tt.errorMsg, ErrorTypeExternal)
			
			isRetryable := embedder.isRetryableError(fakeError)
			if isRetryable != tt.shouldRetry {
				t.Errorf("expected isRetryable=%v for error '%s', got %v", tt.shouldRetry, tt.errorMsg, isRetryable)
			}
		})
	}
}

func TestOpenAIEmbedder_GenerateCacheKey(t *testing.T) {
	config := &EmbeddingConfig{
		Model:  "test-model",
		APIKey: "test-key",
	}

	embedder, err := NewOpenAIEmbedder(config, nil)
	if err != nil {
		t.Fatalf("failed to create embedder: %v", err)
	}
	defer embedder.Close()

	// Test that same inputs generate same key
	key1 := embedder.generateCacheKey("test text", "model1")
	key2 := embedder.generateCacheKey("test text", "model1")
	if key1 != key2 {
		t.Error("same inputs should generate same cache key")
	}

	// Test that different inputs generate different keys
	key3 := embedder.generateCacheKey("different text", "model1")
	if key1 == key3 {
		t.Error("different texts should generate different cache keys")
	}

	key4 := embedder.generateCacheKey("test text", "model2")
	if key1 == key4 {
		t.Error("different models should generate different cache keys")
	}

	// Test key format
	if !strings.HasPrefix(key1, "embed:") {
		t.Errorf("cache key should start with 'embed:', got: %s", key1)
	}
}

func TestDefaultEmbeddingConfig(t *testing.T) {
	config := DefaultEmbeddingConfig()
	
	if config == nil {
		t.Error("expected default config to be created")
		return
	}

	if config.Model != "text-embedding-3-small" {
		t.Errorf("expected default model to be 'text-embedding-3-small', got '%s'", config.Model)
	}

	if config.MaxRetries != 3 {
		t.Errorf("expected default max retries to be 3, got %d", config.MaxRetries)
	}

	if config.BatchSize != 100 {
		t.Errorf("expected default batch size to be 100, got %d", config.BatchSize)
	}

	if config.RateLimit != 60 {
		t.Errorf("expected default rate limit to be 60, got %d", config.RateLimit)
	}
}

// Test that validates our mock structures work correctly
func TestOpenAIEmbedder_MockValidation(t *testing.T) {
	// This test ensures our embedding config structure is valid
	config := &EmbeddingConfig{
		Model:      "text-embedding-3-small",
		APIKey:     "test-key",
		MaxRetries: 3,
		BatchSize:  50,
		RateLimit:  30,
	}

	embedder, err := NewOpenAIEmbedder(config, nil)
	if err != nil {
		t.Fatalf("failed to create embedder with valid config: %v", err)
	}

	if embedder.GetModel() != config.Model {
		t.Error("embedder should return configured model")
	}

	if embedder.GetDimension() != 1536 {
		t.Error("embedder should return correct dimension for text-embedding-3-small")
	}

	embedder.Close()
}