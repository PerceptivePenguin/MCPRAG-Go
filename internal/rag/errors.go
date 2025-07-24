package rag

import (
	"fmt"
)

// Common RAG-related errors
var (
	// Document errors
	ErrDocumentNotFound     = NewRAGError("document not found", ErrorTypeNotFound)
	ErrDocumentExists       = NewRAGError("document already exists", ErrorTypeConflict)
	ErrDocumentTooLarge     = NewRAGError("document size exceeds limit", ErrorTypeValidation)
	ErrDocumentInvalidFormat = NewRAGError("invalid document format", ErrorTypeValidation)
	ErrDocumentEmpty        = NewRAGError("document content is empty", ErrorTypeValidation)
	
	// Embedding errors
	ErrEmbeddingFailed      = NewRAGError("embedding generation failed", ErrorTypeExternal)
	ErrEmbeddingAPIKey      = NewRAGError("invalid or missing API key", ErrorTypeAuth)
	ErrEmbeddingQuotaExceeded = NewRAGError("embedding API quota exceeded", ErrorTypeRateLimit)
	ErrEmbeddingModelNotFound = NewRAGError("embedding model not found", ErrorTypeNotFound)
	ErrEmbeddingDimensionMismatch = NewRAGError("embedding dimension mismatch", ErrorTypeValidation)
	
	// Query errors
	ErrQueryEmpty           = NewRAGError("query text is empty", ErrorTypeValidation)
	ErrQueryTooLong         = NewRAGError("query text is too long", ErrorTypeValidation)
	ErrInvalidTopK          = NewRAGError("topK must be positive", ErrorTypeValidation)
	ErrInvalidThreshold     = NewRAGError("threshold must be between 0 and 1", ErrorTypeValidation)
	ErrInvalidStrategy      = NewRAGError("invalid search strategy", ErrorTypeValidation)
	
	// Cache errors
	ErrCacheKeyNotFound     = NewRAGError("cache key not found", ErrorTypeNotFound)
	ErrCacheFull            = NewRAGError("cache is at maximum capacity", ErrorTypeCapacity)
	ErrCacheCorrupted       = NewRAGError("cache data is corrupted", ErrorTypeInternal)
	ErrCacheSerializationFailed = NewRAGError("cache serialization failed", ErrorTypeInternal)
	ErrCacheClosed          = NewRAGError("cache is closed", ErrorTypeInternal)
	ErrInvalidCacheSize     = NewRAGError("invalid cache size", ErrorTypeValidation)
	
	// Chunking errors
	ErrChunkingFailed       = NewRAGError("document chunking failed", ErrorTypeInternal)
	ErrInvalidChunkSize     = NewRAGError("invalid chunk size", ErrorTypeValidation)
	ErrInvalidOverlap       = NewRAGError("invalid overlap size", ErrorTypeValidation)
	ErrChunkTooLarge        = NewRAGError("chunk size exceeds limit", ErrorTypeValidation)
	
	// Context errors
	ErrContextTooLong       = NewRAGError("context exceeds maximum length", ErrorTypeValidation)
	ErrContextTemplateFailed = NewRAGError("context template rendering failed", ErrorTypeInternal)
	ErrInvalidTemplate      = NewRAGError("invalid context template", ErrorTypeValidation)
	
	// Vector store errors
	ErrVectorStoreNotReady  = NewRAGError("vector store not ready", ErrorTypeInternal)
	ErrVectorStoreFull      = NewRAGError("vector store at capacity", ErrorTypeCapacity)
	ErrVectorSearchFailed   = NewRAGError("vector search failed", ErrorTypeInternal)
	
	// Configuration errors
	ErrInvalidConfig        = NewRAGError("invalid configuration", ErrorTypeValidation)
	ErrMissingConfig        = NewRAGError("missing required configuration", ErrorTypeValidation)
	ErrConfigLoadFailed     = NewRAGError("configuration loading failed", ErrorTypeInternal)
	
	// Network errors
	ErrNetworkTimeout       = NewRAGError("network request timeout", ErrorTypeTimeout)
	ErrNetworkUnavailable   = NewRAGError("network service unavailable", ErrorTypeExternal)
	ErrRateLimitExceeded    = NewRAGError("rate limit exceeded", ErrorTypeRateLimit)
	
	// Internal errors
	ErrInternalError        = NewRAGError("internal server error", ErrorTypeInternal)
	ErrNotImplemented       = NewRAGError("feature not implemented", ErrorTypeNotImplemented)
	ErrResourceExhausted    = NewRAGError("system resources exhausted", ErrorTypeCapacity)
)

// ErrorType represents the category of error
type ErrorType string

const (
	ErrorTypeValidation     ErrorType = "validation"
	ErrorTypeNotFound       ErrorType = "not_found"
	ErrorTypeConflict       ErrorType = "conflict"
	ErrorTypeAuth           ErrorType = "authentication"
	ErrorTypeRateLimit      ErrorType = "rate_limit"
	ErrorTypeTimeout        ErrorType = "timeout"
	ErrorTypeExternal       ErrorType = "external"
	ErrorTypeInternal       ErrorType = "internal"
	ErrorTypeCapacity       ErrorType = "capacity"
	ErrorTypeNotImplemented ErrorType = "not_implemented"
)

// RAGError represents a structured error in the RAG system
type RAGError struct {
	Type       ErrorType         `json:"type"`
	Message    string            `json:"message"`
	Operation  string            `json:"operation,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
	Cause      error             `json:"-"`
}

// Error implements the error interface
func (e *RAGError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("rag %s: %s", e.Operation, e.Message)
	}
	return fmt.Sprintf("rag: %s", e.Message)
}

// Unwrap returns the underlying cause error
func (e *RAGError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error
func (e *RAGError) Is(target error) bool {
	if ragErr, ok := target.(*RAGError); ok {
		return e.Type == ragErr.Type && e.Message == ragErr.Message
	}
	return false
}

// NewRAGError creates a new RAG error with type
func NewRAGError(message string, errorType ErrorType) *RAGError {
	return &RAGError{
		Type:    errorType,
		Message: message,
	}
}

// NewRAGErrorWithOp creates a new RAG error with operation context
func NewRAGErrorWithOp(operation string, message string, errorType ErrorType) *RAGError {
	return &RAGError{
		Type:      errorType,
		Message:   message,
		Operation: operation,
	}
}

// NewRAGErrorWithCause creates a new RAG error wrapping another error
func NewRAGErrorWithCause(message string, errorType ErrorType, cause error) *RAGError {
	return &RAGError{
		Type:    errorType,
		Message: message,
		Cause:   cause,
	}
}

// NewRAGErrorWithDetails creates a new RAG error with additional details
func NewRAGErrorWithDetails(message string, errorType ErrorType, details map[string]string) *RAGError {
	return &RAGError{
		Type:    errorType,
		Message: message,
		Details: details,
	}
}

// WithOperation adds operation context to an error
func (e *RAGError) WithOperation(operation string) *RAGError {
	return &RAGError{
		Type:      e.Type,
		Message:   e.Message,
		Operation: operation,
		Details:   e.Details,
		Cause:     e.Cause,
	}
}

// WithDetails adds details to an error
func (e *RAGError) WithDetails(details map[string]string) *RAGError {
	newDetails := make(map[string]string)
	for k, v := range e.Details {
		newDetails[k] = v
	}
	for k, v := range details {
		newDetails[k] = v
	}
	
	return &RAGError{
		Type:      e.Type,
		Message:   e.Message,
		Operation: e.Operation,
		Details:   newDetails,
		Cause:     e.Cause,
	}
}

// WithCause adds a cause error
func (e *RAGError) WithCause(cause error) *RAGError {
	return &RAGError{
		Type:      e.Type,
		Message:   e.Message,
		Operation: e.Operation,
		Details:   e.Details,
		Cause:     cause,
	}
}

// IsRetryable returns true if the error is retryable
func (e *RAGError) IsRetryable() bool {
	switch e.Type {
	case ErrorTypeTimeout, ErrorTypeExternal, ErrorTypeRateLimit:
		return true
	case ErrorTypeInternal:
		// Some internal errors might be retryable (e.g., temporary resource exhaustion)
		return e.Message == ErrResourceExhausted.Message
	default:
		return false
	}
}

// GetRetryDelay returns the suggested retry delay in seconds
func (e *RAGError) GetRetryDelay() int {
	switch e.Type {
	case ErrorTypeRateLimit:
		return 60 // 1 minute for rate limit
	case ErrorTypeTimeout:
		return 5 // 5 seconds for timeout
	case ErrorTypeExternal:
		return 10 // 10 seconds for external service issues
	default:
		return 30 // default retry delay
	}
}

// HTTPStatusCode returns the appropriate HTTP status code for the error
func (e *RAGError) HTTPStatusCode() int {
	switch e.Type {
	case ErrorTypeValidation:
		return 400 // Bad Request
	case ErrorTypeAuth:
		return 401 // Unauthorized
	case ErrorTypeNotFound:
		return 404 // Not Found
	case ErrorTypeConflict:
		return 409 // Conflict
	case ErrorTypeCapacity:
		return 413 // Payload Too Large
	case ErrorTypeRateLimit:
		return 429 // Too Many Requests
	case ErrorTypeNotImplemented:
		return 501 // Not Implemented
	case ErrorTypeExternal:
		return 502 // Bad Gateway
	case ErrorTypeTimeout:
		return 504 // Gateway Timeout
	default:
		return 500 // Internal Server Error
	}
}

// ValidationError creates a validation error with field information
func ValidationError(field string, message string) *RAGError {
	return NewRAGErrorWithDetails(
		fmt.Sprintf("validation failed for field '%s': %s", field, message),
		ErrorTypeValidation,
		map[string]string{"field": field},
	)
}

// NotFoundError creates a not found error with resource information
func NotFoundError(resource string, id string) *RAGError {
	return NewRAGErrorWithDetails(
		fmt.Sprintf("%s not found", resource),
		ErrorTypeNotFound,
		map[string]string{"resource": resource, "id": id},
	)
}

// ExternalServiceError creates an external service error
func ExternalServiceError(service string, operation string, cause error) *RAGError {
	return &RAGError{
		Type:      ErrorTypeExternal,
		Message:   fmt.Sprintf("%s service error during %s", service, operation),
		Operation: operation,
		Details:   map[string]string{"service": service},
		Cause:     cause,
	}
}

// ConfigurationError creates a configuration error
func ConfigurationError(key string, message string) *RAGError {
	return NewRAGErrorWithDetails(
		fmt.Sprintf("configuration error for '%s': %s", key, message),
		ErrorTypeValidation,
		map[string]string{"config_key": key},
	)
}

// CacheError creates a cache-related error
func CacheError(operation string, cause error) *RAGError {
	return &RAGError{
		Type:      ErrorTypeInternal,
		Message:   fmt.Sprintf("cache operation failed: %s", operation),
		Operation: operation,
		Cause:     cause,
	}
}