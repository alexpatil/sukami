package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// APIKey represents a SaaS tenant's credential and quota.
type APIKey struct {
	Key             string    `json:"key"`
	Name            string    `json:"name"`
	Tier            string    `json:"tier"` // "free" | "pro"
	RateLimitPerMin int       `json:"rate_limit_per_min"`
	CreatedAt       time.Time `json:"created_at"`
	TotalRequests   int64     `json:"total_requests"`
	TokensSaved     int64     `json:"tokens_saved"`
	QuotaLimit      int64     `json:"quota_limit"` // e.g. 100,000 for free tier; -1 for unlimited pro
	QuotaUsed       int64     `json:"quota_used"`  // actual tokens consumed
}

// KeyStore defines the contract for managing API keys and trial quotas (SOLID).
type KeyStore interface {
	Validate(key string) (*APIKey, bool)
	CreateKey(name, tier string) (*APIKey, error)
	RevokeKey(key string) bool
	ListKeys() []*APIKey
	RecordUsage(key string, savedTokens int)
	RecordTokenUsage(key string, usedTokens, savedTokens int)
	CheckQuota(key string) (bool, string)
}

// MemoryKeyStore is a thread-safe in-memory key repository.
type MemoryKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*APIKey
}

// NewMemoryKeyStore initializes the store with a default master trial demo key.
func NewMemoryKeyStore() *MemoryKeyStore {
	store := &MemoryKeyStore{
		keys: make(map[string]*APIKey),
	}

	// Seed default master test key with active free trial quota (100k tokens)
	store.keys["sk-sukami-demo-12345"] = &APIKey{
		Key:             "sk-sukami-demo-12345",
		Name:            "Default Free Trial Key",
		Tier:            "free",
		RateLimitPerMin: 20,
		CreatedAt:       time.Now(),
		QuotaLimit:      100000,
		QuotaUsed:       14250,
		TokensSaved:     8430,
		TotalRequests:   12,
	}

	return store
}

// Validate checks whether the provided API key exists.
func (s *MemoryKeyStore) Validate(key string) (*APIKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apiKey, exists := s.keys[key]
	return apiKey, exists
}

// CheckQuota checks if key has remaining token quota.
func (s *MemoryKeyStore) CheckQuota(key string) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	k, exists := s.keys[key]
	if !exists {
		return false, "Invalid API Key"
	}

	// -1 or 0 indicates unlimited (pro tier)
	if k.QuotaLimit <= 0 {
		return true, ""
	}

	if k.QuotaUsed >= k.QuotaLimit {
		return false, fmt.Sprintf("Free trial quota exhausted (%d/%d tokens used). Upgrade to Pro for unlimited requests.", k.QuotaUsed, k.QuotaLimit)
	}

	return true, ""
}

// CreateKey generates a cryptographically secure tenant API key with quota.
func (s *MemoryKeyStore) CreateKey(name, tier string) (*APIKey, error) {
	if name == "" {
		return nil, errors.New("key name is required")
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	limit := 20
	var quota int64 = 100000 // Free trial: 100k tokens
	if tier == "pro" {
		limit = 120
		quota = -1 // Unlimited
	} else {
		tier = "free"
	}

	newKey := &APIKey{
		Key:             "sk-sukami-" + hex.EncodeToString(b),
		Name:            name,
		Tier:            tier,
		RateLimitPerMin: limit,
		QuotaLimit:      quota,
		QuotaUsed:       0,
		CreatedAt:       time.Now(),
	}

	s.mu.Lock()
	s.keys[newKey.Key] = newKey
	s.mu.Unlock()

	return newKey, nil
}

// RevokeKey removes an API key from active store.
func (s *MemoryKeyStore) RevokeKey(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.keys[key]; exists {
		delete(s.keys, key)
		return true
	}
	return false
}

// ListKeys returns all active tenant keys.
func (s *MemoryKeyStore) ListKeys() []*APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*APIKey, 0, len(s.keys))
	for _, k := range s.keys {
		list = append(list, k)
	}
	return list
}

// RecordUsage increments tenant request and saved token counters.
func (s *MemoryKeyStore) RecordUsage(key string, savedTokens int) {
	s.RecordTokenUsage(key, 0, savedTokens)
}

// RecordTokenUsage atomically records both consumed quota tokens and saved optimization tokens.
func (s *MemoryKeyStore) RecordTokenUsage(key string, usedTokens, savedTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if k, exists := s.keys[key]; exists {
		k.TotalRequests++
		k.QuotaUsed += int64(usedTokens)
		k.TokensSaved += int64(savedTokens)
	}
}
