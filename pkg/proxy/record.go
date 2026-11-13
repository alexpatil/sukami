package proxy

import (
	"encoding/json"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"time"
)

// recordMissMetric records a request that was forwarded upstream. respBytes is the upstream's JSON
// response, or the response synthesized from a stream; when it carries a usage block, those numbers
// replace the proxy's own estimates.
func (h *Handler) recordMissMetric(id string, chatReq models.ChatCompletionRequest, prompt string, comp optimizer.CompressionResult, start time.Time, status string, respBytes []byte) {
	record := metrics.RequestRecord{
		ID:                id,
		Timestamp:         time.Now(),
		Model:             chatReq.Model,
		PromptPreview:     truncate(prompt, 60),
		OriginalTokens:    comp.OriginalTokens,
		SavedTokens:       comp.SavedTokens,
		InputSavedTokens:  comp.SavedTokens,
		OutputSavedTokens: 0,
		BilledTokens:      comp.OptimizedTokens,
		SavingsPercent:    comp.SavingsPercent,
		CacheHit:          false,
		LatencyMs:         time.Since(start).Milliseconds(),
		Status:            status,
	}

	parts := optimizer.AttributeInput(chatReq.Model, chatReq.Messages, chatReq.Tools)
	record.Breakdown = metrics.TokenBreakdown{
		SystemTokens:  int64(parts.System),
		HistoryTokens: int64(parts.History),
		PromptTokens:  int64(parts.Prompt),
		ToolTokens:    int64(parts.Tools),
	}

	var upstream struct {
		Usage models.Usage `json:"usage"`
	}
	if len(respBytes) > 0 && json.Unmarshal(respBytes, &upstream) == nil && upstream.Usage.PromptTokens > 0 {
		usage := upstream.Usage
		totalBilled := usage.TotalTokens
		if totalBilled == 0 {
			totalBilled = usage.PromptTokens + usage.CompletionTokens
		}
		record.BilledTokens = totalBilled
		record.OriginalTokens = totalBilled + comp.SavedTokens
		if record.OriginalTokens > 0 {
			record.SavingsPercent = (float64(record.SavedTokens) / float64(record.OriginalTokens)) * 100.0
		}

		record.Breakdown.UpstreamPromptTokens = int64(usage.PromptTokens)
		record.Breakdown.UpstreamCompletionTokens = int64(usage.CompletionTokens)
		if usage.PromptTokensDetails != nil {
			record.Breakdown.UpstreamCachedTokens = int64(usage.PromptTokensDetails.CachedTokens)
		}
		if usage.CompletionTokensDetails != nil {
			record.Breakdown.UpstreamReasoningTokens = int64(usage.CompletionTokensDetails.ReasoningTokens)
		}
	}

	costUSD, _ := metrics.CalculateSavingsUSD(chatReq.Model, record.InputSavedTokens, record.OutputSavedTokens)
	record.CostSavedUSD = costUSD
	h.tracker.RecordRequest(record)
}
