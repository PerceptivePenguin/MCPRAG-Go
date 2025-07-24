package rag

import (
	"time"
	
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/vector"
)

// Document represents a document with content and metadata
type Document struct {
	ID          string            `json:"id"`
	Content     string            `json:"content"`
	Title       string            `json:"title,omitempty"`
	Source      string            `json:"source,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ChunkIndex  int               `json:"chunk_index,omitempty"`
	ParentID    string            `json:"parent_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Vector      vector.Vector     `json:"vector,omitempty"`
}

// Query represents a retrieval query with parameters
type Query struct {
	Text           string            `json:"text"`
	TopK           int               `json:"top_k"`
	Threshold      float32           `json:"threshold"`
	Filters        map[string]string `json:"filters,omitempty"`
	IncludeVector  bool              `json:"include_vector"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	Strategy       string            `json:"strategy,omitempty"`
}

// RetrievalResult contains the results of a document retrieval
type RetrievalResult struct {
	Query         Query      `json:"query"`
	Documents     []Document `json:"documents"`
	Scores        []float32  `json:"scores"`
	TotalFound    int        `json:"total_found"`
	QueryTime     int64      `json:"query_time_ms"`
	EmbeddingTime int64      `json:"embedding_time_ms"`
	SearchTime    int64      `json:"search_time_ms"`
	Context       string     `json:"context,omitempty"`
}

// ChunkingOptions configures document chunking behavior
type ChunkingOptions struct {
	Strategy     ChunkStrategy `json:"strategy"`
	MaxChunkSize int           `json:"max_chunk_size"`
	Overlap      int           `json:"overlap"`
	Separators   []string      `json:"separators,omitempty"`
	PreserveStructure bool     `json:"preserve_structure"`
}

// ChunkStrategy defines different chunking strategies
type ChunkStrategy string

const (
	ChunkByTokens    ChunkStrategy = "tokens"
	ChunkBySentences ChunkStrategy = "sentences"
	ChunkByParagraphs ChunkStrategy = "paragraphs"
	ChunkByFixedSize ChunkStrategy = "fixed_size"
	ChunkBySemantic  ChunkStrategy = "semantic"
)

// EmbeddingConfig configures the embedding generation
type EmbeddingConfig struct {
	Model       string            `json:"model"`
	APIKey      string            `json:"api_key"`
	BaseURL     string            `json:"base_url,omitempty"`
	MaxRetries  int               `json:"max_retries"`
	Timeout     time.Duration     `json:"timeout"`
	BatchSize   int               `json:"batch_size"`
	RateLimit   int               `json:"rate_limit"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// DefaultEmbeddingConfig returns default embedding configuration
func DefaultEmbeddingConfig() *EmbeddingConfig {
	return &EmbeddingConfig{
		Model:      "text-embedding-3-small",
		MaxRetries: 3,
		Timeout:    30 * time.Second,
		BatchSize:  100,
		RateLimit:  60, // requests per minute
	}
}

// CacheConfig configures the embedding cache
type CacheConfig struct {
	Enabled     bool          `json:"enabled"`
	MaxSize     int           `json:"max_size"`
	TTL         time.Duration `json:"ttl"`
	PersistPath string        `json:"persist_path,omitempty"`
	Strategy    CacheStrategy `json:"strategy"`
}

// CacheStrategy defines cache eviction strategies
type CacheStrategy string

const (
	CacheLRU  CacheStrategy = "lru"
	CacheLFU  CacheStrategy = "lfu"
	CacheFIFO CacheStrategy = "fifo"
)

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enabled:  true,
		MaxSize:  10000,
		TTL:      24 * time.Hour,
		Strategy: CacheLRU,
	}
}

// ContextConfig configures context building behavior
type ContextConfig struct {
	Template         string `json:"template"`
	MaxLength        int    `json:"max_length"`
	IncludeMetadata  bool   `json:"include_metadata"`
	IncludeScores    bool   `json:"include_scores"`
	SeparateChunks   bool   `json:"separate_chunks"`
	TruncateStrategy string `json:"truncate_strategy"`
}

// DefaultContextConfig returns default context configuration
func DefaultContextConfig() *ContextConfig {
	return &ContextConfig{
		Template:         "Context: {{.Content}}",
		MaxLength:        4000,
		IncludeMetadata:  false,
		IncludeScores:    false,
		SeparateChunks:   true,
		TruncateStrategy: "tail",
	}
}

// ProcessingOptions contains options for document processing
type ProcessingOptions struct {
	Language         string   `json:"language,omitempty"`
	RemoveStopWords  bool     `json:"remove_stop_words"`
	Lowercase        bool     `json:"lowercase"`
	RemovePunctuation bool    `json:"remove_punctuation"`
	SupportedFormats []string `json:"supported_formats"`
	MaxDocumentSize  int      `json:"max_document_size"`
}

// DefaultProcessingOptions returns default processing options
func DefaultProcessingOptions() *ProcessingOptions {
	return &ProcessingOptions{
		Language:         "en",
		RemoveStopWords:  false,
		Lowercase:        false,
		RemovePunctuation: false,
		SupportedFormats: []string{"txt", "md", "json"},
		MaxDocumentSize:  1024 * 1024, // 1MB
	}
}

// RetrievalStats contains statistics about retrieval operations
type RetrievalStats struct {
	TotalDocuments   int           `json:"total_documents"`
	TotalChunks      int           `json:"total_chunks"`
	TotalQueries     int64         `json:"total_queries"`
	AverageQueryTime time.Duration `json:"average_query_time"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	EmbeddingsCached int           `json:"embeddings_cached"`
	LastUpdated      time.Time     `json:"last_updated"`
}

// EmbeddingRequest represents a request for generating embeddings
type EmbeddingRequest struct {
	Text      string            `json:"text"`
	Model     string            `json:"model,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	BatchID   string            `json:"batch_id,omitempty"`
	Priority  int               `json:"priority,omitempty"`
}

// EmbeddingResponse represents the response from embedding generation
type EmbeddingResponse struct {
	Vector    vector.Vector     `json:"vector"`
	Model     string            `json:"model"`
	Usage     EmbeddingUsage    `json:"usage"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	Cached    bool              `json:"cached"`
}

// EmbeddingUsage tracks API usage for embeddings
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// CacheEntry represents a cached embedding
type CacheEntry struct {
	Key       string        `json:"key"`
	Vector    vector.Vector `json:"vector"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	AccessedAt time.Time    `json:"accessed_at"`
	AccessCount int          `json:"access_count"`
	Size       int          `json:"size"`
}

// Chunk represents a document chunk with its metadata
type Chunk struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	DocumentID string            `json:"document_id"`
	Index      int               `json:"index"`
	StartPos   int               `json:"start_pos"`
	EndPos     int               `json:"end_pos"`
	TokenCount int               `json:"token_count"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Vector     vector.Vector     `json:"vector,omitempty"`
}

// SearchStrategyType defines different search strategies
type SearchStrategyType string

const (
	SearchSemantic SearchStrategyType = "semantic"
	SearchHybrid   SearchStrategyType = "hybrid" 
	SearchKeyword  SearchStrategyType = "keyword"
	SearchRerank   SearchStrategyType = "rerank"
)

// RetrievalConfig holds the main configuration for the retrieval system
type RetrievalConfig struct {
	Embedding  *EmbeddingConfig   `json:"embedding"`
	Cache      *CacheConfig       `json:"cache"`
	Chunking   *ChunkingOptions   `json:"chunking"`
	Context    *ContextConfig     `json:"context"`
	Processing *ProcessingOptions `json:"processing"`
	VectorStore *vector.Config    `json:"vector_store"`
}

// DefaultRetrievalConfig returns a default retrieval configuration
func DefaultRetrievalConfig() *RetrievalConfig {
	return &RetrievalConfig{
		Embedding:   DefaultEmbeddingConfig(),
		Cache:       DefaultCacheConfig(),
		Chunking:    DefaultChunkingOptions(),
		Context:     DefaultContextConfig(),
		Processing:  DefaultProcessingOptions(),
		VectorStore: vector.DefaultConfig(),
	}
}

// DefaultChunkingOptions returns default chunking options
func DefaultChunkingOptions() *ChunkingOptions {
	return &ChunkingOptions{
		Strategy:     ChunkByTokens,
		MaxChunkSize: 1000,
		Overlap:      100,
		Separators:   []string{"\n\n", "\n", ". ", "? ", "! "},
		PreserveStructure: true,
	}
}