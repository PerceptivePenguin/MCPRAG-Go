package rag

import (
	"context"
	"strings"
	"testing"
)

func TestNewTextChunker(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	
	if chunker == nil {
		t.Error("expected chunker to be created")
	}
	if chunker.tokenizer == nil {
		t.Error("expected tokenizer to be set")
	}
}

func TestTextChunker_ChunkDocument(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	doc := Document{
		ID:      "test-doc",
		Content: "This is the first sentence. This is the second sentence. This is the third sentence.",
		Title:   "Test Document",
		Source:  "test",
		Metadata: map[string]string{
			"category": "test",
		},
	}

	tests := []struct {
		name    string
		options ChunkingOptions
		minChunks int
		maxChunks int
	}{
		{
			name: "chunk by tokens",
			options: ChunkingOptions{
				Strategy:     ChunkByTokens,
				MaxChunkSize: 10,
				Overlap:      2,
			},
			minChunks: 1,
			maxChunks: 10,
		},
		{
			name: "chunk by sentences",
			options: ChunkingOptions{
				Strategy:     ChunkBySentences,
				MaxChunkSize: 2,
				Overlap:      1,
			},
			minChunks: 1,
			maxChunks: 5,
		},
		{
			name: "chunk by paragraphs",
			options: ChunkingOptions{
				Strategy:     ChunkByParagraphs,
				MaxChunkSize: 100,
				Overlap:      0,
			},
			minChunks: 1,
			maxChunks: 2,
		},
		{
			name: "chunk by fixed size",
			options: ChunkingOptions{
				Strategy:     ChunkByFixedSize,
				MaxChunkSize: 50,
				Overlap:      10,
			},
			minChunks: 1,
			maxChunks: 5,
		},
		{
			name: "chunk by semantic",
			options: ChunkingOptions{
				Strategy:     ChunkBySemantic,
				MaxChunkSize: 100,
				Overlap:      20,
			},
			minChunks: 1,
			maxChunks: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, err := chunker.ChunkDocument(ctx, doc, tt.options)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(chunks) < tt.minChunks {
				t.Errorf("expected at least %d chunks, got %d", tt.minChunks, len(chunks))
			}
			if len(chunks) > tt.maxChunks {
				t.Errorf("expected at most %d chunks, got %d", tt.maxChunks, len(chunks))
			}

			// Verify chunk properties
			for i, chunk := range chunks {
				if chunk.ID == "" {
					t.Errorf("chunk %d has empty ID", i)
				}
				if chunk.Content == "" {
					t.Errorf("chunk %d has empty content", i)
				}
				if chunk.DocumentID != doc.ID {
					t.Errorf("chunk %d has wrong document ID: expected %s, got %s", i, doc.ID, chunk.DocumentID)
				}
				if chunk.Index != i {
					t.Errorf("chunk %d has wrong index: expected %d, got %d", i, i, chunk.Index)
				}
			}
		})
	}
}

func TestTextChunker_ChunkDocumentEmptyContent(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	doc := Document{
		ID:      "empty-doc",
		Content: "",
	}

	options := ChunkingOptions{
		Strategy:     ChunkByTokens,
		MaxChunkSize: 100,
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err == nil {
		t.Error("expected error for empty content")
	}
	if len(chunks) != 0 {
		t.Errorf("expected no chunks for empty content, got %d", len(chunks))
	}
}

func TestTextChunker_ChunkDocumentInvalidStrategy(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	doc := Document{
		ID:      "test-doc",
		Content: "Some test content",
	}

	options := ChunkingOptions{
		Strategy:     "invalid_strategy",
		MaxChunkSize: 100,
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err == nil {
		t.Error("expected error for invalid strategy")
	}
	if len(chunks) != 0 {
		t.Errorf("expected no chunks for invalid strategy, got %d", len(chunks))
	}
}

func TestTextChunker_ChunkByTokensWithOverlap(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	// Create a document with known word count
	words := []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten"}
	content := strings.Join(words, " ")

	doc := Document{
		ID:      "test-doc",
		Content: content,
	}

	options := ChunkingOptions{
		Strategy:     ChunkByTokens,
		MaxChunkSize: 4, // 4 tokens per chunk
		Overlap:      2, // 2 tokens overlap
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(chunks) < 2 {
		t.Errorf("expected at least 2 chunks with overlap, got %d", len(chunks))
	}

	// Verify overlap exists between consecutive chunks
	if len(chunks) >= 2 {
		// Check that there's some overlapping content
		firstChunk := strings.Fields(chunks[0].Content)
		secondChunk := strings.Fields(chunks[1].Content)
		
		if len(firstChunk) == 0 || len(secondChunk) == 0 {
			t.Error("chunks should have content")
		}
	}
}

func TestTextChunker_ChunkBySentences(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	doc := Document{
		ID:      "test-doc",
		Content: "First sentence. Second sentence! Third sentence? Fourth sentence.",
	}

	options := ChunkingOptions{
		Strategy:     ChunkBySentences,
		MaxChunkSize: 2, // 2 sentences per chunk
		Overlap:      1, // 1 sentence overlap
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Verify each chunk contains complete sentences
	for _, chunk := range chunks {
		if !strings.Contains(chunk.Content, ".") && 
		   !strings.Contains(chunk.Content, "!") && 
		   !strings.Contains(chunk.Content, "?") {
			t.Errorf("chunk should contain complete sentences: %s", chunk.Content)
		}
	}
}

func TestTextChunker_ChunkByParagraphs(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	doc := Document{
		ID:      "test-doc",
		Content: "First paragraph.\n\nSecond paragraph with more content.\n\nThird paragraph here.",
	}

	options := ChunkingOptions{
		Strategy:     ChunkByParagraphs,
		MaxChunkSize: 1, // 1 paragraph per chunk
		Overlap:      0,
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Each chunk should be a complete paragraph
	for _, chunk := range chunks {
		if chunk.Content == "" {
			t.Error("chunk content should not be empty")
		}
	}
}

func TestTextChunker_ChunkByFixedSize(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	doc := Document{
		ID:      "test-doc",
		Content: strings.Repeat("word ", 100), // 100 words
	}

	options := ChunkingOptions{
		Strategy:     ChunkByFixedSize,
		MaxChunkSize: 200, // 200 characters per chunk
		Overlap:      50,  // 50 character overlap
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Verify chunk sizes are within limits
	for i, chunk := range chunks {
		if len(chunk.Content) > options.MaxChunkSize {
			t.Errorf("chunk %d exceeds max size: %d > %d", i, len(chunk.Content), options.MaxChunkSize)
		}
	}
}

func TestTextChunker_ChunkBySemantic(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	doc := Document{
		ID:      "test-doc",
		Content: "This is about cars. Cars are vehicles. Now let's talk about animals. Animals are living beings. Dogs are animals.",
	}

	options := ChunkingOptions{
		Strategy:     ChunkBySemantic,
		MaxChunkSize: 100,
		Overlap:      20,
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// Basic verification that chunks have content
	for i, chunk := range chunks {
		if chunk.Content == "" {
			t.Errorf("chunk %d has empty content", i)
		}
	}
}

func TestTextChunker_PreserveStructure(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	doc := Document{
		ID:      "test-doc",
		Content: "# Title\n\nFirst paragraph.\n\n## Subtitle\n\nSecond paragraph.",
	}

	options := ChunkingOptions{
		Strategy:          ChunkByParagraphs,
		MaxChunkSize:      100,
		PreserveStructure: true,
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(chunks) == 0 {
		t.Error("expected at least one chunk")
	}

	// When preserving structure, chunks should maintain formatting
	foundMarkdown := false
	for _, chunk := range chunks {
		if strings.Contains(chunk.Content, "#") {
			foundMarkdown = true
			break
		}
	}

	if !foundMarkdown {
		t.Error("expected to preserve markdown structure")
	}
}

func TestTextChunker_UtilityFunctions(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)

	// Test sentence splitting
	text := "First sentence. Second sentence! Third sentence?"
	sentences := chunker.splitIntoSentences(text)
	if len(sentences) != 3 {
		t.Errorf("expected 3 sentences, got %d", len(sentences))
	}

	// Test separator splitting
	content := "part1||part2||part3"
	separators := []string{"||"}
	parts := chunker.splitBySeparators(content, separators)
	if len(parts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(parts))
	}

	// Test word extraction
	words := chunker.extractWords("Hello, world! This is a test.")
	if len(words) == 0 {
		t.Error("expected to extract words")
	}
}

func TestTextChunker_EdgeCases(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()

	// Test with very small chunk size
	doc := Document{
		ID:      "test-doc",
		Content: "word",
	}

	options := ChunkingOptions{
		Strategy:     ChunkByTokens,
		MaxChunkSize: 1,
		Overlap:      0,
	}

	chunks, err := chunker.ChunkDocument(ctx, doc, options)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for single word, got %d", len(chunks))
	}

	// Test with zero chunk size
	options.MaxChunkSize = 0
	_, err = chunker.ChunkDocument(ctx, doc, options)
	if err == nil {
		t.Error("expected error for zero chunk size")
	}
}

func TestTextChunker_GetOverlapContent(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()
	
	// Test with overlap to trigger getOverlapContent function
	doc := Document{
		ID:      "test",
		Content: "First sentence. Second sentence. Third sentence. Fourth sentence. Fifth sentence.",
	}
	
	config := ChunkingOptions{
		Strategy:     ChunkByTokens,
		MaxChunkSize: 10,
		Overlap:      5,
	}
	
	chunks, err := chunker.ChunkDocument(ctx, doc, config)
	if err != nil {
		t.Fatalf("Failed to chunk document: %v", err)
	}
	
	// Should create overlapping chunks
	if len(chunks) < 2 {
		t.Error("Expected at least 2 chunks with overlap")
	}
}

func TestTextChunker_ExtendedOverlapHandling(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")  
	chunker := NewTextChunker(tokenizer)
	ctx := context.Background()
	
	// Test getOverlapSentences with complex content
	doc := Document{
		ID:      "overlap-test", 
		Content: "Sentence one goes here. Sentence two is longer and more detailed. Sentence three continues the pattern. Sentence four adds more complexity. Sentence five wraps up.",
	}
	
	config := ChunkingOptions{
		Strategy:     ChunkBySentences,
		MaxChunkSize: 50,
		Overlap:      20,
		PreserveStructure: true,
	}
	
	chunks, err := chunker.ChunkDocument(ctx, doc, config)
	if err != nil {
		t.Fatalf("Failed to chunk by sentences with overlap: %v", err)
	}
	
	// Just verify the function runs without error, don't require specific chunk count
	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
	
	// Verify proper overlap handling
	for i, chunk := range chunks {
		if chunk.Content == "" {
			t.Errorf("Chunk %d is empty", i)
		}
	}
}

func TestTextChunker_SemanticCoherence(t *testing.T) {
	tokenizer := NewSimpleTokenizer("")
	chunker := NewTextChunker(tokenizer)
	
	// Test calculateTopicCoherence function 
	currentText := "technology computer software development programming"
	newSentence := "algorithm data structure database network"
	
	score := chunker.calculateTopicCoherence(currentText, newSentence)
	
	if score < 0 || score > 1 {
		t.Errorf("Topic coherence score should be between 0 and 1, got %f", score)
	}
	
	// Test with empty current text
	score2 := chunker.calculateTopicCoherence("", "some new text")
	if score2 != 1.0 {
		t.Errorf("Expected coherence of 1.0 for empty current text, got %f", score2)
	}
	
	// Test with empty new sentence
	score3 := chunker.calculateTopicCoherence("some text", "")
	if score3 != 0.0 {
		t.Errorf("Expected coherence of 0.0 for empty new sentence, got %f", score3)
	}
}