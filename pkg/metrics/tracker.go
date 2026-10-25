package metrics

import (
	"sync"
	"time"
)

// RequestRecord tracks a single proxied request and both input/output token savings.
type RequestRecord struct {
	ID                string    `json:"id"`
	Timestamp         time.Time `json:"timestamp"`
	Model             string    `json:"model"`
	PromptPreview     string    `json:"prompt_preview"`
	OriginalTokens    int       `json:"original_tokens"`
	SavedTokens       int       `json:"saved_tokens"`
	InputSavedTokens  int       `json:"input_saved_tokens"`
	OutputSavedTokens int       `json:"output_saved_tokens"`
	BilledTokens      int       `json:"billed_tokens"`
	SavingsPercent    float64   `json:"savings_percent"`
	CacheHit          bool      `json:"cache_hit"`
	LatencyMs         int64     `json:"latency_ms"`
	CostSavedUSD      float64   `json:"cost_saved_usd"`
	Status            string    `json:"status"`
}

// Summary represents aggregated operational metrics.
type Summary struct {
	TotalRequests          int64           `json:"total_requests"`
	CacheHits              int64           `json:"cache_hits"`
	CacheMisses            int64           `json:"cache_misses"`
	CacheHitRate           float64         `json:"cache_hit_rate"`
	TotalOriginalTokens    int64           `json:"total_original_tokens"`
	TotalSavedTokens       int64           `json:"total_saved_tokens"`
	TotalInputSavedTokens  int64           `json:"total_input_saved_tokens"`
	TotalOutputSavedTokens int64           `json:"total_output_saved_tokens"`
	TotalBilledTokens      int64           `json:"total_billed_tokens"`
	OverallSavingsPct      float64         `json:"overall_savings_pct"`
	EstimatedCostSaved     float64         `json:"estimated_cost_saved_usd"`
	RecentRequests         []RequestRecord `json:"recent_requests"`
}

// Tracker manages real-time SaaS metrics.
type Tracker struct {
	mu                     sync.RWMutex
	totalRequests          int64
	cacheHits              int64
	cacheMisses            int64
	totalOriginalTokens    int64
	totalSavedTokens       int64
	totalInputSavedTokens  int64
	totalOutputSavedTokens int64
	totalBilledTokens      int64
	totalCostSavedUSD      float64
	recentRequests         []RequestRecord
	maxRecent              int
}

// NewTracker creates a new metrics tracker instance.
func NewTracker(maxRecent int) *Tracker {
	if maxRecent <= 0 {
		maxRecent = 50
	}
	return &Tracker{
		recentRequests: make([]RequestRecord, 0, maxRecent),
		maxRecent:      maxRecent,
	}
}

// RecordRequest records an incoming request and updates all counters.
func (t *Tracker) RecordRequest(record RequestRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalRequests++
	if record.CacheHit {
		t.cacheHits++
	} else {
		t.cacheMisses++
	}

	t.totalOriginalTokens += int64(record.OriginalTokens)
	t.totalSavedTokens += int64(record.SavedTokens)
	t.totalInputSavedTokens += int64(record.InputSavedTokens)
	t.totalOutputSavedTokens += int64(record.OutputSavedTokens)
	t.totalBilledTokens += int64(record.BilledTokens)

	// Calculate dynamic API cost savings if not pre-populated
	if record.CostSavedUSD <= 0 && (record.InputSavedTokens > 0 || record.OutputSavedTokens > 0) {
		if usd, found := CalculateSavingsUSD(record.Model, record.InputSavedTokens, record.OutputSavedTokens); found {
			record.CostSavedUSD = usd
		}
	}
	t.totalCostSavedUSD += record.CostSavedUSD

	// Prepend newest-first without allocating a new backing array at capacity
	if len(t.recentRequests) < t.maxRecent {
		t.recentRequests = append(t.recentRequests, RequestRecord{})
	}
	copy(t.recentRequests[1:], t.recentRequests)
	t.recentRequests[0] = record
}

// GetSummary compiles the current operational metrics for the dashboard.
func (t *Tracker) GetSummary() Summary {
	t.mu.RLock()
	defer t.mu.RUnlock()

	hitRate := 0.0
	if t.totalRequests > 0 {
		hitRate = (float64(t.cacheHits) / float64(t.totalRequests)) * 100.0
	}

	savingsPct := 0.0
	if t.totalOriginalTokens > 0 {
		savingsPct = (float64(t.totalSavedTokens) / float64(t.totalOriginalTokens)) * 100.0
	}

	recentCopy := make([]RequestRecord, len(t.recentRequests))
	copy(recentCopy, t.recentRequests)

	return Summary{
		TotalRequests:          t.totalRequests,
		CacheHits:              t.cacheHits,
		CacheMisses:            t.cacheMisses,
		CacheHitRate:           hitRate,
		TotalOriginalTokens:    t.totalOriginalTokens,
		TotalSavedTokens:       t.totalSavedTokens,
		TotalInputSavedTokens:  t.totalInputSavedTokens,
		TotalOutputSavedTokens: t.totalOutputSavedTokens,
		TotalBilledTokens:      t.totalBilledTokens,
		OverallSavingsPct:      savingsPct,
		EstimatedCostSaved:     t.totalCostSavedUSD,
		RecentRequests:         recentCopy,
	}
}
