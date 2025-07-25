package config

import (
	"fmt"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/pkg/errors"
	"github.com/PerceptivePenguin/MCPRAG-Go/pkg/types"
)

// CommonConfigs 包含了项目中常用的配置预设
// 这些预设可以被不同模块重用，确保配置的一致性

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	types.BaseConfig
	types.ConnectionConfig
	
	Driver          string `json:"driver" yaml:"driver"`
	Host            string `json:"host" yaml:"host"`
	Port            int    `json:"port" yaml:"port"`
	Database        string `json:"database" yaml:"database"`
	Username        string `json:"username" yaml:"username"`
	Password        string `json:"password" yaml:"password"`
	SSLMode         string `json:"ssl_mode" yaml:"ssl_mode"`
	MaxOpenConns    int    `json:"max_open_conns" yaml:"max_open_conns"`
	MaxIdleConns    int    `json:"max_idle_conns" yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`
}

// DefaultDatabaseConfig 返回默认的数据库配置
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		BaseConfig:       types.DefaultBaseConfig(),
		ConnectionConfig: types.DefaultConnectionConfig(),
		Driver:           "postgres",
		Host:             "localhost",
		Port:             5432,
		SSLMode:          "disable",
		MaxOpenConns:     25,
		MaxIdleConns:     5,
		ConnMaxLifetime:  5 * time.Minute,
	}
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	types.BaseConfig
	types.MonitoringConfig
	
	Host         string        `json:"host" yaml:"host"`
	Port         int           `json:"port" yaml:"port"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
	TLSEnabled   bool          `json:"tls_enabled" yaml:"tls_enabled"`
	CertFile     string        `json:"cert_file" yaml:"cert_file"`
	KeyFile      string        `json:"key_file" yaml:"key_file"`
	CORS         CORSConfig    `json:"cors" yaml:"cors"`
}

// CORSConfig 跨域配置
type CORSConfig struct {
	Enabled          bool     `json:"enabled" yaml:"enabled"`
	AllowedOrigins   []string `json:"allowed_origins" yaml:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods" yaml:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers" yaml:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers" yaml:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials" yaml:"allow_credentials"`
	MaxAge           int      `json:"max_age" yaml:"max_age"`
}

// DefaultServerConfig 返回默认的服务器配置
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		BaseConfig:       types.DefaultBaseConfig(),
		MonitoringConfig: types.DefaultMonitoringConfig(),
		Host:             "0.0.0.0",
		Port:             8080,
		ReadTimeout:      30 * time.Second,
		WriteTimeout:     30 * time.Second,
		IdleTimeout:      120 * time.Second,
		TLSEnabled:       false,
		CORS: CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"*"},
			MaxAge:         3600,
		},
	}
}

// RedisConfig Redis缓存配置
type RedisConfig struct {
	types.BaseConfig
	types.ConnectionConfig
	
	Host         string `json:"host" yaml:"host"`
	Port         int    `json:"port" yaml:"port"`
	Password     string `json:"password" yaml:"password"`
	Database     int    `json:"database" yaml:"database"`
	PoolSize     int    `json:"pool_size" yaml:"pool_size"`
	MinIdleConns int    `json:"min_idle_conns" yaml:"min_idle_conns"`
}

// DefaultRedisConfig 返回默认的Redis配置
func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		BaseConfig:       types.DefaultBaseConfig(),
		ConnectionConfig: types.DefaultConnectionConfig(),
		Host:             "localhost",
		Port:             6379,
		Database:         0,
		PoolSize:         10,
		MinIdleConns:     2,
	}
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"`
	Output     string `json:"output" yaml:"output"`
	Filename   string `json:"filename" yaml:"filename"`
	MaxSize    int    `json:"max_size" yaml:"max_size"`       // MB
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAge     int    `json:"max_age" yaml:"max_age"`         // days
	Compress   bool   `json:"compress" yaml:"compress"`
}

// DefaultLoggingConfig 返回默认的日志配置
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
}

// ApplicationConfig 应用程序主配置
type ApplicationConfig struct {
	App      AppConfig      `json:"app" yaml:"app"`
	Server   ServerConfig   `json:"server" yaml:"server"`
	Database DatabaseConfig `json:"database" yaml:"database"`
	Redis    RedisConfig    `json:"redis" yaml:"redis"`
	Logging  LoggingConfig  `json:"logging" yaml:"logging"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name        string `json:"name" yaml:"name"`
	Version     string `json:"version" yaml:"version"`
	Environment string `json:"environment" yaml:"environment"`
	Debug       bool   `json:"debug" yaml:"debug"`
}

// DefaultApplicationConfig 返回默认的应用配置
func DefaultApplicationConfig() ApplicationConfig {
	return ApplicationConfig{
		App: AppConfig{
			Name:        "mcprag-go",
			Version:     "1.0.0",
			Environment: "development",
			Debug:       true,
		},
		Server:   DefaultServerConfig(),
		Database: DefaultDatabaseConfig(),
		Redis:    DefaultRedisConfig(),
		Logging:  DefaultLoggingConfig(),
	}
}

// Validate 验证应用配置
func (c *ApplicationConfig) Validate() error {
	// 验证应用名称
	if c.App.Name == "" {
		return errors.ValidationError("app.name", "application name is required")
	}
	
	// 验证端口范围
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.ValidationError("server.port", "port must be between 1 and 65535")
	}
	
	// 验证数据库配置
	if c.Database.Host == "" {
		return errors.ValidationError("database.host", "database host is required")
	}
	
	// 验证日志级别
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	levelValid := false
	for _, level := range validLevels {
		if c.Logging.Level == level {
			levelValid = true
			break
		}
	}
	if !levelValid {
		return errors.ValidationError("logging.level", "invalid log level")
	}
	
	return nil
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	switch c.Driver {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.Host, c.Port, c.Username, c.Password, c.Database, c.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			c.Username, c.Password, c.Host, c.Port, c.Database)
	default:
		return ""
	}
}

// GetRedisAddr 获取Redis地址
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}