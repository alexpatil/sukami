package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"sync"
	"testing"
	"time"
)

func TestRunOnce_MeasuresEveryCaseThroughTheProxy(t *testing.T) {
	var mu sync.Mutex
	var seen []models.ChatCompletionRequest
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req models.ChatCompletionRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		seen = append(seen, req)
		n := len(seen)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"x","model":%q,"choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":100,"completion_tokens":%d,"total_tokens":0}}`, req.Model, n*10)
	}))
	defer upstream.Close()

	cfg := benchConfig{upstreamURL: upstream.URL, apiKey: "test-key", model: "bench-model", runs: 1}
	got, err := runOnce(cfg, &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("runOnce: %v", err)
	}

	// Direct, proxy miss and density reach the upstream; the cache hit must not.
	if len(seen) != 3 {
		t.Fatalf("expected 3 upstream calls, got %d", len(seen))
	}
	for _, name := range caseOrder {
		if _, ok := got[name]; !ok {
			t.Errorf("case %q was not measured", name)
		}
	}
	if got[caseCacheHit].completionTokens != got[caseProxy].completionTokens {
		t.Errorf("cache hit must return the proxied response: %d vs %d tokens", got[caseCacheHit].completionTokens, got[caseProxy].completionTokens)
	}
	dense := seen[2].Messages
	if len(dense) == 0 || dense[0].Role != "system" || !strings.Contains(dense[0].Content, optimizer.OutputDensityDirective) {
		t.Errorf("density case did not send the output density directive upstream: %+v", dense)
	}
}

func TestRenderReport_ShowsMedianAndRange(t *testing.T) {
	cfg := benchConfig{upstreamURL: "https://upstream.example/v1", model: "bench-model", runs: 3}
	results := map[string][]sample{
		caseCacheHit: {{promptTokens: 100, completionTokens: 200, latency: 450 * time.Microsecond, finish: "stop"}},
		caseDirect: {
			{promptTokens: 100, completionTokens: 300, latency: 3 * time.Second, finish: "stop"},
			{promptTokens: 100, completionTokens: 100, latency: 1 * time.Second, finish: "length"},
			{promptTokens: 100, completionTokens: 200, latency: 2 * time.Second, finish: "stop"},
		},
	}
	out := renderReport(cfg, results, time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC))

	for _, want := range []string{
		"Date: 2026-11-13. Model: `bench-model`. Upstream: `upstream.example`. Runs per case: 3.",
		"| Direct to upstream | 100 | 200 (100 to 300) | 2.00s (1.00s to 3.00s) | `stop`, `length` |",
		"| Via Sukami, same request again (cache hit) | 100 | 200 | 450µs | `stop` |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report is missing %q:\n%s", want, out)
		}
	}
}
