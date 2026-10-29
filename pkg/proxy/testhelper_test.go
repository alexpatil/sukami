package proxy

import (
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"time"
)

func setupTestHandler() (*Handler, auth.KeyStore, *metrics.Tracker) {
	c := cache.NewMemoryCache(1 * time.Hour)
	t := metrics.NewTracker(10)
	ks := auth.NewMemoryKeyStore()
	limiter := auth.NewRateLimiter()
	h := NewHandler(c, t, ks, limiter, "https://api.groq.com/openai")
	return h, ks, t
}
