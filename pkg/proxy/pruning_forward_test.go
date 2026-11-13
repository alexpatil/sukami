package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"testing"
	"time"
)

// The proxy records pruning savings, so the upstream must receive the pruned prompt
// with default settings, not only when output density is on.
func TestForwardUpstream_SendsThePrunedPromptItReportsSavingsFor(t *testing.T) {
	var received models.ChatCompletionRequest
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":1,"total_tokens":21}}`))
	}))
	defer upstream.Close()

	tracker := metrics.NewTracker(10)
	h := NewHandler(cache.NewMemoryCache(time.Hour), tracker, auth.NewMemoryKeyStore(), auth.NewRateLimiter(), upstream.URL)

	const code = "def f(x):\n    if x:\n        return 1"
	history := "Please note:   two  spaces stay."
	isoReq{auth: "sk-caller", msgs: []models.ChatMessage{
		userMsg("Could you please review this:\n" + code + "   \n\n\n\n"),
		{Role: "assistant", Content: history},
		userMsg("Please continue."),
	}}.send(h)

	if len(received.Messages) != 3 {
		t.Fatalf("upstream received %d messages, want 3", len(received.Messages))
	}
	if want := "Review this:\n" + code; received.Messages[0].Content != want {
		t.Errorf("first user message = %q, want %q", received.Messages[0].Content, want)
	}
	if received.Messages[1].Content != history {
		t.Errorf("assistant message was changed: %q", received.Messages[1].Content)
	}
	if received.Messages[2].Content != "Continue." {
		t.Errorf("last user message = %q, want %q", received.Messages[2].Content, "Continue.")
	}

	summary := tracker.GetSummary()
	if summary.TotalInputSavedTokens <= 0 {
		t.Errorf("expected recorded input savings for the pruned prompt, got %d", summary.TotalInputSavedTokens)
	}
}
