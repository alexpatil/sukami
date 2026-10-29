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

	key.RateLimitPerMin = 1

	chatReq := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello world 1"},
		},
	}
	reqBytes, _ := json.Marshal(chatReq)

	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBytes))
	req1.Header.Set("Authorization", "Bearer "+key.Key)
	rec1 := httptest.NewRecorder()
	h.HandleChatCompletions(rec1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBytes))
	req2.Header.Set("Authorization", "Bearer "+key.Key)
	rec2 := httptest.NewRecorder()
	h.HandleChatCompletions(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429 Too Many Requests, got %d (body: %s)", rec2.Code, rec2.Body.String())
	}
}

func TestHandleChatCompletions_SmartRoute_UpstreamAwareness(t *testing.T) {
	os.Setenv("SUKAMI_DEMO_MODE", "true")
	defer os.Unsetenv("SUKAMI_DEMO_MODE")

	h, _, _ := setupTestHandler()

	testCases := []struct {
		name        string
		upstreamURL string
		inputModel  string
		prompt      string
		wantRouted  string
		wantTier    string
	}{
		{
			name:        "OpenAI gpt-4o downgrade",
			upstreamURL: "https://api.openai.com/v1",
			inputModel:  "gpt-4o",
			prompt:      "What is the capital of France?",
			wantRouted:  "gpt-4o-mini",
			wantTier:    "economy",
		},
		{
			name:        "Anthropic claude-3-5-sonnet downgrade",
			upstreamURL: "https://api.anthropic.com",
			inputModel:  "claude-3-5-sonnet-20241022",
			prompt:      "Summarize this paragraph in simple terms.",
			wantRouted:  "claude-3-5-haiku",
			wantTier:    "economy",
		},
		{
			name:        "Groq llama-3.3-70b downgrade",
			upstreamURL: "https://api.groq.com/openai/v1",
			inputModel:  "llama-3.3-70b-versatile",
			prompt:      "Translate hello to Spanish.",
			wantRouted:  "llama-3.1-8b-instant",
			wantTier:    "economy",
		},
		{
			name:        "OpenRouter namespaced model downgrade",
			upstreamURL: "https://openrouter.ai/api/v1",
			inputModel:  "openai/gpt-4o",
			prompt:      "Define latency.",
			wantRouted:  "openai/gpt-4o-mini",
			wantTier:    "economy",
		},
		{
			name:        "Flagship preserved on code prompt",
			upstreamURL: "https://api.openai.com/v1",
			inputModel:  "gpt-4o",
			prompt:      "Review this code:\n```go\nfunc Worker() error { return nil }\n```",
			wantRouted:  "",
			wantTier:    "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chatReq := models.ChatCompletionRequest{
				Model: tc.inputModel,
				Messages: []models.ChatMessage{
					{Role: "user", Content: tc.prompt},
				},
			}
			reqBytes, _ := json.Marshal(chatReq)

			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBytes))
			req.Header.Set("X-Sukami-Smart-Route", "true")
			if tc.upstreamURL != "" {
				req.Header.Set("X-Upstream-Url", tc.upstreamURL)
			}
			rec := httptest.NewRecorder()

			h.HandleChatCompletions(rec, req)

			routed := rec.Header().Get("X-Sukami-Routed-Model")
			if routed != tc.wantRouted {
				t.Errorf("X-Sukami-Routed-Model = %q; want %q", routed, tc.wantRouted)
			}
			if tc.wantTier != "" {
				if tier := rec.Header().Get("X-Sukami-Route-Tier"); tier != tc.wantTier {
					t.Errorf("X-Sukami-Route-Tier = %q; want %q", tier, tc.wantTier)
				}
			}
		})
	}
}
