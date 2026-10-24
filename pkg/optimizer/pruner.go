package optimizer

import (
	"regexp"
	"strings"
	"sukami/pkg/models"
	"unicode"
)

var (
	// Regex to isolate markdown code blocks so they are NOT pruned.
	codeBlockRegex = regexp.MustCompile("(?s)```.*?```|`.*?`")
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

// EstimateTokens calculates the exact BPE token count for a text string using OpenAI tiktoken.
func EstimateTokens(text string) int {
	return CountExactTokens("gpt-4o", text)
}

// EstimateMessagesTokens computes 100% exact token count across all chat messages.
func EstimateMessagesTokens(messages []models.ChatMessage) int {
	return CountExactMessagesTokens("gpt-4o", messages)
}

// PruneText compresses prompt text without corrupting code or essential semantics.
func PruneText(input string) string {
	if len(input) == 0 {
		return input
	}

	// Step 1: Protect code blocks
	var preservedBlocks []string
	placeholderPattern := "__CODE_BLOCK_%d__"

	cleanText := codeBlockRegex.ReplaceAllStringFunc(input, func(block string) string {
		idx := len(preservedBlocks)
		preservedBlocks = append(preservedBlocks, block)
		return strings.Replace(placeholderPattern, "%d", string(rune('0'+idx)), 1)
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
		placeholder := strings.Replace(placeholderPattern, "%d", string(rune('0'+idx)), 1)
		cleanText = strings.Replace(cleanText, placeholder, block, 1)
	}

	return cleanText
}

// OptimizeMessages processes the slice of chat messages and returns pruned messages with stats.
func OptimizeMessages(messages []models.ChatMessage) CompressionResult {
	origTokens := EstimateMessagesTokens(messages)

	optimized := make([]models.ChatMessage, len(messages))
	for i, msg := range messages {
		optimized[i] = models.ChatMessage{
			Role:    msg.Role,
			Name:    msg.Name,
			Content: PruneText(msg.Content),
		}
	}

	optTokens := EstimateMessagesTokens(optimized)
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
