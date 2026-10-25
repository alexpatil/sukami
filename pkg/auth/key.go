package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// APIKey represents a gateway client credential and metrics.
type APIKey struct {
	Key             string    `json:"key"`
	Name            string    `json:"name"`
	Tier            string    `json:"tier"`
	RateLimitPerMin int       `json:"rate_limit_per_min"`
	CreatedAt       time.Time `json:"created_at"`
	TotalRequests   int64     `json:"total_requests"`
	TokensSaved     int64     `json:"tokens_saved"`
}

// KeyStore defines the contract for managing API keys and rate limits (SOLID).
type KeyStore interface {
	Validate(key string) (*APIKey, bool)
	CreateKey(name, tier string) (*APIKey, error)
	RevokeKey(key string) bool
	ListKeys() []*APIKey
	RecordUsage(key string, savedTokens int)
	RecordTokenUsage(key string, usedTokens, savedTokens int)
}

// MemoryKeyStore is a thread-safe in-memory key repository.
type MemoryKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*APIKey
}

// NewMemoryKeyStore initializes the store with a default master demo key.
func NewMemoryKeyStore() *MemoryKeyStore {
	store := &MemoryKeyStore{
		keys: make(map[string]*APIKey),
	}

	// Seed default master test key with unlimited open-source access
	store.keys["sk-sukami-demo-12345"] = &APIKey{
		Key:             "sk-sukami-demo-12345",
		Name:            "Default Gateway Key",
		Tier:            "standard",
		RateLimitPerMin: 60,
		CreatedAt:       time.Now(),
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

// CreateKey generates a cryptographically secure client API key.
func (s *MemoryKeyStore) CreateKey(name, tier string) (*APIKey, error) {
	if name == "" {
		return nil, errors.New("key name is required")
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	limit := 60
	if tier == "high" || tier == "pro" {
		limit = 300
	} else if tier == "" {
		tier = "standard"
	}

	newKey := &APIKey{
		Key:             "sk-sukami-" + hex.EncodeToString(b),
		Name:            name,
		Tier:            tier,
		RateLimitPerMin: limit,
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

// ListKeys returns all active gateway keys.
func (s *MemoryKeyStore) ListKeys() []*APIKey {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*APIKey, 0, len(s.keys))
	for _, k := range s.keys {
		list = append(list, k)
	}
	return list
}

// RecordUsage increments client request and saved token counters.
func (s *MemoryKeyStore) RecordUsage(key string, savedTokens int) {
	s.RecordTokenUsage(key, 0, savedTokens)
}

// RecordTokenUsage atomically records both processed requests and saved optimization tokens.
func (s *MemoryKeyStore) RecordTokenUsage(key string, usedTokens, savedTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if k, exists := s.keys[key]; exists {
		k.TotalRequests++
		k.TokensSaved += int64(savedTokens)
	}
}
