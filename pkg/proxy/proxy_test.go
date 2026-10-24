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

func TestHandleCreateKey_FreeTrialQuota(t *testing.T) {
	h, _, _ := setupTestHandler()

	body := []byte(`{"name":"Free Trial Developer","tier":"free"}`)
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

	if created.Tier != "free" {
		t.Errorf("expected tier 'free', got %s", created.Tier)
	}
	if created.QuotaLimit != 100000 {
		t.Errorf("expected QuotaLimit 100000, got %d", created.QuotaLimit)
	}
	if created.QuotaUsed != 0 {
		t.Errorf("expected QuotaUsed 0, got %d", created.QuotaUsed)
	}
}

func TestHandleChatCompletions_QuotaExhaustion(t *testing.T) {
	h, ks, _ := setupTestHandler()

	// Create a free key and simulate exhausted quota
	key, err := ks.CreateKey("Quota Test", "free")
	if err != nil {
		t.Fatalf("failed to create key: %v", err)
	}
	ks.RecordTokenUsage(key.Key, 100000, 5000)

	// Make chat request using the exhausted key
	chatReq := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}
	reqBytes, _ := json.Marshal(chatReq)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBytes))
	req.Header.Set("Authorization", "Bearer "+key.Key)
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	// Must reject with 402 Payment Required
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status 402 Payment Required for exhausted quota, got %d (body: %s)", rec.Code, rec.Body.String())
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
