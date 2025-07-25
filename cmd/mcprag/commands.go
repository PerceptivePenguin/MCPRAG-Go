package main

import (
	"fmt"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/agent"
)

// Command 内置命令接口
type Command interface {
	Name() string
	Description() string
	Execute() error
}

// CommandRegistry 命令注册表
type CommandRegistry struct {
	commands map[string]Command
	agent    *agent.Agent
}

// NewCommandRegistry 创建命令注册表
func NewCommandRegistry(agent *agent.Agent) *CommandRegistry {
	registry := &CommandRegistry{
		commands: make(map[string]Command),
		agent:    agent,
	}
	
	// 注册内置命令
	registry.registerBuiltinCommands()
	
	return registry
}

// registerBuiltinCommands 注册内置命令
func (cr *CommandRegistry) registerBuiltinCommands() {
	cr.Register(&HelpCommand{registry: cr})
	cr.Register(&StatsCommand{agent: cr.agent})
	cr.Register(&HealthCommand{agent: cr.agent})
}

// Register 注册命令
func (cr *CommandRegistry) Register(cmd Command) {
	cr.commands[cmd.Name()] = cmd
}

// Handle 处理命令
func (cr *CommandRegistry) Handle(input string) (bool, error) {
	if cmd, exists := cr.commands[input]; exists {
		return true, cmd.Execute()
	}
	return false, nil
}

// GetCommands 获取所有命令
func (cr *CommandRegistry) GetCommands() map[string]Command {
	return cr.commands
}

// HelpCommand 帮助命令
type HelpCommand struct {
	registry *CommandRegistry
}

func (h *HelpCommand) Name() string {
	return "help"
}

func (h *HelpCommand) Description() string {
	return "显示此帮助信息"
}

func (h *HelpCommand) Execute() error {
	fmt.Println("\n可用命令:")
	
	for name, cmd := range h.registry.GetCommands() {
		fmt.Printf("  %-8s - %s\n", name, cmd.Description())
	}
	
	fmt.Println("  exit     - 退出程序")
	fmt.Println("  quit     - 退出程序")
	fmt.Println("\n直接输入问题即可开始对话")
	
	return nil
}

// StatsCommand 统计信息命令
type StatsCommand struct {
	agent *agent.Agent
}

func (s *StatsCommand) Name() string {
	return "stats"
}

func (s *StatsCommand) Description() string {
	return "显示Agent统计信息"
}

func (s *StatsCommand) Execute() error {
	stats := s.agent.GetStats()
	
	fmt.Printf("\n=== Agent 统计信息 ===\n")
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("工具调用: %d\n", stats.TotalToolCalls)
	fmt.Printf("RAG查询: %d\n", stats.TotalRAGQueries)
	fmt.Printf("平均响应时间: %v\n", stats.AverageResponseTime)
	fmt.Printf("并发请求: %d (峰值: %d)\n", stats.ConcurrentRequests, stats.MaxConcurrentRequests)
	fmt.Printf("RAG命中率: %.2f%%\n", stats.RAGHitRate*100)
	fmt.Printf("启动时间: %v\n", stats.StartTime)
	fmt.Println("====================")
	
	return nil
}

// HealthCommand 健康状态命令
type HealthCommand struct {
	agent *agent.Agent
}

func (h *HealthCommand) Name() string {
	return "health"
}

func (h *HealthCommand) Description() string {
	return "显示健康状态"
}

func (h *HealthCommand) Execute() error {
	health := h.agent.Health()
	
	fmt.Printf("\n=== 健康状态 ===\n")
	fmt.Printf("Agent状态: %v\n", health["started"])
	fmt.Printf("运行时长: %v\n", health["uptime"])
	
	if mcpManager, ok := health["mcpManager"]; ok {
		fmt.Printf("MCP管理器: %v\n", mcpManager)
	}
	
	if stats, ok := health["stats"]; ok {
		fmt.Printf("实时统计: %v\n", stats)
	}
	
	if errors, ok := health["errors"]; ok {
		fmt.Printf("错误统计: %v\n", errors)
	}
	
	fmt.Println("===============")
	
	return nil
}