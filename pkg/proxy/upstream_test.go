package proxy

import (
	"net/http"
	"net/http/httptest"
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

