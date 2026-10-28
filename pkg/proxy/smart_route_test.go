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

	// Flagship model should NOT be downgraded
	if routed := rec.Header().Get("X-Sukami-Routed-Model"); routed != "" {
		t.Errorf("expected X-Sukami-Routed-Model to be empty (preserved), got %q", routed)
	}
}

func TestHandlePlayground_SmartRoute(t *testing.T) {
	h, _, _ := setupTestHandler()

	payload := map[string]string{
		"prompt": "What is the capital of Japan?",
		"model":  "gpt-4o",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/playground", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.HandlePlayground(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res struct {
		SmartRouting struct {
			OriginalModel string `json:"original_model"`
			RoutedModel   string `json:"routed_model"`
			Tier          string `json:"tier"`
			IsDowngraded  bool   `json:"is_downgraded"`
		} `json:"smart_routing"`
	}

	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.SmartRouting.OriginalModel != "gpt-4o" {
		t.Errorf("expected original_model 'gpt-4o', got %q", res.SmartRouting.OriginalModel)
	}
	if res.SmartRouting.RoutedModel != "gpt-4o-mini" {
		t.Errorf("expected routed_model 'gpt-4o-mini', got %q", res.SmartRouting.RoutedModel)
	}
	if !res.SmartRouting.IsDowngraded {
		t.Errorf("expected is_downgraded to be true for trivial prompt")
	}
}
