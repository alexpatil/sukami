package optimizer

import (
	"testing"
)

func TestDetectUpstreamProvider(t *testing.T) {
	tests := []struct {
		url      string
		expected UpstreamProvider
	}{
		{"https://api.openai.com/v1", ProviderOpenAI},
		{"http://api.openai.com/v1/chat/completions", ProviderOpenAI},
		{"https://api.anthropic.com/v1/messages", ProviderAnthropic},
		{"api.anthropic.com", ProviderAnthropic},
		{"https://api.groq.com/openai/v1", ProviderGroq},
		{"groq.com/openai", ProviderGroq},
		{"https://api.deepseek.com/v1", ProviderDeepSeek},
		{"https://openrouter.ai/api/v1", ProviderOpenRouter},
		{"http://localhost:11434/v1", ProviderGeneric},
		{"http://127.0.0.1:8000/v1", ProviderGeneric},
		{"", ProviderGeneric},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			got := DetectUpstreamProvider(tc.url)
			if got != tc.expected {
				t.Errorf("DetectUpstreamProvider(%q) = %s; want %s", tc.url, got, tc.expected)
			}
		})
	}
}

func TestResolveSmartModel_AllUpstreams(t *testing.T) {
	tests := []struct {
		name        string
		upstreamURL string
		model       string
		tier        ComplexityTier
		wantModel   string
		wantChanged bool
	}{
		// OpenAI Upstream
		{"OpenAI gpt-4o", "https://api.openai.com/v1", "gpt-4o", TierEconomy, "gpt-4o-mini", true},
		{"OpenAI gpt-4o dated", "https://api.openai.com/v1", "gpt-4o-2024-08-06", TierEconomy, "gpt-4o-mini", true},
		{"OpenAI gpt-4", "https://api.openai.com/v1", "gpt-4", TierEconomy, "gpt-4o-mini", true},
		{"OpenAI gpt-4-turbo", "https://api.openai.com/v1", "gpt-4-turbo", TierEconomy, "gpt-4o-mini", true},
		{"OpenAI o1", "https://api.openai.com/v1", "o1", TierEconomy, "o1-mini", true},
		{"OpenAI o1-preview", "https://api.openai.com/v1", "o1-preview", TierEconomy, "o1-mini", true},
		{"OpenAI already economy gpt-4o-mini", "https://api.openai.com/v1", "gpt-4o-mini", TierEconomy, "gpt-4o-mini", false},
		{"OpenAI already economy o1-mini", "https://api.openai.com/v1", "o1-mini", TierEconomy, "o1-mini", false},
		{"OpenAI unmapped", "https://api.openai.com/v1", "text-embedding-3", TierEconomy, "text-embedding-3", false},

		// Anthropic Upstream
		{"Anthropic claude-3-5-sonnet", "https://api.anthropic.com", "claude-3-5-sonnet-20241022", TierEconomy, "claude-3-5-haiku", true},
		{"Anthropic claude-3.5-sonnet", "https://api.anthropic.com", "claude-3.5-sonnet", TierEconomy, "claude-3-5-haiku", true},
		{"Anthropic claude-3-opus", "https://api.anthropic.com", "claude-3-opus-20240229", TierEconomy, "claude-3-5-haiku", true},
		{"Anthropic already economy haiku", "https://api.anthropic.com", "claude-3-5-haiku", TierEconomy, "claude-3-5-haiku", false},
		{"Anthropic unmapped", "https://api.anthropic.com", "claude-2.1", TierEconomy, "claude-2.1", false},

		// Groq Upstream
		{"Groq llama-3.3-70b", "https://api.groq.com/openai/v1", "llama-3.3-70b-versatile", TierEconomy, "llama-3.1-8b-instant", true},
		{"Groq llama-3.1-70b", "https://api.groq.com/openai/v1", "llama-3.1-70b-versatile", TierEconomy, "llama-3.1-8b-instant", true},
		{"Groq already economy 8b", "https://api.groq.com/openai/v1", "llama-3.1-8b-instant", TierEconomy, "llama-3.1-8b-instant", false},
		{"Groq unmapped", "https://api.groq.com/openai/v1", "whisper-large-v3", TierEconomy, "whisper-large-v3", false},

		// DeepSeek Upstream
		{"DeepSeek reasoner", "https://api.deepseek.com", "deepseek-reasoner", TierEconomy, "deepseek-chat", true},
		{"DeepSeek r1", "https://api.deepseek.com", "deepseek-r1", TierEconomy, "deepseek-chat", true},
		{"DeepSeek already economy chat", "https://api.deepseek.com", "deepseek-chat", TierEconomy, "deepseek-chat", false},
		{"DeepSeek unmapped", "https://api.deepseek.com", "deepseek-coder", TierEconomy, "deepseek-coder", false},

		// OpenRouter Upstream
		{"OpenRouter openai/gpt-4o", "https://openrouter.ai/api/v1", "openai/gpt-4o", TierEconomy, "openai/gpt-4o-mini", true},
		{"OpenRouter openai/gpt-4-turbo", "https://openrouter.ai/api/v1", "openai/gpt-4-turbo", TierEconomy, "openai/gpt-4o-mini", true},
		{"OpenRouter openai/o1", "https://openrouter.ai/api/v1", "openai/o1", TierEconomy, "openai/o1-mini", true},
		{"OpenRouter openai/gpt-4o-mini economy", "https://openrouter.ai/api/v1", "openai/gpt-4o-mini", TierEconomy, "openai/gpt-4o-mini", false},
		{"OpenRouter anthropic/claude-3.5-sonnet", "https://openrouter.ai/api/v1", "anthropic/claude-3.5-sonnet", TierEconomy, "anthropic/claude-3.5-haiku", true},
		{"OpenRouter anthropic/claude-3-5-sonnet", "https://openrouter.ai/api/v1", "anthropic/claude-3-5-sonnet", TierEconomy, "anthropic/claude-3.5-haiku", true},
		{"OpenRouter anthropic/claude-3-opus", "https://openrouter.ai/api/v1", "anthropic/claude-3-opus", TierEconomy, "anthropic/claude-3.5-haiku", true},
		{"OpenRouter anthropic/haiku economy", "https://openrouter.ai/api/v1", "anthropic/claude-3.5-haiku", TierEconomy, "anthropic/claude-3.5-haiku", false},
		{"OpenRouter meta-llama/llama-3.3-70b-instruct", "https://openrouter.ai/api/v1", "meta-llama/llama-3.3-70b-instruct", TierEconomy, "meta-llama/llama-3.1-8b-instruct", true},
		{"OpenRouter meta-llama/llama-3.3-70b", "https://openrouter.ai/api/v1", "meta-llama/llama-3.3-70b", TierEconomy, "meta-llama/llama-3.1-8b", true},
		{"OpenRouter meta-llama/llama-3.1-70b", "https://openrouter.ai/api/v1", "meta-llama/llama-3.1-70b", TierEconomy, "meta-llama/llama-3.1-8b", true},
		{"OpenRouter meta-llama/8b economy", "https://openrouter.ai/api/v1", "meta-llama/llama-3.1-8b", TierEconomy, "meta-llama/llama-3.1-8b", false},
		{"OpenRouter meta-llama unmapped", "https://openrouter.ai/api/v1", "meta-llama/llama-guard", TierEconomy, "meta-llama/llama-guard", false},
		{"OpenRouter deepseek/reasoner", "https://openrouter.ai/api/v1", "deepseek/deepseek-reasoner", TierEconomy, "deepseek/deepseek-chat", true},
		{"OpenRouter deepseek/chat economy", "https://openrouter.ai/api/v1", "deepseek/deepseek-chat", TierEconomy, "deepseek/deepseek-chat", false},
		{"OpenRouter default prefix generic reduction", "https://openrouter.ai/api/v1", "mistralai/mistral-70b", TierEconomy, "mistralai/mistral-8b", true},
		{"OpenRouter default prefix unmapped", "https://openrouter.ai/api/v1", "custom/small-model", TierEconomy, "custom/small-model", false},
		{"OpenRouter unnamespaced fallback", "https://openrouter.ai/api/v1", "gpt-4o", TierEconomy, "gpt-4o-mini", true},

		// Generic / Local / Unknown Upstream
		{"Generic 70b to 8b", "http://localhost:11434", "llama3:70b", TierEconomy, "llama3:8b", true},
		{"Generic llama-3.3-70b", "http://localhost:11434", "llama-3.3-70b", TierEconomy, "llama-3.1-8b", true},
		{"Generic sonnet to haiku", "http://localhost:8000", "claude-3-5-sonnet", TierEconomy, "claude-3-5-haiku", true},
		{"Generic reasoner to chat", "http://localhost:8000", "deepseek-reasoner", TierEconomy, "deepseek-chat", true},
		{"Generic gpt-4o", "http://localhost:8000", "gpt-4o", TierEconomy, "gpt-4o-mini", true},
		{"Generic gpt-4", "http://localhost:8000", "gpt-4", TierEconomy, "gpt-4o-mini", true},
		{"Generic o1", "http://localhost:8000", "o1", TierEconomy, "o1-mini", true},
		{"Generic already economy 8b", "http://localhost:11434", "llama3:8b", TierEconomy, "llama3:8b", false},
		{"Generic already economy mini", "http://localhost:11434", "gpt-4o-mini", TierEconomy, "gpt-4o-mini", false},
		{"Generic unmapped", "http://localhost:11434", "custom-model-v1", TierEconomy, "custom-model-v1", false},

		// Flagship Tier Preservations (MUST NEVER DOWNGRADE)
		{"Flagship OpenAI", "https://api.openai.com/v1", "gpt-4o", TierFlagship, "gpt-4o", false},
		{"Flagship Anthropic", "https://api.anthropic.com", "claude-3-5-sonnet-20241022", TierFlagship, "claude-3-5-sonnet-20241022", false},
		{"Flagship Groq", "https://api.groq.com", "llama-3.3-70b-versatile", TierFlagship, "llama-3.3-70b-versatile", false},
		{"Flagship DeepSeek", "https://api.deepseek.com", "deepseek-reasoner", TierFlagship, "deepseek-reasoner", false},
		{"Flagship OpenRouter", "https://openrouter.ai/api/v1", "openai/gpt-4o", TierFlagship, "openai/gpt-4o", false},
		{"Flagship Generic", "http://localhost:11434", "llama3:70b", TierFlagship, "llama3:70b", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotModel, gotChanged, reason := ResolveSmartModel(tc.upstreamURL, tc.model, tc.tier)
			if gotModel != tc.wantModel {
				t.Errorf("model = %s; want %s (reason: %s)", gotModel, tc.wantModel, reason)
			}
			if gotChanged != tc.wantChanged {
				t.Errorf("changed = %v; want %v (reason: %s)", gotChanged, tc.wantChanged, reason)
			}
		})
	}
}
