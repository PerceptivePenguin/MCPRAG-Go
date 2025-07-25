package main

import (
	"context"
	"fmt"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/agent"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/chat"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/mcp"
	"github.com/PerceptivePenguin/MCPRAG-Go/internal/rag"
)

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
	interactive := NewInteractiveMode(app.agent)
	return interactive.Run(ctx)
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