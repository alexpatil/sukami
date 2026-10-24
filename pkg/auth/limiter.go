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

// NewRateLimiter instantiates the rate limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*bucket),
	}
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

	// Refill tokens based on elapsed duration
	elapsed := now.Sub(b.lastRefill).Seconds()
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
