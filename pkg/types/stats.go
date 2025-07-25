package types

import (
	"sync"
	"time"
)

// Stats 通用统计信息结构
type Stats struct {
	StartTime         time.Time `json:"start_time"`
	TotalRequests     int64     `json:"total_requests"`
	SuccessRequests   int64     `json:"success_requests"`
	FailedRequests    int64     `json:"failed_requests"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastRequestTime   time.Time `json:"last_request_time"`
	
	// 并发统计
	ConcurrentRequests    int32 `json:"concurrent_requests"`
	MaxConcurrentRequests int32 `json:"max_concurrent_requests"`
	
	// 请求统计
	RequestsByType        map[string]int64 `json:"requests_by_type"`
	RequestDurations      map[string]time.Duration `json:"request_durations"`
	
	// 错误统计
	ErrorsByType          map[string]int64 `json:"errors_by_type"`
	
	mu sync.RWMutex
}

// NewStats 创建新的统计信息
func NewStats() *Stats {
	return &Stats{
		StartTime:        time.Now(),
		RequestsByType:   make(map[string]int64),
		RequestDurations: make(map[string]time.Duration),
		ErrorsByType:     make(map[string]int64),
	}
}

// RecordRequest 记录请求统计
func (s *Stats) RecordRequest(requestType string, duration time.Duration, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalRequests++
	if success {
		s.SuccessRequests++
	} else {
		s.FailedRequests++
	}
	s.LastRequestTime = time.Now()
	
	// 计算平均响应时间
	if s.TotalRequests == 1 {
		s.AverageResponseTime = duration
	} else {
		s.AverageResponseTime = time.Duration(
			(int64(s.AverageResponseTime)*int64(s.TotalRequests-1) + int64(duration)) / int64(s.TotalRequests),
		)
	}
	
	// 记录按类型统计
	s.RequestsByType[requestType]++
	
	// 更新平均时长
	if existing, ok := s.RequestDurations[requestType]; ok {
		count := s.RequestsByType[requestType]
		s.RequestDurations[requestType] = time.Duration(
			(int64(existing)*int64(count-1) + int64(duration)) / int64(count),
		)
	} else {
		s.RequestDurations[requestType] = duration
	}
}

// RecordError 记录错误统计
func (s *Stats) RecordError(errorType string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.ErrorsByType[errorType]++
}

// IncrementConcurrentRequests 增加并发请求计数
func (s *Stats) IncrementConcurrentRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.ConcurrentRequests++
	if s.ConcurrentRequests > s.MaxConcurrentRequests {
		s.MaxConcurrentRequests = s.ConcurrentRequests
	}
}

// DecrementConcurrentRequests 减少并发请求计数
func (s *Stats) DecrementConcurrentRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.ConcurrentRequests > 0 {
		s.ConcurrentRequests--
	}
}

// GetStats 获取统计信息副本
func (s *Stats) GetStats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 创建副本
	requestsByType := make(map[string]int64)
	for k, v := range s.RequestsByType {
		requestsByType[k] = v
	}
	
	requestDurations := make(map[string]time.Duration)
	for k, v := range s.RequestDurations {
		requestDurations[k] = v
	}
	
	errorsByType := make(map[string]int64)
	for k, v := range s.ErrorsByType {
		errorsByType[k] = v
	}
	
	return Stats{
		StartTime:             s.StartTime,
		TotalRequests:         s.TotalRequests,
		SuccessRequests:       s.SuccessRequests,
		FailedRequests:        s.FailedRequests,
		AverageResponseTime:   s.AverageResponseTime,
		LastRequestTime:       s.LastRequestTime,
		ConcurrentRequests:    s.ConcurrentRequests,
		MaxConcurrentRequests: s.MaxConcurrentRequests,
		RequestsByType:        requestsByType,
		RequestDurations:      requestDurations,
		ErrorsByType:          errorsByType,
	}
}

// Reset 重置统计信息
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.StartTime = time.Now()
	s.TotalRequests = 0
	s.SuccessRequests = 0
	s.FailedRequests = 0
	s.AverageResponseTime = 0
	s.LastRequestTime = time.Time{}
	s.ConcurrentRequests = 0
	s.MaxConcurrentRequests = 0
	s.RequestsByType = make(map[string]int64)
	s.RequestDurations = make(map[string]time.Duration)
	s.ErrorsByType = make(map[string]int64)
}

// GetSuccessRate 获取成功率
func (s *Stats) GetSuccessRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.SuccessRequests) / float64(s.TotalRequests)
}

// GetUptime 获取运行时间
func (s *Stats) GetUptime() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return time.Since(s.StartTime)
}