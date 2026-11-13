package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"testing"
	"time"
)

func TestResolveUpstreamTarget(t *testing.T) {
	h, _, _ := setupTestHandler()

	// 1. Explicit X-Upstream-Url header
	req1 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req1.Header.Set("X-Upstream-Url", "https://openrouter.ai/api/v1")
	if target := h.resolveUpstreamTarget(req1); target != "https://openrouter.ai/api/v1/chat/completions" {
		t.Errorf("expected OpenRouter target, got %s", target)
	}

	// 2. Explicit X-Base-Url header
	req2 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req2.Header.Set("X-Base-Url", "https://api.deepseek.com/v1")
	if target := h.resolveUpstreamTarget(req2); target != "https://api.deepseek.com/v1/chat/completions" {
		t.Errorf("expected DeepSeek target, got %s", target)
	}

	// 3. Default server fallback
	req3 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if target := h.resolveUpstreamTarget(req3); target != "https://api.groq.com/openai/v1/chat/completions" {
		t.Errorf("expected default Groq target, got %s", target)
	}

	// 4. Unconfigured default upstream returns empty string
	hEmpty := NewHandler(cache.NewMemoryCache(time.Hour), metrics.NewTracker(10), auth.NewMemoryKeyStore(), auth.NewRateLimiter(), "")
	req4 := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	if target := hEmpty.resolveUpstreamTarget(req4); target != "" {
		t.Errorf("expected empty target for unconfigured upstream, got %s", target)
	}
}

func TestForwardUpstream_KeepsUnmodeledRequestFields(t *testing.T) {
	var received map[string]json.RawMessage
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","choices":[]}`))
	}))
	defer up.Close()
	h := NewHandler(cache.NewMemoryCache(time.Hour), metrics.NewTracker(10), auth.NewMemoryKeyStore(), auth.NewRateLimiter(), up.URL)

	body := `{"model":"m","messages":[{"role":"user","content":"hello there stranger"}],"max_completion_tokens":50,"reasoning_effort":"low","metadata":{"a":"b"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer provider-key")
	h.HandleChatCompletions(httptest.NewRecorder(), req)

	for name, want := range map[string]string{"max_completion_tokens": `50`, "reasoning_effort": `"low"`, "metadata": `{"a":"b"}`} {
		if string(received[name]) != want {
			t.Errorf("upstream received %q = %s, want %s", name, received[name], want)
		}
	}
}
