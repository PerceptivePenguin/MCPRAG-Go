package errors

import (
	"fmt"
	"sync"
	"time"
)

// ErrorStats 错误统计信息
type ErrorStats struct {
	mu sync.RWMutex
	
	// 基础统计
	TotalErrors      int64            `json:"total_errors"`
	RetryableErrors  int64            `json:"retryable_errors"`
	
	// 按类型统计
	ErrorsByType     map[ErrorType]int64 `json:"errors_by_type"`
	ErrorsByCode     map[string]int64    `json:"errors_by_code"`
	ErrorsByComponent map[string]int64   `json:"errors_by_component"`
	ErrorsByOperation map[string]int64   `json:"errors_by_operation"`
	
	// 时间统计
	FirstErrorTime   time.Time         `json:"first_error_time"`
	LastErrorTime    time.Time         `json:"last_error_time"`
	ErrorsPerMinute  map[string]int64  `json:"errors_per_minute"`
	
	// 重试统计
	TotalRetries     int64             `json:"total_retries"`
	RetriesByError   map[string]int64  `json:"retries_by_error"`
}

// NewErrorStats 创建新的错误统计
func NewErrorStats() *ErrorStats {
	return &ErrorStats{
		ErrorsByType:      make(map[ErrorType]int64),
		ErrorsByCode:      make(map[string]int64),
		ErrorsByComponent: make(map[string]int64),
		ErrorsByOperation: make(map[string]int64),
		ErrorsPerMinute:   make(map[string]int64),
		RetriesByError:    make(map[string]int64),
	}
}

// RecordError 记录错误统计
func (s *ErrorStats) RecordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	s.TotalErrors++
	
	if s.FirstErrorTime.IsZero() {
		s.FirstErrorTime = now
	}
	s.LastErrorTime = now
	
	// 记录按分钟的错误统计
	minuteKey := now.Format("2006-01-02T15:04")
	s.ErrorsPerMinute[minuteKey]++
	
	// 检查是否为可重试错误
	if IsTemporaryError(err) {
		s.RetryableErrors++
	}
	
	// 如果是BaseError，记录详细统计
	var baseErr *BaseError
	if As(err, &baseErr) {
		s.ErrorsByType[baseErr.Type]++
		
		if baseErr.Code != "" {
			s.ErrorsByCode[baseErr.Code]++
		}
		
		if baseErr.Component != "" {
			s.ErrorsByComponent[baseErr.Component]++
		}
		
		if baseErr.Operation != "" {
			s.ErrorsByOperation[baseErr.Operation]++
		}
	} else {
		// 对于非BaseError，记录为内部错误
		s.ErrorsByType[ErrorTypeInternal]++
		errorType := fmt.Sprintf("%T", err)
		s.ErrorsByCode[errorType]++
	}
}

// RecordRetry 记录重试统计
func (s *ErrorStats) RecordRetry(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalRetries++
	
	var baseErr *BaseError
	if As(err, &baseErr) {
		key := fmt.Sprintf("%s:%s", baseErr.Type, baseErr.Code)
		s.RetriesByError[key]++
	} else {
		errorType := fmt.Sprintf("%T", err)
		s.RetriesByError[errorType]++
	}
}

// GetStats 获取统计信息副本
func (s *ErrorStats) GetStats() ErrorStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 创建副本
	errorsByType := make(map[ErrorType]int64)
	for k, v := range s.ErrorsByType {
		errorsByType[k] = v
	}
	
	errorsByCode := make(map[string]int64)
	for k, v := range s.ErrorsByCode {
		errorsByCode[k] = v
	}
	
	errorsByComponent := make(map[string]int64)
	for k, v := range s.ErrorsByComponent {
		errorsByComponent[k] = v
	}
	
	errorsByOperation := make(map[string]int64)
	for k, v := range s.ErrorsByOperation {
		errorsByOperation[k] = v
	}
	
	errorsPerMinute := make(map[string]int64)
	for k, v := range s.ErrorsPerMinute {
		errorsPerMinute[k] = v
	}
	
	retriesByError := make(map[string]int64)
	for k, v := range s.RetriesByError {
		retriesByError[k] = v
	}
	
	return ErrorStats{
		TotalErrors:       s.TotalErrors,
		RetryableErrors:   s.RetryableErrors,
		ErrorsByType:      errorsByType,
		ErrorsByCode:      errorsByCode,
		ErrorsByComponent: errorsByComponent,
		ErrorsByOperation: errorsByOperation,
		FirstErrorTime:    s.FirstErrorTime,
		LastErrorTime:     s.LastErrorTime,
		ErrorsPerMinute:   errorsPerMinute,
		TotalRetries:      s.TotalRetries,
		RetriesByError:    retriesByError,
	}
}

// Reset 重置统计信息
func (s *ErrorStats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalErrors = 0
	s.RetryableErrors = 0
	s.ErrorsByType = make(map[ErrorType]int64)
	s.ErrorsByCode = make(map[string]int64)
	s.ErrorsByComponent = make(map[string]int64)
	s.ErrorsByOperation = make(map[string]int64)
	s.FirstErrorTime = time.Time{}
	s.LastErrorTime = time.Time{}
	s.ErrorsPerMinute = make(map[string]int64)
	s.TotalRetries = 0
	s.RetriesByError = make(map[string]int64)
}

// GetErrorRate 获取错误率（errors per minute）
func (s *ErrorStats) GetErrorRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.FirstErrorTime.IsZero() || s.LastErrorTime.IsZero() {
		return 0
	}
	
	duration := s.LastErrorTime.Sub(s.FirstErrorTime).Minutes()
	if duration <= 0 {
		return float64(s.TotalErrors)
	}
	
	return float64(s.TotalErrors) / duration
}

// GetRetryRate 获取重试率
func (s *ErrorStats) GetRetryRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.TotalErrors == 0 {
		return 0
	}
	
	return float64(s.TotalRetries) / float64(s.TotalErrors)
}

// GetTopErrorTypes 获取错误类型排行
func (s *ErrorStats) GetTopErrorTypes(limit int) []ErrorTypeStat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var stats []ErrorTypeStat
	for errorType, count := range s.ErrorsByType {
		stats = append(stats, ErrorTypeStat{
			Type:  errorType,
			Count: count,
		})
	}
	
	// 简单排序（按count降序）
	for i := 0; i < len(stats)-1; i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].Count > stats[i].Count {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}
	
	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}
	
	return stats
}

// ErrorTypeStat 错误类型统计
type ErrorTypeStat struct {
	Type  ErrorType `json:"type"`
	Count int64     `json:"count"`
}

// String 返回统计信息的字符串表示
func (s *ErrorStats) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return fmt.Sprintf("ErrorStats: Total=%d, Retryable=%d, Types=%d, Codes=%d, Components=%d, Operations=%d, Retries=%d",
		s.TotalErrors, s.RetryableErrors, len(s.ErrorsByType), len(s.ErrorsByCode), 
		len(s.ErrorsByComponent), len(s.ErrorsByOperation), s.TotalRetries)
}