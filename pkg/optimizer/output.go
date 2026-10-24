package optimizer

import (
	"regexp"
	"strings"
	"sukami/pkg/models"
)

const (
	// OutputDensityDirective instructs LLMs to omit wasteful preambles and postambles,
	// focusing strictly on high-density solutions, complete code, and factual depth.
	OutputDensityDirective = "Output Policy: Direct, dense, and complete answers only. Omit conversational preambles (e.g., 'Sure!', 'Certainly, I can help with that'), conversational sign-offs ('Hope this helps! Let me know if you need anything else'), and redundant summaries. Deliver full depth and code precision immediately."
)

var (
	leadingFluffRegex = regexp.MustCompile(`(?i)^(certainly!?[^\n]*\n*|sure!?[^\n]*\n*|of course!?[^\n]*\n*|i would be happy to help[^\n]*\n*|here is the (code|solution|response)[^\n]*:\s*\n*)`)
	trailingFluffRegex = regexp.MustCompile(`(?i)\n*(hope this helps!?[^\n]*|let me know if you (need|have) (any|more) (other )?questions!?[^\n]*|feel free to ask if you need[^\n]*|happy coding!?[^\n]*)\s*$`)
)

// ApplyOutputOptimization injects output density directives into system prompt,
// forcing upstream LLMs to generate high-density output without conversational fluff.
func ApplyOutputOptimization(messages []models.ChatMessage) ([]models.ChatMessage, bool) {
	if len(messages) == 0 {
		return messages, false
	}

	result := make([]models.ChatMessage, 0, len(messages)+1)
	systemFound := false

	for _, msg := range messages {
		if msg.Role == "system" && !systemFound {
			// Append directive to existing system prompt
			enhancedContent := msg.Content
			if !strings.Contains(enhancedContent, "Output Policy:") {
				enhancedContent = enhancedContent + "\n\n" + OutputDensityDirective
			}
			result = append(result, models.ChatMessage{
				Role:    msg.Role,
				Name:    msg.Name,
				Content: enhancedContent,
			})
			systemFound = true
		} else {
			result = append(result, msg)
		}
	}

	// If no system prompt exists, prepend one
	if !systemFound {
		sysMsg := models.ChatMessage{
			Role:    "system",
			Content: OutputDensityDirective,
		}
		result = append([]models.ChatMessage{sysMsg}, result...)
	}

	return result, true
}

// CleanOutputFluff strips leading and trailing filler phrases from output text
// and returns the cleaned text along with the number of saved output tokens.
func CleanOutputFluff(output string) (string, int) {
	if len(output) == 0 {
		return output, 0
	}

	initialTokens := EstimateTokens(output)

	cleaned := leadingFluffRegex.ReplaceAllString(output, "")
	cleaned = trailingFluffRegex.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)

	afterTokens := EstimateTokens(cleaned)
	saved := initialTokens - afterTokens
	if saved < 0 {
		saved = 0
	}

	return cleaned, saved
}

// CalculateOutputCostSaved estimates dollar savings for output tokens saved.
// ratePerMillion is in USD (e.g., 15.0 for Claude Sonnet, 10.0 for GPT-4o, 0.79 for Groq).
func CalculateOutputCostSaved(savedTokens int, ratePerMillion float64) float64 {
	if savedTokens <= 0 || ratePerMillion <= 0 {
		return 0.0
	}
	return (float64(savedTokens) / 1000000.0) * ratePerMillion
}
