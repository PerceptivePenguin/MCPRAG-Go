package rag

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

// BasicContextBuilder implements the ContextBuilder interface
type BasicContextBuilder struct {
	config    *ContextConfig
	tokenizer Tokenizer
}

// NewBasicContextBuilder creates a new basic context builder
func NewBasicContextBuilder(config *ContextConfig, tokenizer Tokenizer) *BasicContextBuilder {
	if config == nil {
		config = DefaultContextConfig()
	}

	return &BasicContextBuilder{
		config:    config,
		tokenizer: tokenizer,
	}
}

// BuildContext creates context string from retrieval results
func (b *BasicContextBuilder) BuildContext(ctx context.Context, result *RetrievalResult, config ContextConfig) (string, error) {
	if result == nil {
		return "", NewRAGErrorWithOp("build_context", "retrieval result is nil", ErrorTypeValidation)
	}

	if len(result.Documents) == 0 {
		return "", nil // Empty context for no documents
	}

	// Use provided config or fall back to default
	if config.Template == "" {
		config = *b.config
	}

	var contextParts []string

	// Sort documents by relevance score if requested
	documents := result.Documents
	scores := result.Scores
	if len(scores) == len(documents) {
		// Create paired slice for sorting
		type docScore struct {
			doc   Document
			score float32
		}
		
		pairs := make([]docScore, len(documents))
		for i, doc := range documents {
			pairs[i] = docScore{doc: doc, score: scores[i]}
		}

		// Sort by score descending
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].score > pairs[j].score
		})

		// Extract sorted documents
		documents = make([]Document, len(pairs))
		for i, pair := range pairs {
			documents[i] = pair.doc
		}
	}

	// Build context for each document
	for i, doc := range documents {
		docContext, err := b.FormatDocument(doc, config.Template)
		if err != nil {
			return "", NewRAGErrorWithCause("failed to format document", ErrorTypeInternal, err).WithOperation("build_context")
		}

		// Add metadata if requested
		if config.IncludeMetadata && len(doc.Metadata) > 0 {
			metadata := b.formatMetadata(doc.Metadata)
			if metadata != "" {
				docContext = fmt.Sprintf("%s\nMetadata: %s", docContext, metadata)
			}
		}

		// Add scores if requested
		if config.IncludeScores && i < len(scores) {
			docContext = fmt.Sprintf("%s\nRelevance Score: %.3f", docContext, scores[i])
		}

		contextParts = append(contextParts, docContext)
	}

	// Join context parts
	separator := "\n\n"
	if config.SeparateChunks {
		separator = "\n---\n"
	}
	
	fullContext := strings.Join(contextParts, separator)

	// Apply length limits and truncation
	if config.MaxLength > 0 && b.tokenizer != nil {
		tokenCount := b.tokenizer.CountTokens(fullContext)
		if tokenCount > config.MaxLength {
			fullContext, err := b.TruncateContext(fullContext, config.MaxLength, config.TruncateStrategy)
			if err != nil {
				return "", err
			}
			return fullContext, nil
		}
	} else if config.MaxLength > 0 {
		// Fallback to character-based truncation
		if len(fullContext) > config.MaxLength {
			switch config.TruncateStrategy {
			case "head":
				fullContext = fullContext[:config.MaxLength]
			case "tail":
				start := len(fullContext) - config.MaxLength
				if start < 0 {
					start = 0
				}
				fullContext = fullContext[start:]
			case "middle":
				if config.MaxLength < 100 {
					fullContext = fullContext[:config.MaxLength]
				} else {
					halfSize := config.MaxLength / 2
					prefix := fullContext[:halfSize-50]
					suffix := fullContext[len(fullContext)-(halfSize-50):]
					fullContext = prefix + "\n... [truncated] ...\n" + suffix
				}
			default:
				fullContext = fullContext[:config.MaxLength]
			}
		}
	}

	return fullContext, nil
}

// BuildContextFromDocuments creates context from document list
func (b *BasicContextBuilder) BuildContextFromDocuments(ctx context.Context, docs []Document, config ContextConfig) (string, error) {
	if len(docs) == 0 {
		return "", nil
	}

	// Create a mock retrieval result
	scores := make([]float32, len(docs))
	for i := range scores {
		scores[i] = 1.0 // Default score
	}

	result := &RetrievalResult{
		Documents: docs,
		Scores:    scores,
	}

	return b.BuildContext(ctx, result, config)
}

// FormatDocument formats a single document according to template
func (b *BasicContextBuilder) FormatDocument(doc Document, templateStr string) (string, error) {
	if templateStr == "" {
		templateStr = b.config.Template
	}

	// Parse template
	tmpl, err := template.New("document").Parse(templateStr)
	if err != nil {
		return "", NewRAGErrorWithCause("failed to parse template", ErrorTypeValidation, err)
	}

	// Prepare template data
	data := map[string]interface{}{
		"ID":       doc.ID,
		"Content":  doc.Content,
		"Title":    doc.Title,
		"Source":   doc.Source,
		"Metadata": doc.Metadata,
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", NewRAGErrorWithCause("failed to execute template", ErrorTypeInternal, err)
	}

	return buf.String(), nil
}

// TruncateContext truncates context to fit within token limits
func (b *BasicContextBuilder) TruncateContext(context string, maxTokens int, strategy string) (string, error) {
	if b.tokenizer == nil {
		return "", NewRAGErrorWithOp("truncate_context", "tokenizer not available", ErrorTypeInternal)
	}

	currentTokens := b.tokenizer.CountTokens(context)
	if currentTokens <= maxTokens {
		return context, nil
	}

	switch strategy {
	case "head":
		return b.tokenizer.TruncateToTokens(context, maxTokens), nil
	case "tail":
		return b.truncateFromTail(context, maxTokens), nil
	case "middle":
		return b.truncateFromMiddle(context, maxTokens), nil
	default:
		return b.tokenizer.TruncateToTokens(context, maxTokens), nil
	}
}

// Helper methods

func (b *BasicContextBuilder) formatMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	var parts []string
	for key, value := range metadata {
		parts = append(parts, fmt.Sprintf("%s: %s", key, value))
	}
	
	sort.Strings(parts) // Sort for consistent output
	return strings.Join(parts, ", ")
}

func (b *BasicContextBuilder) truncateFromTail(context string, maxTokens int) string {
	// This is a simplified implementation
	// A more sophisticated version would preserve sentence boundaries
	words := strings.Fields(context)
	
	// Rough approximation: 1 token ≈ 0.75 words
	maxWords := int(float64(maxTokens) * 0.75)
	
	if len(words) <= maxWords {
		return context
	}
	
	startIdx := len(words) - maxWords
	if startIdx < 0 {
		startIdx = 0
	}
	
	return strings.Join(words[startIdx:], " ")
}

func (b *BasicContextBuilder) truncateFromMiddle(context string, maxTokens int) string {
	words := strings.Fields(context)
	maxWords := int(float64(maxTokens) * 0.75)
	
	if len(words) <= maxWords {
		return context
	}
	
	// Reserve space for truncation indicator
	truncationTokens := 10 // "... [truncated] ..."
	availableTokens := maxWords - truncationTokens
	
	if availableTokens < 20 {
		// Not enough space for meaningful context
		return b.tokenizer.TruncateToTokens(context, maxTokens)
	}
	
	halfTokens := availableTokens / 2
	
	prefix := strings.Join(words[:halfTokens], " ")
	suffix := strings.Join(words[len(words)-halfTokens:], " ")
	
	return prefix + "\n... [truncated] ...\n" + suffix
}

// TemplateContextBuilder provides advanced template-based context building
type TemplateContextBuilder struct {
	*BasicContextBuilder
	templates map[string]*template.Template
}

// NewTemplateContextBuilder creates a new template-based context builder
func NewTemplateContextBuilder(config *ContextConfig, tokenizer Tokenizer) *TemplateContextBuilder {
	basic := NewBasicContextBuilder(config, tokenizer)
	
	return &TemplateContextBuilder{
		BasicContextBuilder: basic,
		templates:          make(map[string]*template.Template),
	}
}

// RegisterTemplate registers a named template for document formatting
func (t *TemplateContextBuilder) RegisterTemplate(name string, templateStr string) error {
	tmpl, err := template.New(name).Parse(templateStr)
	if err != nil {
		return NewRAGErrorWithCause("failed to parse template", ErrorTypeValidation, err)
	}
	
	t.templates[name] = tmpl
	return nil
}

// FormatDocumentWithTemplate formats a document using a named template
func (t *TemplateContextBuilder) FormatDocumentWithTemplate(doc Document, templateName string) (string, error) {
	tmpl, exists := t.templates[templateName]
	if !exists {
		return "", NewRAGErrorWithOp("format_document", fmt.Sprintf("template '%s' not found", templateName), ErrorTypeNotFound)
	}

	data := map[string]interface{}{
		"ID":       doc.ID,
		"Content":  doc.Content,
		"Title":    doc.Title,
		"Source":   doc.Source,
		"Metadata": doc.Metadata,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", NewRAGErrorWithCause("failed to execute template", ErrorTypeInternal, err)
	}

	return buf.String(), nil
}

// SimpleTokenizer provides basic tokenization functionality
type SimpleTokenizer struct {
	model string
}

// NewSimpleTokenizer creates a new simple tokenizer
func NewSimpleTokenizer(model string) *SimpleTokenizer {
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	
	return &SimpleTokenizer{
		model: model,
	}
}

// CountTokens counts the number of tokens in text
// This is a simplified implementation - for production use, consider using
// a proper tokenizer library like tiktoken
func (s *SimpleTokenizer) CountTokens(text string) int {
	// Rough approximation: 1 token ≈ 4 characters for English text
	return len(text) / 4
}

// Tokenize splits text into tokens (simplified word-based splitting)
func (s *SimpleTokenizer) Tokenize(text string) []string {
	// This is a very basic implementation
	// A proper tokenizer would handle subword tokenization
	words := strings.Fields(text)
	
	var tokens []string
	for _, word := range words {
		// Split on punctuation as well
		tokens = append(tokens, s.splitPunctuation(word)...)
	}
	
	return tokens
}

// TruncateToTokens truncates text to specified token count
func (s *SimpleTokenizer) TruncateToTokens(text string, maxTokens int) string {
	tokens := s.Tokenize(text)
	if len(tokens) <= maxTokens {
		return text
	}
	
	// Join first maxTokens tokens
	return strings.Join(tokens[:maxTokens], " ")
}

// GetModel returns the tokenizer model name
func (s *SimpleTokenizer) GetModel() string {
	return s.model
}

// Helper method for basic punctuation splitting
func (s *SimpleTokenizer) splitPunctuation(word string) []string {
	// Very basic punctuation handling
	punctuation := ".,!?;:"
	
	var result []string
	current := ""
	
	for _, char := range word {
		if strings.ContainsRune(punctuation, char) {
			if current != "" {
				result = append(result, current)
				current = ""
			}
			result = append(result, string(char))
		} else {
			current += string(char)
		}
	}
	
	if current != "" {
		result = append(result, current)
	}
	
	return result
}