package rag

import (
	"errors"
	"testing"
)

func TestRAGError_Error(t *testing.T) {
	tests := []struct {
		name     string
		ragError *RAGError
		expected string
	}{
		{
			name: "error with operation",
			ragError: &RAGError{
				Type:      ErrorTypeValidation,
				Message:   "invalid input",
				Operation: "validate_input",
			},
			expected: "rag validate_input: invalid input",
		},
		{
			name: "error without operation",
			ragError: &RAGError{
				Type:    ErrorTypeNotFound,
				Message: "document not found",
			},
			expected: "rag: document not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ragError.Error()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestRAGError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	ragError := &RAGError{
		Type:    ErrorTypeInternal,
		Message: "wrapped error",
		Cause:   originalErr,
	}

	unwrapped := ragError.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("expected original error, got %v", unwrapped)
	}

	// Test with no cause
	ragErrorNoCause := &RAGError{
		Type:    ErrorTypeInternal,
		Message: "no cause",
	}

	unwrappedNil := ragErrorNoCause.Unwrap()
	if unwrappedNil != nil {
		t.Errorf("expected nil, got %v", unwrappedNil)
	}
}

func TestRAGError_Is(t *testing.T) {
	err1 := &RAGError{
		Type:    ErrorTypeValidation,
		Message: "validation failed",
	}

	err2 := &RAGError{
		Type:    ErrorTypeValidation,
		Message: "validation failed",
	}

	err3 := &RAGError{
		Type:    ErrorTypeNotFound,
		Message: "not found",
	}

	// Same type and message should match
	if !err1.Is(err2) {
		t.Error("expected err1.Is(err2) to be true")
	}

	// Different type or message should not match
	if err1.Is(err3) {
		t.Error("expected err1.Is(err3) to be false")
	}

	// Non-RAGError should not match
	otherErr := errors.New("other error")
	if err1.Is(otherErr) {
		t.Error("expected err1.Is(otherErr) to be false")
	}
}

func TestRAGError_WithDetails(t *testing.T) {
	originalErr := &RAGError{
		Type:    ErrorTypeValidation,
		Message: "validation failed",
		Details: map[string]string{
			"field": "name",
		},
	}

	newDetails := map[string]string{
		"value": "invalid",
		"field": "updated_name", // This should override the original
	}

	updatedErr := originalErr.WithDetails(newDetails)

	if updatedErr.Details["field"] != "updated_name" {
		t.Errorf("expected field to be updated to 'updated_name', got '%s'", updatedErr.Details["field"])
	}

	if updatedErr.Details["value"] != "invalid" {
		t.Errorf("expected value to be 'invalid', got '%s'", updatedErr.Details["value"])
	}

	// Original error should not be modified
	if originalErr.Details["field"] != "name" {
		t.Error("original error should not be modified")
	}
}

func TestRAGError_WithCause(t *testing.T) {
	originalErr := &RAGError{
		Type:    ErrorTypeInternal,
		Message: "internal error",
	}

	cause := errors.New("root cause")
	updatedErr := originalErr.WithCause(cause)

	if updatedErr.Cause != cause {
		t.Error("expected cause to be set")
	}

	// Original error should not be modified
	if originalErr.Cause != nil {
		t.Error("original error should not be modified")
	}
}

func TestRAGError_IsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		errorType ErrorType
		message   string
		retryable bool
	}{
		{
			name:      "timeout error is retryable",
			errorType: ErrorTypeTimeout,
			message:   "timeout occurred",
			retryable: true,
		},
		{
			name:      "external error is retryable",
			errorType: ErrorTypeExternal,
			message:   "external service failed",
			retryable: true,
		},
		{
			name:      "rate limit error is retryable",
			errorType: ErrorTypeRateLimit,
			message:   "rate limit exceeded",
			retryable: true,
		},
		{
			name:      "resource exhausted internal error is retryable",
			errorType: ErrorTypeInternal,
			message:   ErrResourceExhausted.Message,
			retryable: true,
		},
		{
			name:      "other internal error is not retryable",
			errorType: ErrorTypeInternal,
			message:   "other internal error",
			retryable: false,
		},
		{
			name:      "validation error is not retryable",
			errorType: ErrorTypeValidation,
			message:   "validation failed",
			retryable: false,
		},
		{
			name:      "not found error is not retryable",
			errorType: ErrorTypeNotFound,
			message:   "not found",
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &RAGError{
				Type:    tt.errorType,
				Message: tt.message,
			}

			isRetryable := err.IsRetryable()
			if isRetryable != tt.retryable {
				t.Errorf("expected IsRetryable()=%v, got %v", tt.retryable, isRetryable)
			}
		})
	}
}

func TestRAGError_GetRetryDelay(t *testing.T) {
	tests := []struct {
		name          string
		errorType     ErrorType
		expectedDelay int
	}{
		{
			name:          "rate limit error has 60s delay",
			errorType:     ErrorTypeRateLimit,
			expectedDelay: 60,
		},
		{
			name:          "timeout error has 5s delay",
			errorType:     ErrorTypeTimeout,
			expectedDelay: 5,
		},
		{
			name:          "external error has 10s delay",
			errorType:     ErrorTypeExternal,
			expectedDelay: 10,
		},
		{
			name:          "other error has 30s default delay",
			errorType:     ErrorTypeValidation,
			expectedDelay: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &RAGError{
				Type:    tt.errorType,
				Message: "test error",
			}

			delay := err.GetRetryDelay()
			if delay != tt.expectedDelay {
				t.Errorf("expected GetRetryDelay()=%d, got %d", tt.expectedDelay, delay)
			}
		})
	}
}

func TestRAGError_HTTPStatusCode(t *testing.T) {
	tests := []struct {
		name               string
		errorType          ErrorType
		expectedStatusCode int
	}{
		{
			name:               "validation error returns 400",
			errorType:          ErrorTypeValidation,
			expectedStatusCode: 400,
		},
		{
			name:               "auth error returns 401",
			errorType:          ErrorTypeAuth,
			expectedStatusCode: 401,
		},
		{
			name:               "not found error returns 404",
			errorType:          ErrorTypeNotFound,
			expectedStatusCode: 404,
		},
		{
			name:               "conflict error returns 409",
			errorType:          ErrorTypeConflict,
			expectedStatusCode: 409,
		},
		{
			name:               "capacity error returns 413",
			errorType:          ErrorTypeCapacity,
			expectedStatusCode: 413,
		},
		{
			name:               "rate limit error returns 429",
			errorType:          ErrorTypeRateLimit,
			expectedStatusCode: 429,
		},
		{
			name:               "not implemented error returns 501",
			errorType:          ErrorTypeNotImplemented,
			expectedStatusCode: 501,
		},
		{
			name:               "external error returns 502",
			errorType:          ErrorTypeExternal,
			expectedStatusCode: 502,
		},
		{
			name:               "timeout error returns 504",
			errorType:          ErrorTypeTimeout,
			expectedStatusCode: 504,
		},
		{
			name:               "internal error returns 500",
			errorType:          ErrorTypeInternal,
			expectedStatusCode: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &RAGError{
				Type:    tt.errorType,
				Message: "test error",
			}

			statusCode := err.HTTPStatusCode()
			if statusCode != tt.expectedStatusCode {
				t.Errorf("expected HTTPStatusCode()=%d, got %d", tt.expectedStatusCode, statusCode)
			}
		})
	}
}

func TestNotFoundError(t *testing.T) {
	err := NotFoundError("document", "doc123")

	if err.Type != ErrorTypeNotFound {
		t.Errorf("expected error type %s, got %s", ErrorTypeNotFound, err.Type)
	}

	if err.Message != "document not found" {
		t.Errorf("expected message 'document not found', got '%s'", err.Message)
	}

	if err.Details["resource"] != "document" {
		t.Errorf("expected resource 'document', got '%s'", err.Details["resource"])
	}

	if err.Details["id"] != "doc123" {
		t.Errorf("expected id 'doc123', got '%s'", err.Details["id"])
	}
}

func TestExternalServiceError(t *testing.T) {
	cause := errors.New("connection failed")
	err := ExternalServiceError("OpenAI", "embed", cause)

	if err.Type != ErrorTypeExternal {
		t.Errorf("expected error type %s, got %s", ErrorTypeExternal, err.Type)
	}

	if err.Message != "OpenAI service error during embed" {
		t.Errorf("expected specific message, got '%s'", err.Message)
	}

	if err.Operation != "embed" {
		t.Errorf("expected operation 'embed', got '%s'", err.Operation)
	}

	if err.Details["service"] != "OpenAI" {
		t.Errorf("expected service 'OpenAI', got '%s'", err.Details["service"])
	}

	if err.Cause != cause {
		t.Error("expected cause to be set")
	}
}

func TestConfigurationError(t *testing.T) {
	err := ConfigurationError("api_key", "missing required value")

	if err.Type != ErrorTypeValidation {
		t.Errorf("expected error type %s, got %s", ErrorTypeValidation, err.Type)
	}

	expectedMessage := "configuration error for 'api_key': missing required value"
	if err.Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Details["config_key"] != "api_key" {
		t.Errorf("expected config_key 'api_key', got '%s'", err.Details["config_key"])
	}
}

func TestCacheError(t *testing.T) {
	cause := errors.New("cache full")
	err := CacheError("set", cause)

	if err.Type != ErrorTypeInternal {
		t.Errorf("expected error type %s, got %s", ErrorTypeInternal, err.Type)
	}

	expectedMessage := "cache operation failed: set"
	if err.Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Operation != "set" {
		t.Errorf("expected operation 'set', got '%s'", err.Operation)
	}

	if err.Cause != cause {
		t.Error("expected cause to be set")
	}
}

func TestErrorConstructors(t *testing.T) {
	// Test NewRAGError
	err1 := NewRAGError("test message", ErrorTypeValidation)
	if err1.Type != ErrorTypeValidation || err1.Message != "test message" {
		t.Error("NewRAGError failed")
	}

	// Test NewRAGErrorWithOp
	err2 := NewRAGErrorWithOp("test_op", "test message", ErrorTypeInternal)
	if err2.Operation != "test_op" {
		t.Error("NewRAGErrorWithOp failed to set operation")
	}

	// Test NewRAGErrorWithCause
	cause := errors.New("cause")
	err3 := NewRAGErrorWithCause("test message", ErrorTypeExternal, cause)
	if err3.Cause != cause {
		t.Error("NewRAGErrorWithCause failed to set cause")
	}

	// Test NewRAGErrorWithDetails
	details := map[string]string{"key": "value"}
	err4 := NewRAGErrorWithDetails("test message", ErrorTypeValidation, details)
	if err4.Details["key"] != "value" {
		t.Error("NewRAGErrorWithDetails failed to set details")
	}
}

func TestWithOperation(t *testing.T) {
	originalErr := NewRAGError("test error", ErrorTypeValidation)
	updatedErr := originalErr.WithOperation("new_operation")

	if updatedErr.Operation != "new_operation" {
		t.Error("WithOperation failed to set operation")
	}

	// Original should not be modified
	if originalErr.Operation != "" {
		t.Error("WithOperation modified original error")
	}
}

func TestValidationError(t *testing.T) {
	err := ValidationError("email", "invalid format")

	expectedMessage := "validation failed for field 'email': invalid format"
	if err.Message != expectedMessage {
		t.Errorf("expected message '%s', got '%s'", expectedMessage, err.Message)
	}

	if err.Type != ErrorTypeValidation {
		t.Errorf("expected type %s, got %s", ErrorTypeValidation, err.Type)
	}

	if err.Details["field"] != "email" {
		t.Errorf("expected field 'email', got '%s'", err.Details["field"])
	}
}

func TestPredefinedErrors(t *testing.T) {
	// Test that predefined errors have correct types
	tests := []struct {
		name      string
		err       *RAGError
		errorType ErrorType
	}{
		{"ErrDocumentNotFound", ErrDocumentNotFound, ErrorTypeNotFound},
		{"ErrDocumentExists", ErrDocumentExists, ErrorTypeConflict},
		{"ErrDocumentTooLarge", ErrDocumentTooLarge, ErrorTypeValidation},
		{"ErrEmbeddingFailed", ErrEmbeddingFailed, ErrorTypeExternal},
		{"ErrEmbeddingAPIKey", ErrEmbeddingAPIKey, ErrorTypeAuth},
		{"ErrQueryEmpty", ErrQueryEmpty, ErrorTypeValidation},
		{"ErrInvalidTopK", ErrInvalidTopK, ErrorTypeValidation},
		{"ErrCacheKeyNotFound", ErrCacheKeyNotFound, ErrorTypeNotFound},
		{"ErrInvalidConfig", ErrInvalidConfig, ErrorTypeValidation},
		{"ErrNotImplemented", ErrNotImplemented, ErrorTypeNotImplemented},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Type != tt.errorType {
				t.Errorf("expected %s to have type %s, got %s", tt.name, tt.errorType, tt.err.Type)
			}
		})
	}
}