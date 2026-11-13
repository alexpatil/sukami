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

func TestPruneText_CollapsesBlankLinesAndTrailingSpace(t *testing.T) {
	pruned := PruneText("Explain concurrency in Go.   \n\n\n\n\nHow does it work?\t\n\n")
	want := "Explain concurrency in Go.\n\nHow does it work?"
	if pruned != want {
		t.Errorf("got %q, want %q", pruned, want)
	}
}

// Indentation and spacing carry meaning in code, YAML and tables pasted without a fence.
func TestPruneText_KeepsIndentationAndInnerSpacing(t *testing.T) {
	for _, input := range []string{
		"def f(x):\n    if x:\n        return 1\n    return 2",
		"services:\n  web:\n    image: nginx",
		"| a   | b   |\n| --- | --- |",
		"    indented first line\nsecond line",
		"x = \"two  spaces\"",
	} {
		if pruned := PruneText(input); pruned != input {
			t.Errorf("text was changed:\n in: %q\nout: %q", input, pruned)
		}
	}
}

// A polite prefix is removed only at the very start of the prompt, never inside it,
// because text further in may be content the user wants processed as written.
func TestPruneText_StripsFillerOnlyAtPromptStart(t *testing.T) {
	cases := map[string]string{
		"Please summarise this report.":                    "Summarise this report.",
		"Could you please list the steps?":                 "List the steps?",
		"Translate to French:\nPlease wait here.":          "Translate to French:\nPlease wait here.",
		"Reply with the sentence: Hello. Please sit down.": "Reply with the sentence: Hello. Please sit down.",
	}
	for input, want := range cases {
		if pruned := PruneText(input); pruned != want {
			t.Errorf("PruneText(%q) = %q, want %q", input, pruned, want)
		}
	}
}

func TestOptimizeMessages_PrunesOnlyUserMessages(t *testing.T) {
	msgs := []models.ChatMessage{
		{Role: "system", Content: "Please answer briefly.   "},
		{Role: "user", Content: "Please explain channels."},
		{Role: "assistant", Content: "Please note:   channels are typed.   "},
		{Role: "tool", Content: "please   keep\n\n\n\nthis"},
	}
	got := OptimizeMessages(msgs).Messages

	if got[1].Content != "Explain channels." {
		t.Errorf("user message was not pruned: %q", got[1].Content)
	}
	for _, i := range []int{0, 2, 3} {
		if got[i].Content != msgs[i].Content {
			t.Errorf("%s message was changed: %q", msgs[i].Role, got[i].Content)
		}
	}
}

// TestPruneText_EdgeCases tests adversarial and malformed prompt edge cases.
func TestPruneText_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		mustContain string
	}{
		{
			name:        "Adversarial Placeholder In Prompt",
			input:       "Explain the meaning of __CODE_BLOCK_0__ in compilers:\n```go\nfunc run() { println(\"ok\") }\n```",
			mustContain: "Explain the meaning of __CODE_BLOCK_0__",
		},
		{
			name:        "Unclosed Code Block",
			input:       "Review this snippet:\n```python\nplease def test():\n    return True",
			mustContain: "please def test():",
		},
		{
			name:        "Whitespace Only Input",
			input:       "   \n\t  \n  ",
			mustContain: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pruned := PruneText(tc.input)
			if tc.mustContain != "" && !strings.Contains(pruned, tc.mustContain) {
				t.Errorf("FAIL: Expected output to contain %q, but got:\n%s", tc.mustContain, pruned)
			}
		})
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

// TestCountExactTokens verifies pure Go token estimation.
func TestCountExactTokens(t *testing.T) {
	text := "Hello, world! 123"
	count := CountExactTokens("gpt-4o", text)
	if count <= 0 {
		t.Fatalf("Expected positive token count, got %d", count)
	}

	// Empty text must yield 0
	if CountExactTokens("gpt-4o", "") != 0 {
		t.Errorf("Expected 0 tokens for empty string")
	}

	// Gemini model test
	geminiCount := CountExactTokens("gemini-1.5-pro", text)
	if geminiCount <= 0 {
		t.Fatalf("Expected positive Gemini token count, got %d", geminiCount)
	}
}

// TestEncodeToTokenIDs verifies token integer generation in pure Go.
func TestEncodeToTokenIDs(t *testing.T) {
	text := "Hello world"
	ids := EncodeToTokenIDs("gpt-4o", text)
	if len(ids) == 0 {
		t.Fatalf("Expected non-empty token IDs")
	}
	t.Logf("Token IDs for %q: %v", text, ids)
}

// TestOptimizeMessagesForModel verifies model-specific optimization token counts.
func TestOptimizeMessagesForModel(t *testing.T) {
	msgs := []models.ChatMessage{
		{Role: "user", Content: "Could you please explain how Goroutines work? Thank you so much."},
	}

	resGPT := OptimizeMessagesForModel("gpt-4o", msgs)
	resGemini := OptimizeMessagesForModel("gemini-1.5-flash", msgs)

	if resGPT.SavedTokens <= 0 {
		t.Errorf("Expected saved tokens for gpt-4o, got %d", resGPT.SavedTokens)
	}
	if resGemini.SavedTokens <= 0 {
		t.Errorf("Expected saved tokens for gemini, got %d", resGemini.SavedTokens)
	}
}

// BenchmarkPruneText measures execution speed of prompt pruning.
func BenchmarkPruneText(b *testing.B) {
	prompt := "Could you please help me write a function in Go that takes a list of URLs and downloads them concurrently? Please make sure to handle errors."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PruneText(prompt)
	}
}
