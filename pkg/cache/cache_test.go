package cache

import (
	"fmt"
	"testing"
	"time"
)

// TestMemoryCache_SetAndGet verifies that cached items are retrieved properly.
func TestMemoryCache_SetAndGet(t *testing.T) {
	c := NewMemoryCache(10 * time.Minute)

	key := ComputeHash("gpt-4o", "what is Go?")
	payload := []byte(`{"id":"chatcmpl-1","choices":[{"message":{"content":"Go is an open-source programming language."}}]}`)

	c.Set(key, payload, "gpt-4o")

	retrieved, hit := c.Get(key)
	if !hit {
		t.Fatalf("Expected cache HIT, but got MISS")
	}

	if string(retrieved) != string(payload) {
		t.Fatalf("Payload mismatch:\nGot: %s\nExpected: %s", string(retrieved), string(payload))
	}
}

// TestMemoryCache_SemanticNormalization verifies that slight variations match the same cache entry.
func TestMemoryCache_SemanticNormalization(t *testing.T) {
	c := NewMemoryCache(10 * time.Minute)

	q1 := "What is concurrency in Go?  "
	q2 := "what is concurrency in go"
	q3 := "What is concurrency in Go?!"

	key1 := ComputeHash("gpt-4o", q1)
	key2 := ComputeHash("gpt-4o", q2)
	key3 := ComputeHash("gpt-4o", q3)

	if key1 != key2 || key1 != key3 {
		t.Errorf("Normalized hashes should match across minor punctuation/case variations!\nKey1: %s\nKey2: %s\nKey3: %s", key1, key2, key3)
	}

	payload := []byte(`{"result":"concurrency explanation"}`)
	c.Set(key1, payload, "gpt-4o")

	_, hit2 := c.Get(key2)
	if !hit2 {
		t.Errorf("Expected cache HIT for normalized query variation q2")
	}

	_, hit3 := c.Get(key3)
	if !hit3 {
		t.Errorf("Expected cache HIT for normalized query variation q3")
	}
}

// TestMemoryCache_Expiration verifies TTL eviction.
func TestMemoryCache_Expiration(t *testing.T) {
	shortTTL := 50 * time.Millisecond
	c := NewMemoryCache(shortTTL)

	key := ComputeHash("gpt-4o", "temporary query")
	c.Set(key, []byte("temp"), "gpt-4o")

	// Immediate lookup should hit
	if _, hit := c.Get(key); !hit {
		t.Fatalf("Expected immediate cache HIT")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	if _, hit := c.Get(key); hit {
		t.Errorf("Item should have expired, but got cache HIT")
	}
}

// TestMemoryCache_CapacityBound verifies that cache does not grow beyond maxEntries.
func TestMemoryCache_CapacityBound(t *testing.T) {
	c := NewMemoryCacheWithCapacity(10*time.Minute, 5)

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key_%d", i)
		c.Set(key, []byte(fmt.Sprintf("val_%d", i)), "gpt-4o")
	}

	_, _, total := c.Stats()
	if total > 5 {
		t.Errorf("Cache exceeded max capacity bound! Expected <= 5, got %d", total)
	}
}

