package optimizer

import (
	"fmt"
	"regexp"
	"strings"
	"sukami/pkg/models"
	"time"
	"unicode"
)

var (
	// Regex to isolate markdown code blocks (closed and unclosed) so they are NOT pruned.
	codeBlockRegex = regexp.MustCompile("(?s)```(?:.*?```|.*)|`.*?`")
	// More than two consecutive newlines into double newlines.
	multiNewlineRegex = regexp.MustCompile(`\n{3,}`)
	// Common polite/filler prefixes that burn prompt tokens without providing instruction value.
	fillerPrefixes = []string{
		"could you please ",
		"can you please ",
		"please ",
		"would you kindly ",
		"i would like you to ",
		"i want you to ",
		"kindly ",
		"can you help me to ",
		"can you tell me ",
		"please make sure to ",
	}
)

// CompressionResult holds stats on optimized prompt.
type CompressionResult struct {
	OriginalTokens  int
	OptimizedTokens int
	SavedTokens     int
	SavingsPercent  float64
	Messages        []models.ChatMessage
}

// EstimateTokens calculates the token count for a text string in pure Go (defaults to gpt-4o).
func EstimateTokens(text string) int {
	return EstimateTokensForModel("gpt-4o", text)
}

// EstimateTokensForModel calculates the model-aware token count in pure Go.
func EstimateTokensForModel(model, text string) int {
	if model == "" {
		model = "gpt-4o"
	}
	return CountExactTokens(model, text)
}

// EstimateMessagesTokens computes token count across all chat messages (defaults to gpt-4o).
func EstimateMessagesTokens(messages []models.ChatMessage) int {
	return EstimateMessagesTokensForModel("gpt-4o", messages)
}

// EstimateMessagesTokensForModel estimates token count using model-specific tokenization rules.
func EstimateMessagesTokensForModel(model string, messages []models.ChatMessage) int {
	if model == "" {
		model = "gpt-4o"
	}
	return CountExactMessagesTokens(model, messages)
}

// PruneText removes text that carries no instruction: a polite prefix at the very start of the prompt,
// trailing whitespace, and runs of blank lines. Indentation and spacing inside a line are kept, because
// they carry meaning in code, YAML and tables that are pasted without a fence.
func PruneText(input string) string {
	if len(input) == 0 {
		return input
	}

	// Protect code blocks using collision-proof placeholders
	seed := time.Now().UnixNano()
	var preservedBlocks []string

	cleanText := codeBlockRegex.ReplaceAllStringFunc(input, func(block string) string {
		idx := len(preservedBlocks)
		preservedBlocks = append(preservedBlocks, block)
		return fmt.Sprintf("\x1fSUKAMI_%d_%d\x1f", seed, idx)
	})

	lines := strings.Split(cleanText, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	cleanText = strings.Trim(strings.Join(lines, "\n"), "\n")
	cleanText = multiNewlineRegex.ReplaceAllString(cleanText, "\n\n")
	cleanText = stripFillerPrefix(cleanText)

	// Restore preserved code blocks intact
	for idx, block := range preservedBlocks {
		placeholder := fmt.Sprintf("\x1fSUKAMI_%d_%d\x1f", seed, idx)
		cleanText = strings.Replace(cleanText, placeholder, block, 1)
	}

	return cleanText
}

// OptimizeMessages processes chat messages using default model tokenization.
func OptimizeMessages(messages []models.ChatMessage) CompressionResult {
	return OptimizeMessagesForModel("gpt-4o", messages)
}

// OptimizeMessagesForModel processes the slice of chat messages using model-specific token rules.
// Only text written by the user is pruned; system, assistant, tool and multimodal messages pass through untouched.
func OptimizeMessagesForModel(model string, messages []models.ChatMessage) CompressionResult {
	origTokens := EstimateMessagesTokensForModel(model, messages)

	optimized := make([]models.ChatMessage, len(messages))
	for i, msg := range messages {
		optimized[i] = msg // Copy all fields including passthrough (ToolCalls, RawContent, etc.)
		if msg.Role == "user" && models.IsTextContent(msg.RawContent) {
			optimized[i].Content = PruneText(msg.Content)
			optimized[i].RawContent = nil // Force marshal to use pruned text Content
		}
	}

	optTokens := EstimateMessagesTokensForModel(model, optimized)
	saved := origTokens - optTokens
	if saved < 0 {
		saved = 0
		optTokens = origTokens
	}

	savingsPct := 0.0
	if origTokens > 0 {
		savingsPct = (float64(saved) / float64(origTokens)) * 100.0
	}

	return CompressionResult{
		OriginalTokens:  origTokens,
		OptimizedTokens: optTokens,
		SavedTokens:     saved,
		SavingsPercent:  savingsPct,
		Messages:        optimized,
	}
}

// stripFillerPrefix drops one polite prefix from the start of the prompt. Text that starts with
// whitespace is left alone, since an indented first line is more likely pasted content than a request.
func stripFillerPrefix(s string) string {
	lower := strings.ToLower(s)
	for _, filler := range fillerPrefixes {
		if strings.HasPrefix(lower, filler) {
			rest := []rune(s[len(filler):])
			if len(rest) > 0 {
				rest[0] = unicode.ToUpper(rest[0])
			}
			return string(rest)
		}
	}
	return s
}
