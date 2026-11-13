package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"sync"
	"testing"
	"time"
)

func TestLoadChatScenario_BundledScenarioIsUsable(t *testing.T) {
	t.Setenv("BENCH_CHAT_FILE", "")
	s, err := loadChatScenario()
	if err != nil {
		t.Fatalf("loadChatScenario: %v", err)
	}
	if s.System == "" || len(s.Turns) < 2 {
		t.Errorf("bundled scenario needs a system prompt and several turns, got %d turns", len(s.Turns))
	}
}

func TestRunChat_ResendsHistoryAndSumsUpstreamUsage(t *testing.T) {
	var mu sync.Mutex
	var seen []models.ChatCompletionRequest
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		seen = append(seen, req)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"x","model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":"answer %d"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":20,"total_tokens":120,"prompt_tokens_details":{"cached_tokens":64},"completion_tokens_details":{"reasoning_tokens":5}}}`, req.Model, len(req.Messages))
	}))
	defer upstream.Close()

	cfg := benchConfig{upstreamURL: upstream.URL, apiKey: "test-key", model: "bench-model", runs: 1}
	scenario := chatScenario{System: "Answer from the policy.", Turns: []string{"First question?", "Second question?"}}
	got, err := runChat(cfg, &http.Client{Timeout: 10 * time.Second}, scenario, 1)
	if err != nil {
		t.Fatalf("runChat: %v", err)
	}

	// Three paths, two turns each.
	if len(seen) != 6 {
		t.Fatalf("expected 6 upstream calls, got %d", len(seen))
	}
	for _, name := range chatOrder {
		totals := got[name]
		if totals.requests != 2 || totals.promptTokens != 200 || totals.cachedTokens != 128 || totals.completionTokens != 40 || totals.reasoningTokens != 10 {
			t.Errorf("%s: unexpected totals %+v", name, totals)
		}
	}

	// The second turn of a path carries the first question and the answer the upstream gave to it.
	second := seen[1].Messages
	if len(second) != 4 || second[1].Content != "First question?" || second[2].Role != "assistant" || second[2].Content != "answer 2" || second[3].Content != "Second question?" {
		t.Errorf("second turn did not resend the history: %+v", second)
	}
	if seen[0].Messages[0].Content == seen[2].Messages[0].Content {
		t.Errorf("each path must start with its own session tag so paths do not share a provider-side prompt cache")
	}

	if got[chatDirect].breakdown != (metrics.TokenBreakdown{}) {
		t.Errorf("the direct path has no proxy breakdown, got %+v", got[chatDirect].breakdown)
	}
	if b := got[chatProxy].breakdown; b.SystemTokens == 0 || b.HistoryTokens == 0 || b.PromptTokens == 0 || b.UpstreamCachedTokens != 128 {
		t.Errorf("proxy path breakdown is incomplete: %+v", b)
	}
	denseSystem := seen[4].Messages[0].Content
	if !strings.Contains(denseSystem, optimizer.OutputDensityDirective) {
		t.Errorf("density path did not send the output density directive upstream")
	}
}

func TestRenderChatReport_ShowsTotalsPerPath(t *testing.T) {
	cfg := benchConfig{upstreamURL: "https://upstream.example/v1", model: "bench-model", runs: 2}
	results := map[string][]chatTotals{
		chatDirect: {
			{promptTokens: 9000, cachedTokens: 0, completionTokens: 3000, reasoningTokens: 400, finishes: []string{"stop"}},
			{promptTokens: 9200, cachedTokens: 2048, completionTokens: 2600, reasoningTokens: 300, finishes: []string{"stop", "length"}},
		},
		chatDensity: {
			{promptTokens: 8000, cachedTokens: 4096, completionTokens: 1500, reasoningTokens: 200, finishes: []string{"stop"},
				breakdown: metrics.TokenBreakdown{SystemTokens: 5000, HistoryTokens: 2500, PromptTokens: 90}},
		},
	}
	out := renderChatReport(cfg, 5, results, time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC))

	for _, want := range []string{
		"Date: 2026-11-13. Model: `bench-model`. Upstream: `upstream.example`. Conversation: 5 turns. Runs per path: 2.",
		"| Direct to upstream | 9100 (9000 to 9200) | 1024 (0 to 2048) | 2800 (2600 to 3000) | 350 (300 to 400) | 11900 (11800 to 12000) | `stop`, `length` |",
		"| Via Sukami with X-Sukami-Density: true | 8000 | 4096 | 1500 | 200 | 9500 | `stop` |",
		"| Via Sukami with X-Sukami-Density: true | 5000 | 2500 | 90 | 0 |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report is missing %q:\n%s", want, out)
		}
	}
}
