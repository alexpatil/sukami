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
