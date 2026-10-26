package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// CacheEntry holds cached response data, expiration, and semantic vector.
type CacheEntry struct {
	ResponseJSON []byte
	PromptText   string
	Vector       Vector
	ExpiresAt    time.Time
	CachedAt     time.Time
	Hits         int64
	Model        string
}

// MemoryCache is a high-performance in-memory cache supporting semantic similarity.
type MemoryCache struct {
	mu         sync.RWMutex
	entries    map[string]*CacheEntry
	ttl        time.Duration
	maxEntries int
	hits       uint64
	misses     uint64
}

// NewMemoryCache initializes an in-memory cache with standard TTL and default 10,000 capacity.
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	return NewMemoryCacheWithCapacity(defaultTTL, 10000)
}

// NewMemoryCacheWithCapacity initializes an in-memory cache with explicit capacity limit.
func NewMemoryCacheWithCapacity(defaultTTL time.Duration, maxEntries int) *MemoryCache {
	if maxEntries <= 0 {
		maxEntries = 10000
	}
	c := &MemoryCache{
		entries:    make(map[string]*CacheEntry),
		ttl:        defaultTTL,
		maxEntries: maxEntries,
	}
	go c.evictionWorker()
	return c
}

// NormalizePrompt produces a canonical representation for hash fingerprinting.
func NormalizePrompt(prompt string) string {
	normalized := strings.ToLower(prompt)
	normalized = strings.TrimSpace(normalized)
	normalized = strings.TrimRight(normalized, " ?.!")
	return normalized
}

// ComputeHash computes a deterministic SHA-256 fingerprint.
func ComputeHash(model, prompt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(strings.ToLower(model)))
	hasher.Write([]byte(":::"))
	hasher.Write([]byte(NormalizePrompt(prompt)))
	return hex.EncodeToString(hasher.Sum(nil))
}

// Get checks exact or normalized hash key concurrently using RLock.
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.entries[key]
	if !found || time.Now().After(entry.ExpiresAt) {
		atomic.AddUint64(&c.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&entry.Hits, 1)
	atomic.AddUint64(&c.hits, 1)
	return entry.ResponseJSON, true
}

// GetSemantic looks up responses using exact hash first, then parallel cosine similarity matching.
func (c *MemoryCache) GetSemantic(model, prompt string, threshold float64) ([]byte, bool, float64) {
	exactKey := ComputeHash(model, prompt)
	if data, ok := c.Get(exactKey); ok {
		return data, true, 1.0
	}

	// Fast exit if prompt is empty (no semantic vector lookup needed)
	if len(strings.TrimSpace(prompt)) == 0 {
		return nil, false, 0.0
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	queryVec := BuildSemanticVector(prompt)
	var bestMatch *CacheEntry
	highestSimilarity := 0.0

	now := time.Now()
	for _, entry := range c.entries {
		if entry.Model != model || now.After(entry.ExpiresAt) {
			continue
		}

		// Prevent false positives on safety-critical opposites (e.g. backup vs delete)
		if HasConflictingIntent(entry.PromptText, prompt) {
			continue
		}

		// Prevent false-positive collisions on distinct tech entities (e.g. Docker vs Kubernetes)
		if HasEntityConflict(entry.PromptText, prompt) {
			continue
		}

		// Require key-term overlap to prevent boilerplate prompt inflation
		if KeyTermJaccardOverlap(entry.PromptText, prompt) < 0.40 {
			continue
		}

		sim := CosineSimilarity(queryVec, entry.Vector)
		if sim > highestSimilarity && sim >= threshold {
			highestSimilarity = sim
			bestMatch = entry
		}
	}

	if bestMatch != nil {
		atomic.AddInt64(&bestMatch.Hits, 1)
		atomic.AddUint64(&c.hits, 1)
		return bestMatch.ResponseJSON, true, highestSimilarity
	}

	atomic.AddUint64(&c.misses, 1)
	return nil, false, 0.0
}

// Set stores a response with prompt text and semantic vector.
func (c *MemoryCache) Set(key string, response []byte, model string) {
	c.SetSemantic(key, "", response, model)
}

// SetSemantic stores a response along with prompt vector for semantic cosine lookups.
func (c *MemoryCache) SetSemantic(key, prompt string, response []byte, model string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Capacity bound: if at capacity, evict expired entry or oldest entry
	if len(c.entries) >= c.maxEntries {
		now := time.Now()
		var oldestKey string
		var oldestTime time.Time
		evicted := false

		for k, entry := range c.entries {
			if now.After(entry.ExpiresAt) {
				delete(c.entries, k)
				evicted = true
				break
			}
			if oldestKey == "" || entry.CachedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = entry.CachedAt
			}
		}
		if !evicted && oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = &CacheEntry{
		ResponseJSON: response,
		PromptText:   prompt,
		Vector:       BuildSemanticVector(prompt),
		ExpiresAt:    time.Now().Add(c.ttl),
		CachedAt:     time.Now(),
		Hits:         0,
		Model:        model,
	}
}

// Stats returns hit count, miss count, and total items in cache.
func (c *MemoryCache) Stats() (hits uint64, misses uint64, totalItems int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, len(c.entries)
}

// Clear clears all cache entries.
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*CacheEntry)
}

func (c *MemoryCache) evictionWorker() {
	ticker := time.NewTicker(2 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, entry := range c.entries {
			if now.After(entry.ExpiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
