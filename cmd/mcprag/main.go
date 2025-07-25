package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	appName    = "mcprag"
	appVersion = "0.1.1"
)

func main() {
	// 解析命令行参数
	config, err := ParseFlags()
	if err != nil {
		log.Fatalf("配置错误: %v", err)
	}
	
	// 创建上下文和信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setupSignalHandler(cancel)
	
	// 创建并运行应用
	app, err := NewApp(config)
	if err != nil {
		log.Fatalf("创建应用失败: %v", err)
	}
	
	if err := app.Run(ctx); err != nil {
		log.Fatalf("应用运行错误: %v", err)
	}
	
	fmt.Println("应用正常退出")
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