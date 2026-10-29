package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sukami/pkg/models"
	"testing"
)

func TestHandleChatCompletions_EmptyMessagesArray(t *testing.T) {
	h, _, _ := setupTestHandler()

	body := []byte(`{"model":"gpt-4o","messages":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", rec.Code)
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

func TestHandleChatCompletions_SmartRoute_EconomyDowngrade(t *testing.T) {
	os.Setenv("SUKAMI_DEMO_MODE", "true")
	defer os.Unsetenv("SUKAMI_DEMO_MODE")

	h, _, _ := setupTestHandler()

	chatReq := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "What is the capital of France?"},
		},
	}
	reqBytes, _ := json.Marshal(chatReq)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBytes))
	req.Header.Set("X-Sukami-Smart-Route", "true")
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	if orig := rec.Header().Get("X-Sukami-Original-Model"); orig != "gpt-4o" {
		t.Errorf("expected X-Sukami-Original-Model 'gpt-4o', got %q", orig)
	}
	if routed := rec.Header().Get("X-Sukami-Routed-Model"); routed != "gpt-4o-mini" {
		t.Errorf("expected X-Sukami-Routed-Model 'gpt-4o-mini', got %q", routed)
	}
	if tier := rec.Header().Get("X-Sukami-Route-Tier"); tier != "economy" {
		t.Errorf("expected X-Sukami-Route-Tier 'economy', got %q", tier)
	}
}

func TestHandleChatCompletions_SmartRoute_FlagshipPreserved(t *testing.T) {
	os.Setenv("SUKAMI_DEMO_MODE", "true")
	defer os.Unsetenv("SUKAMI_DEMO_MODE")

	h, _, _ := setupTestHandler()

	chatReq := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Please review this code:\n```go\nfunc Worker() error { return nil }\n```"},
		},
	}
	reqBytes, _ := json.Marshal(chatReq)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBytes))
	req.Header.Set("X-Sukami-Smart-Route", "true")
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	if rec.Header().Get("X-Sukami-Routed-Model") != "" {
		t.Errorf("expected no model change for complex code prompt, got %s", rec.Header().Get("X-Sukami-Routed-Model"))
	}
}
