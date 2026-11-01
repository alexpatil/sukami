package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
)

func TestRetry_Transient429_RecoversOnSecondAttempt(t *testing.T) {
	var attempts atomic.Int32
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := attempts.Add(1)
		if call == 1 {
			w.Header().Set("retry-after-ms", "10")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"Rate limit exceeded"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := models.ChatCompletionResponse{
			ID:      "chatcmpl-retry-test",
			Created: time.Now().Unix(),
			Model:   "gpt-4o",
			Choices: []models.ChatChoice{
				{Message: models.ChatMessage{Role: "assistant", Content: "Recovered successfully!"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer upstreamServer.Close()

	c := cache.NewMemoryCache(10 * time.Minute)
	tr := metrics.NewTracker(10)
	ks := auth.NewMemoryKeyStore()
	rl := auth.NewRateLimiter()
	h := NewHandler(c, tr, ks, rl, upstreamServer.URL)
	h.SetMaxRetries(2)

	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"test retry"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test")
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK after retry recovery, got %d: %s", rec.Code, rec.Body.String())
	}
	if attempts.Load() != 2 {
		t.Fatalf("expected 2 attempts total (1 retry), got %d", attempts.Load())
	}
	if retriesHeader := rec.Header().Get("X-Sukami-Retries"); retriesHeader != "1" {
		t.Fatalf("expected X-Sukami-Retries to be 1, got %q", retriesHeader)
	}
}

func TestRetry_Transient503_ExhaustsMaxRetries(t *testing.T) {
	var attempts atomic.Int32
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.Header().Set("retry-after-ms", "5")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"Server overloaded"}}`))
	}))
	defer upstreamServer.Close()

	c := cache.NewMemoryCache(10 * time.Minute)
	tr := metrics.NewTracker(10)
	ks := auth.NewMemoryKeyStore()
	rl := auth.NewRateLimiter()
	h := NewHandler(c, tr, ks, rl, upstreamServer.URL)
	h.SetMaxRetries(2)

	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"test exhaust"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test")
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 after retries exhausted, got %d", rec.Code)
	}
	// 1 initial + 2 retries = 3 attempts total
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 attempts total, got %d", attempts.Load())
	}
	if retriesHeader := rec.Header().Get("X-Sukami-Retries"); retriesHeader != "2" {
		t.Fatalf("expected X-Sukami-Retries to be 2, got %q", retriesHeader)
	}
}

func TestRetry_HeaderOverride_DisablesRetries(t *testing.T) {
	var attempts atomic.Int32
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"Rate limited"}}`))
	}))
	defer upstreamServer.Close()

	c := cache.NewMemoryCache(10 * time.Minute)
	tr := metrics.NewTracker(10)
	ks := auth.NewMemoryKeyStore()
	rl := auth.NewRateLimiter()
	h := NewHandler(c, tr, ks, rl, upstreamServer.URL)
	h.SetMaxRetries(3)

	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"no retry please"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("X-Sukami-Max-Retries", "0") // Disable via request header
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec.Code)
	}
	if attempts.Load() != 1 {
		t.Fatalf("expected exactly 1 attempt with retries disabled, got %d", attempts.Load())
	}
	if retriesHeader := rec.Header().Get("X-Sukami-Retries"); retriesHeader != "0" {
		t.Fatalf("expected X-Sukami-Retries to be 0, got %q", retriesHeader)
	}
}

func TestRetry_NonTransient400_DoesNotRetry(t *testing.T) {
	var attempts atomic.Int32
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"Invalid parameter"}}`))
	}))
	defer upstreamServer.Close()

	c := cache.NewMemoryCache(10 * time.Minute)
	tr := metrics.NewTracker(10)
	ks := auth.NewMemoryKeyStore()
	rl := auth.NewRateLimiter()
	h := NewHandler(c, tr, ks, rl, upstreamServer.URL)
	h.SetMaxRetries(3)

	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"bad request"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer sk-test")
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if attempts.Load() != 1 {
		t.Fatalf("expected exactly 1 attempt for non-transient 400, got %d", attempts.Load())
	}
}

func TestRetry_ParseRetryAfter(t *testing.T) {
	// Case 1: retry-after-ms
	h1 := http.Header{}
	h1.Set("retry-after-ms", "150")
	if d, ok := parseRetryAfter(h1); !ok || d != 150*time.Millisecond {
		t.Fatalf("expected 150ms, got %v (ok=%v)", d, ok)
	}

	// Case 2: integer seconds
	h2 := http.Header{}
	h2.Set("Retry-After", "3")
	if d, ok := parseRetryAfter(h2); !ok || d != 3*time.Second {
		t.Fatalf("expected 3s, got %v (ok=%v)", d, ok)
	}

	// Case 3: Empty header
	h3 := http.Header{}
	if _, ok := parseRetryAfter(h3); ok {
		t.Fatalf("expected false for empty header")
	}
}

func TestRetry_ContextCancellationAborts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := sleepWithContext(ctx, 5*time.Second)
	if err == nil {
		t.Fatalf("expected error from cancelled context, got nil")
	}
}
