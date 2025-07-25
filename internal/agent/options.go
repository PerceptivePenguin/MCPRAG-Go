package agent

import (
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/chat"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/mcp"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/rag"
)

// Options Agent 配置选项
type Options struct {
	// 模块配置
	ChatConfig chat.ClientConfig `json:"chatConfig"`
	MCPConfig  mcp.ManagerConfig `json:"mcpConfig"`
	RAGConfig  rag.RetrieverConfig `json:"ragConfig"`
	
	// 工具调用配置
	MaxToolCalls         int           `json:"maxToolCalls"`
	ToolCallTimeout      time.Duration `json:"toolCallTimeout"`
	MaxConcurrentCalls   int           `json:"maxConcurrentCalls"`
	EnableParallelCalls  bool          `json:"enableParallelCalls"`
	
	// 上下文配置
	MaxContextLength     int    `json:"maxContextLength"`
	SystemPrompt         string `json:"systemPrompt"`
	EnableRAGContext     bool   `json:"enableRAGContext"`
	RAGContextLength     int    `json:"ragContextLength"`
	
	// 性能配置
	EnableMetrics        bool          `json:"enableMetrics"`
	MetricsInterval      time.Duration `json:"metricsInterval"`
	EnableLogging        bool          `json:"enableLogging"`
	LogLevel             string        `json:"logLevel"`
	
	// 重试配置
	MaxRetries           int           `json:"maxRetries"`
	RetryDelay           time.Duration `json:"retryDelay"`
	RetryBackoff         float64       `json:"retryBackoff"`
}

// DefaultOptions 返回默认的 Agent 配置
func DefaultOptions() Options {
	return Options{
		ChatConfig: chat.DefaultClientConfig(),
		MCPConfig:  mcp.DefaultManagerConfig(),
		RAGConfig:  *rag.DefaultRetrieverConfig(),
		
		MaxToolCalls:        10,
		ToolCallTimeout:     30 * time.Second,
		MaxConcurrentCalls:  3,
		EnableParallelCalls: true,
		
		MaxContextLength:    8192,
		SystemPrompt:        "You are a helpful assistant with access to various tools. Use them when needed to provide accurate and comprehensive responses.",
		EnableRAGContext:    true,
		RAGContextLength:    2048,
		
		EnableMetrics:       false,
		MetricsInterval:     60 * time.Second,
		EnableLogging:       true,
		LogLevel:           "info",
		
		MaxRetries:          3,
		RetryDelay:          time.Second,
		RetryBackoff:        2.0,
	}
}

// Option 函数选项类型
type Option func(*Options)

// WithChatConfig 设置 Chat 配置
func WithChatConfig(config chat.ClientConfig) Option {
	return func(o *Options) {
		o.ChatConfig = config
	}
}

// WithMCPConfig 设置 MCP 配置
func WithMCPConfig(config mcp.ManagerConfig) Option {
	return func(o *Options) {
		o.MCPConfig = config
	}
}

// WithRAGConfig 设置 RAG 配置
func WithRAGConfig(config rag.RetrieverConfig) Option {
	return func(o *Options) {
		o.RAGConfig = config
	}
}

// WithMaxToolCalls 设置最大工具调用次数
func WithMaxToolCalls(max int) Option {
	return func(o *Options) {
		o.MaxToolCalls = max
	}
}

// WithToolCallTimeout 设置工具调用超时时间
func WithToolCallTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.ToolCallTimeout = timeout
	}
}

// WithMaxConcurrentCalls 设置最大并发调用数
func WithMaxConcurrentCalls(max int) Option {
	return func(o *Options) {
		o.MaxConcurrentCalls = max
	}
}

// WithParallelCalls 设置是否启用并行调用
func WithParallelCalls(enable bool) Option {
	return func(o *Options) {
		o.EnableParallelCalls = enable
	}
}

// WithMaxContextLength 设置最大上下文长度
func WithMaxContextLength(length int) Option {
	return func(o *Options) {
		o.MaxContextLength = length
	}
}

// WithSystemPrompt 设置系统提示
func WithSystemPrompt(prompt string) Option {
	return func(o *Options) {
		o.SystemPrompt = prompt
	}
}

// WithRAGContext 设置是否启用 RAG 上下文
func WithRAGContext(enable bool, length int) Option {
	return func(o *Options) {
		o.EnableRAGContext = enable
		if length > 0 {
			o.RAGContextLength = length
		}
	}
}

// WithMetrics 设置指标收集
func WithMetrics(enable bool, interval time.Duration) Option {
	return func(o *Options) {
		o.EnableMetrics = enable
		if interval > 0 {
			o.MetricsInterval = interval
		}
	}
}

// WithLogging 设置日志记录
func WithLogging(enable bool, level string) Option {
	return func(o *Options) {
		o.EnableLogging = enable
		if level != "" {
			o.LogLevel = level
		}
	}
}

// WithRetryConfig 设置重试配置
func WithRetryConfig(maxRetries int, delay time.Duration, backoff float64) Option {
	return func(o *Options) {
		o.MaxRetries = maxRetries
		o.RetryDelay = delay
		o.RetryBackoff = backoff
	}
}

// Validate 验证配置选项
func (o *Options) Validate() error {
	if o.MaxToolCalls <= 0 {
		return NewAgentError("validate", "Invalid MaxToolCalls: must be positive", false)
	}
	
	if o.ToolCallTimeout <= 0 {
		return NewAgentError("validate", "Invalid ToolCallTimeout: must be positive", false)
	}
	
	if o.MaxConcurrentCalls <= 0 {
		return NewAgentError("validate", "Invalid MaxConcurrentCalls: must be positive", false)
	}
	
	if o.MaxContextLength <= 0 {
		return NewAgentError("validate", "Invalid MaxContextLength: must be positive", false)
	}
	
	if o.RAGContextLength < 0 {
		return NewAgentError("validate", "Invalid RAGContextLength: cannot be negative", false)
	}
	
	if o.MaxRetries < 0 {
		return NewAgentError("validate", "Invalid MaxRetries: cannot be negative", false)
	}
	
	if o.RetryDelay < 0 {
		return NewAgentError("validate", "Invalid RetryDelay: cannot be negative", false)
	}
	
	if o.RetryBackoff <= 0 {
		return NewAgentError("validate", "Invalid RetryBackoff: must be positive", false)
	}
	
	return nil
}

// Clone 创建配置副本
func (o *Options) Clone() Options {
	return Options{
		ChatConfig:          o.ChatConfig,
		MCPConfig:           o.MCPConfig,
		RAGConfig:           o.RAGConfig,
		MaxToolCalls:        o.MaxToolCalls,
		ToolCallTimeout:     o.ToolCallTimeout,
		MaxConcurrentCalls:  o.MaxConcurrentCalls,
		EnableParallelCalls: o.EnableParallelCalls,
		MaxContextLength:    o.MaxContextLength,
		SystemPrompt:        o.SystemPrompt,
		EnableRAGContext:    o.EnableRAGContext,
		RAGContextLength:    o.RAGContextLength,
		EnableMetrics:       o.EnableMetrics,
		MetricsInterval:     o.MetricsInterval,
		EnableLogging:       o.EnableLogging,
		LogLevel:           o.LogLevel,
		MaxRetries:          o.MaxRetries,
		RetryDelay:          o.RetryDelay,
		RetryBackoff:        o.RetryBackoff,
	}
}

// String 返回配置的字符串表示
func (o *Options) String() string {
	return "Agent Options: " +
		"MaxToolCalls=" + string(rune(o.MaxToolCalls)) + ", " +
		"MaxConcurrentCalls=" + string(rune(o.MaxConcurrentCalls)) + ", " +
		"EnableParallelCalls=" + boolToString(o.EnableParallelCalls) + ", " +
		"EnableRAGContext=" + boolToString(o.EnableRAGContext) + ", " +
		"EnableMetrics=" + boolToString(o.EnableMetrics)
}

// boolToString 转换布尔值为字符串
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}