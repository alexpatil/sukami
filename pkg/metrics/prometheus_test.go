package metrics

import (
	"strings"
	"testing"
)

func TestFormatPrometheus_ZeroState(t *testing.T) {
	tracker := NewTracker(10)
	out := tracker.FormatPrometheus()

	expectedMetrics := []string{
		"# HELP sukami_requests_total Total number of incoming HTTP requests processed.",
		"# TYPE sukami_requests_total counter",
		"sukami_requests_total 0",
		"# HELP sukami_cache_hits_total Total number of requests served directly from semantic cache.",
		"# TYPE sukami_cache_hits_total counter",
		"sukami_cache_hits_total 0",
		"# HELP sukami_cache_misses_total Total number of requests requiring upstream model inference.",
		"# TYPE sukami_cache_misses_total counter",
		"sukami_cache_misses_total 0",
		"# HELP sukami_cache_hit_ratio The ratio of cache hits to total requests (0.0 to 1.0).",
		"# TYPE sukami_cache_hit_ratio gauge",
		"sukami_cache_hit_ratio 0.0000",
		"# HELP sukami_tokens_original_total Total unoptimized prompt tokens received.",
		"# TYPE sukami_tokens_original_total counter",
		"sukami_tokens_original_total 0",
		"# HELP sukami_tokens_saved_total Total prompt tokens saved via pruning and semantic cache.",
		"# TYPE sukami_tokens_saved_total counter",
		"sukami_tokens_saved_total 0",
		"# HELP sukami_tokens_billed_total Total tokens billed by upstream providers.",
		"# TYPE sukami_tokens_billed_total counter",
		"sukami_tokens_billed_total 0",
		"# HELP sukami_token_savings_ratio The ratio of saved tokens to original tokens (0.0 to 1.0).",
		"# TYPE sukami_token_savings_ratio gauge",
		"sukami_token_savings_ratio 0.0000",
		"# HELP sukami_cost_saved_usd_total Estimated cost saved in USD based on dynamic pricing.",
		"# TYPE sukami_cost_saved_usd_total counter",
		"sukami_cost_saved_usd_total 0.000000",
	}

	for _, expected := range expectedMetrics {
		if !strings.Contains(out, expected) {
			t.Errorf("expected output to contain %q, but got:\n%s", expected, out)
		}
	}
}

func TestFormatPrometheus_WithMetrics(t *testing.T) {
	tracker := NewTracker(10)

	// Record a cache miss
	tracker.RecordRequest(RequestRecord{
		Model:             "gpt-4o",
		OriginalTokens:    1000,
		SavedTokens:       200,
		InputSavedTokens:  200,
		OutputSavedTokens: 0,
		BilledTokens:      800,
		CacheHit:          false,
		CostSavedUSD:      0.0005,
	})

	// Record a cache hit
	tracker.RecordRequest(RequestRecord{
		Model:             "gpt-4o",
		OriginalTokens:    1000,
		SavedTokens:       1000,
		InputSavedTokens:  500,
		OutputSavedTokens: 500,
		BilledTokens:      0,
		CacheHit:          true,
		CostSavedUSD:      0.00625,
	})

	out := tracker.FormatPrometheus()

	checks := []string{
		"sukami_requests_total 2",
		"sukami_cache_hits_total 1",
		"sukami_cache_misses_total 1",
		"sukami_cache_hit_ratio 0.5000",
		"sukami_tokens_original_total 2000",
		"sukami_tokens_saved_total 1200",
		"sukami_tokens_billed_total 800",
		"sukami_token_savings_ratio 0.6000",
		"sukami_cost_saved_usd_total 0.006750",
	}

	for _, check := range checks {
		if !strings.Contains(out, check) {
			t.Errorf("expected output to contain %q, but got:\n%s", check, out)
		}
	}
}
