package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/PerceptivePenguin/MCPRAG-Go/pkg/errors"
)

// Config 应用配置
type Config struct {
	// API 配置
	OpenAIAPIKey string
	BaseURL      string
	Model        string
	
	// Agent 配置
	SystemPrompt     string
	MaxToolCalls     int
	MaxContextLength int
	
	// MCP 配置
	EnableSequentialThinking bool
	EnableDeepWiki          bool
	EnableContext7          bool
	
	// RAG 配置
	EnableRAG        bool
	RAGContextLength int
	
	// 服务配置
	Interactive bool
	Verbose     bool
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		// API 默认配置
		Model: "gpt-4o",
		
		// Agent 默认配置
		MaxToolCalls:     10,
		MaxContextLength: 8192,
		
		// MCP 默认配置
		EnableSequentialThinking: true,
		EnableDeepWiki:          true,
		EnableContext7:          true,
		
		// RAG 默认配置
		EnableRAG:        true,
		RAGContextLength: 2048,
		
		// 服务默认配置
		Interactive: true,
		Verbose:     false,
	}
}

// ParseFlags 解析命令行参数
func ParseFlags() (*Config, error) {
	config := DefaultConfig()
	
	// API 配置
	flag.StringVar(&config.OpenAIAPIKey, "api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key")
	flag.StringVar(&config.BaseURL, "base-url", "", "OpenAI API base URL")
	flag.StringVar(&config.Model, "model", config.Model, "OpenAI model to use")
	
	// Agent 配置
	flag.StringVar(&config.SystemPrompt, "system-prompt", "", "Custom system prompt")
	flag.IntVar(&config.MaxToolCalls, "max-tool-calls", config.MaxToolCalls, "Maximum tool calls per conversation")
	flag.IntVar(&config.MaxContextLength, "max-context", config.MaxContextLength, "Maximum context length")
	
	// MCP 配置
	flag.BoolVar(&config.EnableSequentialThinking, "enable-sequential-thinking", config.EnableSequentialThinking, "Enable sequential thinking MCP server")
	flag.BoolVar(&config.EnableDeepWiki, "enable-deepwiki", config.EnableDeepWiki, "Enable DeepWiki MCP server")
	flag.BoolVar(&config.EnableContext7, "enable-context7", config.EnableContext7, "Enable Context7 MCP server")
	
	// RAG 配置
	flag.BoolVar(&config.EnableRAG, "enable-rag", config.EnableRAG, "Enable RAG retrieval")
	flag.IntVar(&config.RAGContextLength, "rag-context", config.RAGContextLength, "RAG context length")
	
	// 服务配置
	flag.BoolVar(&config.Interactive, "interactive", config.Interactive, "Run in interactive mode")
	flag.BoolVar(&config.Verbose, "verbose", config.Verbose, "Enable verbose logging")
	
	// 版本和帮助
	version := flag.Bool("version", false, "Show version")
	help := flag.Bool("help", false, "Show help")
	
	flag.Parse()
	
	// 处理版本和帮助
	if *version {
		showVersion()
		os.Exit(0)
	}
	
	if *help {
		showHelp()
		os.Exit(0)
	}
	
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	return config, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证必需的配置
	if c.OpenAIAPIKey == "" {
		return errors.ValidationError("api_key", "OpenAI API key is required. Set OPENAI_API_KEY environment variable or use -api-key flag")
	}
	
	// 验证数值范围
	if c.MaxToolCalls < 1 || c.MaxToolCalls > 100 {
		return errors.ValidationError("max_tool_calls", "max tool calls must be between 1 and 100")
	}
	
	if c.MaxContextLength < 1024 || c.MaxContextLength > 32768 {
		return errors.ValidationError("max_context_length", "max context length must be between 1024 and 32768")
	}
	
	if c.RAGContextLength < 256 || c.RAGContextLength > 8192 {
		return errors.ValidationError("rag_context_length", "RAG context length must be between 256 and 8192")
	}
	
	return nil
}

// showVersion 显示版本信息
func showVersion() {
	fmt.Printf("%s version %s\n", appName, appVersion)
}

// showHelp 显示帮助信息
func showHelp() {
	fmt.Printf("Usage: %s [options]\n\n", appName)
	fmt.Println("MCPRAG - A high-performance LLM system with MCP and RAG capabilities")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s -api-key YOUR_API_KEY\n", appName)
	fmt.Printf("  %s -model gpt-4o-mini -verbose\n", appName)
	fmt.Printf("  %s -enable-rag=false -interactive=false\n", appName)
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  OPENAI_API_KEY    OpenAI API key (alternative to -api-key)")
}