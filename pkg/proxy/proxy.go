package proxy

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"time"
)

// Cache defines the contract for caching responses (SOLID: Interface Segregation).
type Cache interface {
	Get(key string) ([]byte, bool)
	GetSemantic(model, prompt string, threshold float64) ([]byte, bool, float64)
	Set(key string, response []byte, model string)
	SetSemantic(key, prompt string, response []byte, model string)
}

// RequestTracker defines the contract for operational metrics (SOLID: Dependency Inversion).
type RequestTracker interface {
	RecordRequest(record metrics.RequestRecord)
	GetSummary() metrics.Summary
}

// Handler manages proxy routing and token optimization.
type Handler struct {
	cache       Cache
	tracker     RequestTracker
	authStore   auth.KeyStore
	limiter     *auth.RateLimiter
	upstreamURL string
	httpClient  *http.Client
}

// NewHandler constructs a thread-safe proxy handler.
func NewHandler(cache Cache, tracker RequestTracker, authStore auth.KeyStore, limiter *auth.RateLimiter, upstreamURL string) *Handler {
	if upstreamURL == "" {
		upstreamURL = "https://api.openai.com"
	}
	return &Handler{
		cache:       cache,
		tracker:     tracker,
		authStore:   authStore,
		limiter:     limiter,
		upstreamURL: upstreamURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// writeJSON sends a JSON response with status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError sends an OpenAI-compatible JSON error.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"message": message,
		},
	})
}

// truncate shortens a string for preview logging.
func truncate(s string, maxLen int) string {
	clean := strings.ReplaceAll(s, "\n", " ")
	if len(clean) <= maxLen {
		return clean
	}
	return clean[:maxLen] + "..."
}

// resolveSukamiKey extracts the Sukami tenant key from X-Sukami-Key or Authorization header.
func resolveSukamiKey(r *http.Request) string {
	if sukamiHeader := r.Header.Get("X-Sukami-Key"); sukamiHeader != "" {
		return strings.TrimSpace(strings.TrimPrefix(sukamiHeader, "Bearer "))
	}
	rawAuth := r.Header.Get("Authorization")
	clean := strings.TrimSpace(strings.TrimPrefix(rawAuth, "Bearer "))
	if strings.HasPrefix(clean, "sk-sukami-") {
		return clean
	}
	return ""
}

// resolveUpstreamAuth extracts the upstream provider authorization header.
// It checks:
// 1. Explicit upstream header: X-Upstream-Key, X-Provider-Key, or X-OpenAI-Api-Key.
// 2. Direct provider key in Authorization (when not starting with sk-sukami-).
// 3. Server environment fallback: OPENAI_API_KEY, GROQ_API_KEY, ANTHROPIC_API_KEY, DEEPSEEK_API_KEY, GEMINI_API_KEY.
func resolveUpstreamAuth(r *http.Request) string {
	// 1. Explicit upstream provider headers
	for _, header := range []string{"X-Upstream-Key", "X-Provider-Key", "X-OpenAI-Api-Key"} {
		if val := strings.TrimSpace(r.Header.Get(header)); val != "" {
			return "Bearer " + strings.TrimPrefix(val, "Bearer ")
		}
	}

	// 2. Direct provider key in Authorization header (when client doesn't use sk-sukami- in Authorization)
	reqAuth := strings.TrimSpace(r.Header.Get("Authorization"))
	cleanAuth := strings.TrimPrefix(reqAuth, "Bearer ")
	if cleanAuth != "" && !strings.HasPrefix(cleanAuth, "sk-sukami-") {
		return "Bearer " + cleanAuth
	}

	// 3. Server environment fallback
	for _, envVar := range []string{"OPENAI_API_KEY", "GROQ_API_KEY", "ANTHROPIC_API_KEY", "DEEPSEEK_API_KEY", "GEMINI_API_KEY"} {
		if val := os.Getenv(envVar); val != "" {
			return "Bearer " + val
		}
	}
	return ""
}

func extractUserPrompt(messages []models.ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	if len(messages) > 0 {
		return messages[0].Content
	}
	return ""
}

func buildPromptFingerprint(messages []models.ChatMessage) string {
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(m.Role)
		sb.WriteString(":")
		sb.WriteString(m.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}
