package rag

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/PerceptivePenguin/MCPRAG-Go/internal/vector"
)

func TestNewLRUCache(t *testing.T) {
	tests := []struct {
		name        string
		config      *CacheConfig
		expectError bool
	}{
		{
			name:        "default config",
			config:      nil,
			expectError: false,
		},
		{
			name: "valid config",
			config: &CacheConfig{
				Enabled:  true,
				MaxSize:  100,
				TTL:      time.Hour,
				Strategy: CacheLRU,
			},
			expectError: false,
		},
		{
			name: "invalid config - zero size",
			config: &CacheConfig{
				Enabled:  true,
				MaxSize:  0,
				Strategy: CacheLRU,
			},
			expectError: true,
		},
		{
			name: "invalid config - negative size",
			config: &CacheConfig{
				Enabled:  true,
				MaxSize:  -1,
				Strategy: CacheLRU,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, err := NewLRUCache(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if cache == nil {
					t.Error("expected cache to be created")
				}
				if cache != nil {
					cache.Close()
				}
			}
		})
	}
}

func TestLRUCache_SetAndGet(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		MaxSize:  3,
		Strategy: CacheLRU,
	}
	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	testVector := vector.Vector{1.0, 2.0, 3.0}
	testMetadata := map[string]string{"source": "test"}

	// Test Set
	err = cache.Set(ctx, "key1", testVector, testMetadata)
	if err != nil {
		t.Errorf("failed to set cache entry: %v", err)
	}

	// Test Get
	entry, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("failed to get cache entry: %v", err)
	}
	if entry == nil {
		t.Error("expected cache entry but got nil")
		return
	}

	// Verify entry contents
	if entry.Key != "key1" {
		t.Errorf("expected key 'key1', got '%s'", entry.Key)
	}
	if len(entry.Vector) != len(testVector) {
		t.Errorf("expected vector length %d, got %d", len(testVector), len(entry.Vector))
	}
	for i, v := range testVector {
		if entry.Vector[i] != v {
			t.Errorf("expected vector[%d] = %f, got %f", i, v, entry.Vector[i])
		}
	}
	if entry.Metadata["source"] != "test" {
		t.Errorf("expected metadata source 'test', got '%s'", entry.Metadata["source"])
	}
}

func TestLRUCache_GetNonExistent(t *testing.T) {
	cache, err := NewLRUCache(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	entry, err := cache.Get(ctx, "nonexistent")
	if entry != nil {
		t.Error("expected nil entry for nonexistent key")
	}
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
	if err != ErrCacheKeyNotFound {
		t.Errorf("expected ErrCacheKeyNotFound, got %v", err)
	}

	// Check that miss was recorded
	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
}

func TestLRUCache_LRUEviction(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		MaxSize:  2,
		Strategy: CacheLRU,
	}
	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	testVector1 := vector.Vector{1.0}
	testVector2 := vector.Vector{2.0}
	testVector3 := vector.Vector{3.0}

	// Add two entries
	cache.Set(ctx, "key1", testVector1, nil)
	cache.Set(ctx, "key2", testVector2, nil)

	// Verify both are present
	if _, err := cache.Get(ctx, "key1"); err != nil {
		t.Error("key1 should be present")
	}
	if _, err := cache.Get(ctx, "key2"); err != nil {
		t.Error("key2 should be present")
	}

	// Add third entry, should evict key1 (least recently used)
	cache.Set(ctx, "key3", testVector3, nil)

	// key1 should be evicted
	if _, err := cache.Get(ctx, "key1"); err == nil {
		t.Error("key1 should have been evicted")
	}

	// key2 and key3 should still be present
	if _, err := cache.Get(ctx, "key2"); err != nil {
		t.Error("key2 should still be present")
	}
	if _, err := cache.Get(ctx, "key3"); err != nil {
		t.Error("key3 should be present")
	}

	// Verify size
	if cache.Size() != 2 {
		t.Errorf("expected cache size 2, got %d", cache.Size())
	}
}

func TestLRUCache_Update(t *testing.T) {
	cache, err := NewLRUCache(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	testVector1 := vector.Vector{1.0}
	testVector2 := vector.Vector{2.0}

	// Set initial entry
	cache.Set(ctx, "key1", testVector1, map[string]string{"version": "1"})

	// Update with new vector and metadata
	cache.Set(ctx, "key1", testVector2, map[string]string{"version": "2"})

	// Get and verify updated entry
	entry, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("failed to get updated entry: %v", err)
	}
	if entry.Vector[0] != 2.0 {
		t.Errorf("expected updated vector value 2.0, got %f", entry.Vector[0])
	}
	if entry.Metadata["version"] != "2" {
		t.Errorf("expected updated metadata version '2', got '%s'", entry.Metadata["version"])
	}

	// Should still have size 1
	if cache.Size() != 1 {
		t.Errorf("expected cache size 1, got %d", cache.Size())
	}
}

func TestLRUCache_Delete(t *testing.T) {
	cache, err := NewLRUCache(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	testVector := vector.Vector{1.0}

	// Set entry
	cache.Set(ctx, "key1", testVector, nil)
	if cache.Size() != 1 {
		t.Errorf("expected size 1 after set, got %d", cache.Size())
	}

	// Delete entry
	err = cache.Delete(ctx, "key1")
	if err != nil {
		t.Errorf("failed to delete entry: %v", err)
	}

	// Verify entry is gone
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after delete, got %d", cache.Size())
	}

	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("expected error when getting deleted entry")
	}

	// Delete non-existent key should return error
	err = cache.Delete(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error when deleting non-existent key")
	}
}

func TestLRUCache_Clear(t *testing.T) {
	cache, err := NewLRUCache(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	testVector := vector.Vector{1.0}

	// Add multiple entries
	cache.Set(ctx, "key1", testVector, nil)
	cache.Set(ctx, "key2", testVector, nil)
	cache.Set(ctx, "key3", testVector, nil)

	if cache.Size() != 3 {
		t.Errorf("expected size 3, got %d", cache.Size())
	}

	// Clear cache
	err = cache.Clear(ctx)
	if err != nil {
		t.Errorf("failed to clear cache: %v", err)
	}

	// Verify cache is empty
	if cache.Size() != 0 {
		t.Errorf("expected size 0 after clear, got %d", cache.Size())
	}

	// Verify entries are gone
	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("expected error when getting entry from cleared cache")
	}
}

func TestLRUCache_TTL(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		MaxSize:  10,
		TTL:      100 * time.Millisecond, // Very short TTL for testing
		Strategy: CacheLRU,
	}
	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	testVector := vector.Vector{1.0}

	// Set entry
	cache.Set(ctx, "key1", testVector, nil)

	// Should be retrievable immediately
	_, err = cache.Get(ctx, "key1")
	if err != nil {
		t.Errorf("entry should be available immediately: %v", err)
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should not be retrievable after TTL
	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("entry should have expired")
	}
}

func TestLRUCache_Stats(t *testing.T) {
	cache, err := NewLRUCache(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	testVector := vector.Vector{1.0, 2.0, 3.0}

	// Initial stats
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Size != 0 {
		t.Error("initial stats should be zero")
	}

	// Add entry and get it (hit)
	cache.Set(ctx, "key1", testVector, nil)
	cache.Get(ctx, "key1")

	// Try to get non-existent entry (miss)
	cache.Get(ctx, "nonexistent")

	// Check updated stats
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.Size != 1 {
		t.Errorf("expected size 1, got %d", stats.Size)
	}
	if stats.HitRate != 0.5 {
		t.Errorf("expected hit rate 0.5, got %f", stats.HitRate)
	}
}

func TestLRUCache_ClosedOperations(t *testing.T) {
	cache, err := NewLRUCache(DefaultCacheConfig())
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	// Close the cache
	cache.Close()

	ctx := context.Background()
	testVector := vector.Vector{1.0}

	// All operations should return error after close
	err = cache.Set(ctx, "key1", testVector, nil)
	if err != ErrCacheClosed {
		t.Errorf("expected ErrCacheClosed, got %v", err)
	}

	_, err = cache.Get(ctx, "key1")
	if err != ErrCacheClosed {
		t.Errorf("expected ErrCacheClosed, got %v", err)
	}

	err = cache.Delete(ctx, "key1")
	if err != ErrCacheClosed {
		t.Errorf("expected ErrCacheClosed, got %v", err)
	}

	err = cache.Clear(ctx)
	if err != ErrCacheClosed {
		t.Errorf("expected ErrCacheClosed, got %v", err)
	}
}

func TestLRUCache_Persistence(t *testing.T) {
	// Create temporary file for persistence
	tmpFile, err := os.CreateTemp("", "cache_test_*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	config := &CacheConfig{
		Enabled:     true,
		MaxSize:     10,
		Strategy:    CacheLRU,
		PersistPath: tmpFile.Name(),
	}

	ctx := context.Background()
	testVector := vector.Vector{1.0, 2.0, 3.0}

	// Create cache and add data
	cache1, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	cache1.Set(ctx, "key1", testVector, map[string]string{"test": "data"})
	cache1.Set(ctx, "key2", testVector, nil)

	// Close cache (should persist data)
	cache1.Close()

	// Create new cache with same persistence file
	cache2, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("failed to create second cache: %v", err)
	}
	defer cache2.Close()

	// Data should be loaded from persistence
	entry, err := cache2.Get(ctx, "key1")
	if err != nil {
		t.Errorf("failed to get persisted entry: %v", err)
	}
	if entry == nil {
		t.Error("expected persisted entry")
		return
	}

	if entry.Metadata["test"] != "data" {
		t.Errorf("expected metadata 'data', got '%s'", entry.Metadata["test"])
	}
}

func TestGenerateCacheKey(t *testing.T) {
	key1 := GenerateCacheKey("hello world", "model1")
	key2 := GenerateCacheKey("hello world", "model1")
	key3 := GenerateCacheKey("hello world", "model2")
	key4 := GenerateCacheKey("goodbye world", "model1")

	// Same input should generate same key
	if key1 != key2 {
		t.Error("same input should generate same key")
	}

	// Different model should generate different key
	if key1 == key3 {
		t.Error("different model should generate different key")
	}

	// Different text should generate different key
	if key1 == key4 {
		t.Error("different text should generate different key")
	}

	// Keys should be hex encoded strings
	if len(key1) != 64 { // SHA256 hex = 64 characters
		t.Errorf("expected key length 64, got %d", len(key1))
	}
}

func TestLRUCache_EvictExpiredEntries(t *testing.T) {
	config := &CacheConfig{
		Enabled:  true,
		MaxSize:  10,
		TTL:      50 * time.Millisecond,
		Strategy: CacheLRU,
	}
	cache, err := NewLRUCache(config)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	testVector := vector.Vector{1.0}

	// Add entries
	cache.Set(ctx, "key1", testVector, nil)
	cache.Set(ctx, "key2", testVector, nil)

	// Wait for expiration
	time.Sleep(60 * time.Millisecond)

	// Add one more entry (should not be expired)
	cache.Set(ctx, "key3", testVector, nil)

	// Evict expired entries
	evicted := cache.EvictExpiredEntries(ctx)

	if evicted != 2 {
		t.Errorf("expected 2 evicted entries, got %d", evicted)
	}

	// key3 should still be present
	_, err = cache.Get(ctx, "key3")
	if err != nil {
		t.Error("key3 should still be present")
	}

	// key1 and key2 should be gone
	_, err = cache.Get(ctx, "key1")
	if err == nil {
		t.Error("key1 should have been evicted")
	}
}