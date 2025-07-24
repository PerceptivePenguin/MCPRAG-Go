package rag

import (
	"context"
	"strings"
	"testing"
)

func TestNewBasicContextBuilder(t *testing.T) {
	// Test with nil config (should use default)
	builder := NewBasicContextBuilder(nil, nil)
	if builder == nil {
		t.Error("expected builder to be created")
	}
	if builder.config == nil {
		t.Error("expected default config to be used")
	}

	// Test with custom config
	config := &ContextConfig{
		Template:    "Custom: {{.Content}}",
		MaxLength:   500,
		IncludeMetadata: true,
	}
	builder = NewBasicContextBuilder(config, nil)
	if builder.config.Template != config.Template {
		t.Error("expected custom config to be used")
	}
}

func TestBasicContextBuilder_FormatDocument(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil)
	
	doc := Document{
		ID:      "test-1",
		Content: "This is test content",
		Title:   "Test Document",
		Source:  "test-source",
		Metadata: map[string]string{
			"category": "test",
			"author":   "tester",
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "default template",
			template: "",
			expected: "Context: This is test content",
		},
		{
			name:     "title and content template",
			template: "Title: {{.Title}}\nContent: {{.Content}}",
			expected: "Title: Test Document\nContent: This is test content",
		},
		{
			name:     "with metadata",
			template: "{{.Content}} (Source: {{.Source}})",
			expected: "This is test content (Source: test-source)",
		},
		{
			name:     "ID only",
			template: "Document ID: {{.ID}}",
			expected: "Document ID: test-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.FormatDocument(doc, tt.template)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBasicContextBuilder_FormatDocumentInvalidTemplate(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil)
	
	doc := Document{
		ID:      "test-1",
		Content: "test content",
	}

	// Invalid template syntax
	_, err := builder.FormatDocument(doc, "{{.Content")
	if err == nil {
		t.Error("expected error for invalid template")
	}

	// Reference to non-existent field should not error (Go templates handle this)
	_, err = builder.FormatDocument(doc, "{{.NonExistentField}}")
	if err != nil {
		t.Errorf("unexpected error for non-existent field: %v", err)
	}
}

func TestBasicContextBuilder_BuildContext(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil)
	ctx := context.Background()

	docs := []Document{
		{
			ID:      "doc1",
			Content: "First document content",
			Title:   "Document 1",
			Metadata: map[string]string{"type": "article"},
		},
		{
			ID:      "doc2",
			Content: "Second document content", 
			Title:   "Document 2",
			Metadata: map[string]string{"type": "blog"},
		},
	}

	result := &RetrievalResult{
		Documents: docs,
		Scores:    []float32{0.9, 0.7},
	}

	config := ContextConfig{
		Template:        "{{.Title}}: {{.Content}}",
		IncludeMetadata: false,
		IncludeScores:   false,
		SeparateChunks:  true,
	}

	contextStr, err := builder.BuildContext(ctx, result, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	expectedParts := []string{
		"Document 1: First document content",
		"Document 2: Second document content",
	}
	
	for _, part := range expectedParts {
		if !strings.Contains(contextStr, part) {
			t.Errorf("expected context to contain '%s', got: %s", part, contextStr)
		}
	}

	// Should contain separator
	if !strings.Contains(contextStr, "---") {
		t.Error("expected context to contain chunk separator")
	}
}

func TestBasicContextBuilder_BuildContextWithMetadata(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil)
	ctx := context.Background()

	doc := Document{
		ID:      "doc1",
		Content: "Test content",
		Title:   "Test Doc",
		Metadata: map[string]string{
			"author":   "John Doe",
			"category": "test",
		},
	}

	result := &RetrievalResult{
		Documents: []Document{doc},
		Scores:    []float32{0.9},
	}

	config := ContextConfig{
		Template:        "{{.Content}}",
		IncludeMetadata: true,
		IncludeScores:   false,
	}

	contextStr, err := builder.BuildContext(ctx, result, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !strings.Contains(contextStr, "Test content") {
		t.Error("expected context to contain document content")
	}
	if !strings.Contains(contextStr, "Metadata:") {
		t.Error("expected context to contain metadata section")
	}
	if !strings.Contains(contextStr, "author: John Doe") {
		t.Error("expected context to contain author metadata")
	}
}

func TestBasicContextBuilder_BuildContextWithScores(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil)
	ctx := context.Background()

	doc := Document{
		ID:      "doc1",
		Content: "Test content",
	}

	result := &RetrievalResult{
		Documents: []Document{doc},
		Scores:    []float32{0.85},
	}

	config := ContextConfig{
		Template:      "{{.Content}}",
		IncludeScores: true,
	}

	contextStr, err := builder.BuildContext(ctx, result, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !strings.Contains(contextStr, "Test content") {
		t.Error("expected context to contain document content")
	}
	if !strings.Contains(contextStr, "Relevance Score: 0.850") {
		t.Error("expected context to contain relevance score")
	}
}

func TestBasicContextBuilder_BuildContextFromDocuments(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil)
	ctx := context.Background()

	docs := []Document{
		{
			ID:      "doc1",
			Content: "First document",
		},
		{
			ID:      "doc2",
			Content: "Second document",
		},
	}

	config := ContextConfig{
		Template: "{{.Content}}",
	}

	contextStr, err := builder.BuildContextFromDocuments(ctx, docs, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if !strings.Contains(contextStr, "First document") {
		t.Error("expected context to contain first document")
	}
	if !strings.Contains(contextStr, "Second document") {
		t.Error("expected context to contain second document")
	}
}

func TestBasicContextBuilder_BuildContextEmpty(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil)
	ctx := context.Background()

	// Test with nil result
	contextStr, err := builder.BuildContext(ctx, nil, ContextConfig{})
	if err == nil {
		t.Error("expected error for nil result")
	}

	// Test with empty documents
	result := &RetrievalResult{
		Documents: []Document{},
		Scores:    []float32{},
	}
	contextStr, err = builder.BuildContext(ctx, result, ContextConfig{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if contextStr != "" {
		t.Errorf("expected empty context, got: %s", contextStr)
	}
}

func TestSimpleTokenizer(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	
	if tokenizer.GetModel() != "gpt-3.5-turbo" {
		t.Error("expected default model to be gpt-3.5-turbo")
	}

	text := "Hello world! This is a test."
	
	// Test token counting
	tokenCount := tokenizer.CountTokens(text)
	if tokenCount <= 0 {
		t.Error("expected positive token count")
	}

	// Test tokenization
	tokens := tokenizer.Tokenize(text)
	if len(tokens) == 0 {
		t.Error("expected non-empty token list")
	}

	// Test truncation
	truncated := tokenizer.TruncateToTokens(text, 3)
	truncatedTokens := tokenizer.Tokenize(truncated)
	if len(truncatedTokens) > 3 {
		t.Errorf("expected at most 3 tokens after truncation, got %d", len(truncatedTokens))
	}
}

func TestBasicContextBuilder_TruncateContext(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	builder := NewBasicContextBuilder(nil, tokenizer)

	longText := strings.Repeat("This is a long sentence with many words. ", 20)

	tests := []struct {
		name      string
		strategy  string
		maxTokens int
	}{
		{
			name:      "head truncation",
			strategy:  "head",
			maxTokens: 10,
		},
		{
			name:      "tail truncation", 
			strategy:  "tail",
			maxTokens: 10,
		},
		{
			name:      "middle truncation",
			strategy:  "middle",
			maxTokens: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := builder.TruncateContext(longText, tt.maxTokens, tt.strategy)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultTokenCount := tokenizer.CountTokens(result)
			// Allow some tolerance since our tokenizer is approximate
			if resultTokenCount > tt.maxTokens*2 {
				t.Errorf("truncated text still too long: %d tokens (max: %d)", resultTokenCount, tt.maxTokens)
			}

			if result == "" {
				t.Error("truncated text should not be empty")
			}
		})
	}
}

func TestBasicContextBuilder_TruncateContextNoTokenizer(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil) // No tokenizer

	_, err := builder.TruncateContext("test text", 10, "head")
	if err == nil {
		t.Error("expected error when truncating without tokenizer")
	}
}

func TestTemplateContextBuilder(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	builder := NewTemplateContextBuilder(nil, tokenizer)

	// Register a custom template
	err := builder.RegisterTemplate("custom", "Title: {{.Title}} | Content: {{.Content}}")
	if err != nil {
		t.Errorf("failed to register template: %v", err)
	}

	doc := Document{
		ID:      "test-1",
		Content: "Test content",
		Title:   "Test Title",
	}

	// Use the custom template
	result, err := builder.FormatDocumentWithTemplate(doc, "custom")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	expected := "Title: Test Title | Content: Test content"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}

	// Try to use non-existent template
	_, err = builder.FormatDocumentWithTemplate(doc, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent template")
	}
}

func TestTemplateContextBuilder_RegisterInvalidTemplate(t *testing.T) {
	builder := NewTemplateContextBuilder(nil, nil)

	err := builder.RegisterTemplate("invalid", "{{.Content")
	if err == nil {
		t.Error("expected error for invalid template syntax")
	}
}

func TestContextBuilderEdgeCases(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil)
	ctx := context.Background()

	// Test with mismatched documents and scores
	result := &RetrievalResult{
		Documents: []Document{
			{ID: "doc1", Content: "content1"},
			{ID: "doc2", Content: "content2"},
		},
		Scores: []float32{0.9}, // Only one score for two documents
	}

	config := ContextConfig{
		Template: "{{.Content}}",
		IncludeScores: true,
	}

	contextStr, err := builder.BuildContext(ctx, result, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	// Should handle mismatched scores gracefully
	if !strings.Contains(contextStr, "content1") {
		t.Error("expected context to contain first document")
	}
	if !strings.Contains(contextStr, "content2") {
		t.Error("expected context to contain second document")
	}
}

func TestContextConfig_CharacterBasedTruncation(t *testing.T) {
	builder := NewBasicContextBuilder(nil, nil) // No tokenizer
	ctx := context.Background()

	longContent := strings.Repeat("A", 1000)
	doc := Document{
		ID:      "doc1",
		Content: longContent,
	}

	result := &RetrievalResult{
		Documents: []Document{doc},
		Scores:    []float32{1.0},
	}

	config := ContextConfig{
		Template:         "{{.Content}}",
		MaxLength:        100,
		TruncateStrategy: "head",
	}

	contextStr, err := builder.BuildContext(ctx, result, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(contextStr) > 100 {
		t.Errorf("expected context length <= 100, got %d", len(contextStr))
	}

	// Test tail truncation
	config.TruncateStrategy = "tail"
	contextStr, err = builder.BuildContext(ctx, result, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(contextStr) > 100 {
		t.Errorf("expected context length <= 100, got %d", len(contextStr))
	}

	// Test middle truncation
	config.TruncateStrategy = "middle"
	contextStr, err = builder.BuildContext(ctx, result, config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(contextStr) > 200 { // Middle truncation adds truncation text
		t.Errorf("expected reasonable context length, got %d", len(contextStr))
	}
}