package vector

import (
	"sync"
)

// Vector represents a high-dimensional vector for semantic search
type Vector []float32

// Document represents a document with its associated vector embedding
type Document struct {
	ID      string  `json:"id"`
	Content string  `json:"content"`
	Vector  Vector  `json:"vector"`
	Score   float32 `json:"score,omitempty"` // similarity score for search results
}

// SearchResult represents the result of a vector similarity search
type SearchResult struct {
	Documents []Document `json:"documents"`
	QueryTime int64      `json:"query_time_ms"`
}

// Store defines the interface for vector storage operations
type Store interface {
	// Add adds a document with its vector to the store
	Add(doc Document) error
	
	// AddBatch adds multiple documents in a single operation
	AddBatch(docs []Document) error
	
	// Search performs similarity search and returns top-k most similar documents
	Search(queryVector Vector, topK int) (*SearchResult, error)
	
	// SearchWithThreshold performs similarity search with a minimum similarity threshold
	SearchWithThreshold(queryVector Vector, topK int, threshold float32) (*SearchResult, error)
	
	// Get retrieves a document by its ID
	Get(id string) (*Document, error)
	
	// Delete removes a document from the store
	Delete(id string) error
	
	// Size returns the number of documents in the store
	Size() int
	
	// Clear removes all documents from the store
	Clear() error
	
	// Close releases any resources held by the store
	Close() error
}

// MemoryStore implements Store interface with in-memory storage
type MemoryStore struct {
	mu        sync.RWMutex
	documents map[string]Document
	vectors   []Vector
	ids       []string
	dimension int
}

// Config holds configuration for the vector store
type Config struct {
	Dimension           int     `yaml:"dimension" json:"dimension"`
	SimilarityThreshold float32 `yaml:"similarity_threshold" json:"similarity_threshold"`
	MaxDocuments        int     `yaml:"max_documents" json:"max_documents"`
	EnableSIMD          bool    `yaml:"enable_simd" json:"enable_simd"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Dimension:           1536, // OpenAI text-embedding-3-small dimension
		SimilarityThreshold: 0.7,
		MaxDocuments:        10000,
		EnableSIMD:          true,
	}
}

// Similarity function type for vector similarity calculation
type SimilarityFunc func(a, b Vector) float32

// Common error variables
var (
	ErrDocumentNotFound    = NewVectorError("document not found")
	ErrInvalidDimension    = NewVectorError("invalid vector dimension")
	ErrEmptyVector         = NewVectorError("vector cannot be empty")
	ErrInvalidTopK         = NewVectorError("topK must be positive")
	ErrStoreAtCapacity     = NewVectorError("store has reached maximum capacity")
	ErrInvalidThreshold    = NewVectorError("threshold must be between 0 and 1")
)

// VectorError represents errors specific to vector operations
type VectorError struct {
	Op  string
	Err error
	Msg string
}

func (e *VectorError) Error() string {
	if e.Err != nil {
		return "vector " + e.Op + ": " + e.Err.Error()
	}
	return "vector " + e.Op + ": " + e.Msg
}

func (e *VectorError) Unwrap() error {
	return e.Err
}

// NewVectorError creates a new vector error
func NewVectorError(msg string) *VectorError {
	return &VectorError{Op: "operation", Msg: msg}
}

// NewVectorErrorWithOp creates a new vector error with operation context
func NewVectorErrorWithOp(op string, err error) *VectorError {
	return &VectorError{Op: op, Err: err}
}