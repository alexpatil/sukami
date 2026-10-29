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

func TestCloudSecurity_ZeroCacheLeak_RejectBeforeCache(t *testing.T) {
	// Ensure demo mode is OFF so auth is strictly required
	os.Unsetenv("SUKAMI_DEMO_MODE")
	os.Unsetenv("OPENAI_API_KEY")

	h, _, _ := setupTestHandler()

	prompt := "What is the capital of France?"
	cacheModel := "pub|gpt-4o"
	simulatedCachedResponse := []byte(`{"id":"chatcmpl-cached","choices":[{"message":{"content":"Paris."},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
	h.cache.SetSemantic("dummy_key", prompt, simulatedCachedResponse, cacheModel)

	// Send request with NO API KEY
	chatReq := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: prompt},
		},
	}
	body, _ := json.Marshal(chatReq)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	h.HandleChatCompletions(rec, req)

	// MUST be 401 Unauthorized. Cache MUST NOT be leaked to unauthenticated caller!
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated request on cached item, got %d (BODY: %s)", rec.Code, rec.Body.String())
	}
}

func TestCloudSecurity_GatewayMasterKey_Enforced(t *testing.T) {
	os.Setenv("SUKAMI_DEMO_MODE", "true")
	defer os.Unsetenv("SUKAMI_DEMO_MODE")

	h, authStore, _ := setupTestHandler()
	masterSecret := "sukami_master_secret_pass_999"
	h.SetGatewayKey(masterSecret)

	// Create a tenant key
	tenant, err := authStore.CreateKey("TeamAlpha", "pro")
	if err != nil {
		t.Fatalf("failed to create tenant key: %v", err)
	}

	chatReq := models.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []models.ChatMessage{
			{Role: "user", Content: "Hello world"},
		},
	}
	bodyBytes, _ := json.Marshal(chatReq)

	// Case 1: No key provided -> 401
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()
		h.HandleChatCompletions(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 when no gateway key provided, got %d", rec.Code)
		}
	}

	// Case 2: Wrong key provided -> 401
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer wrong_secret")
		rec := httptest.NewRecorder()
		h.HandleChatCompletions(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 when invalid gateway key provided, got %d", rec.Code)
		}
	}

	// Case 3: Correct master key via Authorization Bearer -> 200
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
		req.Header.Set("Authorization", "Bearer "+masterSecret)
		rec := httptest.NewRecorder()
		h.HandleChatCompletions(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 when valid master key provided, got %d (BODY: %s)", rec.Code, rec.Body.String())
		}
	}

	// Case 4: Correct master key via X-Sukami-Master-Key -> 200
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
		req.Header.Set("X-Sukami-Master-Key", masterSecret)
		rec := httptest.NewRecorder()
		h.HandleChatCompletions(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 when X-Sukami-Master-Key provided, got %d", rec.Code)
		}
	}

	// Case 5: Registered tenant key -> 200
	{
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
		req.Header.Set("X-Sukami-Key", tenant.Key)
		rec := httptest.NewRecorder()
		h.HandleChatCompletions(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 when valid tenant key provided, got %d", rec.Code)
		}
	}
}

func TestCloudSecurity_AdminKeys_Protected(t *testing.T) {
	h, _, _ := setupTestHandler()
	masterSecret := "admin_secret_key_888"
	h.SetGatewayKey(masterSecret)

	// List keys without master auth -> 401
	{
		req := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
		rec := httptest.NewRecorder()
		h.HandleListKeys(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for /api/keys without master key, got %d", rec.Code)
		}
	}

	// List keys with valid master auth -> 200
	{
		req := httptest.NewRequest(http.MethodGet, "/api/keys", nil)
		req.Header.Set("X-Sukami-Master-Key", masterSecret)
		rec := httptest.NewRecorder()
		h.HandleListKeys(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for /api/keys with master key, got %d", rec.Code)
		}
	}
}
