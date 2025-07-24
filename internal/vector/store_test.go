package vector

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestNewMemoryStore(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name:        "default config",
			config:      nil,
			expectError: false,
		},
		{
			name: "custom config",
			config: &Config{
				Dimension:           512,
				SimilarityThreshold: 0.8,
			},
			expectError: false,
		},
		{
			name: "invalid dimension",
			config: &Config{
				Dimension: 0,
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewMemoryStore(tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			if store == nil {
				t.Error("expected store but got nil")
				return
			}
			
			expectedDim := 1536 // default
			if tt.config != nil && tt.config.Dimension > 0 {
				expectedDim = tt.config.Dimension
			}
			
			if store.GetDimension() != expectedDim {
				t.Errorf("expected dimension %d, got %d", expectedDim, store.GetDimension())
			}
		})
	}
}

func TestMemoryStore_Add(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	tests := []struct {
		name        string
		doc         Document
		expectError bool
	}{
		{
			name: "valid document",
			doc: Document{
				ID:      "doc1",
				Content: "test content",
				Vector:  make(Vector, 1536),
			},
			expectError: false,
		},
		{
			name: "empty ID",
			doc: Document{
				ID:      "",
				Content: "test content",
				Vector:  make(Vector, 1536),
			},
			expectError: true,
		},
		{
			name: "empty vector",
			doc: Document{
				ID:      "doc2",
				Content: "test content",
				Vector:  Vector{},
			},
			expectError: true,
		},
		{
			name: "wrong dimension",
			doc: Document{
				ID:      "doc3",
				Content: "test content",
				Vector:  make(Vector, 100),
			},
			expectError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.Add(tt.doc)
			
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			
			// Verify document was added
			retrieved, err := store.Get(tt.doc.ID)
			if err != nil {
				t.Errorf("failed to retrieve added document: %v", err)
				return
			}
			
			if retrieved.ID != tt.doc.ID {
				t.Errorf("expected ID %s, got %s", tt.doc.ID, retrieved.ID)
			}
			
			if retrieved.Content != tt.doc.Content {
				t.Errorf("expected content %s, got %s", tt.doc.Content, retrieved.Content)
			}
		})
	}
}

func TestMemoryStore_AddUpdate(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	// Add initial document
	doc := Document{
		ID:      "doc1",
		Content: "original content",
		Vector:  Vector{1, 0, 0}, // pad to 1536 later
	}
	
	// Pad vector to correct dimension
	for len(doc.Vector) < 1536 {
		doc.Vector = append(doc.Vector, 0)
	}
	
	err = store.Add(doc)
	if err != nil {
		t.Fatal(err)
	}
	
	if store.Size() != 1 {
		t.Errorf("expected size 1, got %d", store.Size())
	}
	
	// Update document
	updatedDoc := Document{
		ID:      "doc1",
		Content: "updated content",
		Vector:  Vector{0, 1, 0}, // different vector
	}
	
	// Pad vector to correct dimension
	for len(updatedDoc.Vector) < 1536 {
		updatedDoc.Vector = append(updatedDoc.Vector, 0)
	}
	
	err = store.Add(updatedDoc)
	if err != nil {
		t.Fatal(err)
	}
	
	// Size should still be 1
	if store.Size() != 1 {
		t.Errorf("expected size 1 after update, got %d", store.Size())
	}
	
	// Verify updated content
	retrieved, err := store.Get("doc1")
	if err != nil {
		t.Fatal(err)
	}
	
	if retrieved.Content != "updated content" {
		t.Errorf("expected updated content, got %s", retrieved.Content)
	}
}

func TestMemoryStore_AddBatch(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	// Create test documents
	docs := make([]Document, 5)
	for i := range docs {
		docs[i] = Document{
			ID:      fmt.Sprintf("doc%d", i),
			Content: fmt.Sprintf("content %d", i),
			Vector:  make(Vector, 1536),
		}
		
		// Set first element to i for differentiation
		docs[i].Vector[0] = float32(i)
	}
	
	err = store.AddBatch(docs)
	if err != nil {
		t.Fatal(err)
	}
	
	if store.Size() != 5 {
		t.Errorf("expected size 5, got %d", store.Size())
	}
	
	// Verify all documents were added
	for i, doc := range docs {
		retrieved, err := store.Get(doc.ID)
		if err != nil {
			t.Errorf("failed to retrieve document %d: %v", i, err)
			continue
		}
		
		if retrieved.Content != doc.Content {
			t.Errorf("document %d: expected content %s, got %s", i, doc.Content, retrieved.Content)
		}
	}
}

func TestMemoryStore_Search(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	// Add test documents with known vectors
	docs := []Document{
		{
			ID:      "doc1",
			Content: "identical vector",
			Vector:  Vector{1, 0, 0}, // will be padded
		},
		{
			ID:      "doc2", 
			Content: "orthogonal vector",
			Vector:  Vector{0, 1, 0}, // will be padded
		},
		{
			ID:      "doc3",
			Content: "opposite vector", 
			Vector:  Vector{-1, 0, 0}, // will be padded
		},
		{
			ID:      "doc4",
			Content: "similar vector",
			Vector:  Vector{0.8, 0.2, 0}, // will be padded
		},
	}
	
	// Pad all vectors to correct dimension
	for i := range docs {
		for len(docs[i].Vector) < 1536 {
			docs[i].Vector = append(docs[i].Vector, 0)
		}
	}
	
	err = store.AddBatch(docs)
	if err != nil {
		t.Fatal(err)
	}
	
	// Search with query vector [1, 0, 0, 0, ...]
	queryVector := make(Vector, 1536)
	queryVector[0] = 1
	
	result, err := store.Search(queryVector, 2)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(result.Documents) != 2 {
		t.Errorf("expected 2 results, got %d", len(result.Documents))
	}
	
	// First result should be doc1 (identical vector, score = 1.0)
	if result.Documents[0].ID != "doc1" {
		t.Errorf("expected first result to be doc1, got %s", result.Documents[0].ID)
	}
	
	// Second result should be doc4 (similar vector)
	if result.Documents[1].ID != "doc4" {
		t.Errorf("expected second result to be doc4, got %s", result.Documents[1].ID)
	}
	
	// Scores should be in descending order
	if result.Documents[0].Score < result.Documents[1].Score {
		t.Error("results not ordered by score")
	}
}

func TestMemoryStore_SearchWithThreshold(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	// Add documents with known similarities
	docs := []Document{
		{
			ID:      "high_sim",
			Content: "high similarity",
			Vector:  Vector{1, 0, 0}, // similarity = 1.0
		},
		{
			ID:      "medium_sim", 
			Content: "medium similarity",
			Vector:  Vector{0.5, 0.5, 0}, // similarity ≈ 0.7
		},
		{
			ID:      "low_sim",
			Content: "low similarity",
			Vector:  Vector{0.1, 0.9, 0}, // similarity ≈ 0.1
		},
	}
	
	// Pad vectors
	for i := range docs {
		for len(docs[i].Vector) < 1536 {
			docs[i].Vector = append(docs[i].Vector, 0)
		}
	}
	
	err = store.AddBatch(docs)
	if err != nil {
		t.Fatal(err)
	}
	
	queryVector := make(Vector, 1536)
	queryVector[0] = 1
	
	// Search with threshold 0.5
	result, err := store.SearchWithThreshold(queryVector, 10, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	
	// Should only return documents with similarity >= 0.5
	if len(result.Documents) != 2 {
		t.Errorf("expected 2 results above threshold, got %d", len(result.Documents))
	}
	
	// All returned documents should have score >= 0.5
	for i, doc := range result.Documents {
		if doc.Score < 0.5 {
			t.Errorf("document %d has score %f below threshold 0.5", i, doc.Score)
		}
	}
}

func TestMemoryStore_Get(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	doc := Document{
		ID:      "test_doc",
		Content: "test content",
		Vector:  make(Vector, 1536),
	}
	
	// Test getting non-existent document
	_, err = store.Get("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent document")
	}
	
	// Add document
	err = store.Add(doc)
	if err != nil {
		t.Fatal(err)
	}
	
	// Test getting existing document
	retrieved, err := store.Get("test_doc")
	if err != nil {
		t.Fatal(err)
	}
	
	if retrieved.ID != doc.ID {
		t.Errorf("expected ID %s, got %s", doc.ID, retrieved.ID)
	}
	
	// Test empty ID
	_, err = store.Get("")
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	doc := Document{
		ID:      "test_doc",
		Content: "test content", 
		Vector:  make(Vector, 1536),
	}
	
	// Test deleting non-existent document
	err = store.Delete("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent document")
	}
	
	// Add document
	err = store.Add(doc)
	if err != nil {
		t.Fatal(err)
	}
	
	if store.Size() != 1 {
		t.Errorf("expected size 1, got %d", store.Size())
	}
	
	// Delete document
	err = store.Delete("test_doc")
	if err != nil {
		t.Fatal(err)
	}
	
	if store.Size() != 0 {
		t.Errorf("expected size 0 after deletion, got %d", store.Size())
	}
	
	// Verify document is gone
	_, err = store.Get("test_doc")
	if err == nil {
		t.Error("document should not exist after deletion")
	}
	
	// Test empty ID
	err = store.Delete("")
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	// Add some documents
	docs := make([]Document, 3)
	for i := range docs {
		docs[i] = Document{
			ID:      fmt.Sprintf("doc%d", i),
			Content: fmt.Sprintf("content %d", i),
			Vector:  make(Vector, 1536),
		}
	}
	
	err = store.AddBatch(docs)
	if err != nil {
		t.Fatal(err)
	}
	
	if store.Size() != 3 {
		t.Errorf("expected size 3, got %d", store.Size())
	}
	
	// Clear store
	err = store.Clear()
	if err != nil {
		t.Fatal(err)
	}
	
	if store.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", store.Size())
	}
}

func TestMemoryStore_Concurrent(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	const numWorkers = 10
	const docsPerWorker = 100
	
	var wg sync.WaitGroup
	
	// Concurrent writes
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			for j := 0; j < docsPerWorker; j++ {
				doc := Document{
					ID:      fmt.Sprintf("worker%d_doc%d", workerID, j),
					Content: fmt.Sprintf("content from worker %d doc %d", workerID, j),
					Vector:  make(Vector, 1536),
				}
				
				doc.Vector[0] = float32(workerID)
				doc.Vector[1] = float32(j)
				
				err := store.Add(doc)
				if err != nil {
					t.Errorf("worker %d failed to add doc %d: %v", workerID, j, err)
				}
			}
		}(i)
	}
	
	wg.Wait()
	
	expectedSize := numWorkers * docsPerWorker
	if store.Size() != expectedSize {
		t.Errorf("expected size %d, got %d", expectedSize, store.Size())
	}
	
	// Concurrent reads
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			queryVector := make(Vector, 1536)
			queryVector[0] = float32(workerID)
			
			result, err := store.Search(queryVector, 10)
			if err != nil {
				t.Errorf("worker %d failed to search: %v", workerID, err)
				return
			}
			
			if len(result.Documents) == 0 {
				t.Errorf("worker %d got no search results", workerID)
			}
		}(i)
	}
	
	wg.Wait()
}

func TestMemoryStore_Stats(t *testing.T) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	
	stats := store.GetStats()
	if stats.DocumentCount != 0 {
		t.Errorf("expected 0 documents, got %d", stats.DocumentCount)
	}
	
	if stats.Dimension != 1536 {
		t.Errorf("expected dimension 1536, got %d", stats.Dimension)
	}
	
	// Add a document
	doc := Document{
		ID:      "test",
		Content: "test content",
		Vector:  make(Vector, 1536),
	}
	
	err = store.Add(doc)
	if err != nil {
		t.Fatal(err)
	}
	
	stats = store.GetStats()
	if stats.DocumentCount != 1 {
		t.Errorf("expected 1 document, got %d", stats.DocumentCount)
	}
	
	if stats.MemoryUsage <= 0 {
		t.Error("expected positive memory usage")
	}
}

// Benchmark tests
func BenchmarkMemoryStore_Add(b *testing.B) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	
	// Pre-generate documents
	docs := make([]Document, b.N)
	for i := 0; i < b.N; i++ {
		docs[i] = Document{
			ID:      strconv.Itoa(i),
			Content: fmt.Sprintf("content %d", i),
			Vector:  make(Vector, 1536),
		}
		
		// Initialize vector with some values
		for j := 0; j < 10; j++ {
			docs[i].Vector[j] = float32(i%100) / 100.0
		}
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Add(docs[i])
	}
}

func BenchmarkMemoryStore_Search(b *testing.B) {
	store, err := NewMemoryStore(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	
	// Add 1000 documents
	for i := 0; i < 1000; i++ {
		doc := Document{
			ID:      strconv.Itoa(i),
			Content: fmt.Sprintf("content %d", i),
			Vector:  make(Vector, 1536),
		}
		
		// Initialize vector
		for j := 0; j < 10; j++ {
			doc.Vector[j] = float32((i+j)%100) / 100.0
		}
		
		store.Add(doc)
	}
	
	queryVector := make(Vector, 1536)
	for i := 0; i < 10; i++ {
		queryVector[i] = float32(i) / 10.0
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Search(queryVector, 10)
	}
}