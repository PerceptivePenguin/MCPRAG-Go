package agent

import (
	"context"
	"sync"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/chat"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/mcp"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/rag"
)

// Agent 中央协调器
type Agent struct {
	// 配置
	options Options
	
	// 组件
	chatClient *chat.ClientWithTools
	mcpManager *mcp.Manager
	ragRetriever rag.Retriever
	
	// 状态
	mu         sync.RWMutex
	started    bool
	toolCalls  int
	
	// 统计
	stats      *Stats
	errorStats *ErrorStats
	
	// 控制
	ctx    context.Context
	cancel context.CancelFunc
}

// Stats 统计信息
type Stats struct {
	StartTime         time.Time `json:"startTime"`
	TotalRequests     int64     `json:"totalRequests"`
	TotalToolCalls    int64     `json:"totalToolCalls"`
	TotalRAGQueries   int64     `json:"totalRAGQueries"`
	AverageResponseTime time.Duration `json:"averageResponseTime"`
	LastRequestTime   time.Time `json:"lastRequestTime"`
	
	// 并发统计
	ConcurrentRequests int32 `json:"concurrentRequests"`
	MaxConcurrentRequests int32 `json:"maxConcurrentRequests"`
	
	// 工具调用统计
	ToolCallsByName   map[string]int64 `json:"toolCallsByName"`
	ToolCallDurations map[string]time.Duration `json:"toolCallDurations"`
	
	// RAG 统计
	RAGHitRate        float64 `json:"ragHitRate"`
	RAGAverageLatency time.Duration `json:"ragAverageLatency"`
	
	mu sync.RWMutex
}

// Request 处理请求
type Request struct {
	ID        string    `json:"id"`
	Query     string    `json:"query"`
	Context   []string  `json:"context,omitempty"`
	EnableRAG bool      `json:"enableRAG"`
	EnableTools bool    `json:"enableTools"`
	Timestamp time.Time `json:"timestamp"`
}

// Response 处理响应
type Response struct {
	ID           string        `json:"id"`
	Content      string        `json:"content"`
	ToolCalls    []ToolCall    `json:"toolCalls,omitempty"`
	RAGContext   []string      `json:"ragContext,omitempty"`
	ResponseTime time.Duration `json:"responseTime"`
	TokenUsage   TokenUsage    `json:"tokenUsage"`
	Timestamp    time.Time     `json:"timestamp"`
	Error        string        `json:"error,omitempty"`
}

// ToolCall 工具调用信息
type ToolCall struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Args     interface{}   `json:"args"`
	Result   string        `json:"result"`
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// TokenUsage Token 使用统计
type TokenUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// StreamResponse 流式响应
type StreamResponse struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	ToolCalls []ToolCall `json:"toolCalls,omitempty"`
	Finished  bool      `json:"finished"`
	Error     error     `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// NewStats 创建新的统计信息
func NewStats() *Stats {
	return &Stats{
		StartTime:         time.Now(),
		ToolCallsByName:   make(map[string]int64),
		ToolCallDurations: make(map[string]time.Duration),
	}
}

// RecordRequest 记录请求统计
func (s *Stats) RecordRequest(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalRequests++
	s.LastRequestTime = time.Now()
	
	// 计算平均响应时间
	if s.TotalRequests == 1 {
		s.AverageResponseTime = duration
	} else {
		s.AverageResponseTime = time.Duration(
			(int64(s.AverageResponseTime)*int64(s.TotalRequests-1) + int64(duration)) / int64(s.TotalRequests),
		)
	}
}

// RecordToolCall 记录工具调用统计
func (s *Stats) RecordToolCall(name string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalToolCalls++
	s.ToolCallsByName[name]++
	
	// 更新平均时长
	if existing, ok := s.ToolCallDurations[name]; ok {
		count := s.ToolCallsByName[name]
		s.ToolCallDurations[name] = time.Duration(
			(int64(existing)*int64(count-1) + int64(duration)) / int64(count),
		)
	} else {
		s.ToolCallDurations[name] = duration
	}
}

// RecordRAGQuery 记录 RAG 查询统计
func (s *Stats) RecordRAGQuery(latency time.Duration, hitRate float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.TotalRAGQueries++
	
	// 更新平均延迟
	if s.TotalRAGQueries == 1 {
		s.RAGAverageLatency = latency
		s.RAGHitRate = hitRate
	} else {
		s.RAGAverageLatency = time.Duration(
			(int64(s.RAGAverageLatency)*int64(s.TotalRAGQueries-1) + int64(latency)) / int64(s.TotalRAGQueries),
		)
		s.RAGHitRate = (s.RAGHitRate*float64(s.TotalRAGQueries-1) + hitRate) / float64(s.TotalRAGQueries)
	}
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
	toolCallsByName := make(map[string]int64)
	for k, v := range s.ToolCallsByName {
		toolCallsByName[k] = v
	}
	
	toolCallDurations := make(map[string]time.Duration)
	for k, v := range s.ToolCallDurations {
		toolCallDurations[k] = v
	}
	
	return Stats{
		StartTime:             s.StartTime,
		TotalRequests:         s.TotalRequests,
		TotalToolCalls:        s.TotalToolCalls,
		TotalRAGQueries:       s.TotalRAGQueries,
		AverageResponseTime:   s.AverageResponseTime,
		LastRequestTime:       s.LastRequestTime,
		ConcurrentRequests:    s.ConcurrentRequests,
		MaxConcurrentRequests: s.MaxConcurrentRequests,
		ToolCallsByName:       toolCallsByName,
		ToolCallDurations:     toolCallDurations,
		RAGHitRate:            s.RAGHitRate,
		RAGAverageLatency:     s.RAGAverageLatency,
	}
}

// Reset 重置统计信息
func (s *Stats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.StartTime = time.Now()
	s.TotalRequests = 0
	s.TotalToolCalls = 0
	s.TotalRAGQueries = 0
	s.AverageResponseTime = 0
	s.LastRequestTime = time.Time{}
	s.ConcurrentRequests = 0
	s.MaxConcurrentRequests = 0
	s.ToolCallsByName = make(map[string]int64)
	s.ToolCallDurations = make(map[string]time.Duration)
	s.RAGHitRate = 0
	s.RAGAverageLatency = 0
}

// String 返回统计信息的字符串表示
func (s *Stats) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	uptime := time.Since(s.StartTime)
	return "Agent Stats: " +
		"Uptime=" + uptime.String() + ", " +
		"Requests=" + string(rune(s.TotalRequests)) + ", " +
		"ToolCalls=" + string(rune(s.TotalToolCalls)) + ", " +
		"RAGQueries=" + string(rune(s.TotalRAGQueries)) + ", " +
		"AvgResponseTime=" + s.AverageResponseTime.String()
}