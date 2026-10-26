package metrics

import (
	"fmt"
	"strings"
)

// FormatPrometheus formats aggregated operational metrics in Prometheus exposition text format (0.0.4).
func (t *Tracker) FormatPrometheus() string {
	s := t.GetSummary()
	var sb strings.Builder

	// sukami_requests_total
	sb.WriteString("# HELP sukami_requests_total Total number of incoming HTTP requests processed.\n")
	sb.WriteString("# TYPE sukami_requests_total counter\n")
	fmt.Fprintf(&sb, "sukami_requests_total %d\n\n", s.TotalRequests)

	// sukami_cache_hits_total
	sb.WriteString("# HELP sukami_cache_hits_total Total number of requests served directly from semantic cache.\n")
	sb.WriteString("# TYPE sukami_cache_hits_total counter\n")
	fmt.Fprintf(&sb, "sukami_cache_hits_total %d\n\n", s.CacheHits)

	// sukami_cache_misses_total
	sb.WriteString("# HELP sukami_cache_misses_total Total number of requests requiring upstream model inference.\n")
	sb.WriteString("# TYPE sukami_cache_misses_total counter\n")
	fmt.Fprintf(&sb, "sukami_cache_misses_total %d\n\n", s.CacheMisses)

	// sukami_cache_hit_ratio
	hitRatio := 0.0
	if s.TotalRequests > 0 {
		hitRatio = float64(s.CacheHits) / float64(s.TotalRequests)
	}
	sb.WriteString("# HELP sukami_cache_hit_ratio The ratio of cache hits to total requests (0.0 to 1.0).\n")
	sb.WriteString("# TYPE sukami_cache_hit_ratio gauge\n")
	fmt.Fprintf(&sb, "sukami_cache_hit_ratio %.4f\n\n", hitRatio)

	// sukami_tokens_original_total
	sb.WriteString("# HELP sukami_tokens_original_total Total unoptimized prompt tokens received.\n")
	sb.WriteString("# TYPE sukami_tokens_original_total counter\n")
	fmt.Fprintf(&sb, "sukami_tokens_original_total %d\n\n", s.TotalOriginalTokens)

	// sukami_tokens_saved_total
	sb.WriteString("# HELP sukami_tokens_saved_total Total prompt tokens saved via pruning and semantic cache.\n")
	sb.WriteString("# TYPE sukami_tokens_saved_total counter\n")
	fmt.Fprintf(&sb, "sukami_tokens_saved_total %d\n\n", s.TotalSavedTokens)

	// sukami_tokens_billed_total
	sb.WriteString("# HELP sukami_tokens_billed_total Total tokens billed by upstream providers.\n")
	sb.WriteString("# TYPE sukami_tokens_billed_total counter\n")
	fmt.Fprintf(&sb, "sukami_tokens_billed_total %d\n\n", s.TotalBilledTokens)

	// sukami_token_savings_ratio
	savingsRatio := 0.0
	if s.TotalOriginalTokens > 0 {
		savingsRatio = float64(s.TotalSavedTokens) / float64(s.TotalOriginalTokens)
	}
	sb.WriteString("# HELP sukami_token_savings_ratio The ratio of saved tokens to original tokens (0.0 to 1.0).\n")
	sb.WriteString("# TYPE sukami_token_savings_ratio gauge\n")
	fmt.Fprintf(&sb, "sukami_token_savings_ratio %.4f\n\n", savingsRatio)

	// sukami_cost_saved_usd_total
	sb.WriteString("# HELP sukami_cost_saved_usd_total Estimated cost saved in USD based on dynamic pricing.\n")
	sb.WriteString("# TYPE sukami_cost_saved_usd_total counter\n")
	fmt.Fprintf(&sb, "sukami_cost_saved_usd_total %.6f\n", s.EstimatedCostSaved)

	return sb.String()
}
