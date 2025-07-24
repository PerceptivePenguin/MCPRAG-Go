package rag

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"sync"
	"time"
	
	"github.com/sashabaranov/go-openai"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/vector"
)

// OpenAIEmbedder implements the Embedder interface using OpenAI's API
type OpenAIEmbedder struct {
	client   *openai.Client
	config   *EmbeddingConfig
	cache    Cache
	metrics  MetricsCollector
	
	// Rate limiting
	mu         sync.Mutex
	lastCall   time.Time
	rateLimiter chan struct{}
}

// NewOpenAIEmbedder creates a new OpenAI embedder
func NewOpenAIEmbedder(config *EmbeddingConfig, cache Cache) (*OpenAIEmbedder, error) {
	if config == nil {
		config = DefaultEmbeddingConfig()
	}
	
	if config.APIKey == "" {
		return nil, NewRAGErrorWithOp("new_embedder", "OpenAI API key is required", ErrorTypeAuth)
	}
	
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}
	
	// Add custom headers if provided
	if len(config.Headers) > 0 {
		// Note: go-openai doesn't directly support custom headers in config
		// This would need to be implemented via custom HTTP client
	}
	
	client := openai.NewClientWithConfig(clientConfig)
	
	// Create rate limiter channel
	rateLimit := config.RateLimit
	if rateLimit <= 0 {
		rateLimit = 60 // Default rate limit
	}
	rateLimiter := make(chan struct{}, rateLimit)
	for i := 0; i < rateLimit; i++ {
		rateLimiter <- struct{}{}
	}
	
	embedder := &OpenAIEmbedder{
		client:      client,
		config:      config,
		cache:       cache,
		rateLimiter: rateLimiter,
	}
	
	// Start rate limiter refill goroutine
	go embedder.refillRateLimiter()
	
	return embedder, nil
}

// Embed generates an embedding for a single text
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) (*EmbeddingResponse, error) {
	if strings.TrimSpace(text) == "" {
		return nil, ErrQueryEmpty.WithOperation("embed")
	}
	
	start := time.Now()
	
	// Generate cache key
	cacheKey := e.generateCacheKey(text, e.config.Model)
	
	// Check cache first
	if e.cache != nil {
		if entry, err := e.cache.Get(ctx, cacheKey); err == nil {
			if e.metrics != nil {
				e.metrics.RecordCacheHit(ctx, cacheKey)
				e.metrics.RecordEmbedding(ctx, text, time.Since(start).Milliseconds(), true)
			}
			
			return &EmbeddingResponse{
				Vector:    entry.Vector,
				Model:     e.config.Model,
				Metadata:  entry.Metadata,
				RequestID: cacheKey,
				Cached:    true,
			}, nil
		} else if e.metrics != nil {
			e.metrics.RecordCacheMiss(ctx, cacheKey)
		}
	}
	
	// Generate embedding via API
	response, err := e.embedWithRetry(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	
	if len(response.Data) == 0 {
		return nil, NewRAGErrorWithOp("embed", "no embedding data returned", ErrorTypeExternal)
	}
	
	embedding := response.Data[0]
	vectorData := make(vector.Vector, len(embedding.Embedding))
	for i, val := range embedding.Embedding {
		vectorData[i] = val
	}
	
	// Cache the result
	if e.cache != nil {
		metadata := map[string]string{
			"model": e.config.Model,
			"text_hash": fmt.Sprintf("%x", md5.Sum([]byte(text))),
		}
		if err := e.cache.Set(ctx, cacheKey, vectorData, metadata); err != nil {
			// Log cache error but don't fail the request
		}
	}
	
	result := &EmbeddingResponse{
		Vector: vectorData,
		Model:  e.config.Model,
		Usage: EmbeddingUsage{
			PromptTokens: response.Usage.PromptTokens,
			TotalTokens:  response.Usage.TotalTokens,
		},
		RequestID: fmt.Sprintf("embed_%d", time.Now().Unix()),
		Cached:    false,
	}
	
	if e.metrics != nil {
		e.metrics.RecordEmbedding(ctx, text, time.Since(start).Milliseconds(), false)
	}
	
	return result, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *OpenAIEmbedder) EmbedBatch(ctx context.Context, texts []string) ([]*EmbeddingResponse, error) {
	if len(texts) == 0 {
		return nil, NewRAGErrorWithOp("embed_batch", "no texts provided", ErrorTypeValidation)
	}
	
	// Filter empty texts
	validTexts := make([]string, 0, len(texts))
	indexMap := make([]int, 0, len(texts))
	
	for i, text := range texts {
		if strings.TrimSpace(text) != "" {
			validTexts = append(validTexts, text)
			indexMap = append(indexMap, i)
		}
	}
	
	if len(validTexts) == 0 {
		return nil, ErrQueryEmpty.WithOperation("embed_batch")
	}
	
	results := make([]*EmbeddingResponse, len(texts))
	
	// Process in batches according to config
	batchSize := e.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	
	for i := 0; i < len(validTexts); i += batchSize {
		end := i + batchSize
		if end > len(validTexts) {
			end = len(validTexts)
		}
		
		batch := validTexts[i:end]
		batchResults, err := e.processBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		
		// Map results back to original indices
		for j, result := range batchResults {
			originalIndex := indexMap[i+j]
			results[originalIndex] = result
		}
	}
	
	return results, nil
}

// EmbedWithOptions generates embedding with specific options
func (e *OpenAIEmbedder) EmbedWithOptions(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	if req.Text == "" {
		return nil, ErrQueryEmpty.WithOperation("embed_with_options")
	}
	
	// Use request model if specified, otherwise fall back to config
	model := req.Model
	if model == "" {
		model = e.config.Model
	}
	
	start := time.Now()
	cacheKey := e.generateCacheKey(req.Text, model)
	
	// Check cache
	if e.cache != nil {
		if entry, err := e.cache.Get(ctx, cacheKey); err == nil {
			response := &EmbeddingResponse{
				Vector:    entry.Vector,
				Model:     model,
				Metadata:  req.Metadata,
				RequestID: req.BatchID,
				Cached:    true,
			}
			
			if e.metrics != nil {
				e.metrics.RecordCacheHit(ctx, cacheKey)
				e.metrics.RecordEmbedding(ctx, req.Text, time.Since(start).Milliseconds(), true)
			}
			
			return response, nil
		}
	}
	
	// Create temporary config with specified model
	tempConfig := *e.config
	tempConfig.Model = model
	
	// Generate embedding
	response, err := e.embedWithCustomConfig(ctx, []string{req.Text}, &tempConfig)
	if err != nil {
		return nil, err
	}
	
	if len(response.Data) == 0 {
		return nil, NewRAGErrorWithOp("embed_with_options", "no embedding data returned", ErrorTypeExternal)
	}
	
	embedding := response.Data[0]
	vectorData := make(vector.Vector, len(embedding.Embedding))
	for i, val := range embedding.Embedding {
		vectorData[i] = val
	}
	
	// Cache with custom metadata
	if e.cache != nil {
		cacheMetadata := make(map[string]string)
		for k, v := range req.Metadata {
			cacheMetadata[k] = v
		}
		cacheMetadata["model"] = model
		
		if err := e.cache.Set(ctx, cacheKey, vectorData, cacheMetadata); err != nil {
			// Log but don't fail
		}
	}
	
	result := &EmbeddingResponse{
		Vector:    vectorData,
		Model:     model,
		Usage: EmbeddingUsage{
			PromptTokens: response.Usage.PromptTokens,
			TotalTokens:  response.Usage.TotalTokens,
		},
		Metadata:  req.Metadata,
		RequestID: req.BatchID,
		Cached:    false,
	}
	
	if e.metrics != nil {
		e.metrics.RecordEmbedding(ctx, req.Text, time.Since(start).Milliseconds(), false)
	}
	
	return result, nil
}

// GetModel returns the current embedding model name
func (e *OpenAIEmbedder) GetModel() string {
	return e.config.Model
}

// GetDimension returns the embedding vector dimension
func (e *OpenAIEmbedder) GetDimension() int {
	// OpenAI text-embedding-3-small has 1536 dimensions
	// This should ideally be fetched from the API or configured
	switch e.config.Model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-ada-002":
		return 1536
	default:
		return 1536 // default assumption
	}
}

// Close releases any resources held by the embedder
func (e *OpenAIEmbedder) Close() error {
	// Close rate limiter channel safely
	if e.rateLimiter != nil {
		select {
		case <-e.rateLimiter:
		default:
		}
		// Don't close the channel as it might cause panic
		// Just let it be garbage collected
		e.rateLimiter = nil
	}
	
	// Close cache if it has a Close method
	if e.cache != nil {
		return e.cache.Close()
	}
	
	return nil
}

// Private helper methods

func (e *OpenAIEmbedder) processBatch(ctx context.Context, texts []string) ([]*EmbeddingResponse, error) {
	// Check cache for each text first
	results := make([]*EmbeddingResponse, len(texts))
	uncachedIndices := make([]int, 0)
	uncachedTexts := make([]string, 0)
	
	for i, text := range texts {
		cacheKey := e.generateCacheKey(text, e.config.Model)
		if e.cache != nil {
			if entry, err := e.cache.Get(ctx, cacheKey); err == nil {
				results[i] = &EmbeddingResponse{
					Vector:    entry.Vector,
					Model:     e.config.Model,
					Metadata:  entry.Metadata,
					Cached:    true,
				}
				if e.metrics != nil {
					e.metrics.RecordCacheHit(ctx, cacheKey)
				}
				continue
			}
			if e.metrics != nil {
				e.metrics.RecordCacheMiss(ctx, cacheKey)
			}
		}
		
		uncachedIndices = append(uncachedIndices, i)
		uncachedTexts = append(uncachedTexts, text)
	}
	
	// Generate embeddings for uncached texts
	if len(uncachedTexts) > 0 {
		response, err := e.embedWithRetry(ctx, uncachedTexts)
		if err != nil {
			return nil, err
		}
		
		for i, embedding := range response.Data {
			originalIndex := uncachedIndices[i]
			text := uncachedTexts[i]
			
			vectorData := make(vector.Vector, len(embedding.Embedding))
			for j, val := range embedding.Embedding {
				vectorData[j] = val
			}
			
			results[originalIndex] = &EmbeddingResponse{
				Vector: vectorData,
				Model:  e.config.Model,
				Usage: EmbeddingUsage{
					PromptTokens: response.Usage.PromptTokens / len(uncachedTexts), // approximate
					TotalTokens:  response.Usage.TotalTokens / len(uncachedTexts),
				},
				Cached: false,
			}
			
			// Cache the result
			if e.cache != nil {
				cacheKey := e.generateCacheKey(text, e.config.Model)
				metadata := map[string]string{
					"model": e.config.Model,
					"text_hash": fmt.Sprintf("%x", md5.Sum([]byte(text))),
				}
				e.cache.Set(ctx, cacheKey, vectorData, metadata)
			}
		}
	}
	
	return results, nil
}

func (e *OpenAIEmbedder) embedWithRetry(ctx context.Context, texts []string) (*openai.EmbeddingResponse, error) {
	return e.embedWithCustomConfig(ctx, texts, e.config)
}

func (e *OpenAIEmbedder) embedWithCustomConfig(ctx context.Context, texts []string, config *EmbeddingConfig) (*openai.EmbeddingResponse, error) {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Rate limiting
		select {
		case <-e.rateLimiter:
			// Rate limit token acquired
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		
		// Create request context with timeout
		reqCtx, cancel := context.WithTimeout(ctx, config.Timeout)
		
		req := openai.EmbeddingRequest{
			Input: texts,
			Model: openai.EmbeddingModel(config.Model),
		}
		
		response, err := e.client.CreateEmbeddings(reqCtx, req)
		cancel()
		
		if err == nil {
			return &response, nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !e.isRetryableError(err) {
			break
		}
		
		// Wait before retry with exponential backoff
		if attempt < config.MaxRetries {
			delay := time.Duration(1<<uint(attempt)) * time.Second
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	
	return nil, ExternalServiceError("OpenAI", "embedding", lastErr)
}

func (e *OpenAIEmbedder) isRetryableError(err error) bool {
	// Check for specific OpenAI error types that are retryable
	errStr := strings.ToLower(err.Error())
	
	retryableErrors := []string{
		"rate limit",
		"timeout",
		"service unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}
	
	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}
	
	return false
}

func (e *OpenAIEmbedder) generateCacheKey(text, model string) string {
	hash := md5.Sum([]byte(fmt.Sprintf("%s:%s", model, text)))
	return fmt.Sprintf("embed:%x", hash)
}

func (e *OpenAIEmbedder) refillRateLimiter() {
	rateLimit := e.config.RateLimit
	if rateLimit <= 0 {
		rateLimit = 60 // Default rate limit
	}
	
	ticker := time.NewTicker(time.Minute / time.Duration(rateLimit))
	defer ticker.Stop()
	
	for range ticker.C {
		select {
		case e.rateLimiter <- struct{}{}:
			// Token added
		default:
			// Channel full, skip
		}
	}
}