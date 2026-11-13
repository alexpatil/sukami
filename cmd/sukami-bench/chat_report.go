package main

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
)

// renderChatReport prints what the upstream billed for the whole conversation on each path.
func renderChatReport(cfg benchConfig, turns int, results map[string][]chatTotals, when time.Time) string {
	host := cfg.upstreamURL
	if u, err := url.Parse(cfg.upstreamURL); err == nil && u.Host != "" {
		host = u.Host
	}
	tokens := func(n int) string { return fmt.Sprintf("%d", n) }
	column := func(runs []chatTotals, pick func(chatTotals) int) string {
		values := make([]int, len(runs))
		for i, r := range runs {
			values[i] = pick(r)
		}
		return spread(values, tokens)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Date: %s. Model: `%s`. Upstream: `%s`. Conversation: %d turns. Runs per path: %d.\n", when.Format("2006-01-02"), cfg.model, host, turns, cfg.runs)
	sb.WriteString("Each cell is the total over all turns of one conversation, as billed by the upstream: the median, with the range across runs in parentheses.\n\n")
	sb.WriteString("| Path | Prompt tokens | of which provider-cached | Completion tokens | of which reasoning | Prompt + completion | Finish reasons |\n")
	sb.WriteString("| :--- | :---: | :---: | :---: | :---: | :---: | :---: |\n")
	for _, name := range chatOrder {
		runs := results[name]
		if len(runs) == 0 {
			continue
		}
		var finishes []string
		for _, r := range runs {
			for _, f := range r.finishes {
				if !slices.Contains(finishes, f) {
					finishes = append(finishes, f)
				}
			}
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s | %s | `%s` |\n", name,
			column(runs, func(t chatTotals) int { return t.promptTokens }),
			column(runs, func(t chatTotals) int { return t.cachedTokens }),
			column(runs, func(t chatTotals) int { return t.completionTokens }),
			column(runs, func(t chatTotals) int { return t.reasoningTokens }),
			column(runs, func(t chatTotals) int { return t.promptTokens + t.completionTokens }),
			strings.Join(finishes, "`, `"))
	}

	sb.WriteString("\nWhere the billed prompt tokens went: the upstream's prompt count for each request, split in the proportions the proxy estimates.\n\n")
	sb.WriteString("| Path | System prompt | History (earlier turns) | New prompt | Tools |\n")
	sb.WriteString("| :--- | :---: | :---: | :---: | :---: |\n")
	for _, name := range chatOrder[1:] {
		runs := results[name]
		if len(runs) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s |\n", name,
			column(runs, func(t chatTotals) int { return int(t.breakdown.SystemTokens) }),
			column(runs, func(t chatTotals) int { return int(t.breakdown.HistoryTokens) }),
			column(runs, func(t chatTotals) int { return int(t.breakdown.PromptTokens) }),
			column(runs, func(t chatTotals) int { return int(t.breakdown.ToolTokens) }))
	}
	return sb.String()
}
