# Go-LLM-MCP-RAG

一个高性能的 Go 语言实现的增强型 LLM 系统，集成聊天、模型上下文协议 (MCP) 和检索增强生成 (RAG) 功能。

## 🚀 项目特性

- **高性能**: 通过 Go 的并发特性实现 3-5x 性能提升
- **框架无关**: 无需 LangChain 或 LlamaIndex 等重型框架
- **模块化架构**: 清晰的分层设计，统一的pkg/目录结构
- **多 MCP 服务器**: 支持多个 MCP 服务器并行工具调用
- **类型安全**: 编译时类型检查，减少运行时错误
- **生产就绪**: 内置监控、优雅关闭、资源管理

## 📋 系统要求

- Go 1.24.1 或更高版本
- OpenAI API 密钥
- 支持的操作系统：Linux, macOS, Windows

## 🛠️ 快速开始

### 安装

```bash
git clone https://github.com/PerceptivePenguin/MCPRAG-Go.git
cd MCPRAG-Go
go mod tidy
```

### 配置

设置 OpenAI API 密钥（必需）：

```bash
# 方式1: 环境变量 (推荐)
export OPENAI_API_KEY="your-openai-api-key"

# 方式2: 命令行参数
./mcprag -api-key "your-openai-api-key"
```

### 构建和运行

```bash
# 构建项目
go build ./cmd/mcprag

# 基础运行 (使用环境变量中的API密钥)
./mcprag

# 使用命令行参数
./mcprag -api-key "your-api-key" -verbose

# 查看所有可用选项
./mcprag --help
```

### 交互模式使用

启动后进入交互模式，支持以下操作：

```bash
> 你好，请介绍一下自己              # 普通聊天
> 请用结构化思维分析人工智能发展      # 使用Sequential Thinking
> 帮我查找React的最新文档           # 使用DeepWiki搜索
> help                           # 查看内置命令
> stats                          # 查看统计信息
> health                         # 查看系统健康状态
> exit                           # 退出应用
```


### 核心模块

- **Agent**: 中央协调器，管理 LLM 与工具的交互
- **Chat**: OpenAI API 客户端，支持流式响应和工具调用
- **MCP**: 模型上下文协议客户端，管理外部工具
- **RAG**: 检索增强生成，提供上下文注入
- **Vector**: 高性能向量存储和相似性搜索

### 通用模块 (pkg/)

- **types/**: 跨模块共享的类型定义
- **errors/**: 统一的错误处理系统
- **config/**: 配置管理和验证
- **utils/**: 通用工具函数库

## ⚙️ 命令行选项

### 基础用法

```bash
# 查看所有选项
./mcprag --help

# 基础运行（需要设置OPENAI_API_KEY环境变量）
./mcprag

# 使用特定模型和详细日志
./mcprag -model gpt-4o-mini -verbose

# 禁用某些MCP服务器
./mcprag -enable-deepwiki=false -enable-context7=false

# 调整上下文和工具调用限制
./mcprag -max-context 4096 -max-tool-calls 5 -rag-context 1024

# 自定义系统提示
./mcprag -system-prompt "你是一个专业的编程助手"
```

### 可用选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `-api-key` | `$OPENAI_API_KEY` | OpenAI API密钥 |
| `-model` | `gpt-4o` | 使用的OpenAI模型 |
| `-base-url` | - | 自定义API基础URL |
| `-max-context` | `8192` | 最大上下文长度 |
| `-max-tool-calls` | `10` | 每次对话最大工具调用数 |
| `-rag-context` | `2048` | RAG上下文长度 |
| `-enable-rag` | `true` | 启用RAG检索 |
| `-enable-sequential-thinking` | `true` | 启用结构化思维服务器 |
| `-enable-deepwiki` | `true` | 启用DeepWiki服务器 |
| `-enable-context7` | `true` | 启用Context7服务器 |
| `-interactive` | `true` | 交互模式 |
| `-verbose` | `false` | 详细日志 |
| `-system-prompt` | - | 自定义系统提示 |

## 🔧 开发指南

### 项目结构

```
├── cmd/mcprag/
│   ├── main.go         # 核心启动逻辑
│   ├── config.go       # 配置解析和验证
│   ├── app.go          # 应用生命周期管理
│   ├── interactive.go  # 交互模式处理
│   └── commands.go     # 内置命令系统
├── internal/
│   ├── agent/          # Agent 协调逻辑
│   ├── chat/           # OpenAI 客户端
│   ├── mcp/            # MCP 协议客户端
│   ├── rag/            # RAG 检索系统
│   └── vector/         # 向量存储
├── pkg/
│   ├── types/          # 跨模块共享类型
│   ├── errors/         # 统一错误处理
│   ├── config/         # 配置管理
│   └── utils/          # 工具函数库
```

### 开发命令

```bash
# 格式化代码
go fmt ./...

# 运行测试
go test ./...

# 静态检查
golangci-lint run

# 性能测试
go test -bench=. ./...

# 竞态检测
go test -race ./...
```

### 使用示例

```bash
# 启动时自动加载所有MCP服务器
./mcprag

# 禁用特定服务器
./mcprag -enable-deepwiki=false

# 在交互模式中使用
> 请用结构化思维分析：如何优化网站性能？    # Sequential Thinking
> 帮我查找React的最新文档                  # DeepWiki
> 查询最新的TypeScript API文档            # Context7
```

## 📊 内置监控

### 统计信息查看

在交互模式中使用 `stats` 命令查看实时统计：

### 健康状态检查

使用 `health` 命令查看系统状态：

## 📝 许可证

本项目采用 MIT 许可证。