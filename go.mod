module github.com/PerceptivePenguin/MCPRAG-Go

go 1.24.1

require (
	github.com/sashabaranov/go-openai v1.35.7
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/prometheus/client_golang v1.21.0
	github.com/prometheus/common v0.61.0
)

// Development and testing dependencies
require (
	github.com/stretchr/testify v1.10.0
)

// Indirect dependencies will be managed by go mod tidy
