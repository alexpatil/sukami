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

	// Case 4: Environment fallback UPSTREAM_API_KEY
	os.Setenv("UPSTREAM_API_KEY", "env-secret-key")
	defer os.Unsetenv("UPSTREAM_API_KEY")
	req4 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if auth := resolveUpstreamAuth(req4); auth != "Bearer env-secret-key" {
		t.Errorf("expected 'Bearer env-secret-key', got '%s'", auth)
	}
}

func TestCloudSecurity_ZeroCacheLeak_RejectBeforeCache(t *testing.T) {
	os.Unsetenv("SUKAMI_DEMO_MODE")
	os.Unsetenv("UPSTREAM_API_KEY")

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
