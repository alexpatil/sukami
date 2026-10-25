package auth

import (
	"strings"
	"testing"
)

func TestKeyStore_CreateAndValidate(t *testing.T) {
	store := NewMemoryKeyStore()

	key, err := store.CreateKey("Test Customer", "standard")
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	if !strings.HasPrefix(key.Key, "sk-sukami-") {
		t.Errorf("Invalid key prefix: %s", key.Key)
	}

	validated, ok := store.Validate(key.Key)
	if !ok || validated.Name != "Test Customer" {
		t.Errorf("Validation failed for created key")
	}

	// Test non-existent key
	_, okFake := store.Validate("sk-fake-key")
	if okFake {
		t.Errorf("Non-existent key should not validate")
	}
}

func TestKeyStore_UsageTracking(t *testing.T) {
	store := NewMemoryKeyStore()

	key, err := store.CreateKey("Developer User", "standard")
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}

	store.RecordUsage(key.Key, 1500)
	store.RecordUsage(key.Key, 2500)

	validated, ok := store.Validate(key.Key)
	if !ok {
		t.Fatalf("Key not found")
	}
	if validated.TotalRequests != 2 {
		t.Errorf("expected 2 total requests, got %d", validated.TotalRequests)
	}
	if validated.TokensSaved != 4000 {
		t.Errorf("expected 4000 tokens saved, got %d", validated.TokensSaved)
	}
}

func TestRateLimiter_TokenBucket(t *testing.T) {
	limiter := NewRateLimiter()
	key := "test-tenant-key"

	limit := 3

	if !limiter.Allow(key, limit) {
		t.Errorf("Request 1 should be allowed")
	}
	if !limiter.Allow(key, limit) {
		t.Errorf("Request 2 should be allowed")
	}
	if !limiter.Allow(key, limit) {
		t.Errorf("Request 3 should be allowed")
	}
	if limiter.Allow(key, limit) {
		t.Errorf("Request 4 should be throttled by rate limit")
	}
}
