package errors

// 预定义的通用错误

var (
	// 验证错误
	ErrInvalidInput       = NewError(ErrorTypeValidation, "invalid input")
	ErrMissingRequired    = NewError(ErrorTypeValidation, "required field is missing")
	ErrInvalidFormat      = NewError(ErrorTypeValidation, "invalid format")
	ErrInvalidRange       = NewError(ErrorTypeValidation, "value out of range")
	ErrInvalidLength      = NewError(ErrorTypeValidation, "invalid length")
	
	// 未找到错误
	ErrNotFound           = NewError(ErrorTypeNotFound, "resource not found")
	ErrEndpointNotFound   = NewError(ErrorTypeNotFound, "endpoint not found")
	ErrMethodNotFound     = NewError(ErrorTypeNotFound, "method not found")
	
	// 冲突错误
	ErrAlreadyExists      = NewError(ErrorTypeConflict, "resource already exists")
	ErrConflict           = NewError(ErrorTypeConflict, "operation conflicts with current state")
	
	// 认证错误
	ErrUnauthorized       = NewError(ErrorTypeAuth, "unauthorized access")
	ErrInvalidCredentials = NewError(ErrorTypeAuth, "invalid credentials")
	ErrTokenExpired       = NewError(ErrorTypeAuth, "token expired")
	ErrInvalidToken       = NewError(ErrorTypeAuth, "invalid token")
	
	// 速率限制错误
	ErrRateLimited        = NewError(ErrorTypeRateLimit, "rate limit exceeded")
	ErrQuotaExceeded      = NewError(ErrorTypeRateLimit, "quota exceeded")
	
	// 超时错误
	ErrTimeout            = NewError(ErrorTypeTimeout, "operation timeout")
	ErrConnectionTimeout  = NewError(ErrorTypeTimeout, "connection timeout")
	ErrRequestTimeout     = NewError(ErrorTypeTimeout, "request timeout")
	
	// 网络错误
	ErrNetworkError       = NewError(ErrorTypeNetwork, "network error")
	ErrConnectionFailed   = NewError(ErrorTypeNetwork, "connection failed")
	ErrConnectionLost     = NewError(ErrorTypeNetwork, "connection lost")
	ErrDNSResolution      = NewError(ErrorTypeNetwork, "DNS resolution failed")
	
	// 外部服务错误
	ErrExternalService    = NewError(ErrorTypeExternal, "external service error")
	ErrServiceUnavailable = NewError(ErrorTypeExternal, "service unavailable")
	ErrBadGateway         = NewError(ErrorTypeExternal, "bad gateway")
	
	// 内部错误
	ErrInternalError      = NewError(ErrorTypeInternal, "internal server error")
	ErrUnexpectedError    = NewError(ErrorTypeInternal, "unexpected error")
	ErrProcessingFailed   = NewError(ErrorTypeInternal, "processing failed")
	
	// 容量错误
	ErrCapacityExceeded   = NewError(ErrorTypeCapacity, "capacity exceeded")
	ErrResourceExhausted  = NewError(ErrorTypeCapacity, "resource exhausted")
	ErrMemoryExhausted    = NewError(ErrorTypeCapacity, "memory exhausted")
	ErrDiskFull           = NewError(ErrorTypeCapacity, "disk full")
	
	// 配置错误
	ErrInvalidConfig      = NewError(ErrorTypeConfiguration, "invalid configuration")
	ErrMissingConfig      = NewError(ErrorTypeConfiguration, "missing configuration")
	ErrConfigLoadFailed   = NewError(ErrorTypeConfiguration, "configuration loading failed")
	
	// 未实现错误
	ErrNotImplemented     = NewError(ErrorTypeNotImplemented, "feature not implemented")
	ErrNotSupported       = NewError(ErrorTypeNotImplemented, "operation not supported")
)

// 业务逻辑错误创建函数

// ValidationError 创建验证错误
func ValidationError(field, message string) *BaseError {
	return NewError(ErrorTypeValidation, message).WithDetails(map[string]string{
		"field": field,
	})
}

// NotFoundError 创建未找到错误
func NotFoundError(resource, id string) *BaseError {
	return NewError(ErrorTypeNotFound, resource+" not found").WithDetails(map[string]string{
		"resource": resource,
		"id":       id,
	})
}

// ConflictError 创建冲突错误
func ConflictError(resource, reason string) *BaseError {
	return NewError(ErrorTypeConflict, reason).WithDetails(map[string]string{
		"resource": resource,
	})
}

// AuthError 创建认证错误
func AuthError(reason string) *BaseError {
	return NewError(ErrorTypeAuth, reason)
}

// RateLimitError 创建速率限制错误
func RateLimitError(limit string, window string) *BaseError {
	return NewError(ErrorTypeRateLimit, "rate limit exceeded").WithDetails(map[string]string{
		"limit":  limit,
		"window": window,
	})
}

// TimeoutError 创建超时错误
func TimeoutError(operation string, timeout string) *BaseError {
	return NewError(ErrorTypeTimeout, operation+" timeout").WithDetails(map[string]string{
		"operation": operation,
		"timeout":   timeout,
	})
}

// NetworkError 创建网络错误
func NetworkError(operation string, cause error) *BaseError {
	return NewErrorWithCause(ErrorTypeNetwork, "network error during "+operation, cause)
}

// ExternalServiceError 创建外部服务错误
func ExternalServiceError(service, operation string, cause error) *BaseError {
	return NewErrorWithCause(ErrorTypeExternal, service+" service error during "+operation, cause).WithDetails(map[string]string{
		"service":   service,
		"operation": operation,
	})
}

// InternalError 创建内部错误
func InternalError(component, operation string, cause error) *BaseError {
	return NewErrorWithCause(ErrorTypeInternal, "internal error in "+component, cause).WithComponent(component).WithOperation(operation)
}

// CapacityError 创建容量错误
func CapacityError(resource, limit string) *BaseError {
	return NewError(ErrorTypeCapacity, resource+" capacity exceeded").WithDetails(map[string]string{
		"resource": resource,
		"limit":    limit,
	})
}

// ConfigurationError 创建配置错误
func ConfigurationError(key, reason string) *BaseError {
	return NewError(ErrorTypeConfiguration, "configuration error").WithDetails(map[string]string{
		"key":    key,
		"reason": reason,
	})
}