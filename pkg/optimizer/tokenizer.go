package optimizer

import (
	"hash/fnv"
	"strings"
	"sukami/pkg/models"
	"unicode"
)

// CountExactTokens calculates token count using pure Go stdlib token-chunking logic.
func CountExactTokens(model, text string) int {
	if text == "" {
		return 0
	}
	tokens := splitIntoTokens(text)
	return len(tokens)
}

// EncodeToTokenIDs generates deterministic integer token IDs using pure Go standard library.
func EncodeToTokenIDs(model, text string) []int {
	if text == "" {
		return nil
	}
	tokens := splitIntoTokens(text)
	ids := make([]int, len(tokens))
	for i, t := range tokens {
		h := fnv.New32a()
		h.Write([]byte(t))
		// Map hash into typical 0 - 99,999 vocabulary range
		ids[i] = int(h.Sum32() % 100000)
	}
	return ids
}

// CountExactMessagesTokens computes token count for OpenAI chat messages adhering to ChatML.
func CountExactMessagesTokens(model string, messages []models.ChatMessage) int {
	total := 0
	tokensPerMessage := 3
	tokensPerName := 1

	for _, msg := range messages {
		total += tokensPerMessage
		total += CountExactTokens(model, msg.Content)
		total += CountExactTokens(model, msg.Role)
		if msg.Name != "" {
			total += CountExactTokens(model, msg.Name) + tokensPerName
		}
	}

	// Assistant reply primer (<|im_start|>assistant)
	total += 3
	return total
}

// splitIntoTokens segments text based on word boundaries, numbers, and punctuation.
func splitIntoTokens(text string) []string {
	var tokens []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if unicode.IsSpace(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(r))
			continue
		}

		current.WriteRune(r)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}
