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

func getEffectiveAuth(reqAuth string) string {
	clean := strings.TrimSpace(strings.TrimPrefix(reqAuth, "Bearer "))
	if strings.HasPrefix(clean, "sk-sukami-") {
		envKey := os.Getenv("OPENAI_API_KEY")
		if envKey == "" {
			envKey = os.Getenv("GROQ_API_KEY")
		}
		if envKey != "" {
			return "Bearer " + envKey
		}
		return ""
	}
	if reqAuth != "" {
		return reqAuth
	}
	envKey := os.Getenv("OPENAI_API_KEY")
	if envKey == "" {
		envKey = os.Getenv("GROQ_API_KEY")
	}
	if envKey != "" {
		return "Bearer " + envKey
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
