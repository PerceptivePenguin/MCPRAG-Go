package types

import "time"

// BaseConfig 基础配置结构
type BaseConfig struct {
	Timeout    time.Duration `json:"timeout" yaml:"timeout"`
	MaxRetries int           `json:"max_retries" yaml:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay" yaml:"retry_delay"`
}

// DefaultBaseConfig 返回默认基础配置
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: time.Second,
	}
}

// ConnectionConfig 连接配置结构
type ConnectionConfig struct {
	BaseURL         string            `json:"base_url" yaml:"base_url"`
	APIKey          string            `json:"api_key" yaml:"api_key"`
	Headers         map[string]string `json:"headers" yaml:"headers"`
	ConnectTimeout  time.Duration     `json:"connect_timeout" yaml:"connect_timeout"`
	RequestTimeout  time.Duration     `json:"request_timeout" yaml:"request_timeout"`
	MaxConnections  int               `json:"max_connections" yaml:"max_connections"`
	EnableKeepAlive bool              `json:"enable_keep_alive" yaml:"enable_keep_alive"`
}

// DefaultConnectionConfig 返回默认连接配置
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		Headers:         make(map[string]string),
		ConnectTimeout:  10 * time.Second,
		RequestTimeout:  30 * time.Second,
		MaxConnections:  10,
		EnableKeepAlive: true,
	}
}

// ProcessingConfig 处理配置结构
type ProcessingConfig struct {
	BatchSize    int           `json:"batch_size" yaml:"batch_size"`
	Workers      int           `json:"workers" yaml:"workers"`
	QueueSize    int           `json:"queue_size" yaml:"queue_size"`
	ProcessingTimeout time.Duration `json:"processing_timeout" yaml:"processing_timeout"`
	EnableAsync  bool          `json:"enable_async" yaml:"enable_async"`
}

// DefaultProcessingConfig 返回默认处理配置
func DefaultProcessingConfig() ProcessingConfig {
	return ProcessingConfig{
		BatchSize:         100,
		Workers:           4,
		QueueSize:         1000,
		ProcessingTimeout: 5 * time.Minute,
		EnableAsync:       true,
	}
}

// MonitoringConfig 监控配置结构
type MonitoringConfig struct {
	EnableMetrics    bool          `json:"enable_metrics" yaml:"enable_metrics"`
	EnableTracing    bool          `json:"enable_tracing" yaml:"enable_tracing"`
	MetricsInterval  time.Duration `json:"metrics_interval" yaml:"metrics_interval"`
	HealthCheckPath  string        `json:"health_check_path" yaml:"health_check_path"`
	LogLevel         string        `json:"log_level" yaml:"log_level"`
}

// DefaultMonitoringConfig 返回默认监控配置
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		EnableMetrics:   true,
		EnableTracing:   false,
		MetricsInterval: time.Minute,
		HealthCheckPath: "/health",
		LogLevel:        "info",
	}
}