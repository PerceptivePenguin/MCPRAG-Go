package rag

import (
	"context"
	"io"
	
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/vector"
)

// Retriever is the main interface for document retrieval
type Retriever interface {
	// Retrieve performs a retrieval query and returns relevant documents
	Retrieve(ctx context.Context, query Query) (*RetrievalResult, error)
	
	// AddDocument adds a single document to the retrieval system
	AddDocument(ctx context.Context, doc Document) error
	
	// AddDocuments adds multiple documents in batch
	AddDocuments(ctx context.Context, docs []Document) error
	
	// UpdateDocument updates an existing document
	UpdateDocument(ctx context.Context, doc Document) error
	
	// DeleteDocument removes a document by ID
	DeleteDocument(ctx context.Context, id string) error
	
	// GetDocument retrieves a document by ID
	GetDocument(ctx context.Context, id string) (*Document, error)
	
	// GetStats returns retrieval system statistics
	GetStats() RetrievalStats
	
	// Close releases any resources held by the retriever
	Close() error
}

// Embedder generates vector embeddings for text
type Embedder interface {
	// Embed generates an embedding for a single text
	Embed(ctx context.Context, text string) (*EmbeddingResponse, error)
	
	// EmbedBatch generates embeddings for multiple texts
	EmbedBatch(ctx context.Context, texts []string) ([]*EmbeddingResponse, error)
	
	// EmbedWithOptions generates embedding with specific options
	EmbedWithOptions(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error)
	
	// GetModel returns the current embedding model name
	GetModel() string
	
	// GetDimension returns the embedding vector dimension
	GetDimension() int
	
	// Close releases any resources held by the embedder
	Close() error
}

// DocumentProcessor handles document preprocessing and chunking
type DocumentProcessor interface {
	// Process processes a document and returns chunks
	Process(ctx context.Context, doc Document, options ChunkingOptions) ([]Chunk, error)
	
	// ProcessFromReader processes document content from a reader
	ProcessFromReader(ctx context.Context, reader io.Reader, docID string, options ChunkingOptions) ([]Chunk, error)
	
	// ExtractText extracts plain text from various document formats
	ExtractText(ctx context.Context, content []byte, format string) (string, error)
	
	// ValidateDocument validates document content and format
	ValidateDocument(doc Document) error
	
	// GetSupportedFormats returns list of supported document formats
	GetSupportedFormats() []string
}

// ContextBuilder builds context from retrieved documents
type ContextBuilder interface {
	// BuildContext creates context string from retrieval results
	BuildContext(ctx context.Context, result *RetrievalResult, config ContextConfig) (string, error)
	
	// BuildContextFromDocuments creates context from document list
	BuildContextFromDocuments(ctx context.Context, docs []Document, config ContextConfig) (string, error)
	
	// FormatDocument formats a single document according to template
	FormatDocument(doc Document, template string) (string, error)
	
	// TruncateContext truncates context to fit within token limits
	TruncateContext(context string, maxTokens int, strategy string) (string, error)
}

// Cache manages embedding vector caching
type Cache interface {
	// Get retrieves a cached embedding by key
	Get(ctx context.Context, key string) (*CacheEntry, error)
	
	// Set stores an embedding in the cache
	Set(ctx context.Context, key string, vector vector.Vector, metadata map[string]string) error
	
	// Delete removes an entry from the cache
	Delete(ctx context.Context, key string) error
	
	// Clear removes all entries from the cache
	Clear(ctx context.Context) error
	
	// Size returns the number of cached entries
	Size() int
	
	// Stats returns cache statistics
	Stats() CacheStats
	
	// Close releases any resources held by the cache
	Close() error
}

// CacheStats contains cache performance statistics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	Size        int     `json:"size"`
	MaxSize     int     `json:"max_size"`
	MemoryUsage int64   `json:"memory_usage_bytes"`
}

// QueryProcessor handles query preprocessing and optimization
type QueryProcessor interface {
	// ProcessQuery preprocesses and optimizes a query
	ProcessQuery(ctx context.Context, query Query) (Query, error)
	
	// ExpandQuery expands query with synonyms or related terms
	ExpandQuery(ctx context.Context, query string) ([]string, error)
	
	// RewriteQuery rewrites query for better retrieval
	RewriteQuery(ctx context.Context, query string) (string, error)
	
	// ExtractKeywords extracts important keywords from query
	ExtractKeywords(ctx context.Context, query string) ([]string, error)
}

// SearchStrategy defines different search strategy implementations
type SearchStrategy interface {
	// Search performs search using the specific strategy
	Search(ctx context.Context, query Query, store vector.Store) (*RetrievalResult, error)
	
	// GetName returns the strategy name
	GetName() string
	
	// GetDescription returns strategy description
	GetDescription() string
}

// RerankStrategy defines document reranking strategies
type RerankStrategy interface {
	// Rerank reorders documents based on query relevance
	Rerank(ctx context.Context, query Query, docs []Document) ([]Document, []float32, error)
	
	// GetName returns the reranking strategy name
	GetName() string
}

// MetricsCollector collects and reports retrieval metrics
type MetricsCollector interface {
	// RecordQuery records a query execution
	RecordQuery(ctx context.Context, query Query, result *RetrievalResult, duration int64)
	
	// RecordEmbedding records an embedding generation
	RecordEmbedding(ctx context.Context, text string, duration int64, cached bool)
	
	// RecordCacheHit records a cache hit
	RecordCacheHit(ctx context.Context, key string)
	
	// RecordCacheMiss records a cache miss
	RecordCacheMiss(ctx context.Context, key string)
	
	// GetMetrics returns current metrics
	GetMetrics() map[string]interface{}
}

// Tokenizer handles text tokenization for chunking and context building
type Tokenizer interface {
	// CountTokens counts the number of tokens in text
	CountTokens(text string) int
	
	// Tokenize splits text into tokens
	Tokenize(text string) []string
	
	// TruncateToTokens truncates text to specified token count
	TruncateToTokens(text string, maxTokens int) string
	
	// GetModel returns the tokenizer model name
	GetModel() string
}

// IndexManager manages the document index and metadata
type IndexManager interface {
	// CreateIndex creates a new document index
	CreateIndex(ctx context.Context, name string, config *RetrievalConfig) error
	
	// DeleteIndex deletes an existing index
	DeleteIndex(ctx context.Context, name string) error
	
	// ListIndexes returns all available indexes
	ListIndexes(ctx context.Context) ([]string, error)
	
	// GetIndexStats returns statistics for an index
	GetIndexStats(ctx context.Context, name string) (*RetrievalStats, error)
	
	// BackupIndex creates a backup of the index
	BackupIndex(ctx context.Context, name string, path string) error
	
	// RestoreIndex restores an index from backup
	RestoreIndex(ctx context.Context, name string, path string) error
}

// EventListener handles retrieval system events
type EventListener interface {
	// OnDocumentAdded is called when a document is added
	OnDocumentAdded(ctx context.Context, doc Document)
	
	// OnDocumentUpdated is called when a document is updated
	OnDocumentUpdated(ctx context.Context, doc Document)
	
	// OnDocumentDeleted is called when a document is deleted
	OnDocumentDeleted(ctx context.Context, docID string)
	
	// OnQueryExecuted is called when a query is executed
	OnQueryExecuted(ctx context.Context, query Query, result *RetrievalResult)
	
	// OnError is called when an error occurs
	OnError(ctx context.Context, err error)
}