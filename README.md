# Go-LLM-MCP-RAG

A high-performance LLM system implemented in Go, integrating Chat, Model Context Protocol (MCP), and Retrieval Augmented Generation (RAG) capabilities.

## 🚀 Features

- **High Performance**: 3-5x performance improvement through Go's concurrency features
- **Framework-Free**: No dependency on heavy frameworks like LangChain or LlamaIndex
- **Modular Architecture**: Clean layered design with unified pkg/ directory structure
- **Multi-MCP Servers**: Support for parallel tool calls across multiple MCP servers
- **Type Safety**: Compile-time type checking reduces runtime errors
- **Production Ready**: Built-in monitoring, graceful shutdown, and resource management

## 📋 System Requirements

- Go 1.24.1 or higher
- OpenAI API key
- Supported OS: Linux, macOS, Windows

## 🛠️ Quick Start

### Installation

```bash
git clone https://github.com/PerceptivePenguin/MCPRAG-Go.git
cd MCPRAG-Go
go mod tidy
```

### Configuration

Set up your OpenAI API key (required):

```bash
# Method 1: Environment variable (recommended)
export OPENAI_API_KEY="your-openai-api-key"

# Method 2: Command line parameter
./mcprag -api-key "your-openai-api-key"
```

### Build and Run

```bash
# Build the project
go build ./cmd/mcprag

# Basic run (using API key from environment variable)
./mcprag

# Using command line parameters
./mcprag -api-key "your-api-key" -verbose

# View all available options
./mcprag --help
```

### Interactive Mode Usage

After startup, enter interactive mode with the following operations:

```bash
> Hello, please introduce yourself              # General chat
> Use structured thinking to analyze AI trends  # Sequential Thinking
> Help me find the latest React documentation   # DeepWiki search
> help                                          # View built-in commands
> stats                                         # View statistics
> health                                        # View system health
> exit                                          # Exit application
```

### Core Modules

- **Agent**: Central coordinator managing LLM and tool interactions
- **Chat**: OpenAI API client with streaming response and tool call support
- **MCP**: Model Context Protocol client managing external tools
- **RAG**: Retrieval Augmented Generation providing context injection
- **Vector**: High-performance vector storage and similarity search

### Common Modules (pkg/)

- **types/**: Shared type definitions across modules
- **errors/**: Unified error handling system
- **config/**: Configuration management and validation
- **utils/**: Common utility functions

## ⚙️ Command Line Options

### Basic Usage

```bash
# View all options
./mcprag --help

# Basic run (requires OPENAI_API_KEY environment variable)
./mcprag

# Use specific model with verbose logging
./mcprag -model gpt-4o-mini -verbose

# Disable certain MCP servers
./mcprag -enable-deepwiki=false -enable-context7=false

# Adjust context and tool call limits
./mcprag -max-context 4096 -max-tool-calls 5 -rag-context 1024

# Custom system prompt
./mcprag -system-prompt "You are a professional programming assistant"
```

### Available Options

| Option | Default | Description |
|--------|---------|-------------|
| `-api-key` | `$OPENAI_API_KEY` | OpenAI API key |
| `-model` | `gpt-4o` | OpenAI model to use |
| `-base-url` | - | Custom API base URL |
| `-max-context` | `8192` | Maximum context length |
| `-max-tool-calls` | `10` | Maximum tool calls per conversation |
| `-rag-context` | `2048` | RAG context length |
| `-enable-rag` | `true` | Enable RAG retrieval |
| `-enable-sequential-thinking` | `true` | Enable structured thinking server |
| `-enable-deepwiki` | `true` | Enable DeepWiki server |
| `-enable-context7` | `true` | Enable Context7 server |
| `-interactive` | `true` | Interactive mode |
| `-verbose` | `false` | Verbose logging |
| `-system-prompt` | - | Custom system prompt |

## 🔧 Development Guide

### Project Structure

```
├── cmd/mcprag/
│   ├── main.go         # Core startup logic
│   ├── config.go       # Configuration parsing and validation
│   ├── app.go          # Application lifecycle management
│   ├── interactive.go  # Interactive mode handling
│   └── commands.go     # Built-in command system
├── internal/
│   ├── agent/          # Agent coordination logic
│   ├── chat/           # OpenAI client
│   ├── mcp/            # MCP protocol client
│   ├── rag/            # RAG retrieval system
│   └── vector/         # Vector storage
├── pkg/
│   ├── types/          # Cross-module shared types
│   ├── errors/         # Unified error handling
│   ├── config/         # Configuration management
│   └── utils/          # Utility functions
```

### Development Commands

```bash
# Format code
go fmt ./...

# Run tests
go test ./...

# Static analysis
golangci-lint run

# Performance tests
go test -bench=. ./...

# Race detection
go test -race ./...
```

### Usage Examples

```bash
# Auto-load all MCP servers on startup
./mcprag

# Disable specific servers
./mcprag -enable-deepwiki=false

# Usage in interactive mode
> Use structured thinking to analyze: How to optimize website performance?  # Sequential Thinking
> Help me find the latest React documentation                              # DeepWiki
> Query the latest TypeScript API documentation                           # Context7
```

## 📊 Built-in Monitoring

### Statistics Viewing

Use the `stats` command in interactive mode to view real-time statistics:

### Health Status Check

Use the `health` command to view system status:

## 📝 License

This project is licensed under the MIT License.