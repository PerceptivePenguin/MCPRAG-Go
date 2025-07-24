package rag

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TextChunker implements document chunking functionality
type TextChunker struct {
	tokenizer Tokenizer
}

// NewTextChunker creates a new text chunker
func NewTextChunker(tokenizer Tokenizer) *TextChunker {
	return &TextChunker{
		tokenizer: tokenizer,
	}
}

// ChunkDocument splits a document into chunks based on the specified options
func (c *TextChunker) ChunkDocument(ctx context.Context, doc Document, options ChunkingOptions) ([]Chunk, error) {
	if doc.Content == "" {
		return nil, ErrDocumentEmpty.WithOperation("chunk_document")
	}
	
	if options.MaxChunkSize <= 0 {
		return nil, ErrInvalidChunkSize.WithOperation("chunk_document")
	}
	
	if options.Overlap < 0 || options.Overlap >= options.MaxChunkSize {
		return nil, ErrInvalidOverlap.WithOperation("chunk_document")
	}
	
	var chunks []Chunk
	var err error
	
	switch options.Strategy {
	case ChunkByTokens:
		chunks, err = c.chunkByTokens(doc, options)
	case ChunkBySentences:
		chunks, err = c.chunkBySentences(doc, options)
	case ChunkByParagraphs:
		chunks, err = c.chunkByParagraphs(doc, options)
	case ChunkByFixedSize:
		chunks, err = c.chunkByFixedSize(doc, options)
	case ChunkBySemantic:
		chunks, err = c.chunkBySemantic(doc, options)
	default:
		return nil, ErrInvalidStrategy.WithOperation("chunk_document").WithDetails(map[string]string{
			"strategy": string(options.Strategy),
		})
	}
	
	if err != nil {
		return nil, err
	}
	
	// Add metadata to chunks
	for i := range chunks {
		chunks[i].DocumentID = doc.ID
		chunks[i].Index = i
		chunks[i].ID = fmt.Sprintf("%s_chunk_%d", doc.ID, i)
		
		if chunks[i].Metadata == nil {
			chunks[i].Metadata = make(map[string]string)
		}
		
		// Copy document metadata to chunks
		for k, v := range doc.Metadata {
			chunks[i].Metadata[k] = v
		}
		
		chunks[i].Metadata["chunk_strategy"] = string(options.Strategy)
		chunks[i].Metadata["source_document"] = doc.ID
		
		if c.tokenizer != nil {
			chunks[i].TokenCount = c.tokenizer.CountTokens(chunks[i].Content)
		}
	}
	
	return chunks, nil
}

// chunkByTokens splits text based on token count
func (c *TextChunker) chunkByTokens(doc Document, options ChunkingOptions) ([]Chunk, error) {
	if c.tokenizer == nil {
		return nil, NewRAGErrorWithOp("chunk_by_tokens", "tokenizer is required for token-based chunking", ErrorTypeValidation)
	}
	
	text := doc.Content
	maxTokens := options.MaxChunkSize
	overlapTokens := options.Overlap
	
	var chunks []Chunk
	startPos := 0
	
	for startPos < len(text) {
		// Find the end position for this chunk
		endPos := len(text)
		chunkText := text[startPos:]
		
		// If chunk is too large, truncate to max tokens
		if c.tokenizer.CountTokens(chunkText) > maxTokens {
			chunkText = c.tokenizer.TruncateToTokens(chunkText, maxTokens)
			endPos = startPos + len(chunkText)
		}
		
		if chunkText == "" {
			break
		}
		
		chunk := Chunk{
			Content:  strings.TrimSpace(chunkText),
			StartPos: startPos,
			EndPos:   endPos,
		}
		
		chunks = append(chunks, chunk)
		
		// Calculate next start position with overlap
		if endPos >= len(text) {
			break
		}
		
		// Find overlap start position
		if overlapTokens > 0 && len(chunks) > 0 {
			overlapText := c.tokenizer.TruncateToTokens(chunkText, overlapTokens)
			overlapStart := strings.LastIndex(chunkText, overlapText)
			if overlapStart > 0 {
				startPos = startPos + overlapStart
			} else {
				startPos = endPos
			}
		} else {
			startPos = endPos
		}
	}
	
	return chunks, nil
}

// chunkBySentences splits text by sentences
func (c *TextChunker) chunkBySentences(doc Document, options ChunkingOptions) ([]Chunk, error) {
	sentences := c.splitIntoSentences(doc.Content)
	if len(sentences) == 0 {
		return nil, ErrDocumentEmpty.WithOperation("chunk_by_sentences")
	}
	
	var chunks []Chunk
	var currentChunk strings.Builder
	var currentSentences []string
	startPos := 0
	
	for i, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		
		// Check if adding this sentence would exceed the limit
		testContent := currentChunk.String()
		if testContent != "" {
			testContent += " "
		}
		testContent += sentence
		
		shouldSplit := false
		if c.tokenizer != nil {
			shouldSplit = c.tokenizer.CountTokens(testContent) > options.MaxChunkSize
		} else {
			shouldSplit = len(testContent) > options.MaxChunkSize
		}
		
		if shouldSplit && currentChunk.Len() > 0 {
			// Create chunk from current sentences
			chunk := Chunk{
				Content:  strings.TrimSpace(currentChunk.String()),
				StartPos: startPos,
				EndPos:   startPos + currentChunk.Len(),
			}
			chunks = append(chunks, chunk)
			
			// Handle overlap
			currentChunk.Reset()
			currentSentences = nil
			
			if options.Overlap > 0 && len(chunks) > 0 {
				// Include some previous sentences for overlap
				overlapSentences := c.getOverlapSentences(sentences[:i], options.Overlap)
				for _, overlapSent := range overlapSentences {
					if currentChunk.Len() > 0 {
						currentChunk.WriteString(" ")
					}
					currentChunk.WriteString(overlapSent)
					currentSentences = append(currentSentences, overlapSent)
				}
			}
			
			startPos = chunk.EndPos
		}
		
		// Add current sentence
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
		currentSentences = append(currentSentences, sentence)
	}
	
	// Add final chunk if there's remaining content
	if currentChunk.Len() > 0 {
		chunk := Chunk{
			Content:  strings.TrimSpace(currentChunk.String()),
			StartPos: startPos,
			EndPos:   len(doc.Content),
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// chunkByParagraphs splits text by paragraphs
func (c *TextChunker) chunkByParagraphs(doc Document, options ChunkingOptions) ([]Chunk, error) {
	separators := options.Separators
	if len(separators) == 0 {
		separators = []string{"\n\n", "\n"}
	}
	
	paragraphs := c.splitBySeparators(doc.Content, separators)
	if len(paragraphs) == 0 {
		return nil, ErrDocumentEmpty.WithOperation("chunk_by_paragraphs")
	}
	
	var chunks []Chunk
	var currentContent strings.Builder
	startPos := 0
	
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		
		// Check if adding this paragraph would exceed the limit
		testContent := currentContent.String()
		if testContent != "" {
			testContent += "\n\n"
		}
		testContent += paragraph
		
		shouldSplit := false
		if c.tokenizer != nil {
			shouldSplit = c.tokenizer.CountTokens(testContent) > options.MaxChunkSize
		} else {
			shouldSplit = len(testContent) > options.MaxChunkSize
		}
		
		if shouldSplit && currentContent.Len() > 0 {
			// Create chunk from current content
			chunk := Chunk{
				Content:  strings.TrimSpace(currentContent.String()),
				StartPos: startPos,
				EndPos:   startPos + currentContent.Len(),
			}
			chunks = append(chunks, chunk)
			
			// Handle overlap
			if options.Overlap > 0 {
				overlapContent := c.getOverlapContent(currentContent.String(), options.Overlap)
				currentContent.Reset()
				currentContent.WriteString(overlapContent)
			} else {
				currentContent.Reset()
			}
			
			startPos = chunk.EndPos
		}
		
		// Add current paragraph
		if currentContent.Len() > 0 {
			currentContent.WriteString("\n\n")
		}
		currentContent.WriteString(paragraph)
	}
	
	// Add final chunk if there's remaining content
	if currentContent.Len() > 0 {
		chunk := Chunk{
			Content:  strings.TrimSpace(currentContent.String()),
			StartPos: startPos,
			EndPos:   len(doc.Content),
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// chunkByFixedSize splits text into fixed-size chunks
func (c *TextChunker) chunkByFixedSize(doc Document, options ChunkingOptions) ([]Chunk, error) {
	text := doc.Content
	chunkSize := options.MaxChunkSize
	overlap := options.Overlap
	
	var chunks []Chunk
	
	for i := 0; i < len(text); i += chunkSize - overlap {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		
		// Ensure we don't split in the middle of a UTF-8 character
		if end < len(text) {
			for end > i && !utf8.ValidString(text[i:end]) {
				end--
			}
		}
		
		chunkText := text[i:end]
		if strings.TrimSpace(chunkText) == "" {
			continue
		}
		
		chunk := Chunk{
			Content:  strings.TrimSpace(chunkText),
			StartPos: i,
			EndPos:   end,
		}
		
		chunks = append(chunks, chunk)
		
		if end >= len(text) {
			break
		}
	}
	
	return chunks, nil
}

// chunkBySemantic splits text based on semantic boundaries (simplified implementation)
func (c *TextChunker) chunkBySemantic(doc Document, options ChunkingOptions) ([]Chunk, error) {
	// This is a simplified implementation
	// A full semantic chunking would use NLP models to detect topic boundaries
	
	// For now, combine sentence and paragraph chunking with topic detection heuristics
	sentences := c.splitIntoSentences(doc.Content)
	if len(sentences) == 0 {
		return nil, ErrDocumentEmpty.WithOperation("chunk_by_semantic")
	}
	
	var chunks []Chunk
	var currentChunk strings.Builder
	startPos := 0
	lastTopicScore := 0.0
	
	for i, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}
		
		// Calculate topic coherence score (simplified)
		topicScore := c.calculateTopicCoherence(currentChunk.String(), sentence)
		
		// Detect topic boundary
		isTopicBoundary := false
		if i > 0 && topicScore < lastTopicScore*0.7 { // Threshold for topic change
			isTopicBoundary = true
		}
		
		// Check size limits
		testContent := currentChunk.String()
		if testContent != "" {
			testContent += " "
		}
		testContent += sentence
		
		exceedsSize := false
		if c.tokenizer != nil {
			exceedsSize = c.tokenizer.CountTokens(testContent) > options.MaxChunkSize
		} else {
			exceedsSize = len(testContent) > options.MaxChunkSize
		}
		
		if (isTopicBoundary || exceedsSize) && currentChunk.Len() > 0 {
			// Create chunk
			chunk := Chunk{
				Content:  strings.TrimSpace(currentChunk.String()),
				StartPos: startPos,
				EndPos:   startPos + currentChunk.Len(),
			}
			chunks = append(chunks, chunk)
			
			// Reset for next chunk
			currentChunk.Reset()
			startPos = chunk.EndPos
		}
		
		// Add current sentence
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)
		lastTopicScore = topicScore
	}
	
	// Add final chunk
	if currentChunk.Len() > 0 {
		chunk := Chunk{
			Content:  strings.TrimSpace(currentChunk.String()),
			StartPos: startPos,
			EndPos:   len(doc.Content),
		}
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// Helper methods

func (c *TextChunker) splitIntoSentences(text string) []string {
	// Simple sentence splitting using regex
	// This could be improved with a proper NLP library
	sentenceRegex := regexp.MustCompile(`[.!?]+\s+`)
	
	// Split by sentence endings
	parts := sentenceRegex.Split(text, -1)
	
	// Find the delimiters to reconstruct sentences properly
	delims := sentenceRegex.FindAllString(text, -1)
	
	var sentences []string
	for i, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		
		sentence := part
		if i < len(delims) {
			sentence += strings.TrimSpace(delims[i])
		}
		
		sentences = append(sentences, sentence)
	}
	
	return sentences
}

func (c *TextChunker) splitBySeparators(text string, separators []string) []string {
	parts := []string{text}
	
	for _, sep := range separators {
		var newParts []string
		for _, part := range parts {
			subParts := strings.Split(part, sep)
			newParts = append(newParts, subParts...)
		}
		parts = newParts
	}
	
	// Filter out empty parts
	var result []string
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			result = append(result, part)
		}
	}
	
	return result
}

func (c *TextChunker) getOverlapSentences(sentences []string, overlapSize int) []string {
	if len(sentences) == 0 || overlapSize <= 0 {
		return nil
	}
	
	// Get last few sentences that fit within overlap size
	var overlap []string
	var currentSize int
	
	for i := len(sentences) - 1; i >= 0; i-- {
		sentence := sentences[i]
		var sentenceSize int
		
		if c.tokenizer != nil {
			sentenceSize = c.tokenizer.CountTokens(sentence)
		} else {
			sentenceSize = len(sentence)
		}
		
		if currentSize+sentenceSize > overlapSize {
			break
		}
		
		overlap = append([]string{sentence}, overlap...)
		currentSize += sentenceSize
	}
	
	return overlap
}

func (c *TextChunker) getOverlapContent(content string, overlapSize int) string {
	if content == "" || overlapSize <= 0 {
		return ""
	}
	
	if c.tokenizer != nil {
		tokens := c.tokenizer.Tokenize(content)
		if len(tokens) <= overlapSize {
			return content
		}
		
		overlapTokens := tokens[len(tokens)-overlapSize:]
		return strings.Join(overlapTokens, " ")
	}
	
	// Fallback to character-based overlap
	if len(content) <= overlapSize {
		return content
	}
	
	return content[len(content)-overlapSize:]
}

func (c *TextChunker) calculateTopicCoherence(currentText, newSentence string) float64 {
	// Simplified topic coherence calculation
	// In a real implementation, this would use embeddings or topic models
	
	if currentText == "" {
		return 1.0
	}
	
	// Calculate word overlap as a simple coherence measure
	currentWords := c.extractWords(strings.ToLower(currentText))
	newWords := c.extractWords(strings.ToLower(newSentence))
	
	if len(newWords) == 0 {
		return 0.0
	}
	
	overlap := 0
	for word := range newWords {
		if currentWords[word] {
			overlap++
		}
	}
	
	return float64(overlap) / float64(len(newWords))
}

func (c *TextChunker) extractWords(text string) map[string]bool {
	words := make(map[string]bool)
	
	// Simple word extraction
	for _, word := range strings.FieldsFunc(text, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}) {
		word = strings.TrimSpace(word)
		if len(word) > 2 { // Filter out very short words
			words[word] = true
		}
	}
	
	return words
}