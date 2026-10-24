package auth

import (
	"strings"
	"testing"
)

func TestKeyStore_CreateAndValidate(t *testing.T) {
	store := NewMemoryKeyStore()

	key, err := store.CreateKey("Test Customer", "pro")
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

func TestKeyStore_QuotaLimitAndUsage(t *testing.T) {
	store := NewMemoryKeyStore()

	// Create free tier key
	key, err := store.CreateKey("Free User", "free")
	if err != nil {
		t.Fatalf("Failed to create key: %v", err)
	}
	if key.QuotaLimit != 100000 {
		t.Errorf("expected 100,000 quota limit, got %d", key.QuotaLimit)
	}

	// Should be allowed initially
	allowed, _ := store.CheckQuota(key.Key)
	if !allowed {
		t.Errorf("new key should have available quota")
	}

	// Simulate usage: 90,000 tokens consumed
	store.RecordTokenUsage(key.Key, 90000, 10000)
	allowed, _ = store.CheckQuota(key.Key)
	if !allowed {
		t.Errorf("key with 90k used out of 100k should still be allowed")
	}

	// Consume remaining 10,000 tokens (now 100,000 consumed)
	store.RecordTokenUsage(key.Key, 10000, 2000)
	allowed, reason := store.CheckQuota(key.Key)
	if allowed {
		t.Errorf("key with 100k consumed should be blocked by quota limit")
	}
	if !strings.Contains(reason, "quota exhausted") {
		t.Errorf("expected quota exhausted reason, got: %s", reason)
	}
}

func TestRateLimiter_TokenBucket(t *testing.T) {
	limiter := NewRateLimiter()
	key := "test-tenant-key"

	// Limit to 3 requests per minute
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

	// 4th request must be blocked
	if limiter.Allow(key, limit) {
		t.Errorf("Request 4 should be rejected by rate limiter!")
	}
}
