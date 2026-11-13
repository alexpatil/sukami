package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"sukami/pkg/proxy"
	"time"
)

//go:embed chat_scenario.json
var defaultChatScenario []byte

// chatScenario is one multi-turn conversation: a system prompt and the user's turns in order.
type chatScenario struct {
	System string   `json:"system"`
	Turns  []string `json:"turns"`
}

// loadChatScenario reads the scenario from BENCH_CHAT_FILE, or the bundled one when it is unset.
func loadChatScenario() (chatScenario, error) {
	raw := defaultChatScenario
	if path := os.Getenv("BENCH_CHAT_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return chatScenario{}, err
		}
		raw = data
	}
	var s chatScenario
	if err := json.Unmarshal(raw, &s); err != nil {
		return chatScenario{}, fmt.Errorf("chat scenario is not valid JSON: %w", err)
	}
	if s.System == "" || len(s.Turns) == 0 {
		return chatScenario{}, fmt.Errorf("chat scenario needs a system prompt and at least one turn")
	}
	return s, nil
}

// chatTotals sums what the upstream billed across every turn of one conversation.
type chatTotals struct {
	requests         int
	promptTokens     int
	cachedTokens     int
	completionTokens int
	reasoningTokens  int
	finishes         []string
	// breakdown is the proxy's own attribution of the forwarded input; zero for the direct path.
	breakdown metrics.TokenBreakdown
}

const (
	chatDirect  = "Direct to upstream"
	chatProxy   = "Via Sukami (defaults)"
	chatDensity = "Via Sukami with X-Sukami-Density: true"
)

var chatOrder = []string{chatDirect, chatProxy, chatDensity}

// playChat sends the conversation turn by turn, resending the history each time as a chat client does.
// The answers that come back become the history of the next turn.
func playChat(cfg benchConfig, client *http.Client, target string, headers map[string]string, scenario chatScenario, sessionTag string) (chatTotals, error) {
	var totals chatTotals
	// A tag unique to this path keeps one path from being served the provider-side prompt cache another warmed.
	messages := []models.ChatMessage{{Role: "system", Content: "Session " + sessionTag + ".\n" + scenario.System}}

	for i, turn := range scenario.Turns {
		messages = append(messages, models.ChatMessage{Role: "user", Content: turn})
		resp, _, _, err := postChat(client, target, cfg.apiKey, models.ChatCompletionRequest{Model: cfg.model, Messages: messages}, headers)
		if err != nil {
			return totals, fmt.Errorf("turn %d: %w", i+1, err)
		}
		totals.requests++
		totals.promptTokens += resp.Usage.PromptTokens
		totals.completionTokens += resp.Usage.CompletionTokens
		if d := resp.Usage.PromptTokensDetails; d != nil {
			totals.cachedTokens += d.CachedTokens
		}
		if d := resp.Usage.CompletionTokensDetails; d != nil {
			totals.reasoningTokens += d.ReasoningTokens
		}
		totals.finishes = append(totals.finishes, resp.Choices[0].FinishReason)
		messages = append(messages, models.ChatMessage{Role: "assistant", Content: resp.Choices[0].Message.Content})
		time.Sleep(cfg.pause)
	}
	return totals, nil
}

// mainChat runs the chat scenario cfg.runs times and prints the report.
func mainChat(cfg benchConfig, client *http.Client) error {
	scenario, err := loadChatScenario()
	if err != nil {
		return err
	}
	results := map[string][]chatTotals{}
	for i := 1; i <= cfg.runs; i++ {
		fmt.Fprintf(os.Stderr, "conversation run %d of %d\n", i, cfg.runs)
		run, err := runChat(cfg, client, scenario, i)
		if err != nil {
			return err
		}
		for name, totals := range run {
			results[name] = append(results[name], totals)
		}
	}
	fmt.Print(renderChatReport(cfg, len(scenario.Turns), results, time.Now()))
	return nil
}

// runChat plays the conversation once on each path. Every proxied path gets a fresh proxy.
func runChat(cfg benchConfig, client *http.Client, scenario chatScenario, run int) (map[string]chatTotals, error) {
	out := map[string]chatTotals{}
	tag := func(path string) string { return fmt.Sprintf("%d-%d-%s", time.Now().UnixNano(), run, path) }

	direct, err := playChat(cfg, client, proxy.CompletionsURL(cfg.upstreamURL), nil, scenario, tag("direct"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", chatDirect, err)
	}
	out[chatDirect] = direct

	for _, path := range []struct {
		name    string
		headers map[string]string
	}{
		{chatProxy, nil},
		{chatDensity, map[string]string{"X-Sukami-Density": "true"}},
	} {
		proxyURL, tracker, stop, err := startTrackedProxy(cfg.upstreamURL)
		if err != nil {
			return nil, err
		}
		totals, err := playChat(cfg, client, proxyURL, path.headers, scenario, tag("proxy"))
		stop()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path.name, err)
		}
		totals.breakdown = tracker.GetSummary().TokenBreakdown
		out[path.name] = totals
	}
	return out, nil
}
