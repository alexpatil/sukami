package optimizer

import (
	"fmt"
	"strings"
)

// UpstreamProvider represents the detected backend provider ecosystem.
type UpstreamProvider string

const (
	ProviderOpenAI     UpstreamProvider = "openai"
	ProviderAnthropic  UpstreamProvider = "anthropic"
	ProviderGroq       UpstreamProvider = "groq"
	ProviderDeepSeek   UpstreamProvider = "deepseek"
	ProviderOpenRouter UpstreamProvider = "openrouter"
	ProviderGeneric    UpstreamProvider = "generic"
)

// DetectUpstreamProvider inspects the upstream URL to determine provider ecosystem.
func DetectUpstreamProvider(upstreamURL string) UpstreamProvider {
	trimmed := strings.ToLower(strings.TrimSpace(upstreamURL))
	if trimmed == "" {
		return ProviderGeneric
	}
	switch {
	case strings.Contains(trimmed, "openai.com"):
		return ProviderOpenAI
	case strings.Contains(trimmed, "anthropic.com"):
		return ProviderAnthropic
	case strings.Contains(trimmed, "groq.com"):
		return ProviderGroq
	case strings.Contains(trimmed, "deepseek.com"):
		return ProviderDeepSeek
	case strings.Contains(trimmed, "openrouter.ai"):
		return ProviderOpenRouter
	default:
		return ProviderGeneric
	}
}

// ResolveSmartModel dynamically maps requested model to an optimal economy model
// within the target upstream provider's catalog for low-complexity queries.
func ResolveSmartModel(upstreamURL string, originalModel string, tier ComplexityTier) (routedModel string, changed bool, reason string) {
	if tier == TierFlagship {
		return originalModel, false, fmt.Sprintf("complex prompt retained on flagship %s", originalModel)
	}

	provider := DetectUpstreamProvider(upstreamURL)
	origLower := strings.ToLower(strings.TrimSpace(originalModel))

	switch provider {
	case ProviderOpenAI:
		return resolveOpenAI(origLower, originalModel)
	case ProviderAnthropic:
		return resolveAnthropic(origLower, originalModel)
	case ProviderGroq:
		return resolveGroq(origLower, originalModel)
	case ProviderDeepSeek:
		return resolveDeepSeek(origLower, originalModel)
	case ProviderOpenRouter:
		return resolveOpenRouter(origLower, originalModel)
	default:
		return resolveGeneric(origLower, originalModel)
	}
}

func resolveOpenAI(origLower, originalModel string) (string, bool, string) {
	if origLower == "gpt-4o-mini" || origLower == "o1-mini" {
		return originalModel, false, "model already optimal economy tier"
	}
	if strings.Contains(origLower, "gpt-4o") && !strings.Contains(origLower, "mini") {
		return "gpt-4o-mini", true, "low-complexity prompt routed to gpt-4o-mini (~94% cost savings)"
	}
	if origLower == "gpt-4" || strings.HasPrefix(origLower, "gpt-4-turbo") || strings.HasPrefix(origLower, "gpt-4-0125") {
		return "gpt-4o-mini", true, "low-complexity prompt routed to gpt-4o-mini (~98% cost savings)"
	}
	if origLower == "o1" || strings.HasPrefix(origLower, "o1-preview") || strings.HasPrefix(origLower, "o1-2024") {
		return "o1-mini", true, "low-complexity prompt routed to o1-mini (~80% cost savings)"
	}
	return originalModel, false, "model already optimal or unmapped"
}

func resolveAnthropic(origLower, originalModel string) (string, bool, string) {
	if strings.Contains(origLower, "haiku") {
		return originalModel, false, "model already optimal economy tier"
	}
	if strings.Contains(origLower, "claude-3-5-sonnet") || strings.Contains(origLower, "claude-3.5-sonnet") || strings.Contains(origLower, "claude-3-opus") {
		return "claude-3-5-haiku", true, "low-complexity prompt routed to claude-3-5-haiku (~80% cost savings)"
	}
	return originalModel, false, "model already optimal or unmapped"
}

func resolveGroq(origLower, originalModel string) (string, bool, string) {
	if strings.Contains(origLower, "8b-instant") || strings.Contains(origLower, "8b") {
		return originalModel, false, "model already optimal economy tier"
	}
	if strings.Contains(origLower, "llama-3.3-70b") || strings.Contains(origLower, "llama-3.1-70b") || strings.Contains(origLower, "llama-3-70b") {
		return "llama-3.1-8b-instant", true, "low-complexity prompt routed to llama-3.1-8b-instant (~87% cost savings)"
	}
	return originalModel, false, "model already optimal or unmapped"
}

func resolveDeepSeek(origLower, originalModel string) (string, bool, string) {
	if strings.Contains(origLower, "chat") {
		return originalModel, false, "model already optimal economy tier"
	}
	if strings.Contains(origLower, "reasoner") || strings.Contains(origLower, "r1") {
		return "deepseek-chat", true, "non-reasoning query routed to deepseek-chat (~75% cost savings)"
	}
	return originalModel, false, "model already optimal or unmapped"
}

func resolveOpenRouter(origLower, originalModel string) (string, bool, string) {
	parts := strings.SplitN(originalModel, "/", 2)
	if len(parts) == 2 {
		prefix := parts[0]
		sub := parts[1]
		subLower := strings.ToLower(sub)

		switch prefix {
		case "openai":
			if subLower == "gpt-4o-mini" || subLower == "o1-mini" {
				return originalModel, false, "model already optimal economy tier"
			}
			if strings.Contains(subLower, "gpt-4o") && !strings.Contains(subLower, "mini") {
				return "openai/gpt-4o-mini", true, "low-complexity prompt routed to openai/gpt-4o-mini (~94% cost savings)"
			}
			if subLower == "gpt-4" || strings.HasPrefix(subLower, "gpt-4-turbo") {
				return "openai/gpt-4o-mini", true, "low-complexity prompt routed to openai/gpt-4o-mini (~98% cost savings)"
			}
			if subLower == "o1" || strings.HasPrefix(subLower, "o1-preview") {
				return "openai/o1-mini", true, "low-complexity prompt routed to openai/o1-mini (~80% cost savings)"
			}
		case "anthropic":
			if strings.Contains(subLower, "haiku") {
				return originalModel, false, "model already optimal economy tier"
			}
			if strings.Contains(subLower, "claude-3.5-sonnet") || strings.Contains(subLower, "claude-3-5-sonnet") || strings.Contains(subLower, "claude-3-opus") {
				target := "anthropic/claude-3.5-haiku"
				return target, true, fmt.Sprintf("low-complexity prompt routed to %s (~80%% cost savings)", target)
			}
		case "meta-llama":
			if strings.Contains(subLower, "8b") {
				return originalModel, false, "model already optimal economy tier"
			}
			if strings.Contains(subLower, "llama-3.3-70b") || strings.Contains(subLower, "llama-3.1-70b") {
				routed := strings.Replace(originalModel, "3.3-70b", "3.1-8b", 1)
				routed = strings.Replace(routed, "3.1-70b", "3.1-8b", 1)
				return routed, true, fmt.Sprintf("low-complexity prompt routed to %s (~87%% cost savings)", routed)
			}
		case "deepseek":
			if strings.Contains(subLower, "chat") {
				return originalModel, false, "model already optimal economy tier"
			}
			if strings.Contains(subLower, "reasoner") || strings.Contains(subLower, "r1") {
				return "deepseek/deepseek-chat", true, "non-reasoning query routed to deepseek/deepseek-chat (~75% cost savings)"
			}
		default:
			if routedSub, changed, reason := resolveGeneric(subLower, sub); changed {
				return prefix + "/" + routedSub, true, reason
			}
		}
		return originalModel, false, "model already optimal or unmapped"
	}

	return resolveGeneric(origLower, originalModel)
}

func resolveGeneric(origLower, originalModel string) (string, bool, string) {
	if strings.Contains(origLower, "mini") || strings.Contains(origLower, "haiku") || strings.Contains(origLower, "chat") ||
		strings.Contains(origLower, "-8b") || strings.HasSuffix(origLower, ":8b") {
		return originalModel, false, "model already optimal economy tier"
	}
	if strings.Contains(origLower, "70b") {
		routed := strings.Replace(originalModel, "3.3-70b", "3.1-8b", 1)
		routed = strings.Replace(routed, "70b", "8b", 1)
		routed = strings.Replace(routed, "70B", "8B", 1)
		return routed, true, fmt.Sprintf("parameter reduction routed to %s (~85%% cost savings)", routed)
	}
	if strings.Contains(origLower, "sonnet") {
		routed := strings.Replace(originalModel, "sonnet", "haiku", 1)
		routed = strings.Replace(routed, "Sonnet", "Haiku", 1)
		return routed, true, fmt.Sprintf("parameter reduction routed to %s (~80%% cost savings)", routed)
	}
	if strings.Contains(origLower, "reasoner") {
		routed := strings.Replace(originalModel, "reasoner", "chat", 1)
		routed = strings.Replace(routed, "Reasoner", "Chat", 1)
		return routed, true, fmt.Sprintf("parameter reduction routed to %s (~75%% cost savings)", routed)
	}
	if strings.Contains(origLower, "gpt-4o") && !strings.Contains(origLower, "mini") {
		return "gpt-4o-mini", true, "low-complexity prompt routed to gpt-4o-mini (~94% cost savings)"
	}
	if origLower == "gpt-4" || strings.HasPrefix(origLower, "gpt-4-turbo") {
		return "gpt-4o-mini", true, "low-complexity prompt routed to gpt-4o-mini (~98% cost savings)"
	}
	if origLower == "o1" || strings.HasPrefix(origLower, "o1-preview") {
		return "o1-mini", true, "low-complexity prompt routed to o1-mini (~80% cost savings)"
	}
	return originalModel, false, "model already optimal or unmapped"
}
