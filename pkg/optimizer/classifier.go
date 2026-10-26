package optimizer

import (
	"regexp"
	"strings"
)

// ComplexityTier represents the computational class required for a prompt.
type ComplexityTier string

const (
	TierEconomy  ComplexityTier = "economy"
	TierFlagship ComplexityTier = "flagship"
)

// ClassificationResult holds the complexity evaluation of a prompt.
type ClassificationResult struct {
	Tier    ComplexityTier
	Signals []string
	Reason  string
}

var (
	classifierCodeBlockRegex = regexp.MustCompile("(?s)```.*?```|`[^`\n]{4,}`")
	stackTraceRegex          = regexp.MustCompile(`(?i)(panic:|traceback \(most recent call last\):|nullpointerexception|goroutine \d+|fatal error:|at [a-z0-9_.]+\([a-z0-9_.]+\.java:\d+\))`)
)

var codeKeywords = []string{
	"func ", "def ", "class ", "import ", "package ", "SELECT ", "FROM ", "WHERE ",
	"const ", "interface ", "struct ", "console.log", "public static void", "return ",
	"try {", "except:", "catch (", "goroutine", "dockerfile", "yaml", "json",
}

var reasoningKeywords = []string{
	"step by step", "step-by-step", "think carefully", "chain of thought",
	"prove that", "mathematical proof", "derive the", "architectural tradeoff",
	"system design", "distributed consensus", "race condition", "memory leak",
	"security vulnerability", "optimize algorithm", "time complexity", "o(n",
}

// ClassifyPromptComplexity analyzes lexical and syntactic signals in sub-millisecond time.
func ClassifyPromptComplexity(prompt string) ClassificationResult {
	trimmed := strings.TrimSpace(prompt)
	if len(trimmed) == 0 {
		return ClassificationResult{
			Tier:    TierEconomy,
			Reason:  "empty prompt default to economy",
			Signals: []string{"empty"},
		}
	}

	var signals []string

	// 1. Check for explicit code blocks or backtick expressions
	if classifierCodeBlockRegex.MatchString(trimmed) {
		signals = append(signals, "code_block")
	}

	// 2. Check for stack traces or runtime crash errors
	if stackTraceRegex.MatchString(trimmed) {
		signals = append(signals, "stack_trace")
	}

	// 3. Check for programming keywords (low-cased check)
	lower := strings.ToLower(trimmed)
	for _, kw := range codeKeywords {
		if strings.Contains(lower, strings.ToLower(kw)) {
			signals = append(signals, "code_syntax:"+strings.TrimSpace(kw))
			break
		}
	}

	// 4. Check for deep analytical / reasoning requirements
	for _, rw := range reasoningKeywords {
		if strings.Contains(lower, rw) {
			signals = append(signals, "reasoning:"+strings.TrimSpace(rw))
			break
		}
	}

	// 5. Check prompt length (> 2000 characters ~ 500+ tokens)
	if len(trimmed) > 2000 {
		signals = append(signals, "long_context")
	}

	// Decision Engine
	if len(signals) > 0 {
		return ClassificationResult{
			Tier:    TierFlagship,
			Signals: signals,
			Reason:  "complex workload requires flagship reasoning (" + strings.Join(signals, ", ") + ")",
		}
	}

	return ClassificationResult{
		Tier:    TierEconomy,
		Signals: []string{"conversational_or_factual"},
		Reason:  "direct factual/conversational query suitable for economy tier",
	}
}

// ResolveSmartModel maps requested model to optimal tier model to reduce cost.
func ResolveSmartModel(originalModel string, tier ComplexityTier) (routedModel string, changed bool, reason string) {
	origLower := strings.ToLower(strings.TrimSpace(originalModel))

	switch {
	case strings.Contains(origLower, "gpt-4o") && !strings.Contains(origLower, "mini"):
		if tier == TierEconomy {
			return "gpt-4o-mini", true, "low-complexity prompt routed to gpt-4o-mini (~94% cost savings)"
		}
		return originalModel, false, "complex prompt retained on flagship gpt-4o"

	case origLower == "gpt-4" || origLower == "gpt-4-turbo":
		if tier == TierEconomy {
			return "gpt-4o-mini", true, "low-complexity prompt routed to gpt-4o-mini (~98% cost savings)"
		}
		return originalModel, false, "complex prompt retained on flagship gpt-4"

	case strings.Contains(origLower, "claude-3-5-sonnet") || strings.Contains(origLower, "claude-3-opus"):
		if tier == TierEconomy {
			if strings.HasPrefix(origLower, "openrouter/") {
				return "openrouter/anthropic/claude-3.5-haiku", true, "low-complexity prompt routed to claude-3.5-haiku (~80% cost savings)"
			}
			return "claude-3-5-haiku", true, "low-complexity prompt routed to claude-3.5-haiku (~80% cost savings)"
		}
		return originalModel, false, "complex prompt retained on flagship claude"

	case strings.Contains(origLower, "llama-3.3-70b") || strings.Contains(origLower, "llama-3-70b"):
		if tier == TierEconomy {
			return "llama-3.1-8b-instant", true, "low-complexity prompt routed to llama-3.1-8b-instant (~87% cost savings)"
		}
		return originalModel, false, "complex prompt retained on flagship 70B model"

	case strings.Contains(origLower, "deepseek-reasoner"):
		if tier == TierEconomy {
			return "deepseek-chat", true, "non-reasoning query routed to deepseek-chat (~75% cost savings)"
		}
		return originalModel, false, "reasoning query retained on deepseek-reasoner"
	}

	return originalModel, false, "model already optimal or unmapped"
}
