package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// CacheEntry holds cached response data, expiration, and semantic vector.
type CacheEntry struct {
	ResponseJSON []byte
	PromptText   string
	Vector       Vector
	ExpiresAt    time.Time
	CachedAt     time.Time
	Hits         int
	Model        string
}

// MemoryCache is a high-performance in-memory cache supporting semantic similarity.
type MemoryCache struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	ttl     time.Duration
	hits    uint64
	misses  uint64
}

// NewMemoryCache initializes an in-memory cache with standard TTL.
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	c := &MemoryCache{
		entries: make(map[string]*CacheEntry),
		ttl:     defaultTTL,
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

// Get checks exact or normalized hash key.
func (c *MemoryCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, found := c.entries[key]
	if !found {
		c.misses++
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, key)
		c.misses++
		return nil, false
	}

	entry.Hits++
	c.hits++
	return entry.ResponseJSON, true
}

// GetSemantic looks up responses using exact hash first, then cosine similarity matching.
func (c *MemoryCache) GetSemantic(model, prompt string, threshold float64) ([]byte, bool, float64) {
	exactKey := ComputeHash(model, prompt)
	if data, ok := c.Get(exactKey); ok {
		return data, true, 1.0
	}

	c.mu.Lock()
	defer c.mu.Unlock()

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

		sim := CosineSimilarity(queryVec, entry.Vector)
		if sim > highestSimilarity && sim >= threshold {
			highestSimilarity = sim
			bestMatch = entry
		}
	}

	if bestMatch != nil {
		bestMatch.Hits++
		c.hits++
		return bestMatch.ResponseJSON, true, highestSimilarity
	}

	c.misses++
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
