package vector

import (
	"fmt"
	"sort"
	"time"
)

// NewMemoryStore creates a new in-memory vector store
func NewMemoryStore(config *Config) (*MemoryStore, error) {
	if config == nil {
		config = DefaultConfig()
	}
	
	if config.Dimension <= 0 {
		return nil, NewVectorErrorWithOp("new_store", ErrInvalidDimension)
	}
	
	return &MemoryStore{
		documents: make(map[string]Document),
		vectors:   make([]Vector, 0),
		ids:       make([]string, 0),
		dimension: config.Dimension,
	}, nil
}

// Add adds a document with its vector to the store
func (s *MemoryStore) Add(doc Document) error {
	if doc.ID == "" {
		return NewVectorErrorWithOp("add", fmt.Errorf("document ID cannot be empty"))
	}
	
	if len(doc.Vector) == 0 {
		return NewVectorErrorWithOp("add", ErrEmptyVector)
	}
	
	if len(doc.Vector) != s.dimension {
		return NewVectorErrorWithOp("add", fmt.Errorf("vector dimension %d does not match store dimension %d", len(doc.Vector), s.dimension))
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if document already exists
	if _, exists := s.documents[doc.ID]; exists {
		// Update existing document
		for i, id := range s.ids {
			if id == doc.ID {
				s.vectors[i] = doc.Vector
				break
			}
		}
	} else {
		// Add new document
		s.vectors = append(s.vectors, doc.Vector)
		s.ids = append(s.ids, doc.ID)
	}
	
	s.documents[doc.ID] = doc
	return nil
}

// AddBatch adds multiple documents in a single operation
func (s *MemoryStore) AddBatch(docs []Document) error {
	if len(docs) == 0 {
		return nil
	}
	
	// Validate all documents first
	for i, doc := range docs {
		if doc.ID == "" {
			return NewVectorErrorWithOp("add_batch", fmt.Errorf("document at index %d has empty ID", i))
		}
		
		if len(doc.Vector) == 0 {
			return NewVectorErrorWithOp("add_batch", fmt.Errorf("document at index %d has empty vector", i))
		}
		
		if len(doc.Vector) != s.dimension {
			return NewVectorErrorWithOp("add_batch", fmt.Errorf("document at index %d has vector dimension %d, expected %d", i, len(doc.Vector), s.dimension))
		}
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for _, doc := range docs {
		// Check if document already exists
		if _, exists := s.documents[doc.ID]; exists {
			// Update existing document
			for i, id := range s.ids {
				if id == doc.ID {
					s.vectors[i] = doc.Vector
					break
				}
			}
		} else {
			// Add new document
			s.vectors = append(s.vectors, doc.Vector)
			s.ids = append(s.ids, doc.ID)
		}
		
		s.documents[doc.ID] = doc
	}
	
	return nil
}

// Search performs similarity search and returns top-k most similar documents
func (s *MemoryStore) Search(queryVector Vector, topK int) (*SearchResult, error) {
	return s.SearchWithThreshold(queryVector, topK, 0.0)
}

// SearchWithThreshold performs similarity search with a minimum similarity threshold
func (s *MemoryStore) SearchWithThreshold(queryVector Vector, topK int, threshold float32) (*SearchResult, error) {
	if topK <= 0 {
		return nil, NewVectorErrorWithOp("search", ErrInvalidTopK)
	}
	
	if threshold < 0.0 || threshold > 1.0 {
		return nil, NewVectorErrorWithOp("search", ErrInvalidThreshold)
	}
	
	if len(queryVector) != s.dimension {
		return nil, NewVectorErrorWithOp("search", fmt.Errorf("query vector dimension %d does not match store dimension %d", len(queryVector), s.dimension))
	}
	
	start := time.Now()
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if len(s.vectors) == 0 {
		return &SearchResult{
			Documents: []Document{},
			QueryTime: time.Since(start).Milliseconds(),
		}, nil
	}
	
	// Calculate similarities for all vectors
	similarities := BatchCosineSimilarity(queryVector, s.vectors)
	
	// Create scored documents
	type scoredDoc struct {
		doc   Document
		score float32
		index int
	}
	
	var candidates []scoredDoc
	for i, score := range similarities {
		if score >= threshold {
			doc := s.documents[s.ids[i]]
			doc.Score = score
			candidates = append(candidates, scoredDoc{
				doc:   doc,
				score: score,
				index: i,
			})
		}
	}
	
	// Sort by similarity score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	
	// Take top-k results
	resultCount := topK
	if resultCount > len(candidates) {
		resultCount = len(candidates)
	}
	
	results := make([]Document, resultCount)
	for i := 0; i < resultCount; i++ {
		results[i] = candidates[i].doc
	}
	
	return &SearchResult{
		Documents: results,
		QueryTime: time.Since(start).Milliseconds(),
	}, nil
}

// Get retrieves a document by its ID
func (s *MemoryStore) Get(id string) (*Document, error) {
	if id == "" {
		return nil, NewVectorErrorWithOp("get", fmt.Errorf("document ID cannot be empty"))
	}
	
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	doc, exists := s.documents[id]
	if !exists {
		return nil, NewVectorErrorWithOp("get", ErrDocumentNotFound)
	}
	
	return &doc, nil
}

// Delete removes a document from the store
func (s *MemoryStore) Delete(id string) error {
	if id == "" {
		return NewVectorErrorWithOp("delete", fmt.Errorf("document ID cannot be empty"))
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.documents[id]; !exists {
		return NewVectorErrorWithOp("delete", ErrDocumentNotFound)
	}
	
	// Find and remove from vectors and ids slices
	for i, docID := range s.ids {
		if docID == id {
			// Remove from vectors slice
			s.vectors = append(s.vectors[:i], s.vectors[i+1:]...)
			// Remove from ids slice
			s.ids = append(s.ids[:i], s.ids[i+1:]...)
			break
		}
	}
	
	// Remove from documents map
	delete(s.documents, id)
	
	return nil
}

// Size returns the number of documents in the store
func (s *MemoryStore) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

// Clear removes all documents from the store
func (s *MemoryStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.documents = make(map[string]Document)
	s.vectors = make([]Vector, 0)
	s.ids = make([]string, 0)
	
	return nil
}

// Close releases any resources held by the store
func (s *MemoryStore) Close() error {
	return s.Clear()
}

// GetDimension returns the vector dimension of the store
func (s *MemoryStore) GetDimension() int {
	return s.dimension
}

// ListIDs returns all document IDs in the store
func (s *MemoryStore) ListIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	ids := make([]string, len(s.ids))
	copy(ids, s.ids)
	return ids
}

// Stats returns statistics about the store
type StoreStats struct {
	DocumentCount int `json:"document_count"`
	Dimension     int `json:"dimension"`
	MemoryUsage   int `json:"memory_usage_bytes"`
}

// GetStats returns statistics about the store
func (s *MemoryStore) GetStats() StoreStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Estimate memory usage
	memoryUsage := len(s.documents) * (32 + s.dimension*4) // rough estimate
	
	return StoreStats{
		DocumentCount: len(s.documents),
		Dimension:     s.dimension,
		MemoryUsage:   memoryUsage,
	}
}