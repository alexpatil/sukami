package proxy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultUpstreamURL is the standard fallback completions base endpoint when no upstream flag or custom routing is specified.
const DefaultUpstreamURL = "https://api.openai.com/v1"

// ProviderConfig defines standard upstream provider endpoints.
type ProviderConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

var knownProviders = map[string]ProviderConfig{
	"openrouter": {
		ID:      "openrouter",
		Name:    "OpenRouter (300+ Models)",
		BaseURL: "https://openrouter.ai/api/v1",
	},
	"groq": {
		ID:      "groq",
		Name:    "Groq Cloud (Ultra-Fast Llama)",
		BaseURL: "https://api.groq.com/openai/v1",
	},
	"openai": {
		ID:      "openai",
		Name:    "OpenAI Official",
		BaseURL: "https://api.openai.com/v1",
	},
	"deepseek": {
		ID:      "deepseek",
		Name:    "DeepSeek Official",
		BaseURL: "https://api.deepseek.com",
	},
	"ollama": {
		ID:      "ollama",
		Name:    "Ollama / Local vLLM",
		BaseURL: "http://localhost:11434/v1",
	},
}

// modelProviderHints maps common model name substrings to their provider for auto-routing.
var modelProviderHints = map[string]string{
	"claude":  "openrouter",
	"gemini":  "openrouter",
	"llama":   "groq",
	"mixtral": "groq",
}

// FetchModelsRequest specifies the provider and credentials to query models from.
type FetchModelsRequest struct {
	Provider    string `json:"provider"`
	UpstreamURL string `json:"upstream_url,omitempty"`
	APIKey      string `json:"api_key,omitempty"`
}

// ModelItem represents a model in the returned catalog.
type ModelItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContextLen  int    `json:"context_length,omitempty"`
	Description string `json:"description,omitempty"`
}

// resolveUpstreamTarget dynamically determines the target completions URL.
// Priority:
// 1. Explicit request header: X-Upstream-Url or X-Base-Url
// 2. Explicit provider header: X-Provider (e.g. "openrouter", "deepseek", "groq")
// 3. Model prefix routing (e.g. "openrouter/...")
// 4. Server default upstreamURL
func (h *Handler) resolveUpstreamTarget(r *http.Request, model string) string {
	// 1. Explicit custom URL header
	if custom := r.Header.Get("X-Upstream-Url"); custom != "" {
		return buildCompletionsURL(custom)
	}
	if custom := r.Header.Get("X-Base-Url"); custom != "" {
		return buildCompletionsURL(custom)
	}

	// 2. Explicit provider header
	if providerID := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Provider"))); providerID != "" {
		if p, ok := knownProviders[providerID]; ok {
			return buildCompletionsURL(p.BaseURL)
		}
	}

	// 3. Model prefix detection
	lowerModel := strings.ToLower(model)
	if strings.HasPrefix(lowerModel, "openrouter/") {
		return buildCompletionsURL(knownProviders["openrouter"].BaseURL)
	}
	if strings.HasPrefix(lowerModel, "deepseek/") || strings.HasPrefix(lowerModel, "deepseek-") {
		return buildCompletionsURL(knownProviders["deepseek"].BaseURL)
	}

	// 3b. Model name hints for common models without provider prefix
	for hint, providerID := range modelProviderHints {
		if strings.Contains(lowerModel, hint) {
			if p, ok := knownProviders[providerID]; ok {
				return buildCompletionsURL(p.BaseURL)
			}
		}
	}

	// 4. Default server upstream
	return buildCompletionsURL(h.upstreamURL)
}

func buildCompletionsURL(base string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(trimmed, "/chat/completions") {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/chat/completions"
	}
	return trimmed + "/v1/chat/completions"
}

// HandleFetchProviderModels queries live models from any provider using the user's API key.
func (h *Handler) HandleFetchProviderModels(w http.ResponseWriter, r *http.Request) {
	var req FetchModelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	baseURL := req.UpstreamURL
	if baseURL == "" {
		if p, ok := knownProviders[strings.ToLower(req.Provider)]; ok {
			baseURL = p.BaseURL
		} else {
			baseURL = h.upstreamURL
		}
	}

	targetURL := strings.TrimRight(baseURL, "/")
	if !strings.HasSuffix(targetURL, "/models") {
		targetURL += "/models"
	}

	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to construct request: %v", err))
		return
	}

	if req.APIKey != "" {
		cleanKey := strings.TrimPrefix(strings.TrimSpace(req.APIKey), "Bearer ")
		httpReq.Header.Set("Authorization", "Bearer "+cleanKey)
	}
	// Recommended OpenRouter headers
	httpReq.Header.Set("HTTP-Referer", "https://sukami.ai")
	httpReq.Header.Set("X-Title", "Sukami AI Gateway")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("Failed to connect to provider at %s: %v", baseURL, err))
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode >= 400 {
		msg := string(bodyBytes)
		if msg == "" {
			msg = resp.Status
		}
		writeError(w, resp.StatusCode, fmt.Sprintf("Provider returned %d: %s", resp.StatusCode, msg))
		return
	}

	var upstreamResp struct {
		Data   []ModelItem `json:"data"`
		Models []ModelItem `json:"models"` // Ollama returns "models"
	}
	_ = json.Unmarshal(bodyBytes, &upstreamResp)

	results := upstreamResp.Data
	if len(results) == 0 && len(upstreamResp.Models) > 0 {
		results = upstreamResp.Models
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"provider": req.Provider,
		"base_url": baseURL,
		"count":    len(results),
		"models":   results,
	})
}
