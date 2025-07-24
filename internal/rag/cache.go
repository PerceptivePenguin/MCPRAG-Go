package rag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/vector"
)

// LRUCache implements the Cache interface using LRU eviction strategy
type LRUCache struct {
	mu          sync.RWMutex
	items       map[string]*cacheNode
	head        *cacheNode
	tail        *cacheNode
	maxSize     int
	currentSize int
	config      *CacheConfig
	metrics     CacheStats
	closed      bool
}

// cacheNode represents a node in the doubly linked list
type cacheNode struct {
	key      string
	entry    *CacheEntry
	prev     *cacheNode
	next     *cacheNode
	expireAt time.Time
}

// NewLRUCache creates a new LRU cache with the given configuration
func NewLRUCache(config *CacheConfig) (*LRUCache, error) {
	if config == nil {
		config = DefaultCacheConfig()
	}

	if config.MaxSize <= 0 {
		return nil, ErrInvalidCacheSize
	}

	cache := &LRUCache{
		items:   make(map[string]*cacheNode),
		maxSize: config.MaxSize,
		config:  config,
		metrics: CacheStats{
			MaxSize: config.MaxSize,
		},
	}

	// Initialize doubly linked list with dummy head and tail
	cache.head = &cacheNode{}
	cache.tail = &cacheNode{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	// Load from persistence if configured
	if config.PersistPath != "" {
		if err := cache.loadFromDisk(); err != nil {
			// Log error but don't fail initialization
			fmt.Printf("Warning: failed to load cache from disk: %v\n", err)
		}
	}

	return cache, nil
}

// Get retrieves a cached embedding by key
func (c *LRUCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrCacheClosed
	}

	node, exists := c.items[key]
	if !exists {
		c.metrics.Misses++
		c.updateHitRate()
		return nil, ErrCacheKeyNotFound
	}

	// Check if entry has expired
	if c.config.TTL > 0 && time.Now().After(node.expireAt) {
		c.removeNode(node)
		delete(c.items, key)
		c.currentSize--
		c.metrics.Misses++
		c.updateHitRate()
		return nil, ErrCacheKeyNotFound
	}

	// Move to head (most recently used)
	c.moveToHead(node)
	
	// Update access information
	node.entry.AccessedAt = time.Now()
	node.entry.AccessCount++

	c.metrics.Hits++
	c.updateHitRate()
	
	return node.entry, nil
}

// Set stores an embedding in the cache
func (c *LRUCache) Set(ctx context.Context, key string, vector vector.Vector, metadata map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	now := time.Now()
	var expireAt time.Time
	if c.config.TTL > 0 {
		expireAt = now.Add(c.config.TTL)
	}

	// Check if key already exists
	if node, exists := c.items[key]; exists {
		// Update existing entry
		node.entry.Vector = vector
		node.entry.Metadata = metadata
		node.entry.AccessedAt = now
		node.entry.AccessCount++
		node.expireAt = expireAt
		c.moveToHead(node)
		return nil
	}

	// Create new entry
	entry := &CacheEntry{
		Key:         key,
		Vector:      vector,
		Metadata:    metadata,
		CreatedAt:   now,
		AccessedAt:  now,
		AccessCount: 1,
		Size:        c.calculateEntrySize(key, vector, metadata),
	}

	node := &cacheNode{
		key:      key,
		entry:    entry,
		expireAt: expireAt,
	}

	// Add to head
	c.addToHead(node)
	c.items[key] = node
	c.currentSize++
	c.metrics.Size = c.currentSize
	c.metrics.MemoryUsage += int64(entry.Size)

	// Evict if necessary
	for c.currentSize > c.maxSize {
		tail := c.removeTail()
		if tail != nil {
			delete(c.items, tail.key)
			c.currentSize--
			c.metrics.Size = c.currentSize
			c.metrics.MemoryUsage -= int64(tail.entry.Size)
		}
	}

	return nil
}

// Delete removes an entry from the cache
func (c *LRUCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	node, exists := c.items[key]
	if !exists {
		return ErrCacheKeyNotFound
	}

	c.removeNode(node)
	delete(c.items, key)
	c.currentSize--
	c.metrics.Size = c.currentSize
	c.metrics.MemoryUsage -= int64(node.entry.Size)

	return nil
}

// Clear removes all entries from the cache
func (c *LRUCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrCacheClosed
	}

	c.items = make(map[string]*cacheNode)
	c.head.next = c.tail
	c.tail.prev = c.head
	c.currentSize = 0
	c.metrics.Size = 0
	c.metrics.MemoryUsage = 0

	return nil
}

// Size returns the number of cached entries
func (c *LRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentSize
}

// Stats returns cache statistics
func (c *LRUCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

// Close releases any resources held by the cache
func (c *LRUCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	// Persist to disk if configured
	if c.config.PersistPath != "" {
		if err := c.saveToDisk(); err != nil {
			return fmt.Errorf("failed to persist cache: %w", err)
		}
	}

	c.closed = true
	return nil
}

// Helper methods for doubly linked list operations

func (c *LRUCache) addToHead(node *cacheNode) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

func (c *LRUCache) removeNode(node *cacheNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

func (c *LRUCache) moveToHead(node *cacheNode) {
	c.removeNode(node)
	c.addToHead(node)
}

func (c *LRUCache) removeTail() *cacheNode {
	lastNode := c.tail.prev
	if lastNode == c.head {
		return nil
	}
	c.removeNode(lastNode)
	return lastNode
}

func (c *LRUCache) updateHitRate() {
	total := c.metrics.Hits + c.metrics.Misses
	if total > 0 {
		c.metrics.HitRate = float64(c.metrics.Hits) / float64(total)
	}
}

func (c *LRUCache) calculateEntrySize(key string, vector vector.Vector, metadata map[string]string) int {
	size := len(key) + len(vector)*4 // float32 is 4 bytes
	for k, v := range metadata {
		size += len(k) + len(v)
	}
	return size
}

// Persistence methods

func (c *LRUCache) saveToDisk() error {
	if c.config.PersistPath == "" {
		return nil
	}

	// Create a serializable representation
	data := make(map[string]*CacheEntry)
	for key, node := range c.items {
		if c.config.TTL == 0 || time.Now().Before(node.expireAt) {
			data[key] = node.entry
		}
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	return ioutil.WriteFile(c.config.PersistPath, jsonData, 0644)
}

func (c *LRUCache) loadFromDisk() error {
	if c.config.PersistPath == "" {
		return nil
	}

	if _, err := os.Stat(c.config.PersistPath); os.IsNotExist(err) {
		return nil // File doesn't exist, that's OK
	}

	jsonData, err := ioutil.ReadFile(c.config.PersistPath)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var data map[string]*CacheEntry
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	// Restore entries to cache
	for key, entry := range data {
		if len(c.items) >= c.maxSize {
			break // Don't exceed max size when loading
		}

		node := &cacheNode{
			key:   key,
			entry: entry,
		}

		c.addToHead(node)
		c.items[key] = node
		c.currentSize++
		c.metrics.Size = c.currentSize
		c.metrics.MemoryUsage += int64(entry.Size)
	}

	return nil
}

// GenerateCacheKey generates a cache key from text content
func GenerateCacheKey(text string, model string) string {
	h := sha256.New()
	h.Write([]byte(text))
	h.Write([]byte(model))
	return hex.EncodeToString(h.Sum(nil))
}

// EvictExpiredEntries removes expired entries from the cache
func (c *LRUCache) EvictExpiredEntries(ctx context.Context) int {
	if c.config.TTL == 0 {
		return 0 // No TTL configured
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0
	}

	now := time.Now()
	evicted := 0
	
	// Iterate through all entries and remove expired ones
	for key, node := range c.items {
		if now.After(node.expireAt) {
			c.removeNode(node)
			delete(c.items, key)
			c.currentSize--
			c.metrics.Size = c.currentSize
			c.metrics.MemoryUsage -= int64(node.entry.Size)
			evicted++
		}
	}

	return evicted
}