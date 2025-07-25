package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/agent"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/chat"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/mcp"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/rag"
)

const (
	appName    = "mcprag"
	appVersion = "0.1.0"
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

func main() {
	// 解析命令行参数
	config := parseFlags()
	
	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// 设置信号处理
	setupSignalHandler(cancel)
	
	// 创建并启动Agent
	app, err := NewApp(config)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}
	
	if err := app.Run(ctx); err != nil {
		log.Fatalf("Application error: %v", err)
	}
	
	fmt.Println("应用正常退出")
}

// parseFlags 解析命令行参数
func parseFlags() *Config {
	config := &Config{}
	
	// API 配置
	flag.StringVar(&config.OpenAIAPIKey, "api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key")
	flag.StringVar(&config.BaseURL, "base-url", "", "OpenAI API base URL")
	flag.StringVar(&config.Model, "model", "gpt-4o", "OpenAI model to use")
	
	// Agent 配置
	flag.StringVar(&config.SystemPrompt, "system-prompt", "", "Custom system prompt")
	flag.IntVar(&config.MaxToolCalls, "max-tool-calls", 10, "Maximum tool calls per conversation")
	flag.IntVar(&config.MaxContextLength, "max-context", 8192, "Maximum context length")
	
	// MCP 配置
	flag.BoolVar(&config.EnableSequentialThinking, "enable-sequential-thinking", true, "Enable sequential thinking MCP server")
	flag.BoolVar(&config.EnableDeepWiki, "enable-deepwiki", true, "Enable DeepWiki MCP server")
	flag.BoolVar(&config.EnableContext7, "enable-context7", true, "Enable Context7 MCP server")
	
	// RAG 配置
	flag.BoolVar(&config.EnableRAG, "enable-rag", true, "Enable RAG retrieval")
	flag.IntVar(&config.RAGContextLength, "rag-context", 2048, "RAG context length")
	
	// 服务配置
	flag.BoolVar(&config.Interactive, "interactive", true, "Run in interactive mode")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	
	// 版本和帮助
	version := flag.Bool("version", false, "Show version")
	help := flag.Bool("help", false, "Show help")
	
	flag.Parse()
	
	if *version {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}
	
	if *help {
		fmt.Printf("Usage: %s [options]\n\n", appName)
		flag.PrintDefaults()
		os.Exit(0)
	}
	
	// 验证必需的配置
	if config.OpenAIAPIKey == "" {
		log.Fatal("OpenAI API key is required. Set OPENAI_API_KEY environment variable or use -api-key flag")
	}
	
	return config
}

// setupSignalHandler 设置信号处理
func setupSignalHandler(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		fmt.Println("\n收到中断信号，正在优雅关闭...")
		cancel()
	}()
}

// App 应用主结构
type App struct {
	config *Config
	agent  *agent.Agent
}

// NewApp 创建新的应用实例
func NewApp(config *Config) (*App, error) {
	// 创建并配置MCP管理器
	mcpConfig := createMCPConfig(config)
	mcpManager := mcp.NewManager(mcpConfig)
	
	// 注册MCP客户端
	if err := configureMCPManager(mcpManager, config); err != nil {
		return nil, fmt.Errorf("failed to configure MCP manager: %w", err)
	}
	
	// 创建Agent选项
	opts := []agent.Option{
		agent.WithChatConfig(createChatConfig(config)),
		agent.WithMCPConfig(mcpConfig),
		agent.WithRAGConfig(createRAGConfig(config)),
		agent.WithMaxToolCalls(config.MaxToolCalls),
		agent.WithMaxContextLength(config.MaxContextLength),
		agent.WithRAGContext(config.EnableRAG, config.RAGContextLength),
	}
	
	// 设置系统提示
	if config.SystemPrompt != "" {
		opts = append(opts, agent.WithSystemPrompt(config.SystemPrompt))
	}
	
	// 创建Agent
	agentInstance, err := agent.NewAgent(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	
	return &App{
		config: config,
		agent:  agentInstance,
	}, nil
}

// createChatConfig 创建Chat配置
func createChatConfig(config *Config) chat.ClientConfig {
	chatConfig := chat.DefaultClientConfig()
	chatConfig.APIKey = config.OpenAIAPIKey
	chatConfig.Model = config.Model
	
	if config.BaseURL != "" {
		chatConfig.BaseURL = config.BaseURL
	}
	
	return chatConfig
}

// createMCPConfig 创建MCP配置
func createMCPConfig(config *Config) mcp.ManagerConfig {
	mcpConfig := mcp.DefaultManagerConfig()
	return mcpConfig
}

// configureMCPManager 配置MCP管理器
func configureMCPManager(manager *mcp.Manager, config *Config) error {
	// 根据配置启用MCP服务器
	if config.EnableSequentialThinking {
		if err := manager.RegisterSequentialThinkingClient(); err != nil {
			return fmt.Errorf("failed to register sequential thinking client: %w", err)
		}
	}
	
	if config.EnableDeepWiki {
		if err := manager.RegisterDeepWikiClient(); err != nil {
			return fmt.Errorf("failed to register deepwiki client: %w", err)
		}
	}
	
	if config.EnableContext7 {
		if err := manager.RegisterContext7Client(); err != nil {
			return fmt.Errorf("failed to register context7 client: %w", err)
		}
	}
	
	return nil
}

// createRAGConfig 创建RAG配置
func createRAGConfig(config *Config) rag.RetrieverConfig {
	ragConfig := *rag.DefaultRetrieverConfig()
	ragConfig.Embedding.APIKey = config.OpenAIAPIKey
	
	if config.BaseURL != "" {
		ragConfig.Embedding.BaseURL = config.BaseURL
	}
	
	return ragConfig
}

// Run 运行应用
func (app *App) Run(ctx context.Context) error {
	fmt.Printf("正在启动 %s v%s...\n", appName, appVersion)
	
	// 启动Agent
	if err := app.agent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}
	defer app.agent.Stop()
	
	fmt.Println("Agent 启动成功")
	
	if app.config.Interactive {
		return app.runInteractive(ctx)
	}
	
	// 非交互模式：等待信号
	<-ctx.Done()
	return nil
}

// runInteractive 运行交互模式
func (app *App) runInteractive(ctx context.Context) error {
	fmt.Println("\n=== MCPRAG Interactive Mode ===")
	fmt.Println("输入您的问题，输入 'exit' 或 'quit' 退出")
	fmt.Println("输入 'help' 查看可用命令")
	fmt.Println("=====================================")
	fmt.Println()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// 读取用户输入
			fmt.Print("> ")
			var input string
			_, err := fmt.Scanln(&input)
			if err != nil {
				if err.Error() == "unexpected newline" {
					continue
				}
				return fmt.Errorf("failed to read input: %w", err)
			}
			
			// 处理特殊命令
			switch input {
			case "exit", "quit":
				return nil
			case "help":
				app.showHelp()
				continue
			case "stats":
				app.showStats()
				continue
			case "health":
				app.showHealth()
				continue
			case "":
				continue
			}
			
			// 处理用户查询
			if err := app.processQuery(ctx, input); err != nil {
				fmt.Printf("错误: %v\n", err)
			}
		}
	}
}

// processQuery 处理用户查询
func (app *App) processQuery(ctx context.Context, query string) error {
	// 创建请求
	req := agent.Request{
		ID:    fmt.Sprintf("req-%d", time.Now().Unix()),
		Query: query,
	}
	
	fmt.Println("\n正在处理您的问题...")
	
	// 发送请求并获取流式响应
	responseStream, err := app.agent.ProcessStream(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to process query: %w", err)
	}
	
	fmt.Print("回答: ")
	
	// 处理流式响应
	for response := range responseStream {
		if response.Error != nil {
			return fmt.Errorf("stream error: %w", response.Error)
		}
		
		if response.Content != "" {
			fmt.Print(response.Content)
		}
		
		if response.Finished {
			fmt.Println()
			break
		}
	}
	
	return nil
}

// showHelp 显示帮助信息
func (app *App) showHelp() {
	fmt.Println("\n可用命令:")
	fmt.Println("  help   - 显示此帮助信息")
	fmt.Println("  stats  - 显示Agent统计信息")
	fmt.Println("  health - 显示健康状态")
	fmt.Println("  exit   - 退出程序")
	fmt.Println("  quit   - 退出程序")
	fmt.Println("\n直接输入问题即可开始对话")
}

// showStats 显示统计信息
func (app *App) showStats() {
	stats := app.agent.GetStats()
	fmt.Printf("\n=== Agent 统计信息 ===\n")
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("工具调用: %d\n", stats.TotalToolCalls)
	fmt.Printf("RAG查询: %d\n", stats.TotalRAGQueries)
	fmt.Printf("平均响应时间: %v\n", stats.AverageResponseTime)
	fmt.Printf("并发请求: %d (峰值: %d)\n", stats.ConcurrentRequests, stats.MaxConcurrentRequests)
	fmt.Printf("RAG命中率: %.2f%%\n", stats.RAGHitRate*100)
	fmt.Printf("启动时间: %v\n", stats.StartTime)
	fmt.Println("====================")
}

// showHealth 显示健康状态
func (app *App) showHealth() {
	health := app.agent.Health()
	fmt.Printf("\n=== 健康状态 ===\n")
	fmt.Printf("Agent状态: %v\n", health["status"])
	fmt.Printf("启动时间: %v\n", health["start_time"])
	fmt.Printf("运行时长: %v\n", health["uptime"])
	fmt.Printf("连接状态: %v\n", health["connections"])
	if stats, ok := health["stats"]; ok {
		fmt.Printf("实时统计: %v\n", stats)
	}
	fmt.Println("===============")
}