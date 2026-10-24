package optimizer

import (
	"strings"
	"sukami/pkg/models"
	"testing"
)

// TestPruneText_CodeIntegrity verifies that code blocks are 100% preserved without any corruption.
func TestPruneText_CodeIntegrity(t *testing.T) {
	rawPrompt := "Could you please review this code:\n```go\nfunc add(a, b int) int {\n    return a + b\n}\n```\nThank you!"

	pruned := PruneText(rawPrompt)

	expectedCode := "```go\nfunc add(a, b int) int {\n    return a + b\n}\n```"
	if !strings.Contains(pruned, expectedCode) {
		t.Fatalf("Code block was corrupted!\nExpected:\n%s\nGot:\n%s", expectedCode, pruned)
	}

	if strings.Contains(pruned, "Could you please") {
		t.Errorf("Filler prefix was not removed from text: %s", pruned)
	}
}

// TestPruneText_WhitespaceCollapsing checks removal of empty newlines and spaces.
func TestPruneText_WhitespaceCollapsing(t *testing.T) {
	input := "Explain    concurrency     in   Go.\n\n\n\n\nHow    does it work?"
	pruned := PruneText(input)

	if strings.Contains(pruned, "    ") {
		t.Errorf("Multiple spaces were not collapsed: %q", pruned)
	}

	if strings.Contains(pruned, "\n\n\n") {
		t.Errorf("Excess newlines were not collapsed: %q", pruned)
	}
}

// TestOptimizeMessages_RealWorldDataset tests realistic developer and business prompts.
func TestOptimizeMessages_RealWorldDataset(t *testing.T) {
	testCases := []struct {
		name        string
		role        string
		content     string
		minSavings  int
	}{
		{
			name:        "Polite Coding Request",
			role:        "user",
			content:     "Could you please help me write a function in Go that connects to PostgreSQL? Please make sure to handle errors properly.",
			minSavings:  3,
		},
		{
			name:        "Bloated Debugging Query",
			role:        "user",
			content:     "Can you please tell me why my Docker container is crashing?    I would like you to analyze this stack trace:\n```\npanic: nil pointer\n```",
			minSavings:  4,
		},
		{
			name:        "Redundant Question Prefix",
			role:        "user",
			content:     "Would you kindly explain the difference between Goroutines and OS Threads?",
			minSavings:  3,
		},
		{
			name:        "Already Compact Code Prompt",
			role:        "user",
			content:     "Benchmark Go slice append vs preallocation.",
			minSavings:  0, // Should not corrupt already clean prompt
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msgs := []models.ChatMessage{{Role: tc.role, Content: tc.content}}
			res := OptimizeMessages(msgs)

			t.Logf("[%s] Orig: %d tokens -> Opt: %d tokens (Saved: %d, %.1f%%)",
				tc.name, res.OriginalTokens, res.OptimizedTokens, res.SavedTokens, res.SavingsPercent)

			if res.SavedTokens < tc.minSavings {
				t.Errorf("Expected at least %d saved tokens, but got %d", tc.minSavings, res.SavedTokens)
			}

			if res.OptimizedTokens > res.OriginalTokens {
				t.Errorf("Optimized tokens (%d) cannot exceed original tokens (%d)",
					res.OptimizedTokens, res.OriginalTokens)
			}
		})
	}
}

// TestCountExactTokens_TiktokenBPE verifies OpenAI BPE token counting.
func TestCountExactTokens_TiktokenBPE(t *testing.T) {
	text := "Hello, world! 123"
	count := CountExactTokens("gpt-4o", text)
	// In OpenAI BPE (o200k_base / cl100k_base), "Hello, world! 123" is 6 tokens
	if count != 6 {
		t.Fatalf("Expected 6 BPE tokens, got %d", count)
	}

	// Empty text must yield 0
	if CountExactTokens("gpt-4o", "") != 0 {
		t.Errorf("Expected 0 tokens for empty string")
	}
}

// TestEncodeToTokenIDs_TiktokenBPE verifies real BPE token integer generation.
func TestEncodeToTokenIDs_TiktokenBPE(t *testing.T) {
	text := "Hello world"
	ids := EncodeToTokenIDs("gpt-4o", text)
	if len(ids) == 0 {
		t.Fatalf("Expected non-empty token IDs from tiktoken")
	}
	// "Hello world" in cl100k_base is [9906, 1917]
	t.Logf("Token IDs for %q: %v", text, ids)
}

// BenchmarkPruneText measures execution speed of prompt pruning.
func BenchmarkPruneText(b *testing.B) {
	prompt := "Could you please help me write a function in Go that takes a list of URLs and downloads them concurrently? Please make sure to handle errors."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PruneText(prompt)
	}
}
