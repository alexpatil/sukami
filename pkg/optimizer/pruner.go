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
	// Multiple spaces or tabs into a single space.
	multiSpaceRegex = regexp.MustCompile(`[ \t]{2,}`)
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

// EstimateMessagesTokensForModel computes exact token count using model-specific tokenization rules.
func EstimateMessagesTokensForModel(model string, messages []models.ChatMessage) int {
	if model == "" {
		model = "gpt-4o"
	}
	return CountExactMessagesTokens(model, messages)
}

// PruneText compresses prompt text without corrupting code or essential semantics.
func PruneText(input string) string {
	if len(input) == 0 {
		return input
	}

	// Step 1: Protect code blocks using collision-proof placeholders
	seed := time.Now().UnixNano()
	var preservedBlocks []string

	cleanText := codeBlockRegex.ReplaceAllStringFunc(input, func(block string) string {
		idx := len(preservedBlocks)
		preservedBlocks = append(preservedBlocks, block)
		return fmt.Sprintf("\x1fSUKAMI_%d_%d\x1f", seed, idx)
	})

	// Step 2: Remove leading conversational filler from lines and sentences
	lines := strings.Split(cleanText, "\n")
	for i, line := range lines {
		line = stripFillerPrefix(line)
		for _, sep := range []string{"? ", ". ", "! "} {
			parts := strings.Split(line, sep)
			if len(parts) > 1 {
				for j := 1; j < len(parts); j++ {
					parts[j] = stripFillerPrefix(parts[j])
				}
				line = strings.Join(parts, sep)
			}
		}
		lines[i] = line
	}
	cleanText = strings.Join(lines, "\n")

	// Step 3: Collapse whitespace and extra newlines
	cleanText = multiSpaceRegex.ReplaceAllString(cleanText, " ")
	cleanText = multiNewlineRegex.ReplaceAllString(cleanText, "\n\n")
	cleanText = strings.TrimSpace(cleanText)

	// Step 4: Restore preserved code blocks intact
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
// Multimodal messages (image arrays) are passed through untouched; system prompts are preserved intact.
func OptimizeMessagesForModel(model string, messages []models.ChatMessage) CompressionResult {
	origTokens := EstimateMessagesTokensForModel(model, messages)

	optimized := make([]models.ChatMessage, len(messages))
	for i, msg := range messages {
		optimized[i] = msg // Copy all fields including passthrough (ToolCalls, RawContent, etc.)
		if msg.Role != "system" && models.IsTextContent(msg.RawContent) {
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

func stripFillerPrefix(s string) string {
	trimmed := strings.TrimSpace(s)
	lower := strings.ToLower(trimmed)
	for _, filler := range fillerPrefixes {
		if strings.HasPrefix(lower, filler) {
			trimmed = trimmed[len(filler):]
			if len(trimmed) > 0 {
				runes := []rune(trimmed)
				runes[0] = unicode.ToUpper(runes[0])
				trimmed = string(runes)
			}
			break
		}
	}
	return trimmed
}
