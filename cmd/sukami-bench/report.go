package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sukami/pkg/models"
	"time"
)

const (
	caseDirect   = "Direct to upstream"
	caseProxy    = "Via Sukami (defaults)"
	caseCacheHit = "Via Sukami, same request again (cache hit)"
	caseDensity  = "Via Sukami with X-Sukami-Density: true"
)

var caseOrder = []string{caseDirect, caseProxy, caseCacheHit, caseDensity}

// sample is one measured request. Token counts come from the upstream's own usage block.
type sample struct {
	promptTokens     int
	completionTokens int
	latency          time.Duration
	finish           string
	cache            string
}

func call(client *http.Client, target, apiKey, model string, headers map[string]string) (sample, error) {
	body, err := json.Marshal(models.ChatCompletionRequest{
		Model:    model,
		Messages: []models.ChatMessage{{Role: "user", Content: benchPrompt}},
	})
	if err != nil {
		return sample{}, err
	}
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return sample{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return sample{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	latency := time.Since(start)
	if err != nil {
		return sample{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return sample{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, raw)
	}

	var parsed models.ChatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return sample{}, fmt.Errorf("response is not a chat completion: %w", err)
	}
	if len(parsed.Choices) == 0 || parsed.Usage.CompletionTokens == 0 {
		return sample{}, fmt.Errorf("response carries no choice or no usage block: %s", raw)
	}
	return sample{
		promptTokens:     parsed.Usage.PromptTokens,
		completionTokens: parsed.Usage.CompletionTokens,
		latency:          latency,
		finish:           parsed.Choices[0].FinishReason,
		cache:            resp.Header.Get("X-Cache"),
	}, nil
}

// spread formats the median of the values with their range.
func spread[T int | time.Duration](values []T, format func(T) string) string {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	median := sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		median = (sorted[len(sorted)/2-1] + median) / 2
	}
	if sorted[0] == sorted[len(sorted)-1] {
		return format(median)
	}
	return fmt.Sprintf("%s (%s to %s)", format(median), format(sorted[0]), format(sorted[len(sorted)-1]))
}

func renderReport(cfg benchConfig, results map[string][]sample, when time.Time) string {
	host := cfg.upstreamURL
	if u, err := url.Parse(cfg.upstreamURL); err == nil && u.Host != "" {
		host = u.Host
	}
	tokens := func(n int) string { return fmt.Sprintf("%d", n) }
	seconds := func(d time.Duration) string {
		if d < time.Second {
			return fmt.Sprintf("%dms", d.Milliseconds())
		}
		return fmt.Sprintf("%.2fs", d.Seconds())
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Date: %s. Model: `%s`. Upstream: `%s`. Runs per case: %d.\n", when.Format("2006-01-02"), cfg.model, host, cfg.runs)
	sb.WriteString("Cells show the median, with the range across runs in parentheses.\n\n")
	sb.WriteString("| Case | Prompt tokens | Completion tokens | Latency | Finish reasons |\n")
	sb.WriteString("| :--- | :---: | :---: | :---: | :---: |\n")
	for _, name := range caseOrder {
		samples := results[name]
		if len(samples) == 0 {
			continue
		}
		var prompt, completion []int
		var latency []time.Duration
		var finishes []string
		for _, s := range samples {
			prompt = append(prompt, s.promptTokens)
			completion = append(completion, s.completionTokens)
			latency = append(latency, s.latency)
			if !slices.Contains(finishes, s.finish) {
				finishes = append(finishes, s.finish)
			}
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | `%s` |\n", name, spread(prompt, tokens), spread(completion, tokens), spread(latency, seconds), strings.Join(finishes, "`, `"))
	}
	return sb.String()
}
