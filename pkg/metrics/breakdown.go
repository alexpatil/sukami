package metrics

// TokenBreakdown says where the tokens of a forwarded request went.
//
// The Upstream fields are the numbers the provider reported in its usage block, and are zero when it
// reported none. The four input parts split the request's input by purpose. The proxy's tokenizer only
// estimates, so when the provider reports its prompt count the parts are that count divided in the
// estimated proportions (see ScaleInputTo) and add up to it exactly. Without a provider count they are
// the raw estimates.
type TokenBreakdown struct {
	SystemTokens  int64 `json:"system_tokens"`
	HistoryTokens int64 `json:"history_tokens"`
	PromptTokens  int64 `json:"prompt_tokens"`
	ToolTokens    int64 `json:"tool_tokens"`

	UpstreamPromptTokens     int64 `json:"upstream_prompt_tokens"`
	UpstreamCachedTokens     int64 `json:"upstream_cached_prompt_tokens"`
	UpstreamCompletionTokens int64 `json:"upstream_completion_tokens"`
	UpstreamReasoningTokens  int64 `json:"upstream_reasoning_tokens"`
}

// ScaleInputTo rescales the four input parts so they keep their proportions and sum to total,
// the prompt token count the provider billed. Rounding leftovers go to the largest part.
func (b *TokenBreakdown) ScaleInputTo(total int64) {
	parts := []*int64{&b.SystemTokens, &b.HistoryTokens, &b.PromptTokens, &b.ToolTokens}
	var estimated int64
	for _, p := range parts {
		estimated += *p
	}
	if estimated <= 0 || total <= 0 {
		return
	}

	largest := parts[0]
	var assigned int64
	for _, p := range parts {
		if *p > *largest {
			largest = p
		}
	}
	for _, p := range parts {
		*p = *p * total / estimated
		assigned += *p
	}
	*largest += total - assigned
}

func (b *TokenBreakdown) add(other TokenBreakdown) {
	b.SystemTokens += other.SystemTokens
	b.HistoryTokens += other.HistoryTokens
	b.PromptTokens += other.PromptTokens
	b.ToolTokens += other.ToolTokens
	b.UpstreamPromptTokens += other.UpstreamPromptTokens
	b.UpstreamCachedTokens += other.UpstreamCachedTokens
	b.UpstreamCompletionTokens += other.UpstreamCompletionTokens
	b.UpstreamReasoningTokens += other.UpstreamReasoningTokens
}

// parts lists the breakdown as label and value pairs, in the order they are exported.
func (b TokenBreakdown) parts() []struct {
	Label string
	Value int64
} {
	return []struct {
		Label string
		Value int64
	}{
		{"system", b.SystemTokens},
		{"history", b.HistoryTokens},
		{"prompt", b.PromptTokens},
		{"tools", b.ToolTokens},
		{"upstream_prompt", b.UpstreamPromptTokens},
		{"upstream_cached_prompt", b.UpstreamCachedTokens},
		{"upstream_completion", b.UpstreamCompletionTokens},
		{"upstream_reasoning", b.UpstreamReasoningTokens},
	}
}
