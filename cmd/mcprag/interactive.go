package main

import (
	"context"
	"fmt"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/agent"
)

// InteractiveMode 交互模式处理器
type InteractiveMode struct {
	agent    *agent.Agent
	commands *CommandRegistry
}

// NewInteractiveMode 创建交互模式处理器
func NewInteractiveMode(agent *agent.Agent) *InteractiveMode {
	return &InteractiveMode{
		agent:    agent,
		commands: NewCommandRegistry(agent),
	}
}

// Run 运行交互模式
func (im *InteractiveMode) Run(ctx context.Context) error {
	im.showWelcome()
	
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// 读取用户输入
			input, err := im.readInput()
			if err != nil {
				if err.Error() == "unexpected newline" {
					continue
				}
				return fmt.Errorf("failed to read input: %w", err)
			}
			
			// 处理输入
			shouldExit, err := im.processInput(ctx, input)
			if err != nil {
				fmt.Printf("错误: %v\n", err)
			}
			
			if shouldExit {
				return nil
			}
		}
	}
}

// showWelcome 显示欢迎信息
func (im *InteractiveMode) showWelcome() {
	fmt.Println("\n=== MCPRAG Interactive Mode ===")
	fmt.Println("输入您的问题，输入 'exit' 或 'quit' 退出")
	fmt.Println("输入 'help' 查看可用命令")
	fmt.Println("=====================================")
	fmt.Println()
}

// readInput 读取用户输入
func (im *InteractiveMode) readInput() (string, error) {
	fmt.Print("> ")
	var input string
	_, err := fmt.Scanln(&input)
	return input, err
}

// processInput 处理用户输入
func (im *InteractiveMode) processInput(ctx context.Context, input string) (bool, error) {
	// 处理空输入
	if input == "" {
		return false, nil
	}
	
	// 处理退出命令
	if input == "exit" || input == "quit" {
		return true, nil
	}
	
	// 处理内置命令
	if handled, err := im.commands.Handle(input); handled {
		return false, err
	}
	
	// 处理用户查询
	return false, im.processQuery(ctx, input)
}

// processQuery 处理用户查询
func (im *InteractiveMode) processQuery(ctx context.Context, query string) error {
	// 创建请求
	req := agent.Request{
		ID:    fmt.Sprintf("req-%d", time.Now().Unix()),
		Query: query,
	}
	
	fmt.Println("\n正在处理您的问题...")
	
	// 发送请求并获取流式响应
	responseStream, err := im.agent.ProcessStream(ctx, req)
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