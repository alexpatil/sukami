package auth

import (
	"sync"
	"time"
)

type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter implements a high-performance token-bucket algorithm using pure Go stdlib.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

// NewRateLimiter instantiates the rate limiter with background stale-bucket eviction.
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
	}
	go rl.evictStale()
	return rl
}

// Allow evaluates if a request is permitted under rate limit per minute.
func (rl *RateLimiter) Allow(key string, limitPerMin int) bool {
	if limitPerMin <= 0 {
		return true // Unlimited
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, found := rl.buckets[key]
	if !found {
		rl.buckets[key] = &bucket{
			tokens:     float64(limitPerMin) - 1.0,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on elapsed duration (guard against negative elapsed time from NTP clock drift)
	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	fillRate := float64(limitPerMin) / 60.0 // tokens per second
	b.tokens += elapsed * fillRate
	if b.tokens > float64(limitPerMin) {
		b.tokens = float64(limitPerMin)
	}
	b.lastRefill = now

	// Check if enough tokens available
	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false // Rate limit exceeded!
}

// evictStale removes rate-limiter buckets that have been inactive for over 10 minutes.
func (rl *RateLimiter) evictStale() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for k, b := range rl.buckets {
			if b.lastRefill.Before(cutoff) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}
