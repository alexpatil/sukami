package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"testing"
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

func TestHandleCreateKey_Standard(t *testing.T) {
	h, _, _ := setupTestHandler()

	body := []byte(`{"name":"Developer Key","tier":"standard"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/keys", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.HandleCreateKey(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d", rec.Code)
	}

	var created auth.APIKey
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if created.Tier != "standard" {
		t.Errorf("expected tier 'standard', got %s", created.Tier)
	}
	if created.RateLimitPerMin != 60 {
		t.Errorf("expected RateLimitPerMin 60, got %d", created.RateLimitPerMin)
	}
}

func TestHandleChatCompletions_RateLimitExceeded(t *testing.T) {
	h, ks, _ := setupTestHandler()

	key, err := ks.CreateKey("Rate Test", "standard")
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}

	// Exhaust rate limit directly
	key.RateLimitPerMin = 1

	chatReq := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello world 1"},
		},
	}
	reqBytes, _ := json.Marshal(chatReq)

	// First request succeeds or simulates
	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBytes))
	req1.Header.Set("Authorization", "Bearer "+key.Key)
	rec1 := httptest.NewRecorder()
	h.HandleChatCompletions(rec1, req1)

	// Second request within the same minute should be throttled (429)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBytes))
	req2.Header.Set("Authorization", "Bearer "+key.Key)
	rec2 := httptest.NewRecorder()
	h.HandleChatCompletions(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 Too Many Requests, got %d (body: %s)", rec2.Code, rec2.Body.String())
	}
}

func TestHandleMetrics_Summary(t *testing.T) {
	h, _, _ := setupTestHandler()

	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	rec := httptest.NewRecorder()

	h.HandleMetrics(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", rec.Code)
	}

	var summary metrics.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("failed to decode metrics summary: %v", err)
	}
}

func TestResolveSukamiKey(t *testing.T) {
	// Case 1: In Authorization header
	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req1.Header.Set("Authorization", "Bearer sk-sukami-test1")
	if key := resolveSukamiKey(req1); key != "sk-sukami-test1" {
		t.Errorf("expected 'sk-sukami-test1', got '%s'", key)
	}

	// Case 2: In X-Sukami-Key header
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req2.Header.Set("X-Sukami-Key", "sk-sukami-header2")
	if key := resolveSukamiKey(req2); key != "sk-sukami-header2" {
		t.Errorf("expected 'sk-sukami-header2', got '%s'", key)
	}

	// Case 3: Direct provider key does NOT register as Sukami key
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req3.Header.Set("Authorization", "Bearer sk-proj-openai123")
	if key := resolveSukamiKey(req3); key != "" {
		t.Errorf("expected empty string for non-sukami key, got '%s'", key)
	}
}

func TestResolveUpstreamAuth(t *testing.T) {
	// Case 1: Explicit X-Upstream-Key
	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req1.Header.Set("X-Upstream-Key", "sk-upstream-secret")
	if auth := resolveUpstreamAuth(req1); auth != "Bearer sk-upstream-secret" {
		t.Errorf("expected 'Bearer sk-upstream-secret', got '%s'", auth)
	}

	// Case 2: Direct provider key in Authorization
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req2.Header.Set("Authorization", "Bearer sk-proj-direct")
	if auth := resolveUpstreamAuth(req2); auth != "Bearer sk-proj-direct" {
		t.Errorf("expected 'Bearer sk-proj-direct', got '%s'", auth)
	}

	// Case 3: Dual headers: X-Sukami-Key + Authorization direct provider
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req3.Header.Set("X-Sukami-Key", "sk-sukami-tenant")
	req3.Header.Set("Authorization", "Bearer sk-proj-provider")
	if key := resolveSukamiKey(req3); key != "sk-sukami-tenant" {
		t.Errorf("expected Sukami key 'sk-sukami-tenant', got '%s'", key)
	}
	if auth := resolveUpstreamAuth(req3); auth != "Bearer sk-proj-provider" {
		t.Errorf("expected upstream auth 'Bearer sk-proj-provider', got '%s'", auth)
	}
}

func TestResolveUpstreamTarget(t *testing.T) {
	h, _, _ := setupTestHandler()

	// 1. Explicit X-Upstream-Url header
	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req1.Header.Set("X-Upstream-Url", "https://openrouter.ai/api/v1")
	if target := h.resolveUpstreamTarget(req1, "gpt-4o"); target != "https://openrouter.ai/api/v1/chat/completions" {
		t.Errorf("expected OpenRouter target, got %s", target)
	}

	// 2. Explicit X-Provider header
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req2.Header.Set("X-Provider", "deepseek")
	if target := h.resolveUpstreamTarget(req2, "deepseek-chat"); target != "https://api.deepseek.com/v1/chat/completions" {
		t.Errorf("expected DeepSeek target, got %s", target)
	}

	// 3. Model prefix auto-routing
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if target := h.resolveUpstreamTarget(req3, "openrouter/anthropic/claude-3.5-sonnet"); target != "https://openrouter.ai/api/v1/chat/completions" {
		t.Errorf("expected OpenRouter target from model prefix, got %s", target)
	}

	// 4. Default server fallback
	req4 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if target := h.resolveUpstreamTarget(req4, "gpt-4o"); target != "https://api.groq.com/openai/v1/chat/completions" {
		t.Errorf("expected default Groq target, got %s", target)
	}
}


