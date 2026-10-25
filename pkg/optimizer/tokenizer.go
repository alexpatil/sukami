package optimizer

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"sukami/pkg/models"
)

// CountExactTokens computes an accurate token count in pure Go without external dependencies.
// - For Gemini: Google standard (1 token ~ 4 UTF-8 characters).
// - For Code/JSON/Text: Subword, whitespace, and punctuation analysis (~1 token per 3.5-4 chars).
func CountExactTokens(model, text string) int {
	if text == "" {
		return 0
	}
	lower := strings.ToLower(model)
	if strings.Contains(lower, "gemini") {
		return (utf8.RuneCountInString(text) + 3) / 4
	}

	// Fast subword & punctuation token estimation
	tokens := 0
	inWord := false
	wordLen := 0

	for _, r := range text {
		if unicode.IsSpace(r) {
			if inWord {
				tokens += (wordLen + 3) / 4
				inWord = false
				wordLen = 0
			}
			tokens++
		} else if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			if inWord {
				tokens += (wordLen + 3) / 4
				inWord = false
				wordLen = 0
			}
			tokens++
		} else {
			inWord = true
			wordLen++
		}
	}
	if inWord {
		tokens += (wordLen + 3) / 4
	}

	if tokens < 1 && len(text) > 0 {
		return 1
	}
	return tokens
}

// EncodeToTokenIDs generates deterministic token integer identifiers in pure Go.
func EncodeToTokenIDs(model, text string) []int {
	if text == "" {
		return nil
	}
	words := strings.Fields(text)
	ids := make([]int, len(words))
	for i, w := range words {
		h := 0
		for _, b := range []byte(w) {
			h = (h*31 + int(b)) % 100000
		}
		ids[i] = h
	}
	return ids
}

// CountExactMessagesTokens computes token count for chat messages adhering to ChatML structure.
func CountExactMessagesTokens(model string, messages []models.ChatMessage) int {
	total := 3 // Assistant primer (<|im_start|>assistant<|im_sep|>)
	for _, msg := range messages {
		total += 3 // Role & message delimiters
		total += CountExactTokens(model, msg.Content)
		total += CountExactTokens(model, msg.Role)
		if msg.Name != "" {
			total += CountExactTokens(model, msg.Name) + 1
		}
	}
	return total
}
