package rag

import (
	"context"
	"testing"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/vector"
)

// Mock implementations for testing

type mockVectorStore struct {
	documents map[string]vector.Document
	closed    bool
}

func newMockVectorStore() *mockVectorStore {
	return &mockVectorStore{
		documents: make(map[string]vector.Document),
	}
}

func (m *mockVectorStore) Add(doc vector.Document) error {
	if m.closed {
		return vector.NewVectorError("store closed")
	}
	m.documents[doc.ID] = doc
	return nil
}

func (m *mockVectorStore) AddBatch(docs []vector.Document) error {
	if m.closed {
		return vector.NewVectorError("store closed")
	}
	for _, doc := range docs {
		m.documents[doc.ID] = doc
	}
	return nil
}

func (m *mockVectorStore) Get(id string) (*vector.Document, error) {
	if m.closed {
		return nil, vector.NewVectorError("store closed")
	}
	doc, exists := m.documents[id]
	if !exists {
		return nil, vector.ErrDocumentNotFound
	}
	return &doc, nil
}

func (m *mockVectorStore) Delete(id string) error {
	if m.closed {
		return vector.NewVectorError("store closed")
	}
	delete(m.documents, id)
	return nil
}

func (m *mockVectorStore) Search(queryVector vector.Vector, topK int) (*vector.SearchResult, error) {
	return m.SearchWithThreshold(queryVector, topK, 0.0)
}

func (m *mockVectorStore) SearchWithThreshold(queryVector vector.Vector, topK int, threshold float32) (*vector.SearchResult, error) {
	if m.closed {
		return nil, vector.NewVectorError("store closed")
	}

	var results []vector.Document

	// Simple mock search - return documents with similarity > threshold
	for _, doc := range m.documents {
		// Mock similarity calculation
		similarity := float32(0.8) // Fixed similarity for testing
		if similarity >= threshold {
			// Create a copy of the document with score
			resultDoc := doc
			resultDoc.Score = similarity
			results = append(results, resultDoc)
		}
	}

	// Limit to TopK
	if len(results) > topK {
		results = results[:topK]
	}

	return &vector.SearchResult{
		Documents: results,
		QueryTime: 10, // Mock query time
	}, nil
}

func (m *mockVectorStore) Size() int {
	return len(m.documents)
}

func (m *mockVectorStore) Clear() error {
	if m.closed {
		return vector.NewVectorError("store closed")
	}
	m.documents = make(map[string]vector.Document)
	return nil
}

func (m *mockVectorStore) Close() error {
	m.closed = true
	return nil
}

type mockEmbedder struct {
	dimension int
	closed    bool
}

func newMockEmbedder() *mockEmbedder {
	return &mockEmbedder{
		dimension: 4, // Small dimension for testing
	}
}

func (m *mockEmbedder) Embed(ctx context.Context, text string) (*EmbeddingResponse, error) {
	if m.closed {
		return nil, NewRAGError("embedder closed", ErrorTypeInternal)
	}

	// Return a mock embedding based on text length
	vector := make(vector.Vector, m.dimension)
	for i := range vector {
		vector[i] = float32(len(text) % 10) // Simple mock calculation
	}

	return &EmbeddingResponse{
		Vector: vector,
		Model:  "mock-model",
		Usage:  EmbeddingUsage{PromptTokens: len(text) / 4, TotalTokens: len(text) / 4},
		Cached: false,
	}, nil
}

func (m *mockEmbedder) EmbedBatch(ctx context.Context, texts []string) ([]*EmbeddingResponse, error) {
	if m.closed {
		return nil, NewRAGError("embedder closed", ErrorTypeInternal)
	}

	results := make([]*EmbeddingResponse, len(texts))
	for i, text := range texts {
		resp, err := m.Embed(ctx, text)
		if err != nil {
			return nil, err
		}
		results[i] = resp
	}
	return results, nil
}

func (m *mockEmbedder) EmbedWithOptions(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	return m.Embed(ctx, req.Text)
}

func (m *mockEmbedder) GetModel() string {
	return "mock-model"
}

func (m *mockEmbedder) GetDimension() int {
	return m.dimension
}

func (m *mockEmbedder) Close() error {
	m.closed = true
	return nil
}

func TestNewBasicRetriever(t *testing.T) {
	tests := []struct {
		name         string
		vectorStore  vector.Store
		embedder     Embedder
		processor    DocumentProcessor
		config       *RetrievalConfig
		expectError  bool
	}{
		{
			name:        "valid retriever",
			vectorStore: newMockVectorStore(),
			embedder:    newMockEmbedder(),
			processor:   nil,
			config:      DefaultRetrievalConfig(),
			expectError: false,
		},
		{
			name:        "nil vector store",
			vectorStore: nil,
			embedder:    newMockEmbedder(),
			processor:   nil,
			config:      DefaultRetrievalConfig(),
			expectError: true,
		},
		{
			name:        "nil embedder",
			vectorStore: newMockVectorStore(),
			embedder:    nil,
			processor:   nil,
			config:      DefaultRetrievalConfig(),
			expectError: true,
		},
		{
			name:        "nil config uses default",
			vectorStore: newMockVectorStore(),
			embedder:    newMockEmbedder(),
			processor:   nil,
			config:      nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retriever, err := NewBasicRetriever(tt.vectorStore, tt.embedder, tt.processor, tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if retriever == nil {
					t.Error("expected retriever to be created")
				}
				if retriever != nil {
					retriever.Close()
				}
			}
		})
	}
}

func TestBasicRetriever_AddDocument(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, DefaultRetrievalConfig())
	if err != nil {
		t.Fatalf("failed to create retriever: %v", err)
	}
	defer retriever.Close()

	ctx := context.Background()
	doc := Document{
		ID:      "test-doc-1",
		Content: "This is a test document for retrieval",
		Title:   "Test Document",
		Source:  "test",
		Metadata: map[string]string{
			"category": "test",
		},
	}

	err = retriever.AddDocument(ctx, doc)
	if err != nil {
		t.Errorf("failed to add document: %v", err)
	}

	// Verify document was stored in vector store
	if vectorStore.Size() != 1 {
		t.Errorf("expected 1 document in vector store, got %d", vectorStore.Size())
	}

	// Check stats
	stats := retriever.GetStats()
	if stats.TotalDocuments != 1 {
		t.Errorf("expected 1 document in stats, got %d", stats.TotalDocuments)
	}
	if stats.TotalChunks != 1 {
		t.Errorf("expected 1 chunk in stats, got %d", stats.TotalChunks)
	}
}

func TestBasicRetriever_AddDocuments(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, DefaultRetrievalConfig())
	if err != nil {
		t.Fatalf("failed to create retriever: %v", err)
	}
	defer retriever.Close()

	ctx := context.Background()
	docs := []Document{
		{
			ID:      "doc1",
			Content: "First document content",
			Title:   "Document 1",
		},
		{
			ID:      "doc2",
			Content: "Second document content",
			Title:   "Document 2",
		},
		{
			ID:      "doc3",
			Content: "Third document content",
			Title:   "Document 3",
		},
	}

	err = retriever.AddDocuments(ctx, docs)
	if err != nil {
		t.Errorf("failed to add documents: %v", err)
	}

	// Verify all documents were stored
	if vectorStore.Size() != 3 {
		t.Errorf("expected 3 documents in vector store, got %d", vectorStore.Size())
	}

	stats := retriever.GetStats()
	if stats.TotalDocuments != 3 {
		t.Errorf("expected 3 documents in stats, got %d", stats.TotalDocuments)
	}
}

func TestBasicRetriever_Retrieve(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, DefaultRetrievalConfig())
	if err != nil {
		t.Fatalf("failed to create retriever: %v", err)
	}
	defer retriever.Close()

	ctx := context.Background()

	// Add test documents
	docs := []Document{
		{
			ID:      "doc1",
			Content: "Machine learning is a subset of artificial intelligence",
			Title:   "ML Document",
		},
		{
			ID:      "doc2",
			Content: "Deep learning uses neural networks with multiple layers",
			Title:   "DL Document",
		},
	}

	for _, doc := range docs {
		if err := retriever.AddDocument(ctx, doc); err != nil {
			t.Fatalf("failed to add document: %v", err)
		}
	}

	// Test retrieval
	query := Query{
		Text:      "artificial intelligence",
		TopK:      5,
		Threshold: 0.5,
	}

	result, err := retriever.Retrieve(ctx, query)
	if err != nil {
		t.Errorf("failed to retrieve: %v", err)
		return
	}

	if result == nil {
		t.Error("expected retrieval result but got nil")
		return
	}

	if len(result.Documents) == 0 {
		t.Error("expected at least one document in results")
	}

	if len(result.Documents) != len(result.Scores) {
		t.Error("documents and scores length mismatch")
	}

	if result.QueryTime < 0 {
		t.Error("expected non-negative query time")
	}

	// Check that stats were updated
	stats := retriever.GetStats()
	if stats.TotalQueries != 1 {
		t.Errorf("expected 1 query in stats, got %d", stats.TotalQueries)
	}
}

func TestBasicRetriever_ValidateQuery(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, DefaultRetrievalConfig())
	if err != nil {
		t.Fatalf("failed to create retriever: %v", err)
	}
	defer retriever.Close()

	tests := []struct {
		name        string
		query       Query
		expectError bool
	}{
		{
			name: "valid query",
			query: Query{
				Text:      "test query",
				TopK:      5,
				Threshold: 0.5,
			},
			expectError: false,
		},
		{
			name: "empty text",
			query: Query{
				Text:      "",
				TopK:      5,
				Threshold: 0.5,
			},
			expectError: true,
		},
		{
			name: "zero TopK",
			query: Query{
				Text:      "test query",
				TopK:      0,
				Threshold: 0.5,
			},
			expectError: true,
		},
		{
			name: "negative TopK",
			query: Query{
				Text:      "test query",
				TopK:      -1,
				Threshold: 0.5,
			},
			expectError: true,
		},
		{
			name: "invalid threshold - negative",
			query: Query{
				Text:      "test query",
				TopK:      5,
				Threshold: -0.1,
			},
			expectError: true,
		},
		{
			name: "invalid threshold - greater than 1",
			query: Query{
				Text:      "test query",
				TopK:      5,
				Threshold: 1.1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := retriever.validateQuery(tt.query)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBasicRetriever_ValidateDocument(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	config := DefaultRetrievalConfig()
	config.Processing.MaxDocumentSize = 100 // Small size for testing
	
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, config)
	if err != nil {
		t.Fatalf("failed to create retriever: %v", err)
	}
	defer retriever.Close()

	tests := []struct {
		name        string
		doc         Document
		expectError bool
	}{
		{
			name: "valid document",
			doc: Document{
				ID:      "doc1",
				Content: "Valid document content",
			},
			expectError: false,
		},
		{
			name: "empty ID",
			doc: Document{
				ID:      "",
				Content: "Valid document content",
			},
			expectError: true,
		},
		{
			name: "empty content",
			doc: Document{
				ID:      "doc1",
				Content: "",
			},
			expectError: true,
		},
		{
			name: "content too large",
			doc: Document{
				ID:      "doc1",
				Content: string(make([]byte, 200)), // Exceeds MaxDocumentSize of 100
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := retriever.validateDocument(tt.doc)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBasicRetriever_UpdateDocument(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, DefaultRetrievalConfig())
	if err != nil {
		t.Fatalf("failed to create retriever: %v", err)
	}
	defer retriever.Close()

	ctx := context.Background()

	// Add initial document
	originalDoc := Document{
		ID:      "doc1",
		Content: "Original content",
		Title:   "Original Title",
	}

	err = retriever.AddDocument(ctx, originalDoc)
	if err != nil {
		t.Fatalf("failed to add original document: %v", err)
	}

	// Update document
	updatedDoc := Document{
		ID:      "doc1",
		Content: "Updated content with new information",
		Title:   "Updated Title",
	}

	err = retriever.UpdateDocument(ctx, updatedDoc)
	if err != nil {
		t.Errorf("failed to update document: %v", err)
	}

	// Since we can't directly verify the update with current interface,
	// we'll verify that the operation completed without error
	// In a real implementation, we'd check that the content was actually updated
}

func TestBasicRetriever_Close(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, DefaultRetrievalConfig())
	if err != nil {
		t.Fatalf("failed to create retriever: %v", err)
	}

	// Close should succeed
	err = retriever.Close()
	if err != nil {
		t.Errorf("failed to close retriever: %v", err)
	}

	// Operations should fail after close
	ctx := context.Background()
	doc := Document{
		ID:      "doc1",
		Content: "test content",
	}

	err = retriever.AddDocument(ctx, doc)
	if err == nil {
		t.Error("expected error when adding document to closed retriever")
	}

	query := Query{
		Text:      "test query",
		TopK:      5,
		Threshold: 0.5,
	}

	_, err = retriever.Retrieve(ctx, query)
	if err == nil {
		t.Error("expected error when retrieving from closed retriever")
	}
}

func TestBasicRetriever_GetStats(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, DefaultRetrievalConfig())
	if err != nil {
		t.Fatalf("failed to create retriever: %v", err)
	}
	defer retriever.Close()

	ctx := context.Background()

	// Initial stats should be zero
	stats := retriever.GetStats()
	if stats.TotalDocuments != 0 || stats.TotalChunks != 0 || stats.TotalQueries != 0 {
		t.Error("initial stats should be zero")
	}

	// Add a document
	doc := Document{
		ID:      "doc1",
		Content: "test content",
	}
	retriever.AddDocument(ctx, doc)

	// Stats should be updated
	stats = retriever.GetStats()
	if stats.TotalDocuments != 1 {
		t.Errorf("expected 1 document, got %d", stats.TotalDocuments)
	}
	if stats.TotalChunks != 1 {
		t.Errorf("expected 1 chunk, got %d", stats.TotalChunks)
	}

	// Perform a query
	query := Query{
		Text:      "test query",
		TopK:      5,
		Threshold: 0.5,
	}
	retriever.Retrieve(ctx, query)

	// Query stats should be updated
	stats = retriever.GetStats()
	if stats.TotalQueries != 1 {
		t.Errorf("expected 1 query, got %d", stats.TotalQueries)
	}
}

func TestBasicRetriever_GetDocument(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}
	defer retriever.Close()

	ctx := context.Background()

	// Test empty ID
	_, err = retriever.GetDocument(ctx, "")
	if err == nil {
		t.Error("Expected error for empty ID")
	}

	// Test non-existent document
	_, err = retriever.GetDocument(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent document")
	}

	// Add a document first
	doc := Document{
		ID:      "test-doc",
		Content: "Test content",
	}
	err = retriever.AddDocument(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Test getting existing document  
	retrieved, err := retriever.GetDocument(ctx, "test-doc_chunk_0")
	if err != nil {
		t.Fatalf("Failed to get document: %v", err)
	}

	if retrieved.Content != "Test content" {
		t.Errorf("Expected content 'Test content', got '%s'", retrieved.Content)
	}
}

func TestBasicRetriever_DeleteDocument(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	retriever, err := NewBasicRetriever(vectorStore, embedder, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create retriever: %v", err)
	}
	defer retriever.Close()

	ctx := context.Background()

	// Test empty ID
	err = retriever.DeleteDocument(ctx, "")
	if err == nil {
		t.Error("Expected error for empty ID")
	}

	// Test deleting non-existent document - mock doesn't return error for non-existent docs
	err = retriever.DeleteDocument(ctx, "non-existent")
	// Note: Mock store doesn't return error for non-existent docs, so we don't assert error
}

func TestHybridRetriever(t *testing.T) {
	vectorStore := newMockVectorStore()
	embedder := newMockEmbedder()
	
	basic, err := NewBasicRetriever(vectorStore, embedder, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create basic retriever: %v", err)
	}
	defer basic.Close()

	// Test mismatched lengths
	strategies := []SearchStrategy{&mockSearchStrategy{}}
	weights := []float32{0.5, 0.5}
	
	_, err = NewHybridRetriever(basic, strategies, weights)
	if err == nil {
		t.Error("Expected error for mismatched strategy and weight lengths")
	}

	// Test valid hybrid retriever
	weights = []float32{1.0}
	hybrid, err := NewHybridRetriever(basic, strategies, weights)
	if err != nil {
		t.Fatalf("Failed to create hybrid retriever: %v", err)
	}

	// Test hybrid retrieve with no strategies
	emptyHybrid, err := NewHybridRetriever(basic, []SearchStrategy{}, []float32{})
	if err != nil {
		t.Fatalf("Failed to create empty hybrid retriever: %v", err)
	}

	query := Query{
		Text:      "test query",
		TopK:      5,
		Threshold: 0.5,
	}

	// Should fall back to basic retriever
	_, err = emptyHybrid.Retrieve(context.Background(), query)
	if err != nil {
		t.Errorf("Failed to retrieve with empty hybrid: %v", err)
	}

	// Test with strategies
	_, err = hybrid.Retrieve(context.Background(), query)
	if err != nil {
		t.Errorf("Failed to retrieve with hybrid: %v", err)
	}
}

// Mock search strategy for testing
type mockSearchStrategy struct{}

func (m *mockSearchStrategy) Search(ctx context.Context, query Query, store vector.Store) (*RetrievalResult, error) {
	return &RetrievalResult{
		Query: query,
		Documents: []Document{
			{ID: "mock-doc", Content: "Mock content"},
		},
		Scores:     []float32{0.9},
		TotalFound: 1,
	}, nil
}

func (m *mockSearchStrategy) GetName() string {
	return "mock"
}

func (m *mockSearchStrategy) GetDescription() string {
	return "Mock search strategy for testing"
}