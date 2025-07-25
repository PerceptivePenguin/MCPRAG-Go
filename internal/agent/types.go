package agent

import (
	"context"
	"sync"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/chat"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/mcp"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/rag"
	"github.com/PerceptivePenguin/MCPRAG-Go/pkg/types"
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
	stats      *AgentStats
	errorStats *ErrorStats
	
	// 控制
	ctx    context.Context
	cancel context.CancelFunc
}

// AgentStats 扩展统计信息，基于通用Stats
type AgentStats struct {
	types.Stats
	
	// Agent 特定统计
	TotalToolCalls    int64 `json:"total_tool_calls"`
	TotalRAGQueries   int64 `json:"total_rag_queries"`
	
	// 工具调用统计
	ToolCallsByName   map[string]int64 `json:"tool_calls_by_name"`
	ToolCallDurations map[string]time.Duration `json:"tool_call_durations"`
	
	// RAG 统计
	RAGHitRate        float64 `json:"rag_hit_rate"`
	RAGAverageLatency time.Duration `json:"rag_average_latency"`
	
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

// ToolCall Agent模块的工具调用结构，扩展了通用ToolCall
type ToolCall struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Args     string        `json:"args"`
	Duration time.Duration `json:"duration,omitempty"`
}

// TokenUsage 使用pkg/types中的通用类型
type TokenUsage = types.TokenUsage

// StreamResponse Agent模块的流式响应结构，使用Agent专用的ToolCall类型
type StreamResponse struct {
	ID        string     `json:"id"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Finished  bool       `json:"finished"`
	Error     error      `json:"error,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
}

// NewAgentStats 创建新的Agent统计信息
func NewAgentStats() *AgentStats {
	return &AgentStats{
		Stats:             *types.NewStats(),
		ToolCallsByName:   make(map[string]int64),
		ToolCallDurations: make(map[string]time.Duration),
	}
}

// RecordRequest 记录请求统计
func (s *AgentStats) RecordRequest(duration time.Duration) {
	// 使用通用Stats的方法记录基础统计
	s.Stats.RecordRequest("agent", duration, true)
}

// RecordToolCall 记录工具调用统计
func (s *AgentStats) RecordToolCall(name string, duration time.Duration) {
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
func (s *AgentStats) RecordRAGQuery(latency time.Duration, hitRate float64) {
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

// GetAgentStats 获取Agent统计信息副本
func (s *AgentStats) GetAgentStats() AgentStats {
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
	
	baseStats := s.Stats.GetStats()
	
	return AgentStats{
		Stats:             baseStats,
		TotalToolCalls:    s.TotalToolCalls,
		TotalRAGQueries:   s.TotalRAGQueries,
		ToolCallsByName:   toolCallsByName,
		ToolCallDurations: toolCallDurations,
		RAGHitRate:        s.RAGHitRate,
		RAGAverageLatency: s.RAGAverageLatency,
	}
}