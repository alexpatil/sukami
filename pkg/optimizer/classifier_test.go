package optimizer

import (
	"strings"
	"testing"
)

func TestClassifyPromptComplexity_EconomyTier(t *testing.T) {
	economyQueries := []string{
		"What is the difference between TCP and UDP?",
		"Translate this sentence to French: Good morning everyone.",
		"Define latency and throughput in simple terms.",
		"Check spelling and grammar: The quick brown fox jumps.",
		"Summarize the main idea of microservices in two sentences.",
		"",
	}

	for _, q := range economyQueries {
		res := ClassifyPromptComplexity(q)
		if res.Tier != TierEconomy {
			t.Errorf("expected query %q to be TierEconomy, got %s (signals: %v)", q, res.Tier, res.Signals)
		}
	}
}

func TestClassifyPromptComplexity_FlagshipTier(t *testing.T) {
	testCases := []struct {
		name   string
		prompt string
		signal string
	}{
		{
			name:   "Markdown Code Block",
			prompt: "Can you review this code:\n```go\nfunc Worker() {}\n```",
			signal: "code_block",
		},
		{
			name:   "Inline Programming Syntax",
			prompt: "How does func main() handle goroutine channels and struct definitions in Go?",
			signal: "code_syntax",
		},
		{
			name:   "Runtime Crash Stack Trace",
			prompt: "Fix this error: panic: runtime error: invalid memory address or nil pointer dereference [goroutine 1 [running]:",
			signal: "stack_trace",
		},
		{
			name:   "Deep Reasoning and Proof",
			prompt: "Explain step by step the architectural tradeoff of Paxos vs Raft and prove that quorum intersection holds.",
			signal: "reasoning",
		},
		{
			name:   "Long Context Workload",
			prompt: strings.Repeat("Lorem ipsum dolor sit amet, consectetur adipiscing elit. ", 50),
			signal: "long_context",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := ClassifyPromptComplexity(tc.prompt)
			if res.Tier != TierFlagship {
				t.Fatalf("expected TierFlagship, got %s", res.Tier)
			}
			foundSignal := false
			for _, sig := range res.Signals {
				if strings.HasPrefix(sig, tc.signal) {
					foundSignal = true
					break
				}
			}
			if !foundSignal {
				t.Errorf("expected signal %q in %v", tc.signal, res.Signals)
			}
		})
	}
}

func BenchmarkClassifyPromptComplexity(b *testing.B) {
	prompt := "What is the difference between concurrency and parallelism in operating systems?"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ClassifyPromptComplexity(prompt)
	}
}
