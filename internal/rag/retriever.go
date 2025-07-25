package rag

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/vector"
)

// RetrieverConfig is an alias for RetrievalConfig for backward compatibility
type RetrieverConfig = RetrievalConfig

// DefaultRetrieverConfig returns a default retriever configuration
func DefaultRetrieverConfig() *RetrieverConfig {
	return DefaultRetrievalConfig()
}

// NewRetriever creates a new retriever with auto-configuration
// This function provides backward compatibility with the agent module
func NewRetriever(config *RetrieverConfig) (Retriever, error) {
	if config == nil {
		config = DefaultRetrieverConfig()
	}

	// Create vector store
	vectorStore, err := vector.NewMemoryStore(config.VectorStore)
	if err != nil {
		return nil, NewRAGErrorWithOp("new_retriever", 
			fmt.Sprintf("failed to create vector store: %v", err), ErrorTypeInternal)
	}

	// Create embedder
	cache, err := NewLRUCache(config.Cache)
	if err != nil {
		return nil, NewRAGErrorWithOp("new_retriever", 
			fmt.Sprintf("failed to create cache: %v", err), ErrorTypeInternal)
	}
	
	embedder, err := NewOpenAIEmbedder(config.Embedding, cache)
	if err != nil {
		return nil, NewRAGErrorWithOp("new_retriever", 
			fmt.Sprintf("failed to create embedder: %v", err), ErrorTypeInternal)
	}

	// Create document processor
	// TODO: Implement proper document processor factory
	var processor DocumentProcessor

	// Create basic retriever
	return NewBasicRetriever(vectorStore, embedder, processor, config)
}

// BasicRetriever implements the Retriever interface using vector search
type BasicRetriever struct {
	vectorStore vector.Store
	embedder    Embedder
	processor   DocumentProcessor
	config      *RetrievalConfig
	stats       RetrievalStats
	mu          sync.RWMutex
	closed      bool
}

// NewBasicRetriever creates a new basic retriever
func NewBasicRetriever(
	vectorStore vector.Store,
	embedder Embedder,
	processor DocumentProcessor,
	config *RetrievalConfig,
) (*BasicRetriever, error) {
	if vectorStore == nil {
		return nil, NewRAGErrorWithOp("new_retriever", "vector store is required", ErrorTypeValidation)
	}
	if embedder == nil {
		return nil, NewRAGErrorWithOp("new_retriever", "embedder is required", ErrorTypeValidation)
	}
	if config == nil {
		config = DefaultRetrievalConfig()
	}

	return &BasicRetriever{
		vectorStore: vectorStore,
		embedder:    embedder,
		processor:   processor,
		config:      config,
		stats: RetrievalStats{
			TotalDocuments: 0,
			TotalChunks:    0,
			TotalQueries:   0,
			LastUpdated:    time.Now(),
		},
	}, nil
}

// Retrieve performs a retrieval query and returns relevant documents
func (r *BasicRetriever) Retrieve(ctx context.Context, query Query) (*RetrievalResult, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, NewRAGErrorWithOp("retrieve", "retriever is closed", ErrorTypeInternal)
	}
	r.mu.RUnlock()

	start := time.Now()
	
	// Validate query
	if err := r.validateQuery(query); err != nil {
		return nil, err
	}

	// Record query start
	embeddingStart := time.Now()

	// Generate query embedding
	embeddingResp, err := r.embedder.Embed(ctx, query.Text)
	if err != nil {
		return nil, NewRAGErrorWithCause("failed to generate query embedding", ErrorTypeExternal, err).WithOperation("retrieve")
	}
	
	embeddingTime := time.Since(embeddingStart).Milliseconds()
	searchStart := time.Now()

	// Perform vector search
	searchResults, err := r.vectorSearch(ctx, embeddingResp.Vector, query)
	if err != nil {
		return nil, err
	}

	searchTime := time.Since(searchStart).Milliseconds()

	// Convert vector search results to documents
	documents := make([]Document, len(searchResults.Documents))
	scores := make([]float32, len(searchResults.Documents))
	
	for i, doc := range searchResults.Documents {
		documents[i] = Document{
			ID:        doc.ID,
			Content:   doc.Content,
			Vector:    doc.Vector,
			CreatedAt: time.Now(), // This should come from stored metadata
			UpdatedAt: time.Now(),
		}
		scores[i] = doc.Score // Use the score from the document
	}

	result := &RetrievalResult{
		Query:         query,
		Documents:     documents,
		Scores:        scores,
		TotalFound:    len(documents),
		QueryTime:     time.Since(start).Milliseconds(),
		EmbeddingTime: embeddingTime,
		SearchTime:    searchTime,
	}

	// Update statistics
	r.updateStats(result)

	return result, nil
}

// AddDocument adds a single document to the retrieval system
func (r *BasicRetriever) AddDocument(ctx context.Context, doc Document) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return NewRAGErrorWithOp("add_document", "retriever is closed", ErrorTypeInternal)
	}
	r.mu.RUnlock()

	// Validate document
	if err := r.validateDocument(doc); err != nil {
		return err
	}

	// Process document into chunks if processor is available
	var chunks []Chunk
	if r.processor != nil {
		var err error
		chunks, err = r.processor.Process(ctx, doc, *r.config.Chunking)
		if err != nil {
			return NewRAGErrorWithCause("failed to process document", ErrorTypeInternal, err).WithOperation("add_document")
		}
	} else {
		// Create single chunk from entire document
		chunks = []Chunk{
			{
				ID:         fmt.Sprintf("%s_chunk_0", doc.ID),
				Content:    doc.Content,
				DocumentID: doc.ID,
				Index:      0,
				StartPos:   0,
				EndPos:     len(doc.Content),
				TokenCount: len(doc.Content) / 4, // rough estimate
				Metadata:   doc.Metadata,
			},
		}
	}

	// Generate embeddings for chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := r.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return NewRAGErrorWithCause("failed to generate embeddings", ErrorTypeExternal, err).WithOperation("add_document")
	}

	// Store chunks with embeddings in vector store
	for i, chunk := range chunks {
		if embeddings[i] == nil {
			continue // Skip empty chunks
		}

		vectorDoc := vector.Document{
			ID:      chunk.ID,
			Vector:  embeddings[i].Vector,
			Content: chunk.Content,
		}

		if err := r.vectorStore.Add(vectorDoc); err != nil {
			return NewRAGErrorWithCause("failed to store document chunk", ErrorTypeInternal, err).WithOperation("add_document")
		}
	}

	// Update statistics
	r.mu.Lock()
	r.stats.TotalDocuments++
	r.stats.TotalChunks += len(chunks)
	r.stats.LastUpdated = time.Now()
	r.mu.Unlock()

	return nil
}

// AddDocuments adds multiple documents in batch
func (r *BasicRetriever) AddDocuments(ctx context.Context, docs []Document) error {
	if len(docs) == 0 {
		return NewRAGErrorWithOp("add_documents", "no documents provided", ErrorTypeValidation)
	}

	// Process documents one by one for now
	// Could be optimized with batch processing
	for _, doc := range docs {
		if err := r.AddDocument(ctx, doc); err != nil {
			return err
		}
	}

	return nil
}

// UpdateDocument updates an existing document
func (r *BasicRetriever) UpdateDocument(ctx context.Context, doc Document) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return NewRAGErrorWithOp("update_document", "retriever is closed", ErrorTypeInternal)
	}
	r.mu.RUnlock()

	// For now, implement as delete + add
	// A more sophisticated implementation would update in place
	if err := r.DeleteDocument(ctx, doc.ID); err != nil {
		// If document doesn't exist, that's OK for update
		if ragErr, ok := err.(*RAGError); ok && ragErr.Type != ErrorTypeNotFound {
			return err
		}
	}

	return r.AddDocument(ctx, doc)
}

// DeleteDocument removes a document by ID
func (r *BasicRetriever) DeleteDocument(ctx context.Context, id string) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return NewRAGErrorWithOp("delete_document", "retriever is closed", ErrorTypeInternal)
	}
	r.mu.RUnlock()

	if id == "" {
		return NewRAGErrorWithOp("delete_document", "document ID is required", ErrorTypeValidation)
	}

	// Use the vector store's Delete method directly
	// This assumes document ID is the same as vector store ID
	if err := r.vectorStore.Delete(id); err != nil {
		return NewRAGErrorWithCause("failed to delete document", ErrorTypeInternal, err).WithOperation("delete_document")
	}

	return nil
}

// GetDocument retrieves a document by ID
func (r *BasicRetriever) GetDocument(ctx context.Context, id string) (*Document, error) {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return nil, NewRAGErrorWithOp("get_document", "retriever is closed", ErrorTypeInternal)
	}
	r.mu.RUnlock()

	if id == "" {
		return nil, NewRAGErrorWithOp("get_document", "document ID is required", ErrorTypeValidation)
	}

	// Get document from vector store
	vectorDoc, err := r.vectorStore.Get(id)
	if err != nil {
		return nil, NewRAGErrorWithCause("failed to get document", ErrorTypeNotFound, err).WithOperation("get_document")
	}

	// Convert vector document to RAG document
	ragDoc := &Document{
		ID:        vectorDoc.ID,
		Content:   vectorDoc.Content,
		Vector:    vectorDoc.Vector,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return ragDoc, nil
}

// GetStats returns retrieval system statistics
func (r *BasicRetriever) GetStats() RetrievalStats {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Update average query time
	if r.stats.TotalQueries > 0 {
		// This is a simplified calculation - in practice we'd maintain a running average
		r.stats.AverageQueryTime = time.Duration(r.stats.TotalQueries) * time.Millisecond
	}
	
	return r.stats
}

// Close releases any resources held by the retriever
func (r *BasicRetriever) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil
	}

	var errs []error

	// Close embedder
	if r.embedder != nil {
		if err := r.embedder.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	// Close vector store
	if r.vectorStore != nil {
		if err := r.vectorStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	r.closed = true

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	return nil
}

// Private helper methods

func (r *BasicRetriever) validateQuery(query Query) error {
	if query.Text == "" {
		return ErrQueryEmpty.WithOperation("retrieve")
	}

	if query.TopK <= 0 {
		return ErrInvalidTopK.WithOperation("retrieve")
	}

	if query.Threshold < 0 || query.Threshold > 1 {
		return ErrInvalidThreshold.WithOperation("retrieve")
	}

	return nil
}

func (r *BasicRetriever) validateDocument(doc Document) error {
	if doc.ID == "" {
		return ValidationError("id", "document ID is required")
	}

	if doc.Content == "" {
		return ErrDocumentEmpty.WithOperation("validate_document")
	}

	if len(doc.Content) > r.config.Processing.MaxDocumentSize {
		return ErrDocumentTooLarge.WithOperation("validate_document")
	}

	return nil
}

func (r *BasicRetriever) vectorSearch(ctx context.Context, queryVector vector.Vector, query Query) (*vector.SearchResult, error) {
	// Use the vector store's SearchWithThreshold method
	result, err := r.vectorStore.SearchWithThreshold(queryVector, query.TopK, query.Threshold)
	if err != nil {
		return nil, NewRAGErrorWithCause("vector search failed", ErrorTypeInternal, err).WithOperation("retrieve")
	}

	return result, nil
}

func (r *BasicRetriever) updateStats(result *RetrievalResult) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stats.TotalQueries++
	r.stats.LastUpdated = time.Now()

	// Update cache hit rate if we have access to cache stats
	// This would require integration with the cache system
}

// HybridRetriever combines multiple search strategies
type HybridRetriever struct {
	*BasicRetriever
	strategies []SearchStrategy
	weights    []float32
}

// NewHybridRetriever creates a retriever that combines multiple search strategies
func NewHybridRetriever(
	basic *BasicRetriever,
	strategies []SearchStrategy,
	weights []float32,
) (*HybridRetriever, error) {
	if len(strategies) != len(weights) {
		return nil, NewRAGErrorWithOp("new_hybrid_retriever", "strategies and weights must have same length", ErrorTypeValidation)
	}

	return &HybridRetriever{
		BasicRetriever: basic,
		strategies:     strategies,
		weights:        weights,
	}, nil
}

// Retrieve implements hybrid search by combining multiple strategies
func (h *HybridRetriever) Retrieve(ctx context.Context, query Query) (*RetrievalResult, error) {
	if len(h.strategies) == 0 {
		return h.BasicRetriever.Retrieve(ctx, query)
	}

	start := time.Now()
	
	// Execute all strategies in parallel
	type strategyResult struct {
		result *RetrievalResult
		err    error
		index  int
	}

	resultChan := make(chan strategyResult, len(h.strategies))
	
	for i, strategy := range h.strategies {
		go func(idx int, strat SearchStrategy) {
			result, err := strat.Search(ctx, query, h.vectorStore)
			resultChan <- strategyResult{result: result, err: err, index: idx}
		}(i, strategy)
	}

	// Collect results
	strategyResults := make([]*RetrievalResult, len(h.strategies))
	for i := 0; i < len(h.strategies); i++ {
		sr := <-resultChan
		if sr.err != nil {
			return nil, NewRAGErrorWithCause(
				fmt.Sprintf("strategy %d failed", sr.index),
				ErrorTypeInternal,
				sr.err,
			).WithOperation("hybrid_retrieve")
		}
		strategyResults[sr.index] = sr.result
	}

	// Combine results using weighted scoring
	combinedResult, err := h.combineResults(strategyResults, h.weights)
	if err != nil {
		return nil, err
	}

	combinedResult.QueryTime = time.Since(start).Milliseconds()
	
	// Update statistics
	h.updateStats(combinedResult)

	return combinedResult, nil
}

func (h *HybridRetriever) combineResults(results []*RetrievalResult, weights []float32) (*RetrievalResult, error) {
	if len(results) == 0 {
		return nil, NewRAGErrorWithOp("combine_results", "no results to combine", ErrorTypeValidation)
	}

	// Create a map to collect document scores across strategies
	docScores := make(map[string]float32)
	docMap := make(map[string]Document)

	// Combine scores from all strategies
	for i, result := range results {
		weight := weights[i]
		for j, doc := range result.Documents {
			score := result.Scores[j] * weight
			
			if existingScore, exists := docScores[doc.ID]; exists {
				docScores[doc.ID] = existingScore + score
			} else {
				docScores[doc.ID] = score
				docMap[doc.ID] = doc
			}
		}
	}

	// Sort documents by combined score
	type docScore struct {
		doc   Document
		score float32
	}

	var sortedDocs []docScore
	for docID, score := range docScores {
		sortedDocs = append(sortedDocs, docScore{
			doc:   docMap[docID],
			score: score,
		})
	}

	sort.Slice(sortedDocs, func(i, j int) bool {
		return sortedDocs[i].score > sortedDocs[j].score
	})

	// Extract final results
	finalDocs := make([]Document, len(sortedDocs))
	finalScores := make([]float32, len(sortedDocs))
	
	for i, ds := range sortedDocs {
		finalDocs[i] = ds.doc
		finalScores[i] = ds.score
	}

	// Use the first result as template for other fields
	template := results[0]
	
	return &RetrievalResult{
		Query:         template.Query,
		Documents:     finalDocs,
		Scores:        finalScores,
		TotalFound:    len(finalDocs),
		EmbeddingTime: template.EmbeddingTime, // Average could be computed
		SearchTime:    template.SearchTime,    // Sum could be computed
	}, nil
}