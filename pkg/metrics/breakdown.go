package metrics

// TokenBreakdown says where the tokens of a forwarded request went.
//
// The four input parts are estimated by the proxy's own tokenizer from the messages it forwarded, so they
// show proportions and will not add up exactly to the upstream's prompt count. The Upstream fields are the
// numbers the provider reported in its usage block, and are zero when it reported none.
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
