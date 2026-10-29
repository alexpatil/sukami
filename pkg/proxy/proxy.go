package proxy

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
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
	FormatPrometheus() string
}

// Handler manages proxy routing and token optimization.
type Handler struct {
	cache       Cache
	tracker     RequestTracker
	authStore   auth.KeyStore
	limiter     *auth.RateLimiter
	upstreamURL string
	gatewayKey  string
	httpClient  *http.Client
}

// NewHandler constructs a thread-safe proxy handler.
func NewHandler(cache Cache, tracker RequestTracker, authStore auth.KeyStore, limiter *auth.RateLimiter, upstreamURL string) *Handler {
	if upstreamURL == "" {
		upstreamURL = DefaultUpstreamURL
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &Handler{
		cache:       cache,
		tracker:     tracker,
		authStore:   authStore,
		limiter:     limiter,
		upstreamURL: upstreamURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   120 * time.Second,
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

// generateReqID creates a cryptographically unique request identifier.
func generateReqID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "req_" + hex.EncodeToString(b)
}

func (h *Handler) recordMissMetric(id, model, prompt string, comp optimizer.CompressionResult, start time.Time, status string, respBytes []byte) {
	record := metrics.RequestRecord{
		ID:                id,
		Timestamp:         time.Now(),
		Model:             model,
		PromptPreview:     truncate(prompt, 60),
		OriginalTokens:    comp.OriginalTokens,
		SavedTokens:       comp.SavedTokens,
		InputSavedTokens:  comp.SavedTokens,
		OutputSavedTokens: 0,
		BilledTokens:      comp.OptimizedTokens,
		SavingsPercent:    comp.SavingsPercent,
		CacheHit:          false,
		LatencyMs:         time.Since(start).Milliseconds(),
		Status:            status,
	}

	// Extract authoritative ground-truth tokens directly from upstream provider response
	if len(respBytes) > 0 {
		var u struct {
			Usage struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
				TotalTokens      int `json:"total_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(respBytes, &u) == nil && u.Usage.PromptTokens > 0 {
			totalBilled := u.Usage.TotalTokens
			if totalBilled == 0 {
				totalBilled = u.Usage.PromptTokens + u.Usage.CompletionTokens
			}
			record.BilledTokens = totalBilled
			record.OriginalTokens = totalBilled + comp.SavedTokens
			if record.OriginalTokens > 0 {
				record.SavingsPercent = (float64(record.SavedTokens) / float64(record.OriginalTokens)) * 100.0
			}
		}
	}
	costUSD, _ := metrics.CalculateSavingsUSD(model, record.InputSavedTokens, record.OutputSavedTokens)
	record.CostSavedUSD = costUSD
	h.tracker.RecordRequest(record)
}
